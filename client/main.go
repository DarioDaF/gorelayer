package main

import (
	"crypto/tls"
	"errors"
	"gorelayer/conf"
	"gorelayer/connev"
	"io"
	"log"
	"net"
	"time"
)

// Check will assert no error occoured and panic otherwise
var Check = connev.Check

var globalEventPipe connev.EventPipe
var globalHolder connev.SockUIDHolder

func handleConn(conn net.Conn) {
	defer conn.Close()
	defer func() {
		uid := globalHolder.RemoveConnUID(conn)
		log.Printf("-%s\n", uid)
	}()
	defer func() { globalEventPipe.Input <- globalHolder.NewEventDisconnect(conn) }()
	buff := make([]byte, 1024) // Buffer for each outgoing call
	for {
		// Should also wait for data to send back or potential disconnects?
		nRead, err := conn.Read(buff)
		if nRead > 0 {
			globalEventPipe.Input <- globalHolder.NewEventData(conn, buff[0:nRead])
		}
		if err != nil {
			// This next line needs go >= 1.16 to avoid hacky dumb code
			// related to issue https://github.com/golang/go/issues/4373
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Printf("Failed read: %s | %d\n", err.Error(), nRead)
			}
			break // Should Log
		}
	}
}

func main() {
	cfg, err := conf.ReadClientConf()
	Check(err)

	tlsConf, err := conf.GetTLSConfig("client", "server")
	Check(err)
	connEvents, err := tls.Dial("tcp", cfg.EventAddr, tlsConf)
	Check(err)
	defer connEvents.Close()

	globalHolder = connev.NewSockUIDHolder()
	globalEventPipe = connev.NewEventHandler(connEvents)

	go func() {
		for {
			globalEventPipe.Input <- connev.NewEventPing()
			time.Sleep(40 * time.Second)
		}
	}()

EventLoop:
	for e := range globalEventPipe.Output {
		//log.Println(e)
		switch e.T {
		case "Connect":
			conn, err := net.Dial("tcp", cfg.TargetAddr)
			Check(err)
			globalHolder.SetConnUID(conn, e.UID)
			log.Printf("+%s\n", globalHolder.GetConnUID(conn))
			go handleConn(conn)
		case "Disconnect":
			conn := globalHolder.GetConnFromUID(e.UID)
			if conn != nil {
				conn.Close()
			}
		case "Data":
			conn := globalHolder.GetConnFromUID(e.UID)
			if conn != nil {
				_, err := globalHolder.GetConnFromUID(e.UID).Write(e.Data)
				Check(err)
			}
		case "Exit":
			break EventLoop
		case "Ping":
			// NOOP
		}
	}
}
