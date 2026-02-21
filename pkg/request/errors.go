package request

import "github.com/tokmz/qi/pkg/errors"

// 4000 段错误码 — HTTP 客户端相关
var (
	// ErrRequestFailed 请求失败
	ErrRequestFailed = errors.New(4001, 500, "请求失败", nil)
	// ErrTimeout 请求超时
	ErrTimeout = errors.New(4002, 504, "请求超时", nil)
	// ErrMarshal 序列化失败
	ErrMarshal = errors.New(4003, 500, "序列化失败", nil)
	// ErrUnmarshal 反序列化失败
	ErrUnmarshal = errors.New(4004, 500, "反序列化失败", nil)
	// ErrMaxRetry 重试次数已用尽
	ErrMaxRetry = errors.New(4005, 502, "重试次数已用尽", nil)
	// ErrInvalidURL 无效的 URL
	ErrInvalidURL = errors.New(4006, 400, "无效的URL", nil)
)
