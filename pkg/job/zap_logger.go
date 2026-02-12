package job

import (
	"go.uber.org/zap"
)

// ZapLogger zap 日志适配器
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger 创建 zap 日志适配器
func NewZapLogger(logger *zap.Logger) *ZapLogger {
	return &ZapLogger{logger: logger}
}

func (l *ZapLogger) Debug(msg string, args ...any) {
	if len(args) == 0 {
		l.logger.Debug(msg)
		return
	}
	l.logger.Sugar().Debugf(msg, args...)
}

func (l *ZapLogger) Info(msg string, args ...any) {
	if len(args) == 0 {
		l.logger.Info(msg)
		return
	}
	l.logger.Sugar().Infof(msg, args...)
}

func (l *ZapLogger) Warn(msg string, args ...any) {
	if len(args) == 0 {
		l.logger.Warn(msg)
		return
	}
	l.logger.Sugar().Warnf(msg, args...)
}

func (l *ZapLogger) Error(msg string, args ...any) {
	if len(args) == 0 {
		l.logger.Error(msg)
		return
	}
	l.logger.Sugar().Errorf(msg, args...)
}
