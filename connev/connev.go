package connev

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/oklog/ulid"
	"go.mongodb.org/mongo-driver/bson"
)

// Check will assert no error occoured and panic otherwise
func Check(e error) {
	if e != nil {
		panic(e.Error())
	}
}

// Event represents connection and data events
type Event struct {
	T    string
	UID  string
	Data []byte
}

// SockUIDHolder is a holder for socket uids
type SockUIDHolder struct {
	socket2uid map[net.Conn]string
	uid2socket map[string]net.Conn
	suidMutex  sync.RWMutex
}

// NewSockUIDHolder creates a new SockUIDHolder
func NewSockUIDHolder() *SockUIDHolder {
	var res SockUIDHolder
	res.socket2uid = make(map[net.Conn]string)
	res.uid2socket = make(map[string]net.Conn)
	return &res
}

// GetConnFromUID returns a connection from the given uid
func (holder *SockUIDHolder) GetConnFromUID(uid string) net.Conn {
	holder.suidMutex.RLock()
	res := holder.uid2socket[uid]
	holder.suidMutex.RUnlock()
	return res
}

// GetConnUID returns a uid from the given connection
func (holder *SockUIDHolder) GetConnUID(conn net.Conn) string {
	holder.suidMutex.RLock()
	res := holder.socket2uid[conn]
	holder.suidMutex.RUnlock()
	return res
}

// SetConnUID adds a connection with given external uid to the list
func (holder *SockUIDHolder) SetConnUID(conn net.Conn, uid string) {
	holder.suidMutex.Lock()
	holder.socket2uid[conn] = uid
	holder.uid2socket[uid] = conn
	holder.suidMutex.Unlock()
}

// CreateConnUID returns a uid from the given connection adding it to the holder if necessary
func (holder *SockUIDHolder) CreateConnUID(conn net.Conn) string {
	holder.suidMutex.RLock()
	uid, ok := holder.socket2uid[conn]
	holder.suidMutex.RUnlock()
	if !ok {
		t := time.Now().UTC()
		entropy := rand.New(rand.NewSource(t.UnixNano()))
		id := ulid.MustNew(ulid.Timestamp(t), entropy)
		uid = fmt.Sprintf("%s-{%s}", conn.RemoteAddr(), id)
		holder.SetConnUID(conn, uid)
	}
	return uid
}

// RemoveConnFromUID removes a connection form the holder returning it (usually to be closed)
func (holder *SockUIDHolder) RemoveConnFromUID(uid string) net.Conn {
	holder.suidMutex.RLock()
	conn, ok := holder.uid2socket[uid]
	holder.suidMutex.RUnlock()
	if ok {
		holder.suidMutex.Lock()
		delete(holder.uid2socket, uid)
		delete(holder.socket2uid, conn)
		holder.suidMutex.Unlock()
	}
	return conn
}

// RemoveConnUID removes a connection form the holder returning its uid (remember to close it afterwards)
func (holder *SockUIDHolder) RemoveConnUID(conn net.Conn) string {
	holder.suidMutex.Lock()
	uid, ok := holder.socket2uid[conn]
	if ok {
		conn.Close()
		delete(holder.uid2socket, uid)
		delete(holder.socket2uid, conn)
	}
	holder.suidMutex.Unlock()
	return uid
}

// NewEventConnect creates a new event of type Connect
func (holder *SockUIDHolder) NewEventConnect(conn net.Conn) Event {
	var e Event
	e.T = "Connect"
	e.UID = holder.CreateConnUID(conn)
	e.Data = nil
	return e
}

// NewEventDisconnect creates a new event of type Disconnect
func (holder *SockUIDHolder) NewEventDisconnect(conn net.Conn) Event {
	var e Event
	e.T = "Disconnect"
	e.UID = holder.GetConnUID(conn)
	e.Data = nil
	return e
}

// NewEventData creates a new event of type Data
func (holder *SockUIDHolder) NewEventData(conn net.Conn, data []byte) Event {
	var e Event
	e.T = "Data"
	e.UID = holder.GetConnUID(conn)
	e.Data = make([]byte, len(data))
	copy(e.Data, data)
	return e
}

// NewEventExit creates a new event of type Exit
func NewEventExit() Event {
	var e Event
	e.T = "Exit"
	e.UID = ""
	e.Data = nil
	return e
}

// NewEventPing creates a new event of type Ping
func NewEventPing() Event {
	var e Event
	e.T = "Ping"
	e.UID = ""
	e.Data = nil
	return e
}

// ReadEvents is a goroutine to push all events from this connection into ch
func readEvents(reader io.Reader, ch chan<- Event) {
	defer close(ch)
	decomp, err := gzip.NewReader(reader)
	if err != nil {
		log.Println("Invalid connection header")
		return // Should also close?
	}
	defer decomp.Close()

	var e Event
	for {
		data, err := bson.NewFromIOReader(decomp)
		if err != nil {
			break // Should log
		}
		e.T = data.Lookup("t").StringValue()
		e.UID = data.Lookup("uid").StringValue()
		_, e.Data, _ = data.Lookup("data").BinaryOK()
		ch <- e
		if e.T == "Exit" {
			break // Normal exit
		}
	}
}

// WriteEvents is a goroutine to push all events from ch into the writer
func writeEvents(writer io.Writer, ch <-chan Event) {
	comp := gzip.NewWriter(writer)
	defer comp.Close()

	for e := range ch {
		bdata, err := bson.Marshal(e)
		Check(err)
		_, err = comp.Write(bdata)
		if err != nil {
			log.Println("Cannot write on event stream")
			return // Should also close?
		}
		err = comp.Flush()
		if err != nil {
			log.Println("Cannot flush event stream")
			return // Should also close?
		}
	}
	// Channel got closed, send Exit?
	bdata, err := bson.Marshal(NewEventExit())
	Check(err)
	_, err = comp.Write(bdata)
	Check(err)
}

// EventPipe structure
type EventPipe struct {
	Input  chan<- Event
	Output <-chan Event
}

// NewEventPipe creates a EventPipe pair
func NewEventPipe() (EventPipe, EventPipe) {
	var p1, p2 EventPipe
	ch1 := make(chan Event)
	p1.Input = ch1
	p2.Output = ch1
	ch2 := make(chan Event)
	p1.Output = ch2
	p2.Input = ch2
	return p1, p2
}

// NewEventHandler create an EventPipe linked to the read write stream provvided
func NewEventHandler(com io.ReadWriter) EventPipe {
	p1, p2 := NewEventPipe()
	go writeEvents(com, p2.Output)
	go readEvents(com, p2.Input)
	return p1
}
