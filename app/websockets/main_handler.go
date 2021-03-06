package websockets

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leavengood/websocket"
)

var stepNumber uint64

func handler(rawConn *websocket.Conn) {
	defer catchPanic()
	defer rawConn.Close()

	step := atomic.AddUint64(&stepNumber, 1)

	// 5 is a security margin in case
	if step == (1<<10 - 5) {
		atomic.StoreUint64(&stepNumber, 0)
	}

	c := &conn{
		rawConn,
		sync.Mutex{},
		step | uint64(time.Now().UnixNano()<<10),
	}

	c.WriteJSON(TypeConnected, nil)

	defer cleanup(c.ID)

	for {
		var i incomingMessage
		err := c.Conn.ReadJSON(&i)
		if _, ok := err.(*websocket.CloseError); ok {
			return
		}
		if err != nil {
			c.WriteJSON(TypeInvalidMessage, err.Error())
			continue
		}
		f, ok := messageHandler[i.Type]
		if !ok {
			c.WriteJSON(TypeInvalidMessage, "invalid message type")
			continue
		}
		f(c, i)
	}
}

type conn struct {
	Conn *websocket.Conn
	Mtx  sync.Mutex
	ID   uint64
}

func (c *conn) WriteJSON(t string, data interface{}) error {
	c.Mtx.Lock()
	err := c.Conn.WriteJSON(newMessage(t, data))
	c.Mtx.Unlock()
	return err
}

var messageHandler = map[string]func(c *conn, message incomingMessage){
	TypeSubscribeScores: SubscribeScores,
}

// Server Message Types
const (
	TypeConnected          = "connected"
	TypeInvalidMessage     = "invalid_message_type"
	TypeSubscribedToScores = "subscribed_to_scores"
	TypeNewScore           = "new_score"
)

// Client Message Types
const (
	TypeSubscribeScores = "subscribe_scores"
)

// Message is the wrapped information for a message sent to the client.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

func newMessage(t string, data interface{}) Message {
	return Message{
		Type: t,
		Data: data,
	}
}

type incomingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
