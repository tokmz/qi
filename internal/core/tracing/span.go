package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanOption Span 配置选项
type SpanOption func(*spanConfig)

type spanConfig struct {
	attributes []attribute.KeyValue
	spanKind   trace.SpanKind
}

// WithAttributes 添加属性
func WithAttributes(attrs ...attribute.KeyValue) SpanOption {
	return func(cfg *spanConfig) {
		cfg.attributes = append(cfg.attributes, attrs...)
	}
}

// WithSpanKind 设置 Span 类型
func WithSpanKind(kind trace.SpanKind) SpanOption {
	return func(cfg *spanConfig) {
		cfg.spanKind = kind
	}
}

// StartSpan 开始一个新的 span
func StartSpan(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, trace.Span) {
	tracer := GetGlobal()
	if tracer == nil || !tracer.IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}

	cfg := &spanConfig{
		spanKind: trace.SpanKindInternal,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(cfg.spanKind),
	}

	if len(cfg.attributes) > 0 {
		spanOpts = append(spanOpts, trace.WithAttributes(cfg.attributes...))
	}

	return tracer.GetTracer().Start(ctx, spanName, spanOpts...)
}

// StartSpanFromContext 从上下文开始一个新的 span
func StartSpanFromContext(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, trace.Span) {
	return StartSpan(ctx, spanName, opts...)
}

// EndSpan 结束 span
func EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// RecordError 记录错误到 span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanStatus 设置 span 状态
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetStatus(code, description)
	}
}

// AddEvent 添加事件到 span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetAttributes 设置 span 属性
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// GetSpan 从上下文获取当前 span
func GetSpan(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// GetTraceID 从上下文获取 Trace ID
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 从上下文获取 Span ID
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsRecording 检查 span 是否正在记录
func IsRecording(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	return span != nil && span.IsRecording()
}

// SpanWrapper 用于包装函数执行并自动创建 span
func SpanWrapper(ctx context.Context, spanName string, fn func(context.Context) error, opts ...SpanOption) error {
	ctx, span := StartSpan(ctx, spanName, opts...)
	defer EndSpan(span)

	err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
	}

	return err
}

// AsyncSpanWrapper 异步函数的 span 包装器
func AsyncSpanWrapper(ctx context.Context, spanName string, fn func(context.Context), opts ...SpanOption) {
	ctx, span := StartSpan(ctx, spanName, opts...)

	go func() {
		defer EndSpan(span)
		fn(ctx)
	}()
}

// 常用的属性键
var (
	// HTTP 相关
	HTTPMethodKey     = attribute.Key("http.method")
	HTTPURLKey        = attribute.Key("http.url")
	HTTPStatusCodeKey = attribute.Key("http.status_code")
	HTTPRouteKey      = attribute.Key("http.route")
	HTTPUserAgentKey  = attribute.Key("http.user_agent")
	HTTPClientIPKey   = attribute.Key("http.client_ip")

	// 数据库相关
	DBSystemKey    = attribute.Key("db.system")
	DBNameKey      = attribute.Key("db.name")
	DBStatementKey = attribute.Key("db.statement")
	DBOperationKey = attribute.Key("db.operation")
	DBTableKey     = attribute.Key("db.table")

	// 缓存相关
	CacheSystemKey = attribute.Key("cache.system")
	CacheKeyKey    = attribute.Key("cache.key")
	CacheHitKey    = attribute.Key("cache.hit")

	// 消息队列相关
	MessagingSystemKey      = attribute.Key("messaging.system")
	MessagingOperationKey   = attribute.Key("messaging.operation")
	MessagingDestinationKey = attribute.Key("messaging.destination")

	// 业务相关
	UserIDKey      = attribute.Key("user.id")
	UserNameKey    = attribute.Key("user.name")
	TenantIDKey    = attribute.Key("tenant.id")
	RequestIDKey   = attribute.Key("request.id")
	CorrelationKey = attribute.Key("correlation.id")
)

// 辅助函数：创建常用属性

// HTTPAttributes 创建 HTTP 相关属性
func HTTPAttributes(method, url, route string, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		HTTPMethodKey.String(method),
		HTTPURLKey.String(url),
		HTTPRouteKey.String(route),
		HTTPStatusCodeKey.Int(statusCode),
	}
}

// DBAttributes 创建数据库相关属性
func DBAttributes(system, name, operation, table, statement string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		DBSystemKey.String(system),
		DBNameKey.String(name),
		DBOperationKey.String(operation),
	}

	if table != "" {
		attrs = append(attrs, DBTableKey.String(table))
	}

	if statement != "" {
		attrs = append(attrs, DBStatementKey.String(statement))
	}

	return attrs
}

// CacheAttributes 创建缓存相关属性
func CacheAttributes(system, key string, hit bool) []attribute.KeyValue {
	return []attribute.KeyValue{
		CacheSystemKey.String(system),
		CacheKeyKey.String(key),
		CacheHitKey.Bool(hit),
	}
}

// UserAttributes 创建用户相关属性
func UserAttributes(userID, userName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}

	if userID != "" {
		attrs = append(attrs, UserIDKey.String(userID))
	}

	if userName != "" {
		attrs = append(attrs, UserNameKey.String(userName))
	}

	return attrs
}

// ErrorAttributes 创建错误相关属性
func ErrorAttributes(err error) []attribute.KeyValue {
	if err == nil {
		return nil
	}

	return []attribute.KeyValue{
		attribute.String("error", "true"),
		attribute.String("error.type", fmt.Sprintf("%T", err)),
		attribute.String("error.message", err.Error()),
	}
}
