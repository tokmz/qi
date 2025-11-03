package tracing

import "errors"

var (
	// ErrTracingDisabled 链路追踪未启用
	ErrTracingDisabled = errors.New("tracing is disabled")

	// ErrInvalidServiceName 无效的服务名称
	ErrInvalidServiceName = errors.New("invalid service name")

	// ErrInvalidSampleRatio 无效的采样率
	ErrInvalidSampleRatio = errors.New("invalid sample ratio, must be between 0 and 1")

	// ErrInvalidOTLPEndpoint 无效的 OTLP 端点
	ErrInvalidOTLPEndpoint = errors.New("invalid OTLP endpoint")

	// ErrInvalidJaegerEndpoint 无效的 Jaeger 端点
	ErrInvalidJaegerEndpoint = errors.New("invalid Jaeger endpoint")

	// ErrInvalidZipkinEndpoint 无效的 Zipkin 端点
	ErrInvalidZipkinEndpoint = errors.New("invalid Zipkin endpoint")

	// ErrUnsupportedExporter 不支持的导出器类型
	ErrUnsupportedExporter = errors.New("unsupported exporter type")

	// ErrTracerNotInitialized Tracer 未初始化
	ErrTracerNotInitialized = errors.New("tracer not initialized")

	// ErrShutdownTimeout 关闭超时
	ErrShutdownTimeout = errors.New("shutdown timeout")
)
