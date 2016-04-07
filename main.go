package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"syscall"
	"time"

	// Golint pls dont break balls
	_ "github.com/go-sql-driver/mysql"
	"github.com/osuripple/api/app"
	"github.com/osuripple/api/common"

	"github.com/rcrowley/goagain"
)

func init() {
	log.SetFlags(log.Ltime)
	log.SetPrefix(fmt.Sprintf("%d|", syscall.Getpid()))
}

func main() {
	conf, halt := common.Load()
	if halt {
		return
	}
	db, err := sql.Open(conf.DatabaseType, conf.DSN)
	if err != nil {
		log.Fatal(err)
	}
	engine := app.Start(conf, db)

	// Inherit a net.Listener from our parent process or listen anew.
	l, err := goagain.Listener()
	if nil != err {

		// Listen on a TCP or a UNIX domain socket (TCP here).
		if conf.Unix {
			l, err = net.Listen("unix", conf.ListenTo)
		} else {
			l, err = net.Listen("tcp", conf.ListenTo)
		}
		if nil != err {
			log.Fatalln(err)
		}

		log.Println("LISTENINGU STARTUATO ON", l.Addr())

		// Accept connections in a new goroutine.
		go http.Serve(l, engine)

	} else {

		// Resume accepting connections in a new goroutine.
		log.Println("LISTENINGU RESUMINGU ON", l.Addr())
		go http.Serve(l, engine)

		// Kill the parent, now that the child has started successfully.
		if err := goagain.Kill(); nil != err {
			log.Fatalln(err)
		}

	}

	// Block the main goroutine awaiting signals.
	if _, err := goagain.Wait(l); nil != err {
		log.Fatalln(err)
	}

	// Do whatever's necessary to ensure a graceful exit like waiting for
	// goroutines to terminate or a channel to become closed.
	//
	// In this case, we'll simply stop listening and wait one second.
	if err := l.Close(); nil != err {
		log.Fatalln(err)
	}
	if err := db.Close(); err != nil {
		log.Fatalln(err)
	}
	time.Sleep(time.Second * 1)

}
