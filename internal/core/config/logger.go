package config

import "fmt"

// DefaultLogger 默认日志实现（输出到标准输出）
type DefaultLogger struct{}

// Info 输出信息日志
func (l *DefaultLogger) Info(msg string) {
	fmt.Printf("[INFO] [Config] %s\n", msg)
}

// Warn 输出警告日志
func (l *DefaultLogger) Warn(msg string) {
	fmt.Printf("[WARN] [Config] %s\n", msg)
}

// Error 输出错误日志
func (l *DefaultLogger) Error(msg string) {
	fmt.Printf("[ERROR] [Config] %s\n", msg)
}

// Debug 输出调试日志
func (l *DefaultLogger) Debug(msg string) {
	fmt.Printf("[DEBUG] [Config] %s\n", msg)
}

