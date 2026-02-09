package middleware

import (
	"time"

	"qi"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerConfig 日志中间件配置
type LoggerConfig struct {
	// Logger zap logger 实例
	Logger *zap.Logger

	// SkipPaths 跳过日志记录的路径
	SkipPaths []string

	// SkipPathPrefixes 跳过日志记录的路径前缀
	SkipPathPrefixes []string
}

// Logger 返回默认日志中间件（使用 zap）
func Logger() qi.HandlerFunc {
	return LoggerWithConfig(LoggerConfig{})
}

// LoggerWithConfig 返回带配置的日志中间件
func LoggerWithConfig(config LoggerConfig) qi.HandlerFunc {
	// 如果没有提供 logger，创建默认的
	logger := config.Logger
	if logger == nil {
		logger = newDefaultLogger()
	}

	// 构建跳过路径的 map
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *qi.Context) {
		// 开始时间
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 跳过指定路径
		if skipPaths[path] {
			return
		}

		// 跳过指定前缀
		for _, prefix := range config.SkipPathPrefixes {
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				return
			}
		}

		// 计算耗时
		latency := time.Since(start)

		// 拼接完整路径
		if raw != "" {
			path = path + "?" + raw
		}

		// 构建日志字段
		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// 添加 TraceID（如果存在）
		if traceID := qi.GetContextTraceID(c); traceID != "" {
			fields = append(fields, zap.String("trace_id", traceID))
		}

		// 添加用户 ID（如果存在）
		if uid := qi.GetContextUid(c); uid != 0 {
			fields = append(fields, zap.Int64("uid", uid))
		}

		// 添加错误信息（如果有）
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		// 根据状态码选择日志级别
		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 400:
			logger.Warn("Client error", fields...)
		case statusCode >= 300:
			logger.Info("Redirection", fields...)
		default:
			logger.Info("Success", fields...)
		}
	}
}

// newDefaultLogger 创建默认的 zap logger（开发模式，带颜色）
func newDefaultLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006/01/02 - 15:04:05")

	logger, _ := config.Build()
	return logger
}

// NewProductionLogger 创建生产环境 logger（JSON 格式）
func NewProductionLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, _ := config.Build()
	return logger
}

// NewDevelopmentLogger 创建开发环境 logger（控制台格式，带颜色）
func NewDevelopmentLogger() *zap.Logger {
	return newDefaultLogger()
}
