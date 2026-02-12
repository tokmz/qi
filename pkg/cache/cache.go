package cache

import (
	"context"
	"time"
)

// Cache 缓存接口（统一抽象）
type Cache interface {
	// 基础操作
	Get(ctx context.Context, key string, value any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 批量操作
	MGet(ctx context.Context, keys []string, values any) error
	MSet(ctx context.Context, items map[string]any, ttl time.Duration) error
	MSetTx(ctx context.Context, items map[string]any, ttl time.Duration) error

	// TTL 管理
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// 原子操作
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// 工具方法
	Ping(ctx context.Context) error
	Close() error
}

// Serializer 序列化接口
type Serializer interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}
