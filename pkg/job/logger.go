package job

import "log"

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// StdLogger 标准库 log 适配器
type StdLogger struct{}

func (l *StdLogger) Debug(msg string, args ...any) {
	if len(args) == 0 {
		log.Print("[DEBUG] " + msg)
	} else {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

func (l *StdLogger) Info(msg string, args ...any) {
	if len(args) == 0 {
		log.Print("[INFO] " + msg)
	} else {
		log.Printf("[INFO] "+msg, args...)
	}
}

func (l *StdLogger) Warn(msg string, args ...any) {
	if len(args) == 0 {
		log.Print("[WARN] " + msg)
	} else {
		log.Printf("[WARN] "+msg, args...)
	}
}

func (l *StdLogger) Error(msg string, args ...any) {
	if len(args) == 0 {
		log.Print("[ERROR] " + msg)
	} else {
		log.Printf("[ERROR] "+msg, args...)
	}
}

// NopLogger 空日志实现
type NopLogger struct{}

func (l *NopLogger) Debug(msg string, args ...any) {}

func (l *NopLogger) Info(msg string, args ...any) {}

func (l *NopLogger) Warn(msg string, args ...any) {}

func (l *NopLogger) Error(msg string, args ...any) {}
