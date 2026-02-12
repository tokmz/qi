package tracing

import (
	"fmt"

	"qi"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "qi.http"
)

// middlewareConfig 中间件配置
type middlewareConfig struct {
	tracerName        string
	spanNameFormatter func(*qi.Context) string
	filter            func(*qi.Context) bool
}

// MiddlewareOption 中间件选项
type MiddlewareOption func(*middlewareConfig)

// WithTracerName 设置 Tracer 名称（默认 "qi.http"）
func WithTracerName(name string) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.tracerName = name
	}
}

// WithSpanNameFormatter 自定义 Span 名称格式
func WithSpanNameFormatter(fn func(*qi.Context) string) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.spanNameFormatter = fn
	}
}

// WithFilter 过滤不需要追踪的请求（如健康检查）
// 返回 true 表示需要追踪，false 表示跳过
func WithFilter(fn func(*qi.Context) bool) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.filter = fn
	}
}

// Middleware 创建链路追踪中间件
// 自动提取/注入 TraceContext，创建 Root Span
func Middleware(opts ...MiddlewareOption) qi.HandlerFunc {
	// 默认配置
	cfg := &middlewareConfig{
		tracerName: tracerName,
		spanNameFormatter: func(c *qi.Context) string {
			return fmt.Sprintf("%s %s", c.Request().Method, c.Request().URL.Path)
		},
		filter: func(c *qi.Context) bool {
			return true // 默认追踪所有请求
		},
	}

	// 应用选项
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *qi.Context) {
		// 过滤不需要追踪的请求
		if !cfg.filter(c) {
			c.Next()
			return
		}

		// 每次请求时获取 tracer 和 propagator，避免 Provider 后初始化导致使用 noop
		tracer := otel.Tracer(cfg.tracerName)
		propagator := otel.GetTextMapPropagator()

		// 从 HTTP Header 提取 TraceContext
		ctx := propagator.Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

		// 创建 Root Span
		spanName := cfg.spanNameFormatter(c)
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request().Method),
				semconv.URLFull(c.Request().URL.String()),
				semconv.HTTPRouteKey.String(c.FullPath()), // 使用路由模板避免高基数
				semconv.URLScheme(c.Request().URL.Scheme),
				semconv.URLPath(c.Request().URL.Path),
				semconv.ServerAddress(c.Request().Host),
				semconv.UserAgentOriginalKey.String(c.Request().UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// 将 TraceID 同步到 qi.Context（保持与现有机制兼容）
		traceID := span.SpanContext().TraceID().String()
		qi.SetContextTraceID(c, traceID)

		// 将 SpanContext 注入到 Request.Context
		c.SetRequestContext(ctx)

		// 执行后续处理
		c.Next()

		// 记录响应状态
		statusCode := c.Writer().Status()
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(statusCode))

		// 如果是 5xx 错误，标记 Span 状态为错误
		if statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		}

		// 将 TraceContext 注入到响应头
		propagator.Inject(ctx, propagation.HeaderCarrier(c.Writer().Header()))
	}
}
