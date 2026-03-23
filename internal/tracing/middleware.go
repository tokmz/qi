package tracing

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const httpTracerName = "qi.http"

// sensitiveHeaders 记录 header 时需过滤的敏感字段（小写）
var sensitiveHeaders = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
	"x-auth-token":  {},
	"x-api-key":     {},
}

// Middleware 返回链路追踪 gin.HandlerFunc。
//
// 职责：
//  1. 从请求头提取上游 trace context（W3C traceparent）
//  2. 创建 server root span，先以 method 命名，Next() 后更新为路由模板（避免高基数）
//  3. 将 trace_id 写入 gin.Context（key="trace_id"），qi 响应自动填充
//  4. 将含 span 的 context 注入请求，供 db/cache 插件读取
//  5. ≥500 状态码标记 span 为 Error
func Middleware(cfg *Config) gin.HandlerFunc {
	if cfg == nil {
		cfg = &Config{}
	}

	skipPaths := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = struct{}{}
	}

	spanNameFunc := cfg.SpanNameFunc
	if spanNameFunc == nil {
		spanNameFunc = func(method, route string) string {
			return method + " " + route
		}
	}

	return func(c *gin.Context) {
		req := c.Request

		// 跳过指定路径
		if _, skip := skipPaths[req.URL.Path]; skip {
			c.Next()
			return
		}

		// 提取上游 trace context
		ctx := otel.GetTextMapPropagator().Extract(
			req.Context(),
			propagation.HeaderCarrier(req.Header),
		)

		// 构建基础 attributes
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		attrs := []attribute.KeyValue{
			semconv.HTTPRequestMethodKey.String(req.Method),
			semconv.URLPath(req.URL.Path),
			semconv.ServerAddress(req.Host),
			semconv.URLScheme(scheme),
		}
		if req.URL.RawQuery != "" {
			attrs = append(attrs, semconv.URLQuery(req.URL.RawQuery))
		}
		if cfg.RecordHeaders {
			for k, vs := range req.Header {
				if _, sensitive := sensitiveHeaders[strings.ToLower(k)]; sensitive {
					continue
				}
				attrs = append(attrs, attribute.StringSlice("http.request.header."+strings.ToLower(k), vs))
			}
		}

		// 先以 method 作为临时 span name，Next() 后更新为路由模板
		ctx, span := otel.Tracer(httpTracerName).Start(ctx, req.Method,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		// 写入 trace_id，qi.Context.respond() 自动读取填充响应 JSON
		if sc := span.SpanContext(); sc.IsValid() {
			c.Set("trace_id", sc.TraceID().String())
		}

		// 注入含 span 的 context
		c.Request = req.WithContext(ctx)

		c.Next()

		// Next() 后路由模板已确定，更新 span name
		if route := c.FullPath(); route != "" {
			span.SetName(spanNameFunc(req.Method, route))
			span.SetAttributes(semconv.HTTPRoute(route))
		}
		// 404 无路由时保持 "METHOD"，符合 OTel 规范

		status := c.Writer.Status()
		span.SetAttributes(semconv.HTTPResponseStatusCode(status))
		if status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
