package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	cacheTracerName = "qi.cache"
)

// tracedCache 链路追踪缓存装饰器
type tracedCache struct {
	Cache
	tracer trace.Tracer
}

// NewTracing 创建带链路追踪的缓存实例
func NewTracing(c Cache) Cache {
	return &tracedCache{
		Cache:  c,
		tracer: otel.Tracer(cacheTracerName),
	}
}

// StartSpan 启动一个新 Span
func (t *tracedCache) StartSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, operation,
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// wrapOperation 包装操作，自动处理 Span
func (t *tracedCache) wrapOperation(
	ctx context.Context,
	operation string,
	key string,
	fn func(ctx context.Context) error,
) error {
	ctx, span := t.StartSpan(ctx, operation)
	defer span.End()

	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.String("cache.operation", operation),
	)

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(attribute.Int64("cache.duration_ms", duration.Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// wrapOperationMultiKeys 包装多键操作
func (t *tracedCache) wrapOperationMultiKeys(
	ctx context.Context,
	operation string,
	keys []string,
	fn func(ctx context.Context) error,
) error {
	ctx, span := t.StartSpan(ctx, operation)
	defer span.End()

	span.SetAttributes(
		attribute.Int("cache.keys_count", len(keys)),
		attribute.String("cache.operation", operation),
	)

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(attribute.Int64("cache.duration_ms", duration.Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Get 获取缓存（带链路追踪）
func (t *tracedCache) Get(ctx context.Context, key string, value any) error {
	var result error
	err := t.wrapOperation(ctx, "cache.Get", key, func(ctx context.Context) error {
		result = t.Cache.Get(ctx, key, value)
		span := trace.SpanFromContext(ctx)
		if result == nil {
			span.SetAttributes(attribute.Bool("cache.hit", true))
		} else if errors.Is(result, ErrCacheNotFound) {
			span.SetAttributes(attribute.Bool("cache.hit", false))
			return nil // cache miss 不算错误
		}
		return result
	})
	if err != nil {
		return err
	}
	return result
}

// Set 设置缓存（带链路追踪）
func (t *tracedCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return t.wrapOperation(ctx, "cache.Set", key, func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		if ttl > 0 {
			span.SetAttributes(attribute.Float64("cache.ttl_seconds", ttl.Seconds()))
		}
		return t.Cache.Set(ctx, key, value, ttl)
	})
}

// Delete 删除缓存（带链路追踪）
func (t *tracedCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 1 {
		return t.wrapOperation(ctx, "cache.Delete", keys[0], func(ctx context.Context) error {
			return t.Cache.Delete(ctx, keys...)
		})
	}
	return t.wrapOperationMultiKeys(ctx, "cache.DeleteMulti", keys, func(ctx context.Context) error {
		return t.Cache.Delete(ctx, keys...)
	})
}

// Exists 检查键是否存在（带链路追踪）
func (t *tracedCache) Exists(ctx context.Context, key string) (bool, error) {
	var result bool
	err := t.wrapOperation(ctx, "cache.Exists", key, func(ctx context.Context) error {
		var err error
		result, err = t.Cache.Exists(ctx, key)
		return err
	})
	return result, err
}

// MGet 批量获取（带链路追踪）
func (t *tracedCache) MGet(ctx context.Context, keys []string, values any) error {
	return t.wrapOperationMultiKeys(ctx, "cache.MGet", keys, func(ctx context.Context) error {
		return t.Cache.MGet(ctx, keys, values)
	})
}

// MSet 批量设置（带链路追踪）
func (t *tracedCache) MSet(ctx context.Context, items map[string]any, ttl time.Duration) error {
	return t.wrapOperation(ctx, "cache.MSet", fmt.Sprintf("%d items", len(items)), func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.Int("cache.items_count", len(items)))
		if ttl > 0 {
			span.SetAttributes(attribute.Float64("cache.ttl_seconds", ttl.Seconds()))
		}
		return t.Cache.MSet(ctx, items, ttl)
	})
}

// MSetTx 批量设置（带链路追踪，事务原子性）
func (t *tracedCache) MSetTx(ctx context.Context, items map[string]any, ttl time.Duration) error {
	return t.wrapOperation(ctx, "cache.MSetTx", fmt.Sprintf("%d items", len(items)), func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.Int("cache.items_count", len(items)))
		if ttl > 0 {
			span.SetAttributes(attribute.Float64("cache.ttl_seconds", ttl.Seconds()))
		}
		return t.Cache.MSetTx(ctx, items, ttl)
	})
}

// TTL 获取剩余生存时间（带链路追踪）
func (t *tracedCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	var result time.Duration
	err := t.wrapOperation(ctx, "cache.TTL", key, func(ctx context.Context) error {
		var err error
		result, err = t.Cache.TTL(ctx, key)
		return err
	})
	return result, err
}

// Expire 设置过期时间（带链路追踪）
func (t *tracedCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return t.wrapOperation(ctx, "cache.Expire", key, func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.Float64("cache.ttl_seconds", ttl.Seconds()))
		return t.Cache.Expire(ctx, key, ttl)
	})
}

// Incr 自增（带链路追踪）
func (t *tracedCache) Incr(ctx context.Context, key string) (int64, error) {
	var result int64
	err := t.wrapOperation(ctx, "cache.Incr", key, func(ctx context.Context) error {
		var err error
		result, err = t.Cache.Incr(ctx, key)
		return err
	})
	return result, err
}

// Decr 自减（带链路追踪）
func (t *tracedCache) Decr(ctx context.Context, key string) (int64, error) {
	var result int64
	err := t.wrapOperation(ctx, "cache.Decr", key, func(ctx context.Context) error {
		var err error
		result, err = t.Cache.Decr(ctx, key)
		return err
	})
	return result, err
}

// IncrBy 增加指定值（带链路追踪）
func (t *tracedCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	var result int64
	err := t.wrapOperation(ctx, "cache.IncrBy", key, func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.Int64("cache.incr_value", value))
		var err error
		result, err = t.Cache.IncrBy(ctx, key, value)
		return err
	})
	return result, err
}

// Ping 检查连接（带链路追踪）
func (t *tracedCache) Ping(ctx context.Context) error {
	return t.wrapOperation(ctx, "cache.Ping", "", func(ctx context.Context) error {
		return t.Cache.Ping(ctx)
	})
}

// Close 关闭连接（带链路追踪）
func (t *tracedCache) Close() error {
	// Close 不需要 context，直接调用
	return t.Cache.Close()
}
