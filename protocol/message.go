package protocol

const (
	// 主️类型
	// OPEN(0), CLOSE(1), PING(2), PONG(3), MESSAGE(4), UPGRADE(5), NOOP(6),
	// 子类型
	// CONNECT(0, true), DISCONNECT(1, true), EVENT(2, true), ACK(3, true),
	// ERROR(4, true), BINARY_EVENT(5, true), BINARY_ACK(6, true);
	/**
	Message with connection options
	*/
	MessageTypeOpen = iota
	/**
	Close connection and destroy all handle routines
	*/
	MessageTypeClose = iota
	/**
	Ping request message
	*/
	MessageTypePing = iota
	/**
	Pong response message
	*/
	MessageTypePong = iota
	/**
	Empty message
	*/
	MessageTypeEmpty = iota
	/**
	Emit request, no response
	*/
	MessageTypeEmit = iota
	/**
	Emit request, wait for response (ack)
	*/
	MessageTypeAckRequest = iota
	/**
	ack response
	*/
	MessageTypeAckResponse = iota
)

type Message struct {
	Type   int
	AckId  int
	Method string
	Args   string
	Source []byte
	Data   []byte
	Num      int
}

