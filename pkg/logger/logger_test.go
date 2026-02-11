package logger

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"qi"
)

// TestNew 测试创建 Logger
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "console output",
			config: &Config{
				Level:   InfoLevel,
				Format:  JSONFormat,
				Console: true,
			},
			wantErr: false,
		},
		{
			name: "file output",
			config: &Config{
				Level:  InfoLevel,
				Format: JSONFormat,
				File:   "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "rotate output",
			config: &Config{
				Level:  InfoLevel,
				Format: JSONFormat,
				Rotate: &RotateConfig{
					Filename: "/tmp/test-rotate.log",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger != nil {
				defer logger.Sync()
			}
		})
	}
}

// TestNewWithOptions 测试使用 Options 创建 Logger
func TestNewWithOptions(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(DebugLevel),
		WithFormat(ConsoleFormat),
		WithConsoleOutput(),
		WithCaller(true),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	if logger.Level() != DebugLevel {
		t.Errorf("Level() = %v, want %v", logger.Level(), DebugLevel)
	}
}

// TestNewProduction 测试创建生产环境 Logger
func TestNewProduction(t *testing.T) {
	logger, err := NewProduction()
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	defer logger.Sync()

	if logger.Level() != InfoLevel {
		t.Errorf("Level() = %v, want %v", logger.Level(), InfoLevel)
	}
}

// TestNewDevelopment 测试创建开发环境 Logger
func TestNewDevelopment(t *testing.T) {
	logger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("NewDevelopment() error = %v", err)
	}
	defer logger.Sync()

	if logger.Level() != DebugLevel {
		t.Errorf("Level() = %v, want %v", logger.Level(), DebugLevel)
	}
}

// TestLoggerBasicMethods 测试基础日志方法
func TestLoggerBasicMethods(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(DebugLevel),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 测试各级别日志
	logger.Debug("debug message", zap.String("key", "value"))
	logger.Info("info message", zap.Int("count", 42))
	logger.Warn("warn message", zap.Duration("duration", time.Second))
	logger.Error("error message", zap.Bool("success", false))
}

// TestLoggerWithContext 测试带 Context 的日志方法
func TestLoggerWithContext(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 测试带 Context 的日志（使用标准 context.Context）
	ctx := context.Background()
	logger.InfoContext(ctx, "user action", zap.String("action", "login"))
	logger.ErrorContext(ctx, "user error", zap.String("error", "invalid password"))

	// 测试使用 qi.Context.RequestContext() 方法
	engine := qi.New()
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest("GET", "/test", nil)
	c := qi.NewContext(ginCtx)

	// 设置 TraceID 和 UID
	qi.SetContextTraceID(c, "trace-123")
	qi.SetContextUid(c, 12345)

	// 获取标准库 context.Context
	ctx2 := c.RequestContext()
	logger.InfoContext(ctx2, "user action with context", zap.String("action", "login"))

	_ = engine // 避免未使用变量警告
}

// TestLoggerWith 测试创建子 Logger
func TestLoggerWith(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 创建子 Logger
	childLogger := logger.With(
		zap.String("module", "user"),
		zap.String("version", "v1"),
	)

	childLogger.Info("child logger message")
}

// TestLoggerWithContextMethod 测试创建带 Context 的子 Logger
func TestLoggerWithContextMethod(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 测试使用 qi.Context.RequestContext() 方法
	engine := qi.New()
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest("GET", "/test", nil)
	c := qi.NewContext(ginCtx)

	// 设置 TraceID 和 UID
	qi.SetContextTraceID(c, "trace-456")
	qi.SetContextUid(c, 67890)

	// 获取标准库 context.Context
	ctx := c.RequestContext()

	// 创建带 Context 的子 Logger
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("context logger message")

	_ = engine // 避免未使用变量警告
}

// TestSetLevel 测试动态调整级别
func TestSetLevel(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 初始级别
	if logger.Level() != InfoLevel {
		t.Errorf("Level() = %v, want %v", logger.Level(), InfoLevel)
	}

	// 调整级别
	logger.SetLevel(DebugLevel)
	if logger.Level() != DebugLevel {
		t.Errorf("Level() = %v, want %v", logger.Level(), DebugLevel)
	}
}

