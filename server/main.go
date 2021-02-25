package main

import (
	"gorelayer/conf"
	"gorelayer/connev"
	"log"
	"net"
	"time"
)

// Check will assert no error occoured and panic otherwise
var Check = connev.Check

var globalEventPipe connev.EventPipe
var globalHolder connev.SockUIDHolder

func handleConn(conn net.Conn) {
	// Send that I just got connected
	globalEventPipe.Input <- globalHolder.NewEventConnect(conn)
	log.Printf("+%s\n", globalHolder.GetConnUID(conn))
	defer conn.Close()
	defer func() {
		uid := globalHolder.RemoveConnUID(conn)
		log.Printf("-%s\n", uid)
	}()
	defer func() { globalEventPipe.Input <- globalHolder.NewEventDisconnect(conn) }()
	buff := make([]byte, 1024) // Buffer for each incomming call
	for {
		// Should also wait for data to send back or potential disconnects
		nRead, err := conn.Read(buff)
		if err != nil {
			log.Printf("Failed read: %s | %d\n", err.Error(), nRead)
			break // Should Log
		}
		globalEventPipe.Input <- globalHolder.NewEventData(conn, buff[0:nRead])
	}
}

func handleEvents() {
	for {
		log.Println("Connecting to pipe")
		if globalEventPipe.Output != nil {
		EventLoop:
			for e := range globalEventPipe.Output {
				//log.Println(e)
				switch e.T {
				case "Connect":
					// Should NEVER HAPPEN!
					panic("Connect event cannot be dealt with by a server")
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
		log.Println("Pipe changed")
		time.Sleep(5 * time.Second)
	}
}

func handleEventServer(eventServer net.Listener) {
	var lastConn net.Conn = nil
	for {
		conn, err := eventServer.Accept()
		Check(err)
		if lastConn != nil {
			lastConn.Close()
		}
		lastConn = conn
		log.Printf("NewHandler: %s\n", lastConn.RemoteAddr())
		globalEventPipe = connev.NewEventHandler(lastConn)
	}
}

func main() {
	conf, err := conf.ReadServerConf()
	Check(err)

	l, err := net.Listen("tcp", conf.ListenAddr)
	Check(err)
	defer l.Close()

	lEvents, err := net.Listen("tcp", conf.EventAddr)
	Check(err)
	defer lEvents.Close()

	globalHolder = connev.NewSockUIDHolder()
	globalEventPipe.Output = nil
	go handleEventServer(lEvents)
	go handleEvents()

	for {
		conn, err := l.Accept()
		Check(err)
		go handleConn(conn)
	}
}
