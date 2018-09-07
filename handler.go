package gosocketio

import (
	"sync"
	"github.com/guhan121/golang-socketio/protocol"

	"fmt"
)

const (
	OnConnection    = "connection"
	OnDisconnection = "disconnection"
	OnError         = "error"
)

/**
System handler function for internal event processing
*/
type systemHandler func(c *Channel)

/**
Contains maps of message processing functions
*/
type methods struct {
	messageHandlers     map[string]*caller
	messageHandlersLock sync.Mutex

	onConnection    systemHandler
	onDisconnection systemHandler
}

/**
create messageHandlers map
*/
func (m *methods) initMethods() {
	m.messageHandlers = make(map[string]*caller)
}

/**
Add message processing function, and bind it to given method
*/
func (m *methods) On(method string, f interface{}) error {
	c, err := newCaller(f)
	if err != nil {
		return err
	}

	m.messageHandlersLock.Lock()
	defer m.messageHandlersLock.Unlock()
	m.messageHandlers[method] = c

	return nil
}

/**
Find message processing function associated with given method
*/
func (m *methods) findMethod(method string) (*caller, bool) {
	m.messageHandlersLock.Lock()
	defer m.messageHandlersLock.Unlock()
	f, ok := m.messageHandlers[method]
	return f, ok
}

func (m *methods) callLoopEvent(c *Channel, event string) {
	if m.onConnection != nil && event == OnConnection {
		m.onConnection(c)
	}
	if m.onDisconnection != nil && event == OnDisconnection {
		m.onDisconnection(c)
	}

	f, ok := m.findMethod(event)
	if !ok {
		return
	}

	f.callFunc(c, &struct{}{})
}

func (m *methods) processIncomingMessage(c *Channel, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MessageTypeEmit:
		f, ok := m.findMethod(msg.Method)
		if !ok {
			fmt.Println("MessageTypeEmit findMethod err:", msg.Method)
			return
		}

		//函数不存在参数直接调用
		if !f.ArgsPresent {
			f.callFunc(c, &struct{}{})
			return
		}

		//fmt.Println("MessageTypeEmit")
		msg.Data = []byte(msg.Args) //MessageTypeEmit 没有byte数据，直接使用json字符串
		f.callFunc(c, &msg.Data)

	case protocol.MessageTypeAckRequest:
		f, ok := m.findMethod(msg.Method)
		if !ok {
			fmt.Println("MessageTypeAckRequest findMethod err:", msg.Method)
			return
		}
		//fmt.Println("MessageTypeAckRequest", "---->", msg.Method, msg)
		f.callFunc(c, &msg.Data)
		return

	case protocol.MessageTypeAckResponse:
		waiter, err := c.ack.getWaiter(msg.AckId)
		if err == nil {
			waiter <- msg.Args
		}
	default:

	}
}
