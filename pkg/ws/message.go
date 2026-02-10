package ws

import (
	"encoding/json"
	"sync"
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	// MessageTypeRequest 请求消息
	MessageTypeRequest MessageType = "request"
	// MessageTypeResponse 响应消息
	MessageTypeResponse MessageType = "response"
	// MessageTypeNotify 通知消息（无需响应）
	MessageTypeNotify MessageType = "notify"
	// MessageTypeError 错误消息
	MessageTypeError MessageType = "error"
)

// Message WebSocket 消息
type Message struct {
	// Type 消息类型
	Type MessageType `json:"type"`

	// Event 事件名称（如 "chat.send", "user.login"）
	Event string `json:"event"`

	// RequestID 请求 ID（用于请求-响应匹配）
	RequestID string `json:"request_id,omitempty"`

	// Data 消息数据（JSON）
	Data json.RawMessage `json:"data,omitempty"`

	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp"`
}

// Response WebSocket 响应消息
type Response struct {
	// Type 固定为 "response"
	Type MessageType `json:"type"`

	// RequestID 对应的请求 ID
	RequestID string `json:"request_id"`

	// Code 业务状态码
	Code int `json:"code"`

	// Message 消息
	Message string `json:"message"`

	// Data 响应数据
	Data any `json:"data,omitempty"`

	// TraceID 链路追踪 ID
	TraceID string `json:"trace_id,omitempty"`

	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp"`
}

// ErrorResponse WebSocket 错误响应
type ErrorResponse struct {
	// Type 固定为 "error"
	Type MessageType `json:"type"`

	// RequestID 对应的请求 ID
	RequestID string `json:"request_id,omitempty"`

	// Code 错误码
	Code int `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// TraceID 链路追踪 ID
	TraceID string `json:"trace_id,omitempty"`

	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp"`
}

// NewMessage 创建消息
//
// 注意：此函数从对象池获取消息对象，使用完毕后必须调用 msg.Release() 释放到对象池。
//
// 使用示例：
//
//	msg, err := NewMessage("chat.send", data)
//	if err != nil {
//	    return err
//	}
//	defer msg.Release()  // 确保释放到对象池
//
//	// 发送消息
//	if err := client.SendJSON(msg); err != nil {
//	    return err
//	}
func NewMessage(event string, data any) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	msg := acquireMessage()
	msg.Type = MessageTypeRequest
	msg.Event = event
	msg.RequestID = generateRequestID()
	msg.Data = dataBytes
	msg.Timestamp = time.Now().Unix()

	return msg, nil
}

// NewMessageSimple 创建消息（不使用对象池）
//
// 此函数不使用对象池，消息对象由 GC 管理，无需手动调用 Release()。
// 适用于不关心性能优化或担心忘记释放的场景。
//
// 使用示例：
//
//	msg, err := NewMessageSimple("chat.send", data)
//	if err != nil {
//	    return err
//	}
//	// 无需调用 Release()，GC 会自动回收
//	if err := client.SendJSON(msg); err != nil {
//	    return err
//	}
func NewMessageSimple(event string, data any) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:      MessageTypeRequest,
		Event:     event,
		RequestID: generateRequestID(),
		Data:      dataBytes,
		Timestamp: time.Now().Unix(),
	}, nil
}

// NewNotifyMessage 创建通知消息
//
// 注意：此函数从对象池获取消息对象，使用完毕后必须调用 msg.Release() 释放到对象池。
//
// 使用示例：
//
//	msg, err := NewNotifyMessage("user.online", data)
//	if err != nil {
//	    return err
//	}
//	defer msg.Release()  // 确保释放到对象池
//
//	// 广播通知
//	if err := manager.BroadcastAll(msg); err != nil {
//	    return err
//	}
func NewNotifyMessage(event string, data any) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	msg := acquireMessage()
	msg.Type = MessageTypeNotify
	msg.Event = event
	msg.Data = dataBytes
	msg.Timestamp = time.Now().Unix()

	return msg, nil
}

// NewNotifyMessageSimple 创建通知消息（不使用对象池）
//
// 此函数不使用对象池，消息对象由 GC 管理，无需手动调用 Release()。
// 适用于不关心性能优化或担心忘记释放的场景。
//
// 使用示例：
//
//	msg, err := NewNotifyMessageSimple("user.online", data)
//	if err != nil {
//	    return err
//	}
//	// 无需调用 Release()，GC 会自动回收
//	if err := manager.BroadcastAll(msg); err != nil {
//	    return err
//	}
func NewNotifyMessageSimple(event string, data any) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:      MessageTypeNotify,
		Event:     event,
		Data:      dataBytes,
		Timestamp: time.Now().Unix(),
	}, nil
}

// Release 释放消息到对象池
func (m *Message) Release() {
	releaseMessage(m)
}

// NewResponse 创建响应
func NewResponse(requestID string, code int, message string, data any) *Response {
	return &Response{
		Type:      MessageTypeResponse,
		RequestID: requestID,
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(requestID string, code int, message string) *ErrorResponse {
	return &ErrorResponse{
		Type:      MessageTypeError,
		RequestID: requestID,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
}

// Unmarshal 解析消息数据
func (m *Message) Unmarshal(v any) error {
	return json.Unmarshal(m.Data, v)
}

// messagePool 消息对象池
var messagePool = sync.Pool{
	New: func() any {
		return &Message{}
	},
}

// acquireMessage 从对象池获取消息
func acquireMessage() *Message {
	return messagePool.Get().(*Message)
}

// releaseMessage 释放消息到对象池
func releaseMessage(msg *Message) {
	msg.Type = ""
	msg.Event = ""
	msg.RequestID = ""
	msg.Data = nil
	msg.Timestamp = 0
	messagePool.Put(msg)
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return generateID("req", &requestIDCounter)
}
