package request

import "context"

// Logger 日志接口 — 最小化设计，解耦具体日志实现
// 使用 key-value 风格参数，任何实现了该接口的日志器均可传入
type Logger interface {
	// InfoContext 记录 Info 级别日志
	InfoContext(ctx context.Context, msg string, keysAndValues ...any)
	// ErrorContext 记录 Error 级别日志
	ErrorContext(ctx context.Context, msg string, keysAndValues ...any)
}
