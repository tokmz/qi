package qi

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokmz/qi/pkg/i18n"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	// Addr 监听地址，默认 ":8080"
	Addr string

	// ReadTimeout 读取超时
	ReadTimeout time.Duration

	// WriteTimeout 写入超时
	WriteTimeout time.Duration

	// IdleTimeout 空闲超时
	IdleTimeout time.Duration

	// MaxHeaderBytes 最大请求头字节数
	MaxHeaderBytes int
}

// ShutdownConfig 关机配置
type ShutdownConfig struct {
	// Timeout 关机超时时间，默认 10 秒
	Timeout time.Duration

	// BeforeShutdown 关机前回调
	BeforeShutdown func()

	// AfterShutdown 关机后回调
	AfterShutdown func()
}

// Config 应用配置
type Config struct {
	// Mode 运行模式：debug, release, test
	Mode string

	// Server 服务器配置
	Server ServerConfig

	// Shutdown 关机配置
	Shutdown ShutdownConfig

	// TrustedProxies 信任的代理 IP
	TrustedProxies []string

	// MaxMultipartMemory 最大 multipart 内存（字节）
	MaxMultipartMemory int64

	// I18n 国际化配置，nil 表示不启用
	I18n *i18n.Config
}

// Option 配置选项函数
type Option func(*Config)

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	return &Config{
		Mode: gin.DebugMode,
		Server: ServerConfig{
			Addr:           ":8080",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1MB
		},
		Shutdown: ShutdownConfig{
			Timeout:        10 * time.Second,
			BeforeShutdown: nil,
			AfterShutdown:  nil,
		},
		TrustedProxies:     nil,
		MaxMultipartMemory: 32 << 20, // 32MB
	}
}

// WithMode 设置运行模式
func WithMode(mode string) Option {
	return func(c *Config) {
		c.Mode = mode
	}
}

// WithAddr 设置监听地址
func WithAddr(addr string) Option {
	return func(c *Config) {
		c.Server.Addr = addr
	}
}

// WithReadTimeout 设置读取超时
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Server.ReadTimeout = timeout
	}
}

// WithWriteTimeout 设置写入超时
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Server.WriteTimeout = timeout
	}
}

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Server.IdleTimeout = timeout
	}
}

// WithMaxHeaderBytes 设置最大请求头字节数
func WithMaxHeaderBytes(size int) Option {
	return func(c *Config) {
		c.Server.MaxHeaderBytes = size
	}
}

// WithShutdownTimeout 设置关机超时时间
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Shutdown.Timeout = timeout
	}
}

// WithBeforeShutdown 设置关机前回调
func WithBeforeShutdown(fn func()) Option {
	return func(c *Config) {
		c.Shutdown.BeforeShutdown = fn
	}
}

// WithAfterShutdown 设置关机后回调
func WithAfterShutdown(fn func()) Option {
	return func(c *Config) {
		c.Shutdown.AfterShutdown = fn
	}
}

// WithTrustedProxies 设置信任的代理
func WithTrustedProxies(proxies ...string) Option {
	return func(c *Config) {
		c.TrustedProxies = proxies
	}
}

// WithMaxMultipartMemory 设置最大 multipart 内存
func WithMaxMultipartMemory(size int64) Option {
	return func(c *Config) {
		c.MaxMultipartMemory = size
	}
}

// WithI18n 设置国际化配置
func WithI18n(cfg *i18n.Config) Option {
	return func(c *Config) {
		c.I18n = cfg
	}
}