// TestContextHelpers 测试 Context 辅助方法
func TestContextHelpers(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 创建 Qi Engine
	engine := qi.New()

	// 使用 Gin 的测试 Context
	ginCtx, _ := gin.CreateTestContext(nil)
	c := qi.NewContext(ginCtx)

	// 设置 Logger
	SetContextLogger(c, logger)

	// 获取 Logger
	ctxLogger := GetContextLogger(c)
	if ctxLogger == nil {
		t.Error("GetContextLogger() returned nil")
	}

	ctxLogger.Info("logger from context")

	_ = engine // 避免未使用变量警告
}

// TestRotateConfig 测试轮转配置
func TestRotateConfig(t *testing.T) {
	config := &RotateConfig{
		Filename: "/tmp/test-rotate.log",
	}
	config.setDefaults()

	if config.MaxSize != 100 {
		t.Errorf("MaxSize = %v, want 100", config.MaxSize)
	}
	if config.MaxAge != 30 {
		t.Errorf("MaxAge = %v, want 30", config.MaxAge)
	}
	if config.MaxBackups != 10 {
		t.Errorf("MaxBackups = %v, want 10", config.MaxBackups)
	}
	if !config.LocalTime {
		t.Error("LocalTime should be true")
	}
}

// TestSamplingConfig 测试采样配置
func TestSamplingConfig(t *testing.T) {
	config := &SamplingConfig{}
	config.setDefaults()

	if config.Initial != 100 {
		t.Errorf("Initial = %v, want 100", config.Initial)
	}
	if config.Thereafter != 100 {
		t.Errorf("Thereafter = %v, want 100", config.Thereafter)
	}
}

// TestLevel 测试日志级别
func TestLevel(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{DPanicLevel, "dpanic"},
		{PanicLevel, "panic"},
		{FatalLevel, "fatal"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormat 测试日志格式
func TestFormat(t *testing.T) {
	tests := []struct {
		format  Format
		isValid bool
	}{
		{JSONFormat, true},
		{ConsoleFormat, true},
		{Format("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.IsValid(); got != tt.isValid {
				t.Errorf("Format.IsValid() = %v, want %v", got, tt.isValid)
			}
		})
	}
}

// TestHook 测试 Hook 机制
func TestHook(t *testing.T) {
	// 创建测试 Hook
	hookCalled := false
	testHook := &testHook{
		onWrite: func() {
			hookCalled = true
		},
	}

	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithConsoleOutput(),
		WithHook(testHook),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 记录日志
	logger.Info("test hook")

	// 验证 Hook 被调用
	if !hookCalled {
		t.Error("Hook was not called")
	}
}

// testHook 测试用 Hook
type testHook struct {
	onWrite func()
}

func (h *testHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) error {
	if h.onWrite != nil {
		h.onWrite()
	}
	return nil
}

// TestFileOutput 测试文件输出
func TestFileOutput(t *testing.T) {
	tmpFile := "/tmp/test-logger-" + time.Now().Format("20060102150405") + ".log"
	defer os.Remove(tmpFile)

	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithFileOutput(tmpFile),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 写入日志
	logger.Info("test file output", zap.String("key", "value"))

	// 验证文件存在
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

// TestMiddleware 测试日志中间件
func TestMiddleware(t *testing.T) {
	// 创建 Logger
	logger, err := NewWithOptions(
		WithLevel(InfoLevel),
		WithFormat(JSONFormat),
		WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer logger.Sync()

	// 创建中间件
	middleware := Middleware(logger)

	// 验证中间件函数不为 nil
	if middleware == nil {
		t.Error("Middleware returned nil")
	}

	// 创建测试 Engine 并注册中间件
	engine := qi.New()
	engine.Use(middleware)

	// 注册测试路由
	r := engine.RouterGroup()
	r.GET("/test", func(c *qi.Context) {
		c.Success("ok")
	})
}
