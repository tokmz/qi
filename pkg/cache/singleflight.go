package cache

import (
	"context"
	"time"

	"golang.org/x/sync/singleflight"
)

// SingleflightCache 单flight 缓存（防击穿）
// 内部持有 singleflight.Group，确保同一 key 的并发请求只执行一次
type SingleflightCache struct {
	Cache
	group singleflight.Group
}

// NewSingleflightCache 创建单flight 缓存装饰器
// 参数 c 是底层缓存实现
func NewSingleflightCache(c Cache) *SingleflightCache {
	return &SingleflightCache{Cache: c}
}

// Do 执行缓存操作（防击穿）
// 多个相同 key 的并发请求只会执行一次 fn，所有请求返回相同结果
func (s *SingleflightCache) Do(
	ctx context.Context,
	key string,
	ttl time.Duration,
	fn func() (any, error),
) (any, error) {
	v, err, _ := s.group.Do(key, func() (interface{}, error) {
		var r interface{}
		checkErr := s.Cache.Get(ctx, key, &r)
		if checkErr == nil {
			return r, nil
		}
		r, checkErr = fn()
		if checkErr != nil {
			return nil, checkErr
		}
		_ = s.Cache.Set(ctx, key, r, ttl)
		return r, nil
	})
	return v, err
}

// Forget 清除缓存（强制刷新）
// 删除 singleflight 缓存状态，下次请求会重新执行 fn
func (s *SingleflightCache) Forget(key string) {
	s.group.Forget(key)
}

// Remember 标准缓存操作（不防击穿）
// 适合非热点数据
func Remember[T any](
	ctx context.Context,
	c Cache,
	key string,
	ttl time.Duration,
	fn func() (T, error),
) (T, error) {
	var result T
	if err := c.Get(ctx, key, &result); err == nil {
		return result, nil
	}
	result, err := fn()
	if err != nil {
		return result, err
	}
	_ = c.Set(ctx, key, result, ttl)
	return result, nil
}

// RememberWithLock 带锁的缓存操作（防击穿）
// 使用 singleflight 确保同一 key 的多个并发请求只执行一次
// 适合热点数据，防止缓存击穿
func RememberWithLock[T any](
	ctx context.Context,
	sf *SingleflightCache,
	key string,
	ttl time.Duration,
	fn func() (T, error),
) (T, error) {
	v, err := sf.Do(ctx, key, ttl, func() (any, error) {
		return fn()
	})
	if err != nil {
		var zero T
		return zero, err
	}
	result, ok := v.(T)
	if !ok {
		var zero T
		return zero, ErrCacheSerialization.WithMessage("invalid result type")
	}
	return result, nil
}

// DoTyped 类型安全的 Do 操作（防击穿）
func (s *SingleflightCache) DoTyped(
	ctx context.Context,
	key string,
	ttl time.Duration,
	fn func() (any, error),
) (any, error) {
	return s.Do(ctx, key, ttl, fn)
}

// GetTyped 泛型 Get
func GetTyped[T any](ctx context.Context, c Cache, key string) (T, error) {
	var result T
	err := c.Get(ctx, key, &result)
	return result, err
}

// SetTyped 泛型 Set
func SetTyped[T any](ctx context.Context, c Cache, key string, value T, ttl time.Duration) error {
	return c.Set(ctx, key, value, ttl)
}
