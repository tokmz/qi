package request

import "errors"

// HTTP 客户端相关错误
var (
	// ErrRequestFailed 请求失败
	ErrRequestFailed = errors.New("请求失败")
	// ErrTimeout 请求超时
	ErrTimeout = errors.New("请求超时")
	// ErrMarshal 序列化失败
	ErrMarshal = errors.New("序列化失败")
	// ErrUnmarshal 反序列化失败
	ErrUnmarshal = errors.New("反序列化失败")
	// ErrMaxRetry 重试次数已用尽
	ErrMaxRetry = errors.New("重试次数已用尽")
	// ErrInvalidURL 无效的 URL
	ErrInvalidURL = errors.New("无效的URL")
)
