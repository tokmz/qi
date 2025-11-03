// Package tracing 提供基于 OpenTelemetry 的分布式链路追踪功能
package tracing

import (
	"time"
)

// Config 链路追踪配置
type Config struct {
	// 是否启用链路追踪
	Enabled bool `mapstructure:"enabled" json:"enabled"`

	// 服务名称
	ServiceName string `mapstructure:"service_name" json:"service_name"`

	// 服务版本
	ServiceVersion string `mapstructure:"service_version" json:"service_version"`

	// 部署环境 (development, staging, production)
	Environment string `mapstructure:"environment" json:"environment"`

	// 采样配置
	Sampler SamplerConfig `mapstructure:"sampler" json:"sampler"`

	// 导出器配置
	Exporter ExporterConfig `mapstructure:"exporter" json:"exporter"`

	// 批处理配置
	BatchSpanProcessor BatchSpanProcessorConfig `mapstructure:"batch_span_processor" json:"batch_span_processor"`

	// 资源属性
	ResourceAttributes map[string]string `mapstructure:"resource_attributes" json:"resource_attributes"`
}

// SamplerConfig 采样器配置
type SamplerConfig struct {
	// 采样类型: always_on, always_off, trace_id_ratio, parent_based
	Type string `mapstructure:"type" json:"type"`

	// 采样率 (0.0 - 1.0)，仅在 trace_id_ratio 类型时有效
	// 1.0 表示 100% 采样，0.1 表示 10% 采样
	Ratio float64 `mapstructure:"ratio" json:"ratio"`
}

// ExporterConfig 导出器配置
type ExporterConfig struct {
	// 导出器类型: otlp, jaeger, zipkin, stdout
	Type string `mapstructure:"type" json:"type"`

	// OTLP 导出器配置
	OTLP OTLPConfig `mapstructure:"otlp" json:"otlp"`

	// Jaeger 导出器配置
	Jaeger JaegerConfig `mapstructure:"jaeger" json:"jaeger"`

	// Zipkin 导出器配置
	Zipkin ZipkinConfig `mapstructure:"zipkin" json:"zipkin"`

	// Stdout 导出器配置（用于调试）
	Stdout StdoutConfig `mapstructure:"stdout" json:"stdout"`
}

// OTLPConfig OTLP 导出器配置
type OTLPConfig struct {
	// OTLP 端点地址
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`

	// 是否使用不安全的连接（HTTP）
	Insecure bool `mapstructure:"insecure" json:"insecure"`

	// 请求头
	Headers map[string]string `mapstructure:"headers" json:"headers"`

	// 超时时间
	Timeout time.Duration `mapstructure:"timeout" json:"timeout"`

	// 压缩方式: gzip, none
	Compression string `mapstructure:"compression" json:"compression"`

	// 协议: grpc, http
	Protocol string `mapstructure:"protocol" json:"protocol"`
}

// JaegerConfig Jaeger 导出器配置
type JaegerConfig struct {
	// Jaeger Agent 端点
	AgentEndpoint string `mapstructure:"agent_endpoint" json:"agent_endpoint"`

	// Jaeger Collector 端点
	CollectorEndpoint string `mapstructure:"collector_endpoint" json:"collector_endpoint"`

	// 用户名（HTTP Basic Auth）
	Username string `mapstructure:"username" json:"username"`

	// 密码（HTTP Basic Auth）
	Password string `mapstructure:"password" json:"password"`
}

// ZipkinConfig Zipkin 导出器配置
type ZipkinConfig struct {
	// Zipkin 端点
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
}

// StdoutConfig Stdout 导出器配置
type StdoutConfig struct {
	// 是否美化输出
	PrettyPrint bool `mapstructure:"pretty_print" json:"pretty_print"`
}

// BatchSpanProcessorConfig 批处理器配置
type BatchSpanProcessorConfig struct {
	// 批处理最大队列大小
	MaxQueueSize int `mapstructure:"max_queue_size" json:"max_queue_size"`

	// 批处理最大导出批次大小
	MaxExportBatchSize int `mapstructure:"max_export_batch_size" json:"max_export_batch_size"`

	// 批处理调度延迟
	ScheduleDelay time.Duration `mapstructure:"schedule_delay" json:"schedule_delay"`

	// 导出超时
	ExportTimeout time.Duration `mapstructure:"export_timeout" json:"export_timeout"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		ServiceName:    "qi-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Sampler: SamplerConfig{
			Type:  "parent_based",
			Ratio: 1.0,
		},
		Exporter: ExporterConfig{
			Type: "otlp",
			OTLP: OTLPConfig{
				Endpoint:    "localhost:4318",
				Insecure:    true,
				Timeout:     10 * time.Second,
				Compression: "gzip",
				Protocol:    "http",
			},
		},
		BatchSpanProcessor: BatchSpanProcessorConfig{
			MaxQueueSize:       2048,
			MaxExportBatchSize: 512,
			ScheduleDelay:      5 * time.Second,
			ExportTimeout:      30 * time.Second,
		},
		ResourceAttributes: make(map[string]string),
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.ServiceName == "" {
		return ErrInvalidServiceName
	}

	// 验证采样率
	if c.Sampler.Type == "trace_id_ratio" {
		if c.Sampler.Ratio < 0 || c.Sampler.Ratio > 1 {
			return ErrInvalidSampleRatio
		}
	}

	// 验证导出器配置
	switch c.Exporter.Type {
	case "otlp":
		if c.Exporter.OTLP.Endpoint == "" {
			return ErrInvalidOTLPEndpoint
		}
	case "jaeger":
		if c.Exporter.Jaeger.AgentEndpoint == "" && c.Exporter.Jaeger.CollectorEndpoint == "" {
			return ErrInvalidJaegerEndpoint
		}
	case "zipkin":
		if c.Exporter.Zipkin.Endpoint == "" {
			return ErrInvalidZipkinEndpoint
		}
	case "stdout":
		// stdout 不需要额外配置
	default:
		return ErrUnsupportedExporter
	}

	return nil
}
