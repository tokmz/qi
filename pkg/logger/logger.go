package logger

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ContextKey 日志上下文键
type contextKey string

const (
	traceIDKey contextKey = "trace_id"
	uidKey     contextKey = "uid"
)

// Logger 日志接口
type Logger interface {
	// 基础日志方法
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	DPanic(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)

	// 带 Context 的日志方法（自动提取 TraceID、UID）
	// 使用标准库 context.Context，支持 Service 层传递请求上下文
	DebugContext(ctx context.Context, msg string, fields ...zap.Field)
	InfoContext(ctx context.Context, msg string, fields ...zap.Field)
	WarnContext(ctx context.Context, msg string, fields ...zap.Field)
	ErrorContext(ctx context.Context, msg string, fields ...zap.Field)

	// 工具方法
	With(fields ...zap.Field) Logger       // 创建子 Logger
	WithContext(ctx context.Context) Logger // 创建带 Context 的子 Logger
	Sync() error                           // 刷新缓冲区
	SetLevel(level Level)                  // 动态调整级别
	Level() Level                          // 获取当前级别
}

// logger 日志实现
type logger struct {
	zap   *zap.Logger
	level atomic.Value // 存储 zapcore.Level
	hooks []Hook
}

// New 创建 Logger（使用 Config）
func New(config *Config) (Logger, error) {
	if config == nil {
		config = &Config{}
	}
	config.setDefaults()

	// 创建 Encoder
	encoder := buildEncoder(config)

	// 创建 WriteSyncer
	writers, err := buildWriters(config)
	if err != nil {
		return nil, err
	}
	if len(writers) == 0 {
		return nil, fmt.Errorf("no output configured")
	}

	// 创建 Core
	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), config.Level.toZapLevel())

	// 应用采样
	if config.Sampling != nil {
		config.Sampling.setDefaults()
		core = zapcore.NewSamplerWithOptions(
			core,
			1, // 1 秒
			config.Sampling.Initial,
			config.Sampling.Thereafter,
		)
	}

	// 应用 Hooks
	if len(config.Hooks) > 0 {
		core = &hookCore{Core: core, hooks: config.Hooks}
	}

	// 创建 zap.Logger
	opts := []zap.Option{}
	if config.EnableCaller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
	}
	if config.EnableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	zapLogger := zap.New(core, opts...)

	l := &logger{
		zap:   zapLogger,
		hooks: config.Hooks,
	}
	l.level.Store(config.Level.toZapLevel())

	return l, nil
}

// NewWithOptions 创建 Logger（使用 Options 模式）
func NewWithOptions(opts ...Option) (Logger, error) {
	config := &Config{}
	for _, opt := range opts {
		opt(config)
	}
	return New(config)
}

// Default 创建默认 Logger（开发环境配置）
func Default() Logger {
	l, _ := NewDevelopment()
	return l
}

// NewProduction 创建生产环境 Logger
func NewProduction() (Logger, error) {
	return NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithConsoleOutput(),
		WithCaller(false), // 生产环境禁用 Caller 以提升性能
		WithStacktrace(true),
	)
}

// NewDevelopment 创建开发环境 Logger
func NewDevelopment() (Logger, error) {
	return NewWithOptions(
		WithLevel(DebugLevel),
		WithFormat(ConsoleFormat),
		WithConsoleOutput(),
		WithCaller(true),
		WithStacktrace(true),
	)
}

