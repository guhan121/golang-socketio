package protocol

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	//"fmt"
	"fmt"
)

// Type of packet.
type Type byte

const (
	// Connect type
	Connect Type = iota
	// Disconnect type
	Disconnect
	// Event type
	Event
	// Ack type
	Ack  //3
	// Error type
	Error

	// BinaryEvent type
	binaryEvent  //5 - 3 = 2
	// BinaryAck type
	binaryAck  //6
	typeMax
)

const (
	open         = "0"
	CloseMessage = "1"
	PingMessage  = "2"
	PongMessage  = "3"
	msg          = "4"

	ConnectMessage     = "40"
	EventMessage       = "42" //4 means websocket msg, 2 means socket.io msg, not ack
	ackMessage         = "43"
	binaryEventMessage = "45"
	binaryAckMessage   = "46" //6
)

var (
	ErrorWrongMessageType = errors.New("Wrong message type")
	ErrorWrongPacket      = errors.New("Wrong packet")
)

func typeToText(msgType int) (string, error) {
	switch msgType {
	case MessageTypeOpen:
		return open, nil
	case MessageTypeClose:
		return CloseMessage, nil
	case MessageTypePing:
		return PingMessage, nil
	case MessageTypePong:
		return PongMessage, nil
	case MessageTypeEmpty:
		return ConnectMessage, nil
	case MessageTypeEmit:
		return binaryEventMessage, nil
	case MessageTypeAckRequest:
		return EventMessage, nil
	case MessageTypeAckResponse:
		return ackMessage, nil
	}
	return "", ErrorWrongMessageType
}

func Encode(msg *Message) (string, error) {
	result, err := typeToText(msg.Type)
	if err != nil {
		return "", err
	}

	if msg.Type == MessageTypeEmpty || msg.Type == MessageTypePing ||
		msg.Type == MessageTypePong {
		return result, nil
	}

	if msg.Type == MessageTypeAckRequest || msg.Type == MessageTypeAckResponse {
		result += strconv.Itoa(msg.AckId)
	}

	if msg.Type == MessageTypeEmit {
		result += strconv.Itoa(msg.Num)
		result += "-"
	}
	if msg.Type == MessageTypeOpen || msg.Type == MessageTypeClose {
		return result + msg.Args, nil
	}

	if msg.Type == MessageTypeAckResponse {
		return result + "[" + msg.Args + "]", nil
	}

	jsonMethod, err := json.Marshal(&msg.Method)
	if err != nil {
		return "", err
	}

	return result + "[" + string(jsonMethod) + "," + msg.Args + "]", nil
}

func MustEncode(msg *Message) string {
	result, err := Encode(msg)
	if err != nil {
		panic(err)
	}

	return result
}

func getMessageType(databuf []byte) (int, int, error) {
	if len(databuf) == 0 {
		return 0, 0, ErrorWrongMessageType
	}
	data := string(databuf)
	switch data[0:1] {
	case open:
		return MessageTypeOpen, 0, nil
	case CloseMessage:
		return MessageTypeClose, 0, nil
	case PingMessage:
		return MessageTypePing, 0, nil
	case PongMessage:
		return MessageTypePong, 0, nil
	case msg:
		if len(data) == 1 {
			return 0, 0, ErrorWrongMessageType
		}
		switch data[0:2] {
		case ConnectMessage:
			return MessageTypeEmpty, 0, nil
		case EventMessage:
			return MessageTypeEmit, 0, nil
		case binaryEventMessage:
			i := strings.Index(string(data), "-")
			x := string(data[2:i])
			v, _ := strconv.Atoi(x)
			//fmt.Println("num>>",v)
			return MessageTypeAckRequest, v, nil
		case ackMessage:
			return MessageTypeAckResponse, 0, nil
		case binaryAckMessage:
			i := strings.Index(string(data), "-")
			x := string(data[2:i])
			v, _ := strconv.Atoi(x)
			//fmt.Println("num>>>",v)
			return MessageTypeAckResponse, v, nil
		}
	}
	return 0, 0, ErrorWrongMessageType
}

/**
Get ack id of current packet, if present
*/
func getAck(text string) (ackId int, restText string, err error) {
	if len(text) < 4 {
		return 0, "", ErrorWrongPacket
	}
	text = text[2:]

	pos := strings.IndexByte(text, '[')
	if pos == -1 {
		return 0, "", ErrorWrongPacket
	}

	ack, _ := strconv.Atoi(text[0:pos])
	//if err != nil {
	//	return 0, text[pos:], err
	//}

	return ack, text[pos:], nil
}

/**
Get message method of current packet, if present
*/
func getMethod(text string) (method, restText string, err error) {
	//var start, end, rest, countQuote int
	//
	//for i, c := range text {
	//	if c == '"' {
	//		switch countQuote {
	//		case 0:
	//			start = i + 1
	//		case 1:
	//			end = i
	//			rest = i + 1
	//		default:
	//			return "", "", ErrorWrongPacket
	//		}
	//		countQuote++
	//	}
	//	if c == ',' {
	//		if countQuote < 2 {
	//			continue
	//		}
	//		rest = i + 1
	//		break
	//	}
	//}
	//
	//if (end < start) || (rest >= len(text)) {
	//	return "", "", ErrorWrongPacket
	//}
	//
	//return text[start:end], text[rest: len(text)-1], nil
	//
	arr := []string{}
	json.Unmarshal([]byte(text),&arr)
	return arr[0],arr[1],nil
}

func Decode(data []byte) (*Message, error) {
	var err error
	msg := &Message{}
	msg.Source = data

	msg.Type, msg.Num, err = getMessageType(data)
	if err != nil {
		return nil, err
	}

	if msg.Type == MessageTypeOpen {
		msg.Args = string(data)[1:]
		//fmt.Println("MessageTypeOpen",msg.Args)
		return msg, nil
	}

	if msg.Type == MessageTypeClose || msg.Type == MessageTypePing ||
		msg.Type == MessageTypePong || msg.Type == MessageTypeEmpty {
		//fmt.Println("msg.Source  >>> ",msg.Type, string(msg.Source))
		return msg, nil
	}

	ack, rest, err := getAck(string(data))
	msg.AckId = ack
	if msg.Type == MessageTypeAckResponse {
		if err != nil {
			return nil, err
		}
		msg.Args = rest[1: len(rest)-1]
		return msg, nil
	}

	if err != nil {
		msg.Type = MessageTypeEmit
		rest = string(data)[2:]
	}

	msg.Method, msg.Args, err = getMethod(rest)

	if err != nil {
		return nil, err
	}
	//fmt.Println("-->msg.Method",msg.Method,msg.Args,msg.Type)
	return msg, nil
}
