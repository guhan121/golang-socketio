package transport

import (
	"net/http"
	"time"
	"github.com/gorilla/websocket"
)

/**
End-point connection for given transport
*/
type Connection interface {
	/**
	Receive one more message, block until received
	*/
	GetMessage() (message []byte,messageType int, err error)

	/**
	Send given message, block until sent
	*/
	WriteMessage(message string) error
	WriteBytes(bytes []byte) error
	/**
	Close current connection
	*/
	Close()

	/**
	Get ping time interval and ping request timeout
	*/
	PingParams() (interval, timeout time.Duration)

	GetSocket()(*websocket.Conn)
}

/**
Connection factory for given transport
*/
type Transport interface {
	/**
	Get client connection
	*/
	Connect(url string) (conn Connection, err error)

	/**
	Handle one server connection
	*/
	HandleConnection(w http.ResponseWriter, r *http.Request) (conn Connection, err error)

	/**
	Serve HTTP request after making connection and events setup
	*/
	Serve(w http.ResponseWriter, r *http.Request)
}
