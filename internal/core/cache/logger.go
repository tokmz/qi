package cache

import (
	"fmt"
	"log"
)

// Logger 日志接口
type Logger interface {
	// Debug 调试日志
	Debug(msg string, fields ...interface{})
	// Info 信息日志
	Info(msg string, fields ...interface{})
	// Warn 警告日志
	Warn(msg string, fields ...interface{})
	// Error 错误日志
	Error(msg string, fields ...interface{})
}

// DefaultLogger 默认日志实现（使用标准库 log）
type DefaultLogger struct{}

// Debug 调试日志
func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		log.Printf("[DEBUG] %s %v", msg, fields)
	} else {
		log.Printf("[DEBUG] %s", msg)
	}
}

// Info 信息日志
func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		log.Printf("[INFO] %s %v", msg, fields)
	} else {
		log.Printf("[INFO] %s", msg)
	}
}

// Warn 警告日志
func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		log.Printf("[WARN] %s %v", msg, fields)
	} else {
		log.Printf("[WARN] %s", msg)
	}
}

// Error 错误日志
func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		log.Printf("[ERROR] %s %v", msg, fields)
	} else {
		log.Printf("[ERROR] %s", msg)
	}
}

// NoopLogger 空日志实现（不输出任何日志）
type NoopLogger struct{}

// Debug 调试日志
func (l *NoopLogger) Debug(msg string, fields ...interface{}) {}

// Info 信息日志
func (l *NoopLogger) Info(msg string, fields ...interface{}) {}

// Warn 警告日志
func (l *NoopLogger) Warn(msg string, fields ...interface{}) {}

// Error 错误日志
func (l *NoopLogger) Error(msg string, fields ...interface{}) {}

// DebugLogger 详细调试日志实现
type DebugLogger struct{}

// Debug 调试日志
func (l *DebugLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %s", msg, formatFields(fields...))
}

// Info 信息日志
func (l *DebugLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %s", msg, formatFields(fields...))
}

// Warn 警告日志
func (l *DebugLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %s", msg, formatFields(fields...))
}

// Error 错误日志
func (l *DebugLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %s", msg, formatFields(fields...))
}

// formatFields 格式化字段
func formatFields(fields ...interface{}) string {
	if len(fields) == 0 {
		return ""
	}
	return fmt.Sprintf("%+v", fields)
}

