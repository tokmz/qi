package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Carrier 用于传播 trace context 的载体接口
// 实现了 propagation.TextMapCarrier 接口
type Carrier map[string]string

// Get 获取键对应的值
func (c Carrier) Get(key string) string {
	return c[key]
}

// Set 设置键值对
func (c Carrier) Set(key, value string) {
	c[key] = value
}

// Keys 返回所有的键
func (c Carrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectContext 将 trace context 注入到 carrier 中
// 用于跨进程/服务传播 trace context
func InjectContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, carrier)
}

// ExtractContext 从 carrier 中提取 trace context
// 用于接收跨进程/服务的 trace context
func ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, carrier)
}

// InjectHTTPHeaders 将 trace context 注入到 HTTP 请求头
func InjectHTTPHeaders(ctx context.Context, headers map[string]string) {
	carrier := Carrier(headers)
	InjectContext(ctx, carrier)
}

// ExtractHTTPHeaders 从 HTTP 请求头提取 trace context
func ExtractHTTPHeaders(ctx context.Context, headers map[string]string) context.Context {
	carrier := Carrier(headers)
	return ExtractContext(ctx, carrier)
}

// InjectGRPCMetadata 将 trace context 注入到 gRPC metadata
func InjectGRPCMetadata(ctx context.Context, md map[string]string) {
	carrier := Carrier(md)
	InjectContext(ctx, carrier)
}

// ExtractGRPCMetadata 从 gRPC metadata 提取 trace context
func ExtractGRPCMetadata(ctx context.Context, md map[string]string) context.Context {
	carrier := Carrier(md)
	return ExtractContext(ctx, carrier)
}

// NewCarrier 创建新的 Carrier
func NewCarrier() Carrier {
	return make(Carrier)
}

// ContextWithSpan 将 span 放入 context
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// SpanContextFromContext 从 context 获取 SpanContext
func SpanContextFromContext(ctx context.Context) trace.SpanContext {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return trace.SpanContext{}
	}
	return span.SpanContext()
}

// IsValidSpanContext 检查 SpanContext 是否有效
func IsValidSpanContext(ctx context.Context) bool {
	sc := SpanContextFromContext(ctx)
	return sc.IsValid()
}

// GetPropagator 获取全局的 TextMapPropagator
func GetPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// SetPropagator 设置全局的 TextMapPropagator
func SetPropagator(propagator propagation.TextMapPropagator) {
	otel.SetTextMapPropagator(propagator)
}

// NewCompositePropagator 创建组合的 Propagator
// 支持 W3C Trace Context 和 Baggage
func NewCompositePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context
		propagation.Baggage{},      // W3C Baggage
	)
}
