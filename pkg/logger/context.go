package logger

import "qi"

const (
	// contextKeyLogger Context 中存储 Logger 的 key
	contextKeyLogger = "qi:logger"
)

// SetContextLogger 设置 Context 中的 Logger
func SetContextLogger(ctx *qi.Context, logger Logger) {
	ctx.Set(contextKeyLogger, logger)
}

// GetContextLogger 获取 Context 中的 Logger
func GetContextLogger(ctx *qi.Context) Logger {
	if logger, exists := ctx.Get(contextKeyLogger); exists {
		if l, ok := logger.(Logger); ok {
			return l
		}
	}
	return nil
}
