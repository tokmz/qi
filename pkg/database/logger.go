package database

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// zapLogger 将 zap.Logger 适配为 gorm logger.Interface
type zapLogger struct {
	zap       *zap.Logger
	level     logger.LogLevel
	slowThreshold time.Duration
}

func newZapLogger(z *zap.Logger, cfg *Config) logger.Interface {
	return &zapLogger{
		zap:           z.WithOptions(zap.AddCallerSkip(3)),
		level:         logger.LogLevel(cfg.LogLevel),
		slowThreshold: cfg.SlowThreshold,
	}
}

func (l *zapLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *zapLogger) Info(_ context.Context, msg string, args ...any) {
	if l.level >= logger.Info {
		l.zap.Sugar().Infof(msg, args...)
	}
}

func (l *zapLogger) Warn(_ context.Context, msg string, args ...any) {
	if l.level >= logger.Warn {
		l.zap.Sugar().Warnf(msg, args...)
	}
}

func (l *zapLogger) Error(_ context.Context, msg string, args ...any) {
	if l.level >= logger.Error {
		l.zap.Sugar().Errorf(msg, args...)
	}
}

func (l *zapLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	switch {
	case err != nil && !errors.Is(err, logger.ErrRecordNotFound) && l.level >= logger.Error:
		l.zap.Error("gorm trace", append(fields, zap.Error(err))...)
	case l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= logger.Warn:
		l.zap.Warn("gorm slow query", fields...)
	case l.level >= logger.Info:
		l.zap.Info("gorm trace", fields...)
	}
}
