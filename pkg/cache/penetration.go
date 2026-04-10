package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

const nullKeyPrefix = "__null__:"

// penetrationGuard 防缓存穿透装饰器（实现 Cache 接口）
//
// 双重防护：
//   1. Bloom filter 快速拦截确定不存在的 key（可选）
//   2. 空值标记：fn 返回 ErrNotFound 时，以独立 key（__null__:<key>）标记，
//      避免序列化歧义，后续 Get 直接拦截不穿透后端
type penetrationGuard struct {
	inner   Cache
	bloom   *bloom.BloomFilter // nil = 不使用 bloom filter
	bloomMu sync.RWMutex
	nullTTL time.Duration
}

func newPenetrationGuard(c Cache, cfg *PenetrationConfig, _ Serializer) (*penetrationGuard, error) {
	g := &penetrationGuard{
		inner:   c,
		nullTTL: cfg.NullTTL,
	}
	if cfg.EnableBloom {
		g.bloom = bloom.NewWithEstimates(cfg.BloomN, cfg.BloomFP)
	}
	return g, nil
}

// nullKey 返回空值标记的缓存 key
func (g *penetrationGuard) nullKey(key string) string {
	return nullKeyPrefix + key
}

// bloomTest 线程安全地测试 bloom filter
func (g *penetrationGuard) bloomTest(key string) bool {
	g.bloomMu.RLock()
	defer g.bloomMu.RUnlock()
	return g.bloom.TestString(key)
}

// bloomAdd 线程安全地向 bloom filter 添加 key
func (g *penetrationGuard) bloomAdd(key string) {
	g.bloomMu.Lock()
	defer g.bloomMu.Unlock()
	g.bloom.AddString(key)
}

func (g *penetrationGuard) Get(ctx context.Context, key string, dest any) error {
	// Bloom filter 快速拦截：确定不存在则直接返回
	if g.bloom != nil && !g.bloomTest(key) {
		return ErrNotFound
	}

	// 检查空值标记（独立 key，无序列化问题）
	if ok, _ := g.inner.Exists(ctx, g.nullKey(key)); ok {
		return ErrNotFound
	}

	return g.inner.Get(ctx, key, dest)
}

func (g *penetrationGuard) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	// 写入真实值：更新 bloom filter，并清除可能存在的空值标记
	if g.bloom != nil {
		g.bloomAdd(key)
	}
	_ = g.inner.Del(ctx, g.nullKey(key))
	return g.inner.Set(ctx, key, value, ttl)
}

func (g *penetrationGuard) Del(ctx context.Context, keys ...string) error {
	// 同时删除空值标记
	nulls := make([]string, len(keys))
	for i, k := range keys {
		nulls[i] = g.nullKey(k)
	}
	_ = g.inner.Del(ctx, nulls...)
	return g.inner.Del(ctx, keys...)
}

func (g *penetrationGuard) Exists(ctx context.Context, key string) (bool, error) {
	if g.bloom != nil && !g.bloomTest(key) {
		return false, nil
	}
	if ok, _ := g.inner.Exists(ctx, g.nullKey(key)); ok {
		return false, nil
	}
	return g.inner.Exists(ctx, key)
}

func (g *penetrationGuard) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return g.inner.Expire(ctx, key, ttl)
}

func (g *penetrationGuard) TTL(ctx context.Context, key string) (time.Duration, error) {
	return g.inner.TTL(ctx, key)
}

func (g *penetrationGuard) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	filtered := make([]string, 0, len(keys))
	for _, k := range keys {
		if g.bloom != nil && !g.bloomTest(k) {
			continue
		}
		// 跳过有空值标记的 key
		if ok, _ := g.inner.Exists(ctx, g.nullKey(k)); ok {
			continue
		}
		filtered = append(filtered, k)
	}
	if len(filtered) == 0 {
		return map[string][]byte{}, nil
	}
	return g.inner.MGet(ctx, filtered)
}

func (g *penetrationGuard) MSet(ctx context.Context, kvs map[string]any, ttl time.Duration) error {
	if g.bloom != nil {
		for k := range kvs {
			g.bloomAdd(k)
		}
	}
	// 清除所有相关空值标记
	nulls := make([]string, 0, len(kvs))
	for k := range kvs {
		nulls = append(nulls, g.nullKey(k))
	}
	_ = g.inner.Del(ctx, nulls...)
	return g.inner.MSet(ctx, kvs, ttl)
}

// GetOrSet 含防穿透：fn 返回 ErrNotFound 时写入空值标记 key，阻止后续穿透
func (g *penetrationGuard) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	// 注意：不在 GetOrSet 中使用 bloom filter 短路。
	// GetOrSet 的语义是"未命中则调用 fn 加载并回写"，bloom 短路会导致新 key 的回调永远不执行。
	// bloom filter 仅适用于纯读操作（Get/Exists/MGet），那里的语义是"确定不存在就不查了"。

	if ok, _ := g.inner.Exists(ctx, g.nullKey(key)); ok {
		return ErrNotFound
	}

	return g.inner.GetOrSet(ctx, key, dest, ttl, func() (any, error) {
		val, err := fn()
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// 写入空值标记（独立 key），不影响原 key 的类型
				_ = g.inner.Set(ctx, g.nullKey(key), true, g.nullTTL)
			}
			return nil, err
		}
		if g.bloom != nil {
			g.bloomAdd(key)
		}
		return val, nil
	})
}

func (g *penetrationGuard) Incr(ctx context.Context, key string) (int64, error) {
	return g.inner.Incr(ctx, key)
}

func (g *penetrationGuard) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return g.inner.IncrBy(ctx, key, delta)
}

func (g *penetrationGuard) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return g.inner.DecrBy(ctx, key, delta)
}

func (g *penetrationGuard) Flush(ctx context.Context) error {
	if g.bloom != nil {
		g.bloomMu.Lock()
		g.bloom.ClearAll()
		g.bloomMu.Unlock()
	}
	return g.inner.Flush(ctx)
}

func (g *penetrationGuard) Close() error {
	return g.inner.Close()
}
