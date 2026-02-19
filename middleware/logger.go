package middleware

import (
	"strconv"
	"time"

	"github.com/tokmz/qi"
	"github.com/tokmz/qi/pkg/logger"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// LoggerConfig 日志中间件配置
type LoggerConfig struct {
	// Logger 日志实例（必填）
	Logger logger.Logger

	// SkipFunc 跳过日志的函数
	SkipFunc func(c *qi.Context) bool

	// ExcludePaths 排除的路径（不记录日志）
	ExcludePaths []string
}

// Logger 创建日志中间件
// 记录请求方法、路径、客户端 IP、状态码、耗时、TraceID、SpanID 等信息
func Logger(log logger.Logger, cfgs ...*LoggerConfig) qi.HandlerFunc {
	cfg := &LoggerConfig{
		Logger: log,
	}
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
		if cfg.Logger == nil {
			cfg.Logger = log
		}
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
			zap.String("latency", formatLatency(latency)),
			zap.String("client_ip", clientIP),
		}

		// 从请求 context 中提取链路追踪信息
		spanCtx := trace.SpanFromContext(c.RequestContext()).SpanContext()
		if spanCtx.IsValid() {
			fields = append(fields,
				zap.String("trace_id", spanCtx.TraceID().String()),
				zap.String("span_id", spanCtx.SpanID().String()),
			)
		} else if traceID := qi.GetContextTraceID(c); traceID != "" {
			// 回退：从 qi.Context 获取 TraceID
			fields = append(fields, zap.String("trace_id", traceID))
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

// formatLatency 将耗时格式化为人类可读的字符串
func formatLatency(d time.Duration) string {
	switch {
	case d >= time.Second:
		return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + "s"
	case d >= time.Millisecond:
		return strconv.FormatFloat(float64(d)/float64(time.Millisecond), 'f', 2, 64) + "ms"
	default:
		return strconv.FormatFloat(float64(d)/float64(time.Microsecond), 'f', 2, 64) + "µs"
	}
}
