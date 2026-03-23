package tracing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const DefaultTracerName = "qi"

// Init 初始化全局 OTel TracerProvider 并设置 W3C 传播器。
// 返回的 shutdown 须在服务退出时调用（flush + 关闭连接）。
func Init(cfg *Config) (shutdown func(context.Context) error, err error) {
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.setDefaults()

	ctx := context.Background()

	// ErrPartialResource 在容器环境下常见（部分探测失败），resource 仍有效
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.Env),
		),
	)
	if err != nil && !errors.Is(err, resource.ErrPartialResource) {
		return nil, fmt.Errorf("tracing: create resource: %w", err)
	}

	var sampler sdktrace.Sampler
	switch {
	case cfg.SampleRate >= 1.0:
		sampler = sdktrace.AlwaysSample()
	case cfg.SampleRate <= 0:
		sampler = sdktrace.NeverSample()
	default:
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
	}

	var tp *sdktrace.TracerProvider
	if cfg.Exporter == ExporterNoop {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
			sdktrace.WithResource(res),
		)
	} else {
		exp, err := newExporter(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("tracing: create exporter: %w", err)
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sampler),
		)
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func newExporter(ctx context.Context, cfg *Config) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())

	case ExporterOTLPGRPC:
		if cfg.Endpoint == "" {
			return nil, fmt.Errorf("tracing: endpoint required for otlp_grpc")
		}
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithDialOption(
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
		}
		return otlptracegrpc.New(ctx, opts...)

	case ExporterOTLPHTTP:
		if cfg.Endpoint == "" {
			return nil, fmt.Errorf("tracing: endpoint required for otlp_http")
		}
		endpoint := cfg.Endpoint
		var opts []otlptracehttp.Option
		if ep, ok := strings.CutPrefix(endpoint, "http://"); ok {
			endpoint = ep
			opts = append(opts, otlptracehttp.WithInsecure())
		} else if ep, ok := strings.CutPrefix(endpoint, "https://"); ok {
			endpoint = ep
		}
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
		return otlptracehttp.New(ctx, opts...)

	default:
		return nil, fmt.Errorf("tracing: unknown exporter %q", cfg.Exporter)
	}
}

// TraceIDFromContext 从 context 提取 TraceID 字符串。
func TraceIDFromContext(ctx context.Context) string {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanFromContext 返回当前 span。
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// Start 开启子 span（语法糖）。
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(DefaultTracerName).Start(ctx, name, opts...)
}
