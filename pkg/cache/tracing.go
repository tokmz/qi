package cache

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "qi.cache"

// tracingCache 链路追踪装饰器（最外层，包裹所有驱动和装饰器）
type tracingCache struct {
	inner  Cache
	tracer trace.Tracer
}

func newTracingCache(c Cache) Cache {
	return &tracingCache{
		inner:  c,
		tracer: otel.Tracer(tracerName),
	}
}

func (t *tracingCache) start(ctx context.Context, op string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "cache."+op,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.String("cache.operation", op)),
	)
}

func endSpan(span trace.Span, err error) {
	if err != nil && !errors.Is(err, ErrNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

func (t *tracingCache) Get(ctx context.Context, key string, dest any) error {
	ctx, span := t.start(ctx, "Get")
	span.SetAttributes(attribute.String("cache.key", key))
	err := t.inner.Get(ctx, key, dest)
	if errors.Is(err, ErrNotFound) {
		span.SetAttributes(attribute.Bool("cache.hit", false))
	} else if err == nil {
		span.SetAttributes(attribute.Bool("cache.hit", true))
	}
	endSpan(span, err)
	return err
}

func (t *tracingCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	ctx, span := t.start(ctx, "Set")
	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.String("cache.ttl", ttl.String()),
	)
	err := t.inner.Set(ctx, key, value, ttl)
	endSpan(span, err)
	return err
}

func (t *tracingCache) Del(ctx context.Context, keys ...string) error {
	ctx, span := t.start(ctx, "Del")
	span.SetAttributes(attribute.Int("cache.key_count", len(keys)))
	err := t.inner.Del(ctx, keys...)
	endSpan(span, err)
	return err
}

func (t *tracingCache) Exists(ctx context.Context, key string) (bool, error) {
	ctx, span := t.start(ctx, "Exists")
	span.SetAttributes(attribute.String("cache.key", key))
	ok, err := t.inner.Exists(ctx, key)
	endSpan(span, err)
	return ok, err
}

func (t *tracingCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	ctx, span := t.start(ctx, "Expire")
	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.String("cache.ttl", ttl.String()),
	)
	err := t.inner.Expire(ctx, key, ttl)
	endSpan(span, err)
	return err
}

func (t *tracingCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ctx, span := t.start(ctx, "TTL")
	span.SetAttributes(attribute.String("cache.key", key))
	d, err := t.inner.TTL(ctx, key)
	endSpan(span, err)
	return d, err
}

func (t *tracingCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	ctx, span := t.start(ctx, "MGet")
	span.SetAttributes(attribute.Int("cache.key_count", len(keys)))
	res, err := t.inner.MGet(ctx, keys)
	if err == nil {
		span.SetAttributes(attribute.Int("cache.hit_count", len(res)))
	}
	endSpan(span, err)
	return res, err
}

func (t *tracingCache) MSet(ctx context.Context, kvs map[string]any, ttl time.Duration) error {
	ctx, span := t.start(ctx, "MSet")
	span.SetAttributes(
		attribute.Int("cache.key_count", len(kvs)),
		attribute.String("cache.ttl", ttl.String()),
	)
	err := t.inner.MSet(ctx, kvs, ttl)
	endSpan(span, err)
	return err
}

func (t *tracingCache) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	ctx, span := t.start(ctx, "GetOrSet")
	span.SetAttributes(attribute.String("cache.key", key))
	err := t.inner.GetOrSet(ctx, key, dest, ttl, fn)
	endSpan(span, err)
	return err
}

func (t *tracingCache) Incr(ctx context.Context, key string) (int64, error) {
	ctx, span := t.start(ctx, "Incr")
	span.SetAttributes(attribute.String("cache.key", key))
	n, err := t.inner.Incr(ctx, key)
	endSpan(span, err)
	return n, err
}

func (t *tracingCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	ctx, span := t.start(ctx, "IncrBy")
	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.Int64("cache.delta", delta),
	)
	n, err := t.inner.IncrBy(ctx, key, delta)
	endSpan(span, err)
	return n, err
}

func (t *tracingCache) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	ctx, span := t.start(ctx, "DecrBy")
	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.Int64("cache.delta", delta),
	)
	n, err := t.inner.DecrBy(ctx, key, delta)
	endSpan(span, err)
	return n, err
}

func (t *tracingCache) Flush(ctx context.Context) error {
	ctx, span := t.start(ctx, "Flush")
	err := t.inner.Flush(ctx)
	endSpan(span, err)
	return err
}

func (t *tracingCache) Close() error {
	return t.inner.Close()
}
