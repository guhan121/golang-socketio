package gosocketio

import (
	"encoding/json"
	"errors"
	"log"
	"time"
	"github.com/guhan121/golang-socketio/protocol"
	"strings"
)

var (
	ErrorSendTimeout     = errors.New("Timeout")
	ErrorSocketOverflood = errors.New("Socket overflood")
)

/**
Send message packet to socket
*/
func send(msg *protocol.Message, c *Channel, args interface{}) error {
	//preventing json/encoding "index out of range" panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("socket.io send panic: ", r)
		}
	}()

	if args != nil {
		json, err := json.Marshal(&args)
		if err != nil {
			return err
		}

		msg.Args = string(json)
	}

	command, err := protocol.Encode(msg)
	if err != nil {
		return err
	}

	if len(c.out) == queueBufferSize {
		return ErrorSocketOverflood
	}
	//fmt.Println("write datas",command)
	replace := strings.Replace(command, "\\\"", "\"", -1)
	replace = strings.Replace(replace, "}\"", "}", -1)
	replace = strings.Replace(replace, "\"{", "{", -1)
	err1 := c.conn.WriteMessage(strings.Replace(replace," ","",-1))
	if err1 != nil{
		return err1
	}
	if msg.Data != nil{
		err1 = c.conn.WriteBytes(append([]byte{4}, msg.Data...))
	}
	return err1
}

/**
Create packet based on given data and send it
*/
func (c *Channel) Emit(method string, args interface{}) error {
	msg := &protocol.Message{
		Type:   protocol.MessageTypeEmit,
		Method: method,
	}

	return send(msg, c, args)
}

/**
Create packet based on given data and send it
*/
func (c *Channel) EmitData(method string, data []byte,args interface{}) error {


	msg := &protocol.Message{
		Type:   protocol.MessageTypeEmit,
		Method: method,
	}
	if data!=nil {
		msg.Data = data
		msg.Num = 1
	}
	return send(msg, c, args)
}


/**
Create ack packet based on given data and send it and receive response
*/
func (c *Channel) Ack(method string, args interface{}, timeout time.Duration) (string, error) {
	msg := &protocol.Message{
		Type:   protocol.MessageTypeAckRequest,
		AckId:  c.ack.getNextId(),
		Method: method,
	}

	waiter := make(chan string)
	c.ack.addWaiter(msg.AckId, waiter)

	err := send(msg, c, args)
	if err != nil {
		c.ack.removeWaiter(msg.AckId)
	}

	select {
	case result := <-waiter:
		return result, nil
	case <-time.After(timeout):
		c.ack.removeWaiter(msg.AckId)
		return "", ErrorSendTimeout
	}
}
