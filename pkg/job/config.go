package job

import (
	"time"
)

// Config 调度器配置
type Config struct {
	// 日志器
	Logger Logger

	// 调度器配置
	// 并发执行的任务数
	ConcurrentRuns int
	// 默认任务超时时间
	JobTimeout time.Duration
	// 默认重试间隔
	RetryDelay time.Duration
	// 是否自动启动
	AutoStart bool

	// 性能优化配置
	// 是否启用批量更新（减少数据库访问）
	EnableBatchUpdate bool
	// 批量更新大小
	BatchSize int
	// 批量更新刷新间隔
	BatchFlushInterval time.Duration

	// 缓存配置
	// 是否启用 LRU 缓存
	EnableCache bool
	// 缓存容量
	CacheCapacity int
	// 缓存 TTL
	CacheTTL time.Duration
	// 缓存清理间隔
	CacheCleanupInterval time.Duration
}

// Option 配置选项
type Option func(*Config)

// WithConcurrentRuns 设置并发执行数
func WithConcurrentRuns(n int) Option {
	return func(c *Config) {
		c.ConcurrentRuns = n
	}
}

// WithJobTimeout 设置任务超时时间
func WithJobTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.JobTimeout = timeout
	}
}

// WithRetryDelay 设置重试间隔
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.RetryDelay = delay
	}
}

// WithAutoStart 设置是否自动启动
func WithAutoStart(auto bool) Option {
	return func(c *Config) {
		c.AutoStart = auto
	}
}

// WithLogger 设置日志器
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithEnableBatchUpdate 设置是否启用批量更新
func WithEnableBatchUpdate(enable bool) Option {
	return func(c *Config) {
		c.EnableBatchUpdate = enable
	}
}

// WithBatchSize 设置批量更新大小
func WithBatchSize(size int) Option {
	return func(c *Config) {
		c.BatchSize = size
	}
}

// WithBatchFlushInterval 设置批量更新刷新间隔
func WithBatchFlushInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.BatchFlushInterval = interval
	}
}

// WithEnableCache 设置是否启用缓存
func WithEnableCache(enable bool) Option {
	return func(c *Config) {
		c.EnableCache = enable
	}
}

// WithCacheCapacity 设置缓存容量
func WithCacheCapacity(capacity int) Option {
	return func(c *Config) {
		c.CacheCapacity = capacity
	}
}

// WithCacheTTL 设置缓存 TTL
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.CacheTTL = ttl
	}
}

// WithCacheCleanupInterval 设置缓存清理间隔
func WithCacheCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CacheCleanupInterval = interval
	}
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ConcurrentRuns:       DefaultConcurrentRuns,
		JobTimeout:           DefaultJobTimeout,
		RetryDelay:           DefaultRetryDelay,
		AutoStart:            false,
		Logger:               &StdLogger{},
		EnableBatchUpdate:    false,
		BatchSize:            DefaultBatchSize,
		BatchFlushInterval:   DefaultBatchFlushInterval,
		EnableCache:          false,
		CacheCapacity:        DefaultCacheCapacity,
		CacheTTL:             DefaultCacheTTL,
		CacheCleanupInterval: DefaultCacheCleanupInterval,
	}
}
