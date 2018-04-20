package gosocketio

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
	"github.com/guhan121/golang-socketio/transport"
	"github.com/guhan121/golang-socketio/protocol"
	"github.com/gorilla/websocket"
)

const (
	queueBufferSize = 500
)

var (
	ErrorWrongHeader = errors.New("Wrong header")
)

/**
engine.io header to send or receive
*/
type Header struct {
	Sid          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}

/**
socket.io connection handler

use IsAlive to check that handler is still working
use Dial to connect to websocket
use In and Out channels for message exchange
Close message means channel is closed
ping is automatic
*/
type Channel struct {
	conn transport.Connection

	out    chan byte
	Header Header

	alive     bool
	aliveLock sync.Mutex

	ack ackProcessor

	server        *Server
	ip            string
	requestHeader http.Header
}

/**
create channel, map, and set active
*/
func (c *Channel) initChannel() {
	//TODO: queueBufferSize from constant to server or client variable
	c.out = make(chan byte, queueBufferSize)
	c.ack.resultWaiters = make(map[int](chan string))
	c.alive = true
}

/**
Get id of current socket connection
*/
func (c *Channel) Id() string {
	return c.Header.Sid
}

/**
Checks that Channel is still alive
*/
func (c *Channel) IsAlive() bool {
	c.aliveLock.Lock()
	defer c.aliveLock.Unlock()

	return c.alive
}

/**
Close channel
*/
func closeChannel(c *Channel, m *methods, args ...interface{}) error {
	c.aliveLock.Lock()
	defer c.aliveLock.Unlock()

	if !c.alive {
		//already closed
		return nil
	}

	c.conn.Close()
	c.alive = false

	//clean outloop
	for len(c.out) > 0 {
		<-c.out
	}
	rs := []byte(protocol.CloseMessage)
	for _, r := range rs {
		c.out <- r
	}
	m.callLoopEvent(c, OnDisconnection)

	overfloodedLock.Lock()
	delete(overflooded, c)
	overfloodedLock.Unlock()

	return nil
}

//incoming messages loop, puts incoming messages to In channel
func inLoop(c *Channel, m *methods) error {
	for {
		pkg, messageType, err := c.conn.GetMessage()
		if err != nil {
			//fmt.Println(",,,,,,,",err)
			return closeChannel(c, m, err)
		}
		if messageType != websocket.BinaryMessage {
			msg, err := protocol.Decode(pkg)
			if err != nil {
				closeChannel(c, m, protocol.ErrorWrongPacket)
				return err
			}

			for i := 0; i < msg.Num; i++ {
				pkg1, messageType1, err := c.conn.GetMessage()
				if err != nil || messageType1 != websocket.BinaryMessage {
					//fmt.Println("---------",err)
					return closeChannel(c, m, err)
				}
				//fmt.Println("read %d --- pkg1",i,pkg1)
				msg.Data = append(msg.Data, pkg1...)
			}

			switch msg.Type {
			case protocol.MessageTypeOpen:
				if err := json.Unmarshal([]byte(msg.Source[1:]), &c.Header); err != nil {
					closeChannel(c, m, ErrorWrongHeader)
				}
				if c.server == nil {
					//客户端
					//fmt.Print(c.conn.PingParams())
				}
				m.callLoopEvent(c, OnConnection)
			case protocol.MessageTypePing:

				rs := []byte(protocol.PongMessage)
				for _, r := range rs {
					c.out <- r
				}
			case protocol.MessageTypePong:
			default:
				go m.processIncomingMessage(c, msg)
			}
		} else {
			return errors.New("Binary must read after text!")
		}
	}
	return nil
}

var overflooded map[*Channel]struct{} = make(map[*Channel]struct{})
var overfloodedLock sync.Mutex

func AmountOfOverflooded() int64 {
	overfloodedLock.Lock()
	defer overfloodedLock.Unlock()

	return int64(len(overflooded))
}

/**
outgoing messages loop, sends messages from channel to socket
*/
func outLoop(c *Channel, m *methods) error {
	for {
		outBufferLen := len(c.out)
		if outBufferLen >= queueBufferSize-1 {
			return closeChannel(c, m, ErrorSocketOverflood)
		} else if outBufferLen > int(queueBufferSize/2) {
			overfloodedLock.Lock()
			overflooded[c] = struct{}{}
			overfloodedLock.Unlock()
		} else {
			overfloodedLock.Lock()
			delete(overflooded, c)
			overfloodedLock.Unlock()
		}

		msg := <-c.out
		if string(msg) == protocol.CloseMessage {
			return nil
		}

		err := c.conn.WriteMessage(string(msg))
		if err != nil {
			return closeChannel(c, m, err)
		}
	}
	return nil
}

/**
Pinger sends ping messages for keeping connection alive
*/
func pinger(c *Channel) {
	for {
		interval, _ := c.conn.PingParams()
		time.Sleep(interval)
		if !c.IsAlive() {
			return
		}

		rs := []byte(protocol.PingMessage)
		for _, r := range rs {
			c.out <- r
		}
	}
}