// buildEncoder 构建 Encoder
func buildEncoder(config *Config) zapcore.Encoder {
	encoderConfig := config.EncoderConfig
	if encoderConfig == nil {
		encoderConfig = &zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	switch config.Format {
	case ConsoleFormat:
		return zapcore.NewConsoleEncoder(*encoderConfig)
	case JSONFormat:
		return zapcore.NewJSONEncoder(*encoderConfig)
	default:
		return zapcore.NewJSONEncoder(*encoderConfig)
	}
}

// buildWriters 构建 WriteSyncer
func buildWriters(config *Config) ([]zapcore.WriteSyncer, error) {
	var writers []zapcore.WriteSyncer

	// 控制台输出
	if config.Console {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 文件输出
	if config.File != "" {
		writer, _, err := zap.Open(config.File)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", config.File, err)
		}
		writers = append(writers, writer)
	}

	// 文件轮转输出
	if config.Rotate != nil {
		config.Rotate.setDefaults()
		rotateWriter := &lumberjack.Logger{
			Filename:   config.Rotate.Filename,
			MaxSize:    config.Rotate.MaxSize,
			MaxAge:     config.Rotate.MaxAge,
			MaxBackups: config.Rotate.MaxBackups,
			LocalTime:  config.Rotate.LocalTime,
			Compress:   config.Rotate.Compress,
		}
		writers = append(writers, zapcore.AddSync(rotateWriter))
	}

	return writers, nil
}

// Debug 记录调试日志
func (l *logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

// Info 记录信息日志
func (l *logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

// Warn 记录警告日志
func (l *logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

// Error 记录错误日志
func (l *logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

// DPanic 记录 DPanic 日志
func (l *logger) DPanic(msg string, fields ...zap.Field) {
	l.zap.DPanic(msg, fields...)
}

// Panic 记录 Panic 日志
func (l *logger) Panic(msg string, fields ...zap.Field) {
	l.zap.Panic(msg, fields...)
}

// Fatal 记录 Fatal 日志
func (l *logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

// DebugContext 记录带 Context 的调试日志
func (l *logger) DebugContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.zap.Debug(msg, l.contextFields(ctx, fields)...)
}

// InfoContext 记录带 Context 的信息日志
func (l *logger) InfoContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.zap.Info(msg, l.contextFields(ctx, fields)...)
}

// WarnContext 记录带 Context 的警告日志
func (l *logger) WarnContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.zap.Warn(msg, l.contextFields(ctx, fields)...)
}

// ErrorContext 记录带 Context 的错误日志
func (l *logger) ErrorContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.zap.Error(msg, l.contextFields(ctx, fields)...)
}

// contextFields 从标准库 context.Context 提取字段
func (l *logger) contextFields(ctx context.Context, fields []zap.Field) []zap.Field {
	contextFields := make([]zap.Field, 0, len(fields)+3)

	// 从 context.Context 提取 TraceID
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		contextFields = append(contextFields, zap.String("trace_id", traceID))
	}

	// 从 context.Context 提取 OpenTelemetry SpanID
	if spanID := extractSpanID(ctx); spanID != "" {
		contextFields = append(contextFields, zap.String("span_id", spanID))
	}

	// 从 context.Context 提取 UID
	if uid, ok := ctx.Value(uidKey).(int64); ok && uid != 0 {
		contextFields = append(contextFields, zap.Int64("uid", uid))
	}

	// 添加用户字段
	contextFields = append(contextFields, fields...)

	return contextFields
}

// With 创建子 Logger
func (l *logger) With(fields ...zap.Field) Logger {
	return &logger{
		zap:   l.zap.With(fields...),
		level: l.level,
		hooks: l.hooks,
	}
}

// WithContext 创建带 Context 的子 Logger
func (l *logger) WithContext(ctx context.Context) Logger {
	fields := make([]zap.Field, 0, 2)

	// 从 context.Context 提取 TraceID
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// 从 context.Context 提取 UID
	if uid, ok := ctx.Value(uidKey).(int64); ok && uid != 0 {
		fields = append(fields, zap.Int64("uid", uid))
	}

	return l.With(fields...)
}

// Sync 刷新缓冲区
func (l *logger) Sync() error {
	return l.zap.Sync()
}

// SetLevel 动态调整级别
func (l *logger) SetLevel(level Level) {
	l.level.Store(level.toZapLevel())
}

// Level 获取当前级别
func (l *logger) Level() Level {
	return fromZapLevel(l.level.Load().(zapcore.Level))
}

// extractSpanID 从 context.Context 提取 OpenTelemetry SpanID
func extractSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// hookCore 实现 Hook 机制的 Core
type hookCore struct {
	zapcore.Core
	hooks []Hook
}

// Write 写入日志时调用 Hooks
func (c *hookCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// 先调用 Hooks
	for _, hook := range c.hooks {
		if err := hook.OnWrite(entry, fields); err != nil {
			return err
		}
	}
	// 再写入日志
	return c.Core.Write(entry, fields)
}

// With 创建带字段的 Core
func (c *hookCore) With(fields []zapcore.Field) zapcore.Core {
	return &hookCore{
		Core:  c.Core.With(fields),
		hooks: c.hooks,
	}
}

// Check 检查日志级别
func (c *hookCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}
