package transport

import (
	"errors"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"time"
	"sync"
)

const (
	upgradeFailed = "Upgrade failed: "

	WsDefaultPingInterval   = 30 * time.Second
	WsDefaultPingTimeout    = 60 * time.Second
	WsDefaultReceiveTimeout = 60 * time.Second
	WsDefaultSendTimeout    = 60 * time.Second
	WsDefaultBufferSize     = 1024 * 32
)

var (
	ErrorBinaryMessage     = errors.New("Binary messages are not supported")
	ErrorBadBuffer         = errors.New("Buffer error")
	ErrorPacketWrong       = errors.New("Wrong packet type error")
	ErrorMethodNotAllowed  = errors.New("Method not allowed")
	ErrorHttpUpgradeFailed = errors.New("Http upgrade failed")
)

type WebsocketConnection struct {
	socket    *websocket.Conn
	transport *WebsocketTransport
	writeLock     *sync.Mutex
}

func (wsc *WebsocketConnection) GetMessage() (message []byte, messageType int, err error) {
	wsc.socket.SetReadDeadline(time.Now().Add(wsc.transport.ReceiveTimeout))
	msgType, reader, err := wsc.socket.NextReader()
	if err != nil {
		//fmt.Println("wsc.socket.NextReader")
		return nil, msgType, err
	}

	//fmt.Println(msgType)

	//support only text messages exchange
	if msgType != websocket.TextMessage {
		//fmt.Println("---> msgType", msgType)
		//		return "", ErrorBinaryMessage
	}

	data, err := ioutil.ReadAll(reader)
	//fmt.Println(data)
	if err != nil {
		//fmt.Println("ErrorBadBuffer")
		return nil, msgType, ErrorBadBuffer
	}
	return data, msgType, nil
}

func (wsc *WebsocketConnection) WriteMessage(message string) error {
	wsc.writeLock.Lock()
	defer wsc.writeLock.Unlock()
	wsc.socket.SetWriteDeadline(time.Now().Add(wsc.transport.SendTimeout))
	writer, err := wsc.socket.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	if _, err := writer.Write([]byte(message)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return nil
}

func (wsc *WebsocketConnection) WriteBytes(bytes []byte) error {
	wsc.writeLock.Lock()
	defer wsc.writeLock.Unlock()
	wsc.socket.SetWriteDeadline(time.Now().Add(wsc.transport.SendTimeout))
	writer, err := wsc.socket.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	if _, err := writer.Write(bytes); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return nil
}
func (wsc *WebsocketConnection) Close() {
	wsc.socket.Close()
}

func (wsc *WebsocketConnection) PingParams() (interval, timeout time.Duration) {
	return wsc.transport.PingInterval, wsc.transport.PingTimeout
}

type WebsocketTransport struct {
	PingInterval   time.Duration
	PingTimeout    time.Duration
	ReceiveTimeout time.Duration
	SendTimeout    time.Duration

	BufferSize int

	RequestHeader http.Header
}

func (wst *WebsocketTransport) Connect(url string) (conn Connection, err error) {
	dialer := websocket.Dialer{}
	socket, _, err := dialer.Dial(url, wst.RequestHeader)
	if err != nil {
		return nil, err
	}

	return &WebsocketConnection{socket, wst,&sync.Mutex{}}, nil
}

func (wst *WebsocketTransport) HandleConnection(
	w http.ResponseWriter, r *http.Request) (conn Connection, err error) {

	if r.Method != "GET" {
		http.Error(w, upgradeFailed+ErrorMethodNotAllowed.Error(), 503)
		return nil, ErrorMethodNotAllowed
	}

	socket, err := websocket.Upgrade(w, r, nil, wst.BufferSize, wst.BufferSize)
	if err != nil {
		http.Error(w, upgradeFailed+err.Error(), 503)
		return nil, ErrorHttpUpgradeFailed
	}

	return &WebsocketConnection{socket, wst,&sync.Mutex{}}, nil
}

/**
Websocket connection do not require any additional processing
*/
func (wst *WebsocketTransport) Serve(w http.ResponseWriter, r *http.Request) {}

/**
Returns websocket connection with default params
*/
func GetDefaultWebsocketTransport() *WebsocketTransport {
	return &WebsocketTransport{
		PingInterval:   WsDefaultPingInterval,
		PingTimeout:    WsDefaultPingTimeout,
		ReceiveTimeout: WsDefaultReceiveTimeout,
		SendTimeout:    WsDefaultSendTimeout,
		BufferSize:     WsDefaultBufferSize,
	}
}

func (wsc *WebsocketConnection) GetSocket()(*websocket.Conn) {
	return wsc.socket
}