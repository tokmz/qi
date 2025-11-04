package cache

import (
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 缓存管理器配置
type Config struct {
	// Redis Redis 客户端
	Redis *redis.Client

	// DefaultExpiration 默认过期时间
	DefaultExpiration time.Duration

	// CleanupInterval 清理间隔
	CleanupInterval time.Duration

	// Serializer 序列化器类型
	Serializer SerializerType

	// KeyPrefix 键前缀
	KeyPrefix string

	// NullCache 空值缓存配置
	NullCache NullCacheConfig

	// Stats 统计配置
	Stats StatsConfig
}

// NullCacheConfig 空值缓存配置
type NullCacheConfig struct {
	// Enabled 是否启用空值缓存
	Enabled bool

	// Expiration 空值过期时间
	Expiration time.Duration
}

// StatsConfig 统计配置
type StatsConfig struct {
	// Enabled 是否启用统计
	Enabled bool

	// ReportInterval 统计上报间隔
	ReportInterval time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
		Serializer:        SerializerJSON,
		KeyPrefix:         "qi:",
		NullCache: NullCacheConfig{
			Enabled:    true,
			Expiration: 1 * time.Minute,
		},
		Stats: StatsConfig{
			Enabled:        true,
			ReportInterval: 1 * time.Minute,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证 Redis 客户端
	if c.Redis == nil {
		return ErrRedisClientRequired
	}

	// 验证序列化器
	if c.Serializer == "" {
		c.Serializer = SerializerJSON
	}

	// 验证过期时间
	if c.DefaultExpiration <= 0 {
		c.DefaultExpiration = 5 * time.Minute
	}

	// 验证清理间隔
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = 10 * time.Minute
	}

	// 验证空值缓存配置
	if c.NullCache.Enabled && c.NullCache.Expiration <= 0 {
		c.NullCache.Expiration = 1 * time.Minute
	}

	// 验证统计配置
	if c.Stats.Enabled && c.Stats.ReportInterval <= 0 {
		c.Stats.ReportInterval = 1 * time.Minute
	}

	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

