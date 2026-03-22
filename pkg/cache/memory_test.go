package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestMemory 创建测试用内存缓存
func newTestMemory(maxSize int, defaultTTL time.Duration) *memoryCache {
	cfg := &Config{
		Serializer: JSONSerializer{},
		DefaultTTL: defaultTTL,
		Memory: &MemoryConfig{
			MaxSize:         maxSize,
			CleanupInterval: time.Hour, // 测试中不主动清理
		},
	}
	c, _ := newMemoryCache(cfg)
	return c
}

// ===== 基础操作 =====

func TestMemoryCache_GetMiss(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()

	var v string
	if err := c.Get(context.Background(), "k", &v); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestMemoryCache_SetGet(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
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

func TestMemoryCache_SetNilValue(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()

	if err := c.Set(context.Background(), "k", nil, 0); err != ErrNilValue {
		t.Fatalf("want ErrNilValue, got %v", err)
	}
}

func TestMemoryCache_Del(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
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

func TestMemoryCache_Exists(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("should not exist")
	}
	c.Set(ctx, "k", 1, 0)
	if ok, _ := c.Exists(ctx, "k"); !ok {
		t.Fatal("should exist")
	}
}

func TestMemoryCache_Expire(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", "v", 0) // 永不过期
	if err := c.Expire(ctx, "k", 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("should have expired")
	}
}

func TestMemoryCache_Expire_NotFound(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()

	if err := c.Expire(context.Background(), "missing", time.Minute); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	// 永不过期
	c.Set(ctx, "permanent", 1, 0)
	ttl, err := c.TTL(ctx, "permanent")
	if err != nil || ttl != -1 {
		t.Fatalf("permanent key: want ttl=-1, err=nil, got ttl=%v, err=%v", ttl, err)
	}

	// 有过期时间
	c.Set(ctx, "expiring", 1, time.Minute)
	ttl, err = c.TTL(ctx, "expiring")
	if err != nil || ttl <= 0 || ttl > time.Minute {
		t.Fatalf("expiring key: unexpected ttl=%v err=%v", ttl, err)
	}

	// 不存在
	_, err = c.TTL(ctx, "missing")
	if err != ErrNotFound {
		t.Fatalf("missing key: want ErrNotFound, got %v", err)
	}
}

// ===== TTL 过期 =====

func TestMemoryCache_TTLExpiry_Get(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", "v", 50*time.Millisecond)
	var v string
	if err := c.Get(ctx, "k", &v); err != nil {
		t.Fatal("should exist before expiry")
	}
	time.Sleep(80 * time.Millisecond)
	if err := c.Get(ctx, "k", &v); err != ErrNotFound {
		t.Fatalf("should be expired, got err=%v", err)
	}
}

func TestMemoryCache_DefaultTTL(t *testing.T) {
	c := newTestMemory(0, 50*time.Millisecond)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", "v", 0) // 使用默认 TTL
	time.Sleep(80 * time.Millisecond)
	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("should have expired via default TTL")
	}
}

// ===== 批量操作 =====

func TestMemoryCache_MGet(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
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
		t.Fatal("missing key should not be in result")
	}
}

