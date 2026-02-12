package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTracerName = "qi.tracing"
)

// StartSpan 从 context.Context 启动新 Span
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(defaultTracerName)
	return tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext 从 context.Context 获取当前 Span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan 将 Span 注入 context.Context
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// RecordError 记录错误到 Span
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetAttributes 批量设置 Span 属性
func SetAttributes(span trace.Span, attrs map[string]any) {
	if len(attrs) == 0 {
		return
	}

	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kvs = append(kvs, convertToAttribute(k, v))
	}
	span.SetAttributes(kvs...)
}

// AddEvent 添加 Span 事件
func AddEvent(span trace.Span, name string, attrs map[string]any) {
	if len(attrs) == 0 {
		span.AddEvent(name)
		return
	}

	kvs := make([]trace.EventOption, 0, len(attrs))
	for k, v := range attrs {
		kvs = append(kvs, trace.WithAttributes(convertToAttribute(k, v)))
	}
	span.AddEvent(name, kvs...)
}

// convertToAttribute 将 interface{} 转换为 attribute.KeyValue
func convertToAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	default:
		// 默认转换为字符串
		return attribute.String(key, fmt.Sprint(v))
	}
}
