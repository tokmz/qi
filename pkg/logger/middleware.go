package logger

import (
	"time"

	"qi"

	"go.uber.org/zap"
)

// Middleware 创建日志中间件
func Middleware(logger Logger) qi.HandlerFunc {
	return func(c *qi.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 记录请求日志
		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		}

		// 使用标准库 context.Context 获取上下文
		ctx := c.RequestContext()

		// 根据状态码选择日志级别
		switch {
		case status >= 500:
			logger.ErrorContext(ctx, "HTTP Request", fields...)
		case status >= 400:
			logger.WarnContext(ctx, "HTTP Request", fields...)
		default:
			logger.InfoContext(ctx, "HTTP Request", fields...)
		}
	}
}
