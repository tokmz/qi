package job

import (
	"time"

	"gorm.io/gorm"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeMemory StorageType = "memory" // 内存存储
	StorageTypeGorm   StorageType = "gorm"   // GORM 持久化存储
)

// Config 调度器配置
type Config struct {
	// 存储类型
	StorageType StorageType

	// GORM 数据库实例（用于持久化存储）
	DB *gorm.DB

	// 表名前缀
	TablePrefix string

	// 自定义表名
	JobTableName string
	RunTableName string

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
}

// Option 配置选项
type Option func(*Config)

// WithStorageType 设置存储类型
func WithStorageType(t StorageType) Option {
	return func(c *Config) {
		c.StorageType = t
	}
}

// WithGormDB 设置 GORM 数据库实例
func WithGormDB(db *gorm.DB) Option {
	return func(c *Config) {
		c.DB = db
		c.StorageType = StorageTypeGorm
	}
}

// WithTablePrefix 设置表名前缀
func WithTablePrefix(prefix string) Option {
	return func(c *Config) {
		c.TablePrefix = prefix
	}
}

// WithJobTableName 设置任务表名
func WithJobTableName(name string) Option {
	return func(c *Config) {
		c.JobTableName = name
	}
}

// WithRunTableName 设置执行记录表名
func WithRunTableName(name string) Option {
	return func(c *Config) {
		c.RunTableName = name
	}
}

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

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		StorageType:    StorageTypeMemory,
		ConcurrentRuns: 5,
		JobTimeout:     5 * time.Minute,
		RetryDelay:     time.Second * 5,
		AutoStart:      false,
		Logger:         &StdLogger{},
	}
}
