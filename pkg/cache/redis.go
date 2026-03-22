package cache

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type redisCache struct {
	client     redis.UniversalClient
	serializer Serializer
	prefix     string
	defaultTTL time.Duration
	disableJitter bool
	sf         singleflight.Group
}

func newRedisCache(cfg *Config) (*redisCache, error) {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        redisAddrs(cfg.Redis),
		MasterName:   cfg.Redis.Master,
		Username:     cfg.Redis.Username,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("cache: redis ping failed: %w", err)
	}
	return &redisCache{
		client:        client,
		serializer:    cfg.Serializer,
		prefix:        cfg.KeyPrefix,
		defaultTTL:    cfg.DefaultTTL,
		disableJitter: cfg.Redis.DisableJitter,
	}, nil
}

func redisAddrs(cfg *RedisConfig) []string {
	if len(cfg.Addrs) > 0 {
		return cfg.Addrs
	}
	if cfg.Addr != "" {
		return []string{cfg.Addr}
	}
	return []string{"127.0.0.1:6379"}
}

func (c *redisCache) k(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + key
}

func (c *redisCache) effectiveTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	if ttl <= 0 || c.disableJitter {
		return ttl
	}
	// ±10% 随机抖动，防雪崩；ttl/5 最小为 1 防止 rand panic
	base := int64(ttl / 5)
	if base < 1 {
		base = 1
	}
	jitter := time.Duration(rand.Int63n(base)) - ttl/10
	return ttl + jitter
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) error {
	b, err := c.client.Get(ctx, c.k(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrNotFound
		}
		return err
	}
	return c.serializer.Unmarshal(b, dest)
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if value == nil {
		return ErrNilValue
	}
	b, err := c.serializer.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.k(key), b, c.effectiveTTL(ttl)).Err()
}

func (c *redisCache) Del(ctx context.Context, keys ...string) error {
	prefixed := make([]string, len(keys))
	for i, k := range keys {
		prefixed[i] = c.k(k)
	}
	return c.client.Del(ctx, prefixed...).Err()
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.k(key)).Result()
	return n > 0, err
}

func (c *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	// 使用 PExpire 支持毫秒级精度（EXPIRE 只支持秒级）
	ok, err := c.client.PExpire(ctx, c.k(key), ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (c *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	// 使用 PTTL 获取毫秒级剩余时间
	d, err := c.client.PTTL(ctx, c.k(key)).Result()
	if err != nil {
		return -1, err
	}
	// PTTL 返回: -2 (ns) 表示 key 不存在，-1 (ns) 表示无 TTL，正数为剩余毫秒数
	// go-redis v9 直接返回 Redis 原始整数作为 time.Duration（单位 ns，非 ms）
	if d == -2 {
		return -1, ErrNotFound
	}
	if d == -1 {
		return -1, nil // key 存在但永不过期
	}
	return d, nil
}

func (c *redisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	prefixed := make([]string, len(keys))
	for i, k := range keys {
		prefixed[i] = c.k(k)
	}
	vals, err := c.client.MGet(ctx, prefixed...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string][]byte, len(keys))
	for i, v := range vals {
		if v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		result[keys[i]] = []byte(s)
	}
	return result, nil
}

func (c *redisCache) MSet(ctx context.Context, kvs map[string]any, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	effTTL := c.effectiveTTL(ttl)
	for k, v := range kvs {
		b, err := c.serializer.Marshal(v)
		if err != nil {
			return err
		}
		pipe.Set(ctx, c.k(k), b, effTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *redisCache) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	if err := c.Get(ctx, key, dest); err == nil {
		return nil
	}
	v, err, _ := c.sf.Do(c.k(key), func() (any, error) {
		if err2 := c.Get(ctx, key, dest); err2 == nil {
			return nil, nil
		}
		val, err2 := fn()
		if err2 != nil {
			return nil, err2
		}
		_ = c.Set(ctx, key, val, ttl)
		return val, nil
	})
	if err != nil {
		return err
	}
	if v == nil {
		return c.Get(ctx, key, dest)
	}
	b, err2 := c.serializer.Marshal(v)
	if err2 != nil {
		return err2
	}
	return c.serializer.Unmarshal(b, dest)
}

func (c *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, c.k(key)).Result()
}

func (c *redisCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return c.client.IncrBy(ctx, c.k(key), delta).Result()
}

func (c *redisCache) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return c.client.DecrBy(ctx, c.k(key), delta).Result()
}

func (c *redisCache) Flush(ctx context.Context) error {
	if c.prefix == "" {
		// 无前缀时拒绝执行，防止误清整个 Redis DB
		// 若确实需要清库，请直接操作 Redis 客户端
		return fmt.Errorf("cache: Flush refused: KeyPrefix is empty, operation would wipe the entire DB")
	}
	// 仅删除带前缀的 key
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, c.prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (c *redisCache) Close() error {
	return c.client.Close()
}

// NewLocker 创建基于 Redis 的分布式锁
func NewLocker(cfg *RedisConfig, prefix string) (Locker, error) {
	ser := JSONSerializer{}
	cacheCfg := &Config{Driver: DriverRedis, Redis: cfg, KeyPrefix: prefix, Serializer: ser}
	cacheCfg.setDefaults()
	c, err := newRedisCache(cacheCfg)
	if err != nil {
		return nil, err
	}
	return &redisLocker{client: c.client, prefix: prefix}, nil
}
