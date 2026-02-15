package qi

import (
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"go.uber.org/zap"
	"github.com/tokmz/qi/pkg/logger"
)

// LoggerConfig 日志中间件配置
type LoggerConfig struct {
	// Logger 日志实例（必填）
	Logger logger.Logger

	// SkipFunc 跳过日志的函数
	SkipFunc func(c *Context) bool

	// ExcludePaths 排除的路径（不记录日志）
	ExcludePaths []string
}

// defaultLoggerConfig 返回默认配置
func defaultLoggerConfig(log logger.Logger) *LoggerConfig {
	return &LoggerConfig{
		Logger:       log,
		ExcludePaths: nil,
	}
}

// Logger 创建日志中间件
// 记录请求方法、路径、客户端 IP、状态码、耗时等信息
func Logger(log logger.Logger, cfgs ...*LoggerConfig) HandlerFunc {
	cfg := defaultLoggerConfig(log)
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	// 构建跳过路径 map
	skipMap := make(map[string]bool)
	for _, path := range cfg.ExcludePaths {
		skipMap[path] = true
	}

	return func(c *Context) {
		// 检查是否跳过
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}
		if skipMap[c.Request().URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request().URL.Path
		method := c.Request().Method
		clientIP := c.ClientIP()

		c.Next()

		// 计算耗时
		latency := time.Since(start)
		status := c.Writer().Status()

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		}

		// 根据状态码选择日志级别
		if status >= 500 {
			cfg.Logger.Error("request", fields...)
		} else if status >= 400 {
			cfg.Logger.Warn("request", fields...)
		} else {
			cfg.Logger.Info("request", fields...)
		}
	}
}

// defaultLogger 创建默认日志中间件（无需配置）
func defaultLogger() HandlerFunc {
	// 自动创建默认日志实例
	log, err := logger.NewDevelopment()
	if err != nil {
		panic("qi: failed to create default logger: " + err.Error())
	}
	return Logger(log)
}

// Recovery 创建 panic 恢复中间件
// panic 时返回 qi 统一响应格式（500），并记录错误日志
func Recovery(logs ...logger.Logger) HandlerFunc {
	var log logger.Logger
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	} else {
		var err error
		log, err = logger.NewDevelopment()
		if err != nil {
			panic("qi: failed to create recovery logger: " + err.Error())
		}
	}

	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否为断开的连接（客户端主动断开）
				if isBrokenPipe(err) {
					log.Error("broken pipe",
						zap.Any("error", err),
						zap.String("path", c.Request().URL.Path),
					)
					c.Abort()
					return
				}

				// 获取堆栈信息
				stack := string(debug.Stack())

				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request().Method),
					zap.String("path", c.Request().URL.Path),
					zap.String("client_ip", c.ClientIP()),
					zap.String("stack", stack),
				)

				c.Fail(http.StatusInternalServerError, "Internal Server Error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// isBrokenPipe 检查是否为断开的连接错误
func isBrokenPipe(err any) bool {
	ne, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	se, ok := ne.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	msg := strings.ToLower(se.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}
