package tracing

// ExporterType 链路追踪导出器类型
type ExporterType string

const (
	ExporterNoop     ExporterType = "noop"      // 禁用追踪（默认）
	ExporterStdout   ExporterType = "stdout"    // 输出到控制台（开发调试）
	ExporterOTLPGRPC ExporterType = "otlp_grpc" // OTLP gRPC（生产）
	ExporterOTLPHTTP ExporterType = "otlp_http" // OTLP HTTP/protobuf（生产）
)

// Config 链路追踪配置
type Config struct {
	// 服务元信息
	ServiceName    string
	ServiceVersion string
	Env            string

	// 导出器
	Exporter ExporterType
	// gRPC 示例: "otel-collector:4317"
	// HTTP 示例: "http://host:4318" 或 "https://host:4318"
	Endpoint string
	Insecure bool // gRPC 禁用 TLS

	// 采样率 0.0~1.0，默认 1.0
	SampleRate float64

	// 中间件选项
	SkipPaths     []string                        // 跳过追踪的路径（如 /ping）
	RecordHeaders bool                             // 是否记录请求 header（过滤敏感字段）
	SpanNameFunc  func(method, route string) string // 自定义 span 名称，nil 使用默认
}

func (c *Config) setDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "unknown_service"
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = "v0.0.0"
	}
	if c.Exporter == "" {
		c.Exporter = ExporterNoop
	}
	if c.SampleRate <= 0 {
		c.SampleRate = 1.0
	}
	if c.SampleRate > 1.0 {
		c.SampleRate = 1.0
	}
}
