package middleware

import (
	"fmt"

	"qi/internal/core/tracing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware Gin 框架的链路追踪中间件
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := tracing.GetGlobal()
		if tracer == nil || !tracer.IsEnabled() {
			c.Next()
			return
		}

		// 从请求头提取 trace context
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		ctx := tracing.ExtractHTTPHeaders(c.Request.Context(), headers)

		// 构建 span 名称
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if c.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		// 创建 span
		attrs := []attribute.KeyValue{
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPURL(c.Request.URL.String()),
			semconv.HTTPRoute(c.FullPath()),
			semconv.HTTPScheme(c.Request.URL.Scheme),
			semconv.HTTPTarget(c.Request.URL.Path),
			semconv.NetHostName(c.Request.Host),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		}

		// 添加端口信息（如果存在）
		if port := c.Request.URL.Port(); port != "" {
			attrs = append(attrs, attribute.String("net.host.port", port))
		}

		ctx, span := tracer.GetTracer().Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		// 将 context 注入到 gin.Context
		c.Request = c.Request.WithContext(ctx)

		// 将 trace ID 和 span ID 添加到响应头（方便调试）
		c.Header("X-Trace-ID", span.SpanContext().TraceID().String())
		c.Header("X-Span-ID", span.SpanContext().SpanID().String())

		// 处理请求
		c.Next()

		// 记录响应状态码
		statusCode := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		// 根据状态码设置 span 状态
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// 如果有错误，记录错误信息
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}
	}
}

// TracingMiddlewareWithConfig 使用自定义配置的 Gin 链路追踪中间件
func TracingMiddlewareWithConfig(config TracingMiddlewareConfig) gin.HandlerFunc {
	if config.Skipper == nil {
		config.Skipper = defaultSkipper
	}

	if config.SpanNameFormatter == nil {
		config.SpanNameFormatter = defaultSpanNameFormatter
	}

	return func(c *gin.Context) {
		// 跳过检查
		if config.Skipper(c) {
			c.Next()
			return
		}

		tracer := tracing.GetGlobal()
		if tracer == nil || !tracer.IsEnabled() {
			c.Next()
			return
		}

		// 提取 trace context
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		ctx := tracing.ExtractHTTPHeaders(c.Request.Context(), headers)

		// 创建 span
		spanName := config.SpanNameFormatter(c)
		ctx, span := tracer.GetTracer().Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// 添加基础属性
		span.SetAttributes(
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPURL(c.Request.URL.String()),
			semconv.HTTPRoute(c.FullPath()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		)

		// 添加自定义属性
		if config.AttributesExtractor != nil {
			for _, attr := range config.AttributesExtractor(c) {
				span.SetAttributes(attr)
			}
		}

		c.Request = c.Request.WithContext(ctx)

		// 添加 trace 信息到响应头
		if config.IncludeTraceHeaders {
			c.Header("X-Trace-ID", span.SpanContext().TraceID().String())
			c.Header("X-Span-ID", span.SpanContext().SpanID().String())
		}

		c.Next()

		// 记录响应状态
		statusCode := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// 记录错误
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}
	}
}

// TracingMiddlewareConfig Gin 链路追踪中间件配置
type TracingMiddlewareConfig struct {
	// Skipper 定义跳过中间件的函数
	Skipper func(*gin.Context) bool

	// SpanNameFormatter 自定义 span 名称格式化函数
	SpanNameFormatter func(*gin.Context) string

	// AttributesExtractor 提取自定义属性的函数
	AttributesExtractor func(*gin.Context) []attribute.KeyValue

	// IncludeTraceHeaders 是否在响应头中包含 trace 信息
	IncludeTraceHeaders bool
}

// defaultSkipper 默认的跳过函数
func defaultSkipper(c *gin.Context) bool {
	// 跳过健康检查等路径
	path := c.Request.URL.Path
	return path == "/health" || path == "/metrics" || path == "/ping"
}

// defaultSpanNameFormatter 默认的 span 名称格式化函数
func defaultSpanNameFormatter(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}
	return fmt.Sprintf("%s %s", c.Request.Method, route)
}

// HTTPClientMiddleware HTTP 客户端中间件示例
// 用于在发送 HTTP 请求时自动注入 trace context
func HTTPClientMiddleware() func(req interface{}) {
	return func(req interface{}) {
		// 这里是示例，实际使用时需要根据具体的 HTTP 客户端实现
		// 例如使用 go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
	}
}

// GetTraceIDFromGin 从 Gin Context 获取 Trace ID
func GetTraceIDFromGin(c *gin.Context) string {
	return tracing.GetTraceID(c.Request.Context())
}

// GetSpanIDFromGin 从 Gin Context 获取 Span ID
func GetSpanIDFromGin(c *gin.Context) string {
	return tracing.GetSpanID(c.Request.Context())
}

// StartSpanFromGin 从 Gin Context 开始一个新的 span
func StartSpanFromGin(c *gin.Context, spanName string, opts ...tracing.SpanOption) trace.Span {
	ctx, span := tracing.StartSpan(c.Request.Context(), spanName, opts...)
	c.Request = c.Request.WithContext(ctx)
	return span
}

// RecordErrorToGin 记录错误到 Gin Context 的 span
func RecordErrorToGin(c *gin.Context, err error) {
	tracing.RecordError(c.Request.Context(), err)
}

// AddEventToGin 添加事件到 Gin Context 的 span
func AddEventToGin(c *gin.Context, name string, attrs ...attribute.KeyValue) {
	tracing.AddEvent(c.Request.Context(), name, attrs...)
}

// SetAttributesToGin 设置属性到 Gin Context 的 span
func SetAttributesToGin(c *gin.Context, attrs ...attribute.KeyValue) {
	tracing.SetAttributes(c.Request.Context(), attrs...)
}
