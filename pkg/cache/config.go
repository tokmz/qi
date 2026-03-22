package cache

import "time"

// DriverType 缓存驱动类型
type DriverType string

const (
	DriverMemory     DriverType = "memory"
	DriverRedis      DriverType = "redis"
	DriverMultiLevel DriverType = "multilevel"
)

// Config 缓存总配置
type Config struct {
	Driver     DriverType    // 驱动类型
	KeyPrefix  string        // key 统一前缀，如 "app:"
	DefaultTTL time.Duration // 全局默认过期时间（0 = 永不过期）
	Serializer Serializer    // nil → 使用 JSONSerializer

	Memory      *MemoryConfig      // Driver=memory / multilevel 时有效
	Redis       *RedisConfig       // Driver=redis  / multilevel 时有效
	Penetration *PenetrationConfig // nil → 不启用防穿透

	// 链路追踪：需外部提前初始化 OTel TracerProvider
	TracingEnabled bool // 是否启用 OpenTelemetry 链路追踪
}

// MemoryConfig 内存缓存配置
type MemoryConfig struct {
	MaxSize         int           // LRU 最大条目数（0 = 不限）
	CleanupInterval time.Duration // 后台扫描过期 key 的间隔（默认 1m）
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	// 三选一：单机 / 哨兵 / 集群
	Addr   string   // 单机地址，如 "127.0.0.1:6379"
	Addrs  []string // 集群节点列表
	Master string   // 哨兵模式 master 名称（非空时启用哨兵）

	Username string
	Password string
	DB       int // 单机/哨兵有效

	// 连接池
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// TTL 抖动：在 Set 时对 TTL 随机 ±10%，防雪崩（默认开启）
	DisableJitter bool
}

// PenetrationConfig 防缓存穿透配置
type PenetrationConfig struct {
	EnableBloom bool          // 是否启用 Bloom filter（推荐开启）
	BloomN      uint          // 预期最大 key 数量
	BloomFP     float64       // Bloom filter 误判率（建议 0.01）
	NullTTL     time.Duration // 空值缓存时长（默认 60s）
}

// setDefaults 补全默认值
func (c *Config) setDefaults() {
	if c.Serializer == nil {
		c.Serializer = JSONSerializer{}
	}
	if c.Memory != nil {
		if c.Memory.CleanupInterval <= 0 {
			c.Memory.CleanupInterval = time.Minute
		}
	}
	if c.Penetration != nil {
		if c.Penetration.NullTTL <= 0 {
			c.Penetration.NullTTL = 60 * time.Second
		}
		if c.Penetration.EnableBloom {
			if c.Penetration.BloomN == 0 {
				c.Penetration.BloomN = 100_000
			}
			if c.Penetration.BloomFP == 0 {
				c.Penetration.BloomFP = 0.01
			}
		}
	}
}

// DefaultConfig 返回合理默认配置（内存驱动）
func DefaultConfig() *Config {
	return &Config{
		Driver: DriverMemory,
		Memory: &MemoryConfig{
			MaxSize:         10_000,
			CleanupInterval: time.Minute,
		},
	}
}
