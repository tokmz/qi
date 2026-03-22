package cache

import (
	"context"
	"time"
)

// multiLevelCache L1(内存) + L2(Redis) 多级缓存
type multiLevelCache struct {
	l1 *memoryCache
	l2 *redisCache
	// L1 TTL 取 L2 TTL 的 20%，防止 L1 长期持有旧数据
	l1TTLRatio float64
}

func newMultiLevelCache(cfg *Config) (*multiLevelCache, error) {
	l1, err := newMemoryCache(cfg)
	if err != nil {
		return nil, err
	}
	l2, err := newRedisCache(cfg)
	if err != nil {
		l1.Close()
		return nil, err
	}
	return &multiLevelCache{l1: l1, l2: l2, l1TTLRatio: 0.2}, nil
}

func (c *multiLevelCache) l1TTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	d := time.Duration(float64(ttl) * c.l1TTLRatio)
	if d < time.Second {
		d = time.Second
	}
	return d
}

func (c *multiLevelCache) Get(ctx context.Context, key string, dest any) error {
	// L1 命中
	if err := c.l1.Get(ctx, key, dest); err == nil {
		return nil
	}
	// L2 查询
	if err := c.l2.Get(ctx, key, dest); err != nil {
		return err
	}
	// 回填 L1：优先用 key 在 L2 的实际剩余 TTL，避免 L1 持有已过期数据
	l1TTL := c.l1TTL(c.l2.defaultTTL)
	if remainTTL, err := c.l2.TTL(ctx, key); err == nil && remainTTL > 0 {
		l1TTL = c.l1TTL(remainTTL)
	}
	_ = c.l1.Set(ctx, key, dest, l1TTL)
	return nil
}

func (c *multiLevelCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := c.l2.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	_ = c.l1.Set(ctx, key, value, c.l1TTL(ttl))
	return nil
}

func (c *multiLevelCache) Del(ctx context.Context, keys ...string) error {
	_ = c.l1.Del(ctx, keys...)
	return c.l2.Del(ctx, keys...)
}

func (c *multiLevelCache) Exists(ctx context.Context, key string) (bool, error) {
	if ok, err := c.l1.Exists(ctx, key); err == nil && ok {
		return true, nil
	}
	return c.l2.Exists(ctx, key)
}

func (c *multiLevelCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	_ = c.l1.Expire(ctx, key, c.l1TTL(ttl))
	return c.l2.Expire(ctx, key, ttl)
}

func (c *multiLevelCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	// TTL 以 L2 为准
	return c.l2.TTL(ctx, key)
}

func (c *multiLevelCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))
	missed := make([]string, 0, len(keys))

	// 先查 L1
	l1res, _ := c.l1.MGet(ctx, keys)
	for _, k := range keys {
		if v, ok := l1res[k]; ok {
			result[k] = v
		} else {
			missed = append(missed, k)
		}
	}
	if len(missed) == 0 {
		return result, nil
	}

	// 再查 L2
	l2res, err := c.l2.MGet(ctx, missed)
	if err != nil {
		return result, err
	}
	for k, v := range l2res {
		result[k] = v
	}
	return result, nil
}

func (c *multiLevelCache) MSet(ctx context.Context, kvs map[string]any, ttl time.Duration) error {
	if err := c.l2.MSet(ctx, kvs, ttl); err != nil {
		return err
	}
	_ = c.l1.MSet(ctx, kvs, c.l1TTL(ttl))
	return nil
}

func (c *multiLevelCache) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	if err := c.Get(ctx, key, dest); err == nil {
		return nil
	}
	// 委托给 L2 的 singleflight
	return c.l2.GetOrSet(ctx, key, dest, ttl, func() (any, error) {
		val, err := fn()
		if err != nil {
			return nil, err
		}
		_ = c.l1.Set(ctx, key, val, c.l1TTL(ttl))
		return val, nil
	})
}

func (c *multiLevelCache) Incr(ctx context.Context, key string) (int64, error) {
	_ = c.l1.Del(ctx, key) // 计数器操作使 L1 失效，以 L2 为准
	return c.l2.Incr(ctx, key)
}

func (c *multiLevelCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	_ = c.l1.Del(ctx, key)
	return c.l2.IncrBy(ctx, key, delta)
}

func (c *multiLevelCache) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	_ = c.l1.Del(ctx, key)
	return c.l2.DecrBy(ctx, key, delta)
}

func (c *multiLevelCache) Flush(ctx context.Context) error {
	_ = c.l1.Flush(ctx)
	return c.l2.Flush(ctx)
}

func (c *multiLevelCache) Close() error {
	_ = c.l1.Close()
	return c.l2.Close()
}
