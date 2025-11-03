package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Tracer 链路追踪管理器
type Tracer struct {
	config   *Config
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	mu       sync.RWMutex
}

var (
	// globalTracer 全局 Tracer 实例
	globalTracer *Tracer
	once         sync.Once
)

// New 创建新的 Tracer 实例
func New(cfg *Config) (*Tracer, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 如果未启用，返回 noop tracer
	if !cfg.Enabled {
		return &Tracer{
			config: cfg,
			tracer: noop.NewTracerProvider().Tracer(cfg.ServiceName),
		}, nil
	}

	t := &Tracer{
		config: cfg,
	}

	// 初始化 TracerProvider
	if err := t.initProvider(); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	return t, nil
}

// InitGlobal 初始化全局 Tracer（单例模式）
func InitGlobal(cfg *Config) error {
	var err error
	once.Do(func() {
		globalTracer, err = New(cfg)
		if err != nil {
			return
		}

		// 设置全局 TracerProvider
		if globalTracer.provider != nil {
			otel.SetTracerProvider(globalTracer.provider)

			// 设置全局 Propagator（支持 W3C Trace Context）
			otel.SetTextMapPropagator(
				propagation.NewCompositeTextMapPropagator(
					propagation.TraceContext{}, // W3C Trace Context
					propagation.Baggage{},      // W3C Baggage
				),
			)
		}
	})

	return err
}

// GetGlobal 获取全局 Tracer 实例
func GetGlobal() *Tracer {
	return globalTracer
}

// initProvider 初始化 TracerProvider
func (t *Tracer) initProvider() error {
	// 创建资源
	res, err := t.createResource()
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建导出器
	exporter, err := t.createExporter()
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// 创建采样器
	sampler := t.createSampler()

	// 创建批处理器
	bsp := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithMaxQueueSize(t.config.BatchSpanProcessor.MaxQueueSize),
		sdktrace.WithMaxExportBatchSize(t.config.BatchSpanProcessor.MaxExportBatchSize),
		sdktrace.WithBatchTimeout(t.config.BatchSpanProcessor.ScheduleDelay),
		sdktrace.WithExportTimeout(t.config.BatchSpanProcessor.ExportTimeout),
	)

	// 创建 TracerProvider
	t.provider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(bsp),
	)

	// 创建 Tracer
	t.tracer = t.provider.Tracer(
		t.config.ServiceName,
		trace.WithInstrumentationVersion(t.config.ServiceVersion),
	)

	return nil
}

// createResource 创建资源
func (t *Tracer) createResource() (*resource.Resource, error) {
	// 基础属性
	attrs := []attribute.KeyValue{
		semconv.ServiceName(t.config.ServiceName),
		semconv.ServiceVersion(t.config.ServiceVersion),
		attribute.String("environment", t.config.Environment),
	}

	// 添加自定义属性
	for k, v := range t.config.ResourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	// 创建资源
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attrs...,
		),
	)
}

// createSampler 创建采样器
func (t *Tracer) createSampler() sdktrace.Sampler {
	switch t.config.Sampler.Type {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "trace_id_ratio":
		return sdktrace.TraceIDRatioBased(t.config.Sampler.Ratio)
	case "parent_based":
		// 父级采样器：如果有父 span，则遵循父 span 的采样决策
		// 否则使用 TraceIDRatioBased 采样器
		return sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(t.config.Sampler.Ratio),
		)
	default:
		// 默认使用父级采样器
		return sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(t.config.Sampler.Ratio),
		)
	}
}

// GetTracer 获取 OpenTelemetry Tracer
func (t *Tracer) GetTracer() trace.Tracer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tracer
}

// GetProvider 获取 TracerProvider
func (t *Tracer) GetProvider() *sdktrace.TracerProvider {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.provider
}

// Shutdown 关闭 Tracer
// 确保所有 span 都被导出
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider == nil {
		return nil
	}

	// 创建带超时的 context
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 关闭 provider
	if err := t.provider.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	return nil
}

// ForceFlush 强制刷新所有待处理的 span
func (t *Tracer) ForceFlush(ctx context.Context) error {
	if t.provider == nil {
		return nil
	}

	flushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := t.provider.ForceFlush(flushCtx); err != nil {
		return fmt.Errorf("failed to flush spans: %w", err)
	}

	return nil
}

// IsEnabled 检查链路追踪是否启用
func (t *Tracer) IsEnabled() bool {
	return t.config.Enabled
}

// GetConfig 获取配置
func (t *Tracer) GetConfig() *Config {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config
}
