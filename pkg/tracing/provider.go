package tracing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	globalProvider *trace.TracerProvider
	providerMu     sync.Mutex
)

// NewTracerProvider 创建并初始化 TracerProvider
func NewTracerProvider(cfg *Config) (*trace.TracerProvider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 如果禁用追踪，返回 noop provider
	if !cfg.Enabled {
		cfg.ExporterType = "noop"
	}

	// 创建导出器
	ctx := context.Background()
	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// 创建资源
	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建采样器
	sampler := newSampler(cfg)

	// 创建批处理器
	batchProcessor := trace.NewBatchSpanProcessor(
		exporter,
		trace.WithBatchTimeout(cfg.BatchTimeout),
		trace.WithMaxExportBatchSize(cfg.MaxExportBatchSize),
		trace.WithMaxQueueSize(cfg.MaxQueueSize),
	)

	// 创建 TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithSampler(sampler),
		trace.WithSpanProcessor(batchProcessor),
		trace.WithResource(res),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 设置全局 Propagator（W3C Trace Context）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 保存全局引用
	providerMu.Lock()
	globalProvider = tp
	providerMu.Unlock()

	return tp, nil
}

// newResource 创建资源（包含服务信息和自定义属性）
func newResource(cfg *Config) (*resource.Resource, error) {
	// 基础属性
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	}

	// 添加环境标识
	if cfg.Environment != "" {
		attrs = append(attrs, resource.WithAttributes(
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		))
	}

	// 添加自定义属性
	if len(cfg.ResourceAttributes) > 0 {
		customAttrs := make([]attribute.KeyValue, 0, len(cfg.ResourceAttributes))
		for k, v := range cfg.ResourceAttributes {
			customAttrs = append(customAttrs, attribute.String(k, v))
		}
		attrs = append(attrs, resource.WithAttributes(customAttrs...))
	}

	// 从环境变量读取资源属性
	if envAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); envAttrs != "" {
		attrs = append(attrs, resource.WithAttributes(parseResourceAttributes(envAttrs)...))
	}

	// 合并默认资源（包含 host、process 等信息）
	attrs = append(attrs, resource.WithFromEnv(), resource.WithTelemetrySDK())

	return resource.New(context.Background(), attrs...)
}

// parseResourceAttributes 解析环境变量中的资源属性
// 格式: key1=value1,key2=value2
func parseResourceAttributes(envAttrs string) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	pairs := strings.Split(envAttrs, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			attrs = append(attrs, attribute.String(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])))
		}
	}
	return attrs
}

// Shutdown 优雅关闭 TracerProvider
// 确保所有 Span 导出完成
func Shutdown(ctx context.Context) error {
	providerMu.Lock()
	tp := globalProvider
	providerMu.Unlock()

	if tp == nil {
		return nil
	}

	return tp.Shutdown(ctx)
}

// GetTracerProvider 获取全局 TracerProvider
func GetTracerProvider() *trace.TracerProvider {
	providerMu.Lock()
	defer providerMu.Unlock()
	return globalProvider
}
