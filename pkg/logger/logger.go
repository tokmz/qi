package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置
type Config struct {
	// Level 日志级别：debug, info, warn, error
	Level string

	// Mode 输出模式：console（开发）, json（生产）
	Mode string

	// Filename 日志文件路径，为空则只输出到控制台
	Filename string

	// MaxSize 单个日志文件最大大小（MB），超过后触发分割
	MaxSize int

	// MaxAge 日志文件保留天数，超过后自动删除
	MaxAge int

	// MaxBackups 保留的旧日志文件最大数量
	MaxBackups int

	// Compress 是否压缩旧日志文件（gzip）
	Compress bool

	// ConsoleOutput 是否同时输出到控制台（文件模式下）
	ConsoleOutput bool

	// ShowCaller 是否显示调用位置（文件名:行号）
	ShowCaller bool

	// CallerSkip 跳过的调用栈层数（用于封装时调整）
	CallerSkip int
}

var (
	// globalLogger 用于 L() 直接调用
	globalLogger *zap.Logger

	// shortcutLogger 用于快捷方法（CallerSkip+1）
	shortcutLogger *zap.Logger

	// mu 保护全局 logger 的并发安全
	mu sync.RWMutex

	// initialized 标记是否已初始化
	initialized bool
)

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Level:         "info",
		Mode:          "console",
		Filename:      "",
		MaxSize:       100,
		MaxAge:        30,
		MaxBackups:    10,
		Compress:      true,
		ConsoleOutput: true,
		ShowCaller:    true,
		CallerSkip:    0,
	}
}

// Init 初始化全局 logger（并发安全）
func Init(config Config) {
	mu.Lock()
	defer mu.Unlock()

	core := buildCore(config)

	// 构建选项
	opts := []zap.Option{}
	if config.ShowCaller {
		opts = append(opts, zap.AddCaller())
	}

	// globalLogger 用于 L() 直接调用
	if config.CallerSkip > 0 {
		globalLogger = zap.New(core, append(opts, zap.AddCallerSkip(config.CallerSkip))...)
	} else {
		globalLogger = zap.New(core, opts...)
	}

	// shortcutLogger 用于快捷方法，额外跳过 1 层
	if config.ShowCaller {
		shortcutLogger = zap.New(core, append(opts, zap.AddCallerSkip(config.CallerSkip+1))...)
	} else {
		shortcutLogger = zap.New(core, opts...)
	}

	initialized = true
}

// New 创建独立的 logger 实例
func New(config Config) *zap.Logger {
	core := buildCore(config)

	opts := []zap.Option{}
	if config.ShowCaller {
		opts = append(opts, zap.AddCaller())
		if config.CallerSkip > 0 {
			opts = append(opts, zap.AddCallerSkip(config.CallerSkip))
		}
	}

	return zap.New(core, opts...)
}

// buildCore 构建 zap core
func buildCore(config Config) zapcore.Core {
	// 日志级别
	level := parseLevel(config.Level)

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006/01/02 - 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 开发模式使用彩色输出
	if config.Mode == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// 编码器
	var encoder zapcore.Encoder
	if config.Mode == "json" {
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 输出目标
	var writeSyncer zapcore.WriteSyncer
	if config.Filename != "" {
		// 文件输出（带分割）
		fileWriter := &lumberjack.Logger{
			Filename:   config.Filename,
			MaxSize:    config.MaxSize,
			MaxAge:     config.MaxAge,
			MaxBackups: config.MaxBackups,
			Compress:   config.Compress,
			LocalTime:  true,
		}

		if config.ConsoleOutput {
			// 同时输出到文件和控制台
			writeSyncer = zapcore.NewMultiWriteSyncer(
				zapcore.AddSync(fileWriter),
				zapcore.AddSync(os.Stdout),
			)
		} else {
			// 仅输出到文件
			writeSyncer = zapcore.AddSync(fileWriter)
		}
	} else {
		// 仅输出到控制台
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	return zapcore.NewCore(encoder, writeSyncer, level)
}

// parseLevel 解析日志级别
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// L 获取全局 logger（并发安全）
func L() *zap.Logger {
	mu.RLock()
	if initialized {
		logger := globalLogger
		mu.RUnlock()
		return logger
	}
	mu.RUnlock()

	// 未初始化，使用默认配置初始化
	Init(DefaultConfig())
	return globalLogger
}

// S 获取全局 SugaredLogger
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// ensureInitialized 确保 logger 已初始化（内部使用）
func ensureInitialized() {
	mu.RLock()
	if initialized {
		mu.RUnlock()
		return
	}
	mu.RUnlock()

	Init(DefaultConfig())
}

// Debug 记录 debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	ensureInitialized()
	shortcutLogger.Debug(msg, fields...)
}

// Info 记录 info 级别日志
func Info(msg string, fields ...zap.Field) {
	ensureInitialized()
	shortcutLogger.Info(msg, fields...)
}

// Warn 记录 warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	ensureInitialized()
	shortcutLogger.Warn(msg, fields...)
}

// Error 记录 error 级别日志
func Error(msg string, fields ...zap.Field) {
	ensureInitialized()
	shortcutLogger.Error(msg, fields...)
}

// Fatal 记录 fatal 级别日志并退出程序
func Fatal(msg string, fields ...zap.Field) {
	ensureInitialized()
	shortcutLogger.Fatal(msg, fields...)
}

// Panic 记录 panic 级别日志并触发 panic
func Panic(msg string, fields ...zap.Field) {
	ensureInitialized()
	shortcutLogger.Panic(msg, fields...)
}

// Sync 刷新缓冲区（程序退出前调用）
func Sync() error {
	mu.RLock()
	defer mu.RUnlock()

	var err error
	if globalLogger != nil {
		if syncErr := globalLogger.Sync(); syncErr != nil {
			err = syncErr
		}
	}
	if shortcutLogger != nil {
		if syncErr := shortcutLogger.Sync(); syncErr != nil {
			err = syncErr
		}
	}
	return err
}
