package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testRedisAddr = "127.0.0.1:6379"
const testRedisPrefix = "qi_test:"

func newTestRedis(t *testing.T) *redisCache {
	t.Helper()
	cfg := &Config{
		Serializer: JSONSerializer{},
		KeyPrefix:  testRedisPrefix,
		Redis: &RedisConfig{
			Addr:          testRedisAddr,
			DisableJitter: true, // 测试中禁用抖动，TTL 精确可测
		},
	}
	cfg.setDefaults()
	c, err := newRedisCache(cfg)
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	// 清理测试前缀的 key
	c.Flush(context.Background())
	t.Cleanup(func() { c.Flush(context.Background()); c.Close() })
	return c
}

// ===== 基础操作 =====

func TestRedisCache_SetGet(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", "hello", 0); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := c.Get(ctx, "k", &v); err != nil {
		t.Fatal(err)
	}
	if v != "hello" {
		t.Fatalf("want hello, got %s", v)
	}
}

func TestRedisCache_GetMiss(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	var v string
	if err := c.Get(ctx, "missing", &v); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRedisCache_SetNilValue(t *testing.T) {
	c := newTestRedis(t)
	if err := c.Set(context.Background(), "k", nil, 0); !errors.Is(err, ErrNilValue) {
		t.Fatalf("want ErrNilValue, got %v", err)
	}
}

func TestRedisCache_Del(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "a", 1, 0)
	c.Set(ctx, "b", 2, 0)
	c.Del(ctx, "a", "b")

	for _, key := range []string{"a", "b"} {
		if ok, _ := c.Exists(ctx, key); ok {
			t.Errorf("%s should be deleted", key)
		}
	}
}

func TestRedisCache_Exists(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("should not exist")
	}
	c.Set(ctx, "k", 1, 0)
	if ok, _ := c.Exists(ctx, "k"); !ok {
		t.Fatal("should exist")
	}
}

func TestRedisCache_TTLExpiry(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "k", "v", 100*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("should have expired")
	}
}

func TestRedisCache_Expire(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "k", "v", 0)
	if err := c.Expire(ctx, "k", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("should have expired after Expire")
	}
}

func TestRedisCache_Expire_NotFound(t *testing.T) {
	c := newTestRedis(t)
	if err := c.Expire(context.Background(), "missing", time.Minute); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRedisCache_TTL(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "k", "v", time.Minute)
	ttl, err := c.TTL(ctx, "k")
	if err != nil || ttl <= 0 || ttl > time.Minute {
		t.Fatalf("unexpected ttl=%v err=%v", ttl, err)
	}

	_, err = c.TTL(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound for missing key, got %v", err)
	}
}

// ===== 批量操作 =====

func TestRedisCache_MGet(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "a", "1", 0)
	c.Set(ctx, "b", "2", 0)

	res, err := c.MGet(ctx, []string{"a", "b", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("want 2 results, got %d", len(res))
	}
	if _, ok := res["missing"]; ok {
		t.Fatal("missing key should not appear in result")
	}
}

func TestRedisCache_MSet(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	err := c.MSet(ctx, map[string]any{"x": 1, "y": 2}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"x", "y"} {
		if ok, _ := c.Exists(ctx, k); !ok {
			t.Errorf("%s should exist after MSet", k)
		}
	}
}

// ===== 计数器 =====

func TestRedisCache_Incr(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		n, err := c.Incr(ctx, "cnt")
		if err != nil || n != i {
			t.Fatalf("iteration %d: want %d got %d err %v", i, i, n, err)
		}
	}
}

func TestRedisCache_IncrBy(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	n, _ := c.IncrBy(ctx, "cnt", 10)
	if n != 10 {
		t.Fatalf("want 10, got %d", n)
	}
	n, _ = c.IncrBy(ctx, "cnt", 5)
	if n != 15 {
		t.Fatalf("want 15, got %d", n)
	}
}

func TestRedisCache_DecrBy(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.IncrBy(ctx, "cnt", 10)
	n, err := c.DecrBy(ctx, "cnt", 3)
	if err != nil || n != 7 {
		t.Fatalf("want 7, got %d err %v", n, err)
	}
}

// ===== GetOrSet =====

func TestRedisCache_GetOrSet_Miss(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	var v string
	called := false
	err := c.GetOrSet(ctx, "k", &v, time.Hour, func() (any, error) {
		called = true
		return "loaded", nil
	})
	if err != nil || !called || v != "loaded" {
		t.Fatalf("miss: called=%v v=%s err=%v", called, v, err)
	}
}

func TestRedisCache_GetOrSet_Hit(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "k", "cached", 0)
	called := false
	var v string
	c.GetOrSet(ctx, "k", &v, time.Hour, func() (any, error) {
		called = true
		return "loaded", nil
	})
	if called {
		t.Fatal("fn should not be called on hit")
	}
	if v != "cached" {
		t.Fatalf("want cached, got %s", v)
	}
}

func TestRedisCache_GetOrSet_Singleflight(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	var callCount int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var v string
			err := c.GetOrSet(ctx, "sf", &v, time.Hour, func() (any, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(30 * time.Millisecond)
				return "value", nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if callCount > 2 {
		t.Errorf("singleflight should limit fn calls, got %d", callCount)
	}
}

// ===== Flush =====

func TestRedisCache_Flush_RequiresPrefix(t *testing.T) {
	cfg := &Config{
		Serializer: JSONSerializer{},
		Redis:      &RedisConfig{Addr: testRedisAddr, DisableJitter: true},
	}
	cfg.setDefaults()
	c, err := newRedisCache(cfg)
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer c.Close()

	if err := c.Flush(context.Background()); err == nil {
		t.Fatal("Flush without prefix should return error")
	}
}

func TestRedisCache_Flush_WithPrefix(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "a", 1, 0)
	c.Set(ctx, "b", 2, 0)
	if err := c.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"a", "b"} {
		if ok, _ := c.Exists(ctx, k); ok {
			t.Errorf("%s should be flushed", k)
		}
	}
}

// ===== KeyPrefix =====

func TestRedisCache_KeyPrefix(t *testing.T) {
	c := newTestRedis(t)
	ctx := context.Background()

	c.Set(ctx, "k", "v", 0)

	// 直接用底层 client 验证 key 带前缀
	exists, err := c.client.Exists(ctx, testRedisPrefix+"k").Result()
	if err != nil || exists == 0 {
		t.Fatal("key should be stored with prefix in Redis")
	}
	// 不带前缀的 key 不存在
	exists, err = c.client.Exists(ctx, "k").Result()
	if err != nil || exists != 0 {
		t.Fatal("key should NOT exist without prefix")
	}
}
