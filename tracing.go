package qi

import itrace "github.com/tokmz/qi/internal/tracing"

// TracingConfig 链路追踪配置，参见 internal/tracing.Config。
type TracingConfig = itrace.Config

// TracingExporterType 导出器类型
type TracingExporterType = itrace.ExporterType

const (
	TracingExporterNoop     = itrace.ExporterNoop
	TracingExporterStdout   = itrace.ExporterStdout
	TracingExporterOTLPGRPC = itrace.ExporterOTLPGRPC
	TracingExporterOTLPHTTP = itrace.ExporterOTLPHTTP
)

// WithTracing 配置链路追踪。
// Engine 启动时自动初始化 OTel TracerProvider 并注册追踪中间件；
// 优雅关闭时自动 flush span 数据。
func WithTracing(cfg *TracingConfig) Option {
	return func(c *Config) {
		c.tracingConfig = cfg
	}
}
