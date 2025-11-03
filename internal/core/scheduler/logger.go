package scheduler

import (
	"fmt"
)

// Logger 日志接口
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Debug(msg string)
}

// cronLogger 适配器，将 cron.Logger 转换为我们的 Logger
type cronLogger struct {
	logger Logger
}

// newCronLogger 创建 cron 日志适配器
func newCronLogger(logger Logger) *cronLogger {
	return &cronLogger{logger: logger}
}

// Info 实现 cron.Logger 接口
func (l *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	if l.logger != nil {
		l.logger.Info(formatLog(msg, keysAndValues...))
	}
}

// Error 实现 cron.Logger 接口
func (l *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if l.logger != nil {
		fullMsg := formatLog(msg, keysAndValues...)
		if err != nil {
			fullMsg = fmt.Sprintf("%s: %v", fullMsg, err)
		}
		l.logger.Error(fullMsg)
	}
}

// formatLog 格式化日志消息
func formatLog(msg string, keysAndValues ...interface{}) string {
	if len(keysAndValues) == 0 {
		return msg
	}

	result := msg
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			result += fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}
	return result
}

// DefaultLogger 默认日志实现（输出到标准输出）
type DefaultLogger struct{}

// Info 输出信息日志
func (l *DefaultLogger) Info(msg string) {
	fmt.Printf("[INFO] [Scheduler] %s\n", msg)
}

// Warn 输出警告日志
func (l *DefaultLogger) Warn(msg string) {
	fmt.Printf("[WARN] [Scheduler] %s\n", msg)
}

// Error 输出错误日志
func (l *DefaultLogger) Error(msg string) {
	fmt.Printf("[ERROR] [Scheduler] %s\n", msg)
}

// Debug 输出调试日志
func (l *DefaultLogger) Debug(msg string) {
	fmt.Printf("[DEBUG] [Scheduler] %s\n", msg)
}
