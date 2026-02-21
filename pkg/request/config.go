package request

import (
	"crypto/tls"
	"net/http"
	"time"
)

// Config HTTP 客户端配置
type Config struct {
	BaseURL             string            // 基础 URL
	Timeout             time.Duration     // 全局超时（默认 30s）
	Headers             map[string]string // 全局默认请求头
	MaxIdleConns        int               // 最大空闲连接数（默认 100）
	MaxIdleConnsPerHost int               // 每 Host 最大空闲连接（默认 10）
	MaxConnsPerHost     int               // 每 Host 最大连接（默认 100）
	IdleConnTimeout     time.Duration     // 空闲连接超时（默认 90s）
	Retry               *RetryConfig      // 重试配置（nil 不重试）
	Interceptors        []Interceptor     // 拦截器链
	Logger              Logger            // 日志器（nil 不记录）
	EnableTracing       bool              // 启用 OpenTelemetry 追踪
	InsecureSkipVerify  bool              // 跳过 TLS 验证
	Transport           http.RoundTripper // 自定义 Transport（覆盖连接池配置）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:             30 * time.Second,
		Headers:             make(map[string]string),
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
	}
}

// buildTransport 根据配置构建 http.Transport
func (c *Config) buildTransport() http.RoundTripper {
	if c.Transport != nil {
		return c.Transport
	}

	t := &http.Transport{
		MaxIdleConns:        c.MaxIdleConns,
		MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
		MaxConnsPerHost:     c.MaxConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
	}

	if c.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return t
}

// Option 配置选项函数
type Option func(*Config)

// WithBaseURL 设置基础 URL
func WithBaseURL(url string) Option {
	return func(c *Config) { c.BaseURL = url }
}

// WithTimeout 设置全局超时
func WithTimeout(d time.Duration) Option {
	return func(c *Config) { c.Timeout = d }
}

// WithHeader 设置全局默认请求头
func WithHeader(key, value string) Option {
	return func(c *Config) { c.Headers[key] = value }
}

// WithHeaders 批量设置全局默认请求头
func WithHeaders(headers map[string]string) Option {
	return func(c *Config) {
		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithMaxIdleConns 设置最大空闲连接数
func WithMaxIdleConns(n int) Option {
	return func(c *Config) { c.MaxIdleConns = n }
}

// WithMaxIdleConnsPerHost 设置每 Host 最大空闲连接
func WithMaxIdleConnsPerHost(n int) Option {
	return func(c *Config) { c.MaxIdleConnsPerHost = n }
}

// WithMaxConnsPerHost 设置每 Host 最大连接
func WithMaxConnsPerHost(n int) Option {
	return func(c *Config) { c.MaxConnsPerHost = n }
}

// WithIdleConnTimeout 设置空闲连接超时
func WithIdleConnTimeout(d time.Duration) Option {
	return func(c *Config) { c.IdleConnTimeout = d }
}

// WithRetry 设置重试配置
func WithRetry(cfg *RetryConfig) Option {
	return func(c *Config) { c.Retry = cfg }
}

// WithInterceptor 添加拦截器
func WithInterceptor(i Interceptor) Option {
	return func(c *Config) { c.Interceptors = append(c.Interceptors, i) }
}

// WithLogger 设置日志器
func WithLogger(l Logger) Option {
	return func(c *Config) { c.Logger = l }
}

// WithTracing 启用 OpenTelemetry 追踪
func WithTracing(enable bool) Option {
	return func(c *Config) { c.EnableTracing = enable }
}

// WithInsecureSkipVerify 跳过 TLS 验证
func WithInsecureSkipVerify(skip bool) Option {
	return func(c *Config) { c.InsecureSkipVerify = skip }
}

// WithTransport 设置自定义 Transport
func WithTransport(t http.RoundTripper) Option {
	return func(c *Config) { c.Transport = t }
}
