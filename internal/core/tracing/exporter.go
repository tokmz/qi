package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createExporter 根据配置创建相应的导出器
func (t *Tracer) createExporter() (trace.SpanExporter, error) {
	switch t.config.Exporter.Type {
	case "otlp":
		return t.createOTLPExporter()
	case "stdout":
		return t.createStdoutExporter()
	case "jaeger":
		// Jaeger 使用 OTLP 协议
		return t.createJaegerExporter()
	case "zipkin":
		// Zipkin 需要单独的包，这里暂不实现
		return nil, fmt.Errorf("zipkin exporter not implemented yet")
	default:
		return nil, ErrUnsupportedExporter
	}
}

// createOTLPExporter 创建 OTLP 导出器
func (t *Tracer) createOTLPExporter() (trace.SpanExporter, error) {
	cfg := t.config.Exporter.OTLP

	// 根据协议类型创建不同的客户端
	switch cfg.Protocol {
	case "grpc":
		return t.createOTLPGRPCExporter()
	case "http", "":
		return t.createOTLPHTTPExporter()
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s", cfg.Protocol)
	}
}

// createOTLPHTTPExporter 创建 OTLP HTTP 导出器
func (t *Tracer) createOTLPHTTPExporter() (trace.SpanExporter, error) {
	cfg := t.config.Exporter.OTLP

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithTimeout(cfg.Timeout),
	}

	// 是否使用不安全连接
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	// 添加自定义请求头
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	// 压缩方式
	switch cfg.Compression {
	case "gzip":
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	case "none", "":
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
	}

	return otlptracehttp.New(context.Background(), opts...)
}

// createOTLPGRPCExporter 创建 OTLP gRPC 导出器
func (t *Tracer) createOTLPGRPCExporter() (trace.SpanExporter, error) {
	cfg := t.config.Exporter.OTLP

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(cfg.Timeout),
	}

	// 是否使用不安全连接
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
		opts = append(opts, otlptracegrpc.WithDialOption(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		))
	}

	// 添加自定义请求头
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	// 压缩方式
	switch cfg.Compression {
	case "gzip":
		opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
	}

	return otlptracegrpc.New(context.Background(), opts...)
}

// createJaegerExporter 创建 Jaeger 导出器（通过 OTLP）
func (t *Tracer) createJaegerExporter() (trace.SpanExporter, error) {
	cfg := t.config.Exporter.Jaeger

	var opts []otlptracegrpc.Option

	// 优先使用 Collector 端点
	if cfg.CollectorEndpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint))
	} else if cfg.AgentEndpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.AgentEndpoint))
	}

	// 添加认证信息
	if cfg.Username != "" && cfg.Password != "" {
		headers := map[string]string{
			"Authorization": fmt.Sprintf("Basic %s:%s", cfg.Username, cfg.Password),
		}
		opts = append(opts, otlptracegrpc.WithHeaders(headers))
	}

	opts = append(opts, otlptracegrpc.WithInsecure())

	return otlptracegrpc.New(context.Background(), opts...)
}

// createStdoutExporter 创建标准输出导出器（用于调试）
func (t *Tracer) createStdoutExporter() (trace.SpanExporter, error) {
	cfg := t.config.Exporter.Stdout

	opts := []stdouttrace.Option{}

	if cfg.PrettyPrint {
		opts = append(opts, stdouttrace.WithPrettyPrint())
	}

	return stdouttrace.New(opts...)
}

// GetExporter 获取当前使用的导出器类型
func (t *Tracer) GetExporter() string {
	return t.config.Exporter.Type
}
