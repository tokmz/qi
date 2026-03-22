package cache

import (
	"context"
	"fmt"
	"time"
)

// Cache 统一缓存接口
type Cache interface {
	// 基础操作
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 批量操作
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, kvs map[string]any, ttl time.Duration) error

	// 原子加载：命中直接返回；未命中调用 fn 加载并回写（内置 singleflight 防击穿）
	GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error

	// 计数器
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	DecrBy(ctx context.Context, key string, delta int64) (int64, error)

	// 生命周期
	Flush(ctx context.Context) error
	Close() error
}

// Locker 分布式锁接口（仅 Redis 驱动支持）
type Locker interface {
	// Lock 阻塞直到获取锁或 ctx 取消，返回解锁函数
	Lock(ctx context.Context, key string, ttl time.Duration) (unlock func(), err error)
	// TryLock 非阻塞尝试获取锁
	TryLock(ctx context.Context, key string, ttl time.Duration) (ok bool, unlock func(), err error)
}

// New 根据 Config 创建缓存实例
func New(cfg *Config) (Cache, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	cfg.setDefaults()

	var (
		c   Cache
		err error
	)

	switch cfg.Driver {
	case DriverMemory:
		if cfg.Memory == nil {
			cfg.Memory = &MemoryConfig{MaxSize: 10_000, CleanupInterval: time.Minute}
		}
		c, err = newMemoryCache(cfg)
	case DriverRedis:
		if cfg.Redis == nil {
			return nil, fmt.Errorf("cache: redis config is required for driver %q", DriverRedis)
		}
		c, err = newRedisCache(cfg)
	case DriverMultiLevel:
		if cfg.Memory == nil {
			cfg.Memory = &MemoryConfig{MaxSize: 1_000, CleanupInterval: time.Minute}
		}
		if cfg.Redis == nil {
			return nil, fmt.Errorf("cache: redis config is required for driver %q", DriverMultiLevel)
		}
		c, err = newMultiLevelCache(cfg)
	default:
		return nil, fmt.Errorf("cache: unknown driver %q", cfg.Driver)
	}

	if err != nil {
		return nil, err
	}

	// 防穿透装饰器
	if cfg.Penetration != nil {
		c, err = newPenetrationGuard(c, cfg.Penetration, cfg.Serializer)
		if err != nil {
			return nil, err
		}
	}

	// 链路追踪装饰器（最外层）
	if cfg.TracingEnabled {
		c = newTracingCache(c)
	}

	return c, nil
}
