package middleware

import (
	"time"

	"go.uber.org/zap"
	"qi"
	"qi/pkg/logger"
)

// LoggerConfig 日志中间件配置
type LoggerConfig struct {
	// Logger 日志实例（必填）
	Logger logger.Logger

	// SkipFunc 跳过日志的函数
	SkipFunc func(c *qi.Context) bool

	// IncludeRequestBody 是否记录请求体（默认 false）
	IncludeRequestBody bool

	// IncludeResponseBody 是否记录响应体（默认 false）
	IncludeResponseBody bool

	// ExcludePaths 排除的路径（不记录日志）
	ExcludePaths []string
}

// DefaultLoggerConfig 返回默认配置
func DefaultLoggerConfig(log logger.Logger) *LoggerConfig {
	return &LoggerConfig{
		Logger:             log,
		IncludeRequestBody: false,
		IncludeResponseBody: false,
		ExcludePaths:       nil,
	}
}

// Logger 创建日志中间件
// 记录请求方法、路径、客户端 IP、状态码、耗时等信息
func Logger(log logger.Logger, cfgs ...*LoggerConfig) qi.HandlerFunc {
	cfg := DefaultLoggerConfig(log)
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	// 构建跳过路径 map
	skipMap := make(map[string]bool)
	for _, path := range cfg.ExcludePaths {
		skipMap[path] = true
	}

	return func(c *qi.Context) {
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
		query := c.Request().URL.RawQuery
		method := c.Request().Method
		clientIP := c.ClientIP()

		// 记录请求
		cfg.Logger.Info("request started",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", c.Request().UserAgent()),
		)

		c.Next()

		// 计算耗时
		latency := time.Since(start)
		status := c.Writer().Status()

		// 根据状态码选择日志级别
		if status >= 500 {
			cfg.Logger.Error("request completed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("client_ip", clientIP),
			)
		} else if status >= 400 {
			cfg.Logger.Warn("request completed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("client_ip", clientIP),
			)
		} else {
			cfg.Logger.Info("request completed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("client_ip", clientIP),
			)
		}
	}
}

// defaultLogger 默认日志中间件
// 使用 logger 包，自动创建默认配置（输出到终端）
func defaultLogger() qi.HandlerFunc {
	// 自动创建默认日志实例
	log, _ := logger.NewDevelopment()

	return Logger(log)
}