func TestMemoryCache_MSet(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	err := c.MSet(ctx, map[string]any{
		"x": 1,
		"y": 2,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Exists(ctx, "x"); !ok {
		t.Fatal("x should exist")
	}
	if ok, _ := c.Exists(ctx, "y"); !ok {
		t.Fatal("y should exist")
	}
}

// ===== GetOrSet =====

func TestMemoryCache_GetOrSet_Miss(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	var v string
	called := false
	err := c.GetOrSet(ctx, "k", &v, time.Hour, func() (any, error) {
		called = true
		return "loaded", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("fn should be called on miss")
	}
	if v != "loaded" {
		t.Fatalf("want loaded, got %s", v)
	}
}

func TestMemoryCache_GetOrSet_Hit(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", "cached", 0)

	var v string
	called := false
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

func TestMemoryCache_GetOrSet_Singleflight(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
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
				time.Sleep(20 * time.Millisecond)
				return "value", nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	// singleflight 保证 fn 只被调用 1 次（偶尔因调度可能 2 次，但绝不是 20 次）
	if callCount > 2 {
		t.Errorf("fn should be called at most 2 times due to singleflight, got %d", callCount)
	}
}

func TestMemoryCache_GetOrSet_FnError(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	var v string
	err := c.GetOrSet(ctx, "k", &v, time.Hour, func() (any, error) {
		return nil, ErrNotFound
	})
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// ===== 计数器 =====

func TestMemoryCache_IncrBy(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	n, err := c.IncrBy(ctx, "cnt", 5)
	if err != nil || n != 5 {
		t.Fatalf("want 5, got %d, err %v", n, err)
	}
	n, err = c.IncrBy(ctx, "cnt", 3)
	if err != nil || n != 8 {
		t.Fatalf("want 8, got %d, err %v", n, err)
	}
}

func TestMemoryCache_Incr(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		n, err := c.Incr(ctx, "cnt")
		if err != nil || n != i {
			t.Fatalf("iteration %d: want %d, got %d, err %v", i, i, n, err)
		}
	}
}

func TestMemoryCache_DecrBy(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	c.IncrBy(ctx, "cnt", 10)
	n, err := c.DecrBy(ctx, "cnt", 3)
	if err != nil || n != 7 {
		t.Fatalf("want 7, got %d, err %v", n, err)
	}
}

func TestMemoryCache_IncrBy_PreservesTTL(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	// 用整数设置初始值（JSON 序列化为 "5"，strconv 可解析）
	c.Set(ctx, "cnt", 5, 80*time.Millisecond)

	n, _ := c.IncrBy(ctx, "cnt", 1)
	if n != 6 {
		t.Fatalf("want 6, got %d", n)
	}

	// TTL 应被保留，等到过期
	time.Sleep(120 * time.Millisecond)
	if ok, _ := c.Exists(ctx, "cnt"); ok {
		t.Fatal("counter should have expired (TTL not preserved properly)")
	}
}

// ===== LRU 淘汰 =====

func TestMemoryCache_LRU_Eviction(t *testing.T) {
	c := newTestMemory(3, 0) // 最多 3 条
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "a", 1, 0)
	c.Set(ctx, "b", 2, 0)
	c.Set(ctx, "c", 3, 0)

	// 访问 a，使其成为最近使用：LRU 顺序 [a, c, b]
	var v int
	c.Get(ctx, "a", &v)

	// 插入 d，应淘汰最久未使用的 b
	c.Set(ctx, "d", 4, 0)

	if ok, _ := c.Exists(ctx, "b"); ok {
		t.Error("b should have been evicted (LRU)")
	}
	for _, key := range []string{"a", "c", "d"} {
		if ok, _ := c.Exists(ctx, key); !ok {
			t.Errorf("%s should still exist", key)
		}
	}
}

func TestMemoryCache_LRU_UpdateExisting(t *testing.T) {
	c := newTestMemory(2, 0)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "a", 1, 0)
	c.Set(ctx, "b", 2, 0)

	// 更新 a（已存在），不应新增条目
	c.Set(ctx, "a", 99, 0)

	// b 不应被淘汰
	if ok, _ := c.Exists(ctx, "b"); !ok {
		t.Error("b should still exist after updating a")
	}
}

func TestMemoryCache_LRU_Unlimited(t *testing.T) {
	c := newTestMemory(0, 0) // maxSize=0 不限制
	defer c.Close()
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		c.Set(ctx, string(rune('a'+i%26))+string(rune('0'+i%10)), i, 0)
	}
	// 不 panic 即通过
}

// ===== Flush / Close =====

func TestMemoryCache_Flush(t *testing.T) {
	c := newTestMemory(0, 0)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "a", 1, 0)
	c.Set(ctx, "b", 2, 0)
	c.Flush(ctx)

	for _, key := range []string{"a", "b"} {
		if ok, _ := c.Exists(ctx, key); ok {
			t.Errorf("%s should be flushed", key)
		}
	}
}

func TestMemoryCache_KeyPrefix(t *testing.T) {
	cfg := &Config{
		Serializer: JSONSerializer{},
		KeyPrefix:  "test:",
		Memory:     &MemoryConfig{CleanupInterval: time.Hour},
	}
	c, _ := newMemoryCache(cfg)
	defer c.Close()
	ctx := context.Background()

	c.Set(ctx, "k", "v", 0)

	// 内部 key 应带前缀
	c.mu.Lock()
	_, withPrefix := c.items["test:k"]
	_, withoutPrefix := c.items["k"]
	c.mu.Unlock()

	if !withPrefix {
		t.Error("key should be stored with prefix")
	}
	if withoutPrefix {
		t.Error("key should not be stored without prefix")
	}
}
