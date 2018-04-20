package gosocketio

import (
	"encoding/json"
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
		//todo 读取二进制数据
		f, ok := m.findMethod(msg.Method)
		if !ok {
			fmt.Println("MessageTypeEmit findMethod err:",msg.Method)
			return
		}

		//函数不存在参数直接调用
		if !f.ArgsPresent {
			f.callFunc(c, &struct{}{})
			return
		}

		//解析参数
		data := f.getArgs()
		err := json.Unmarshal([]byte(msg.Args), &data)
		if err != nil {
			//fmt.Println("json.Unmarshal",msg.Method, err)
			return
		}

		f.callFunc(c, data)

	case protocol.MessageTypeAckRequest:
		f, ok := m.findMethod(msg.Method)
		if !ok {
			fmt.Println("MessageTypeAckRequest findMethod err:",msg.Method)
			return
		}
		f.callFunc(c, &msg.Data)
		return
		//var result []reflect.Value
		//
		//if f.ArgsPresent {
		//	//data type should be defined for unmarshall
		//	data := f.getArgs()
		//	err := json.Unmarshal([]byte(msg.Args), &data)
		//	if err != nil {
		//		return
		//	}
		//
		//	result = f.callFunc(c, data)
		//} else {
		//	result = f.callFunc(c, &struct{}{})
		//}
		//
		//ack := &protocol.Message{
		//	Type:  protocol.MessageTypeAckResponse,
		//	AckId: msg.AckId,
		//}
		//send(ack, c, result[0].Interface())

	case protocol.MessageTypeAckResponse:
		waiter, err := c.ack.getWaiter(msg.AckId)
		if err == nil {
			waiter <- msg.Args
		}
	}
}
