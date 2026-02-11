package logger

import (
	"testing"

	"go.uber.org/zap"
)

// BenchmarkLogger 基准测试 - JSON 格式
func BenchmarkLoggerJSON(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(false),
	)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message",
				zap.String("key", "value"),
				zap.Int("count", 42),
			)
		}
	})
}

// BenchmarkLoggerConsole 基准测试 - Console 格式
func BenchmarkLoggerConsole(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(ConsoleFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(false),
	)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message",
				zap.String("key", "value"),
				zap.Int("count", 42),
			)
		}
	})
}

// BenchmarkLoggerWithFields 基准测试 - 带字段
func BenchmarkLoggerWithFields(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(false),
	)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message",
				zap.String("string", "value"),
				zap.Int("int", 42),
				zap.Int64("int64", 12345),
				zap.Float64("float64", 3.14),
				zap.Bool("bool", true),
			)
		}
	})
}

// BenchmarkLoggerWithContext 基准测试 - 带 Context
func BenchmarkLoggerWithContext(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(false),
	)
	defer logger.Sync()

	// 创建子 Logger（模拟 WithContext）
	ctxLogger := logger.With(
		zap.String("trace_id", "trace-123"),
		zap.Int64("uid", 12345),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctxLogger.Info("benchmark message",
				zap.String("key", "value"),
			)
		}
	})
}

// BenchmarkLoggerDisabled 基准测试 - 禁用级别
func BenchmarkLoggerDisabled(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(ErrorLevel), // 只记录 Error 及以上
		WithFormat(JSONFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(false),
	)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message", // Info 级别被禁用
				zap.String("key", "value"),
			)
		}
	})
}

// BenchmarkLoggerWithCaller 基准测试 - 启用 Caller
func BenchmarkLoggerWithCaller(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(true),
	)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message",
				zap.String("key", "value"),
			)
		}
	})
}

// BenchmarkLoggerSampling 基准测试 - 采样
func BenchmarkLoggerSampling(b *testing.B) {
	logger, _ := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithFileOutput("/tmp/bench.log"),
		WithCaller(false),
		WithSampling(&SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}),
	)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message",
				zap.String("key", "value"),
			)
		}
	})
}
