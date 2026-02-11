package logger

import "go.uber.org/zap/zapcore"

// Hook 日志钩子接口
type Hook interface {
	// OnWrite 在日志写入时调用
	OnWrite(entry zapcore.Entry, fields []zapcore.Field) error
}
