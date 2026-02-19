package middleware

import (
	"fmt"
	"strings"

	"github.com/tokmz/qi"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig 链路追踪中间件配置
type TracingConfig struct {
	// TracerName Tracer 名称（默认 "qi.http"）
	TracerName string

	// SpanNameFormatter 自定义 Span 名称格式
	SpanNameFormatter func(c *qi.Context) string

	// Filter 过滤不需要追踪的请求
	// 返回 true 表示需要追踪，false 表示跳过
	Filter func(c *qi.Context) bool

	// ExcludePaths 排除的路径（不追踪）
	ExcludePaths []string
}

// DefaultTracingConfig 返回默认配置
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		TracerName: "qi.http",
		SpanNameFormatter: func(c *qi.Context) string {
			return fmt.Sprintf("%s %s", c.Request().Method, c.FullPath())
		},
	}
}

// Tracing 创建链路追踪中间件
// 自动提取/注入 TraceContext，创建 HTTP Server Span
// OTel 会自动生成 TraceID 和 SpanID
func Tracing(cfgs ...*TracingConfig) qi.HandlerFunc {
	cfg := DefaultTracingConfig()
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
		if cfg.Filter != nil && !cfg.Filter(c) {
			c.Next()
			return
		}
		if skipMap[c.Request().URL.Path] {
			c.Next()
			return
		}

		// 每次请求时获取 tracer 和 propagator，避免 Provider 后初始化导致使用 noop
		tracer := otel.Tracer(cfg.TracerName)
		propagator := otel.GetTextMapPropagator()

		// 从 HTTP Header 提取上游 TraceContext（如果有）
		ctx := propagator.Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

		// 创建 Server Span（OTel 自动生成 TraceID/SpanID）
		spanName := cfg.SpanNameFormatter(c)
		if spanName == "" || strings.HasSuffix(spanName, " ") {
			// FullPath() 未匹配路由时返回空字符串，回退到 URL.Path
			spanName = fmt.Sprintf("%s %s", c.Request().Method, c.Request().URL.Path)
		}
		spanAttrs := []attribute.KeyValue{
			semconv.HTTPRequestMethodKey.String(c.Request().Method),
			semconv.URLPath(c.Request().URL.Path),
			semconv.ServerAddress(c.Request().Host),
			semconv.UserAgentOriginalKey.String(c.Request().UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		}
		if fullPath := c.FullPath(); fullPath != "" {
			spanAttrs = append(spanAttrs, semconv.HTTPRouteKey.String(fullPath))
		}
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(spanAttrs...),
		)
		defer span.End()

		// 将 OTel 生成的 TraceID 同步到 qi.Context
		traceID := span.SpanContext().TraceID().String()
		qi.SetContextTraceID(c, traceID)

		// 将 SpanContext 注入到 Request.Context
		c.SetRequestContext(ctx)

		// 执行后续处理
		c.Next()

		// 记录响应状态
		statusCode := c.Writer().Status()
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(statusCode))

		if statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		}

		// 将 TraceContext 注入到响应头（供下游服务使用）
		propagator.Inject(ctx, propagation.HeaderCarrier(c.Writer().Header()))
	}
}
