package tracing

import (
	"time"
)

// Config 链路追踪配置
type Config struct {
	// 服务名称（必填）
	ServiceName string

	// 服务版本
	ServiceVersion string

	// 环境（dev/staging/prod）
	Environment string

	// 导出器类型（otlp/stdout/noop）
	ExporterType string

	// 导出器端点（如 OTLP Collector URL）
	ExporterEndpoint string

	// 导出器请求头（用于认证）
	ExporterHeaders map[string]string

	// 是否使用非 TLS 连接（默认 false，即使用 HTTPS）
	Insecure bool

	// 采样率（0.0-1.0，1.0 表示全量采集）
	SamplingRate float64

	// 采样类型（always/never/ratio/parent_based）
	SamplingType string

	// 是否启用（默认 true）
	Enabled bool

	// 资源属性（自定义标签）
	ResourceAttributes map[string]string

	// 批处理配置
	BatchTimeout       time.Duration // 批量导出超时（默认 5s）
	MaxExportBatchSize int           // 最大批量大小（默认 512）
	MaxQueueSize       int           // 最大队列大小（默认 2048）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ServiceName:        "qi-service",
		ServiceVersion:     "1.0.0",
		Environment:        "development",
		ExporterType:       "stdout",
		SamplingRate:       1.0,
		SamplingType:       "parent_based",
		Enabled:            true,
		ResourceAttributes: make(map[string]string),
		BatchTimeout:       5 * time.Second,
		MaxExportBatchSize: 512,
		MaxQueueSize:       2048,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return ErrInvalidConfig("service name is required")
	}

	if c.SamplingRate < 0 || c.SamplingRate > 1 {
		return ErrInvalidConfig("sampling rate must be between 0.0 and 1.0")
	}

	validExporters := map[string]bool{
		"otlp":   true,
		"stdout": true,
		"noop":   true,
	}
	if !validExporters[c.ExporterType] {
		return ErrInvalidConfig("invalid exporter type: " + c.ExporterType)
	}

	return nil
}

// ErrInvalidConfig 配置错误
type ConfigError struct {
	message string
}

func (e *ConfigError) Error() string {
	return "tracing config error: " + e.message
}

func ErrInvalidConfig(message string) error {
	return &ConfigError{message: message}
}
