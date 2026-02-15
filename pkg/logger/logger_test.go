package logger_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/tokmz/qi"
	"github.com/tokmz/qi/pkg/logger"
)

// TestNew 测试创建 Logger
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *logger.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "console output",
			config: &logger.Config{
				Level:   logger.InfoLevel,
				Format:  logger.JSONFormat,
				Console: true,
			},
			wantErr: false,
		},
		{
			name: "file output",
			config: &logger.Config{
				Level:  logger.InfoLevel,
				Format: logger.JSONFormat,
				File:   "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "rotate output",
			config: &logger.Config{
				Level:  logger.InfoLevel,
				Format: logger.JSONFormat,
				Rotate: &logger.RotateConfig{
					Filename: "/tmp/test-rotate.log",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := logger.New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if l != nil {
				defer l.Sync()
			}
		})
	}
}

// TestNewWithOptions 测试使用 Options 创建 Logger
func TestNewWithOptions(t *testing.T) {
	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.DebugLevel),
		logger.WithFormat(logger.ConsoleFormat),
		logger.WithConsoleOutput(),
		logger.WithCaller(true),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	if l.Level() != logger.DebugLevel {
		t.Errorf("Level() = %v, want %v", l.Level(), logger.DebugLevel)
	}
}

// TestNewProduction 测试创建生产环境 Logger
func TestNewProduction(t *testing.T) {
	l, err := logger.NewProduction()
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	defer l.Sync()

	if l.Level() != logger.InfoLevel {
		t.Errorf("Level() = %v, want %v", l.Level(), logger.InfoLevel)
	}
}

// TestNewDevelopment 测试创建开发环境 Logger
func TestNewDevelopment(t *testing.T) {
	l, err := logger.NewDevelopment()
	if err != nil {
		t.Fatalf("NewDevelopment() error = %v", err)
	}
	defer l.Sync()

	if l.Level() != logger.DebugLevel {
		t.Errorf("Level() = %v, want %v", l.Level(), logger.DebugLevel)
	}
}

// TestLoggerBasicMethods 测试基础日志方法
func TestLoggerBasicMethods(t *testing.T) {
	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.DebugLevel),
		logger.WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 测试各级别日志
	l.Debug("debug message", zap.String("key", "value"))
	l.Info("info message", zap.Int("count", 42))
	l.Warn("warn message", zap.Duration("duration", time.Second))
	l.Error("error message", zap.Bool("success", false))
}

// TestLoggerWithContext 测试带 Context 的日志方法
func TestLoggerWithContext(t *testing.T) {
	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.InfoLevel),
		logger.WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 测试带 Context 的日志（使用标准 context.Context）
	ctx := context.Background()
	l.InfoContext(ctx, "user action", zap.String("action", "login"))
	l.ErrorContext(ctx, "user error", zap.String("error", "invalid password"))

	// 测试使用 qi.Context.RequestContext() 方法
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest("GET", "/test", nil)
	c := qi.NewContext(ginCtx)

	// 设置 TraceID 和 UID
	qi.SetContextTraceID(c, "trace-123")
	qi.SetContextUid(c, 12345)

	// 获取标准库 context.Context
	ctx2 := c.RequestContext()
	l.InfoContext(ctx2, "user action with context", zap.String("action", "login"))
}

// TestLoggerWith 测试创建子 Logger
func TestLoggerWith(t *testing.T) {
	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.InfoLevel),
		logger.WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 创建子 Logger
	childLogger := l.With(
		zap.String("module", "user"),
		zap.String("version", "v1"),
	)

	childLogger.Info("child logger message")
}

// TestLoggerWithContextMethod 测试创建带 Context 的子 Logger
func TestLoggerWithContextMethod(t *testing.T) {
	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.InfoLevel),
		logger.WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 测试使用 qi.Context.RequestContext() 方法
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest("GET", "/test", nil)
	c := qi.NewContext(ginCtx)

	// 设置 TraceID 和 UID
	qi.SetContextTraceID(c, "trace-456")
	qi.SetContextUid(c, 67890)

	// 获取标准库 context.Context
	ctx := c.RequestContext()

	// 创建带 Context 的子 Logger
	ctxLogger := l.WithContext(ctx)
	ctxLogger.Info("context logger message")
}

// TestSetLevel 测试动态调整级别
func TestSetLevel(t *testing.T) {
	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.InfoLevel),
		logger.WithConsoleOutput(),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 初始级别
	if l.Level() != logger.InfoLevel {
		t.Errorf("Level() = %v, want %v", l.Level(), logger.InfoLevel)
	}

	// 调整级别
	l.SetLevel(logger.DebugLevel)
	if l.Level() != logger.DebugLevel {
		t.Errorf("Level() = %v, want %v", l.Level(), logger.DebugLevel)
	}
}

// TestLevel 测试日志级别
func TestLevel(t *testing.T) {
	tests := []struct {
		level logger.Level
		want  string
	}{
		{logger.DebugLevel, "debug"},
		{logger.InfoLevel, "info"},
		{logger.WarnLevel, "warn"},
		{logger.ErrorLevel, "error"},
		{logger.DPanicLevel, "dpanic"},
		{logger.PanicLevel, "panic"},
		{logger.FatalLevel, "fatal"},
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
		format  logger.Format
		isValid bool
	}{
		{logger.JSONFormat, true},
		{logger.ConsoleFormat, true},
		{logger.Format("invalid"), false},
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
	hook := &testHook{
		onWrite: func() {
			hookCalled = true
		},
	}

	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.InfoLevel),
		logger.WithConsoleOutput(),
		logger.WithHook(hook),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 记录日志
	l.Info("test hook")

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

	l, err := logger.NewWithOptions(
		logger.WithLevel(logger.InfoLevel),
		logger.WithFormat(logger.JSONFormat),
		logger.WithFileOutput(tmpFile),
	)
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	defer l.Sync()

	// 写入日志
	l.Info("test file output", zap.String("key", "value"))

	// 验证文件存在
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}
