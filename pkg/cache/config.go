package cache

import "time"

// DriverType 驱动类型
type DriverType string

const (
	DriverRedis  DriverType = "redis"
	DriverMemory DriverType = "memory"
)

// RedisMode Redis 模式
type RedisMode string

const (
	RedisStandalone RedisMode = "standalone"
	RedisCluster    RedisMode = "cluster"
	RedisSentinel   RedisMode = "sentinel"
)

// Config 缓存配置
type Config struct {
	// 驱动类型
	Driver DriverType

	// Redis 配置
	Redis *RedisConfig

	// Memory 配置
	Memory *MemoryConfig

	// 序列化器
	Serializer Serializer

	// 键前缀（避免冲突）
	KeyPrefix string

	// 默认 TTL
	DefaultTTL time.Duration
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr         string        // 地址（单机）
	Addrs        []string      // 地址列表（集群/哨兵）
	Mode         RedisMode     // standalone, cluster, sentinel
	Username     string        // 用户名（Redis 6.0+）
	Password     string        // 密码
	DB           int           // 数据库编号
	PoolSize     int           // 连接池大小
	MinIdleConns int           // 最小空闲连接
	MaxRetries   int           // 最大重试次数
	DialTimeout  time.Duration // 连接超时
	ReadTimeout  time.Duration // 读超时
	WriteTimeout time.Duration // 写超时

	// 哨兵模式配置
	MasterName string // 主节点名称
}

// MemoryConfig 内存缓存配置
type MemoryConfig struct {
	DefaultExpiration time.Duration // 默认过期时间
	CleanupInterval   time.Duration // 清理间隔
	MaxEntries        int           // 最大条目数（0 表示无限制）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Driver:     DriverMemory,
		Serializer: &JSONSerializer{},
		KeyPrefix:  "",
		DefaultTTL: 10 * time.Minute,
		Memory: &MemoryConfig{
			DefaultExpiration: 10 * time.Minute,
			CleanupInterval:   5 * time.Minute,
			MaxEntries:        0,
		},
	}
}

// DefaultRedisConfig 返回默认 Redis 配置
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		Mode:         RedisStandalone,
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// DefaultMemoryConfig 返回默认 Memory 配置
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		DefaultExpiration: 10 * time.Minute,
		CleanupInterval:   5 * time.Minute,
		MaxEntries:        0,
	}
}

// Option 配置选项
type Option func(*Config)

// WithRedis 设置 Redis 配置
func WithRedis(cfg *RedisConfig) Option {
	return func(c *Config) {
		c.Driver = DriverRedis
		c.Redis = cfg
	}
}

// WithMemory 设置 Memory 配置
func WithMemory(cfg *MemoryConfig) Option {
	return func(c *Config) {
		c.Driver = DriverMemory
		c.Memory = cfg
	}
}

// WithSerializer 设置序列化器
func WithSerializer(s Serializer) Option {
	return func(c *Config) {
		c.Serializer = s
	}
}

// WithKeyPrefix 设置键前缀
func WithKeyPrefix(prefix string) Option {
	return func(c *Config) {
		c.KeyPrefix = prefix
	}
}

// WithDefaultTTL 设置默认 TTL
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证驱动类型
	if c.Driver != DriverRedis && c.Driver != DriverMemory {
		return ErrCacheInvalidConfig.WithMessage("invalid driver type")
	}

	// 验证序列化器
	if c.Serializer == nil {
		return ErrCacheInvalidConfig.WithMessage("serializer is required")
	}

	// 验证 Redis 配置
	if c.Driver == DriverRedis {
		if c.Redis == nil {
			return ErrCacheInvalidConfig.WithMessage("redis config is required")
		}

		switch c.Redis.Mode {
		case RedisStandalone:
			if c.Redis.Addr == "" {
				return ErrCacheInvalidConfig.WithMessage("redis addr is required for standalone mode")
			}
		case RedisCluster:
			if len(c.Redis.Addrs) < 3 {
				return ErrCacheInvalidConfig.WithMessage("redis cluster requires at least 3 nodes")
			}
		case RedisSentinel:
			if len(c.Redis.Addrs) == 0 {
				return ErrCacheInvalidConfig.WithMessage("redis sentinel requires at least 1 sentinel node")
			}
			if c.Redis.MasterName == "" {
				return ErrCacheInvalidConfig.WithMessage("redis sentinel requires master name")
			}
		default:
			return ErrCacheInvalidConfig.WithMessage("invalid redis mode")
		}
	}

	// 验证 Memory 配置
	if c.Driver == DriverMemory {
		if c.Memory == nil {
			return ErrCacheInvalidConfig.WithMessage("memory config is required")
		}
	}

	return nil
}
