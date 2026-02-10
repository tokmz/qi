package ws

import "errors"

// 错误定义
var (
	// 连接相关错误
	ErrTooManyConnections = errors.New("ws: too many connections")
	ErrClientIDExists     = errors.New("ws: client id already exists")
	ErrClientNotFound     = errors.New("ws: client not found")
	ErrConnectionClosed   = errors.New("ws: connection closed")

	// 房间相关错误
	ErrRoomNotFound  = errors.New("ws: room not found")
	ErrRoomExists    = errors.New("ws: room already exists")
	ErrRoomFull      = errors.New("ws: room is full")
	ErrAlreadyInRoom = errors.New("ws: already in room")

	// 消息相关错误
	ErrHandlerNotFound    = errors.New("ws: handler not found")
	ErrHandlerExists      = errors.New("ws: handler already exists")
	ErrInvalidMessage     = errors.New("ws: invalid message format")
	ErrMessageTooLarge    = errors.New("ws: message too large")
	ErrChannelFull        = errors.New("ws: send channel full")
	ErrRouterFrozen       = errors.New("ws: router is frozen")
	ErrInvalidMessageType = errors.New("ws: invalid message type")
	ErrBroadcastTimeout   = errors.New("ws: broadcast timeout")

	// 配置相关错误
	ErrInvalidConfig = errors.New("ws: invalid config")
)

// ClientError 客户端错误
type ClientError struct {
	Message string
}

func (e *ClientError) Error() string {
	return "ws client error: " + e.Message
}

// NewClientError 创建客户端错误
func NewClientError(msg string) error {
	return &ClientError{Message: msg}
}
