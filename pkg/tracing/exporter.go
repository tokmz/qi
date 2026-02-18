package tracing

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

// newExporter 根据配置创建导出器
func newExporter(ctx context.Context, cfg *Config) (trace.SpanExporter, error) {
	switch cfg.ExporterType {
	case "otlp":
		return newOTLPExporter(ctx, cfg)
	case "stdout":
		return newStdoutExporter()
	case "noop":
		return newNoopExporter(), nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}
}

// newOTLPExporter 创建 OTLP HTTP 导出器
func newOTLPExporter(ctx context.Context, cfg *Config) (trace.SpanExporter, error) {
	opts := []otlptracehttp.Option{}

	// 设置端点（配置优先，环境变量次之）
	endpoint := cfg.ExporterEndpoint
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
	}

	// 非 TLS 连接
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	// 设置请求头（用于认证）
	if len(cfg.ExporterHeaders) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.ExporterHeaders))
	}

	return otlptracehttp.New(ctx, opts...)
}

// newStdoutExporter 创建标准输出导出器（用于开发调试）
func newStdoutExporter() (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
}

// newNoopExporter 创建空导出器（禁用追踪）
func newNoopExporter() trace.SpanExporter {
	return &noopExporter{}
}

// noopExporter 空导出器实现
type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}
