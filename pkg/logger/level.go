package logger

import "go.uber.org/zap/zapcore"

// Level 日志级别
type Level int8

const (
	// DebugLevel 调试信息（开发环境）
	DebugLevel Level = iota - 1
	// InfoLevel 常规信息（默认级别）
	InfoLevel
	// WarnLevel 警告信息（需要关注但不影响运行）
	WarnLevel
	// ErrorLevel 错误信息（影响功能但不致命）
	ErrorLevel
	// DPanicLevel 开发环境 panic（生产环境记录错误）
	DPanicLevel
	// PanicLevel 记录后 panic
	PanicLevel
	// FatalLevel 记录后退出程序
	FatalLevel
)

// String 返回级别名称
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case DPanicLevel:
		return "dpanic"
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

// toZapLevel 转换为 zap 级别
func (l Level) toZapLevel() zapcore.Level {
	return zapcore.Level(l)
}

// fromZapLevel 从 zap 级别转换
func fromZapLevel(level zapcore.Level) Level {
	return Level(level)
}
