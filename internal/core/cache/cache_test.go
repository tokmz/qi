package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 测试辅助函数
func setupTestManager(t *testing.T) (*Manager, *redis.Client) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	cfg := DefaultConfig()
	cfg.Redis = rdb
	cfg.KeyPrefix = "test:"

	manager, err := New(cfg, &NoopLogger{})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	return manager, rdb
}

func cleanupTestManager(t *testing.T, manager *Manager, rdb *redis.Client) {
	ctx := context.Background()
	rdb.FlushDB(ctx)
	manager.Close()
	rdb.Close()
}

func TestSetAndGet(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	type testData struct {
		ID   int64
		Name string
	}

	original := &testData{ID: 123, Name: "test"}

	// 设置缓存
	err := manager.Set(ctx, "test:key", original, time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 获取缓存
	var result testData
	err = manager.Get(ctx, "test:key", &result)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if result.ID != original.ID || result.Name != original.Name {
		t.Errorf("Data mismatch: got %+v, want %+v", result, original)
	}
}

func TestCacheMiss(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	var result string
	err := manager.Get(ctx, "nonexistent", &result)
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	// 设置缓存
	manager.Set(ctx, "test:delete", "value", time.Minute)

	// 删除缓存
	err := manager.Delete(ctx, "test:delete")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// 验证已删除
	var result string
	err = manager.Get(ctx, "test:delete", &result)
	if err != ErrCacheMiss {
		t.Error("Cache should be deleted")
	}
}

func TestGetOrLoad(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	loadCount := 0
	loader := func() (interface{}, error) {
		loadCount++
		return "loaded value", nil
	}

	// 第一次调用，会执行 loader
	var result1 string
	err := manager.GetOrLoad(ctx, "test:loader", &result1, loader, time.Minute)
	if err != nil {
		t.Fatalf("GetOrLoad failed: %v", err)
	}

	if result1 != "loaded value" {
		t.Errorf("Expected 'loaded value', got %s", result1)
	}

	if loadCount != 1 {
		t.Errorf("Loader should be called once, called %d times", loadCount)
	}

	// 第二次调用，应该从缓存获取
	var result2 string
	err = manager.GetOrLoad(ctx, "test:loader", &result2, loader, time.Minute)
	if err != nil {
		t.Fatalf("GetOrLoad failed: %v", err)
	}

	if loadCount != 1 {
		t.Errorf("Loader should still be called once, called %d times", loadCount)
	}
}

func TestBatchOperations(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	// 批量设置
	items := map[string]interface{}{
		"batch:1": "value1",
		"batch:2": "value2",
		"batch:3": "value3",
	}

	err := manager.SetMulti(ctx, items, time.Minute)
	if err != nil {
		t.Fatalf("SetMulti failed: %v", err)
	}

	// 批量获取
	keys := []string{"batch:1", "batch:2", "batch:3"}
	results, err := manager.GetMulti(ctx, keys)
	if err != nil {
		t.Fatalf("GetMulti failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// 批量删除
	err = manager.DeleteMulti(ctx, keys)
	if err != nil {
		t.Fatalf("DeleteMulti failed: %v", err)
	}
}

func TestIncrement(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	// 自增
	count1, err := manager.Incr(ctx, "counter:test")
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	if count1 != 1 {
		t.Errorf("Expected count 1, got %d", count1)
	}

	count2, err := manager.Incr(ctx, "counter:test")
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	if count2 != 2 {
		t.Errorf("Expected count 2, got %d", count2)
	}

	// 增加指定值
	count3, err := manager.IncrBy(ctx, "counter:test", 10)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}

	if count3 != 12 {
		t.Errorf("Expected count 12, got %d", count3)
	}
}

func TestExpire(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	// 设置缓存
	manager.Set(ctx, "expire:test", "value", time.Minute)

	// 更新过期时间
	err := manager.Expire(ctx, "expire:test", 2*time.Second)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}

	// 获取 TTL
	ttl, err := manager.TTL(ctx, "expire:test")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 || ttl > 2*time.Second {
		t.Errorf("Expected TTL around 2s, got %v", ttl)
	}
}

func TestStats(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	// 执行一些操作
	manager.Set(ctx, "stats:1", "value", time.Minute)
	
	var result string
	manager.Get(ctx, "stats:1", &result) // 命中
	manager.Get(ctx, "stats:2", &result) // 未命中

	stats := manager.GetStats()

	if stats.Requests != 2 {
		t.Errorf("Expected 2 requests, got %d", stats.Requests)
	}

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Sets != 1 {
		t.Errorf("Expected 1 set, got %d", stats.Sets)
	}

	hitRate := manager.GetHitRate()
	if hitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", hitRate)
	}
}

func TestHealth(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	err := manager.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*Config)
		expectError  error
	}{
		{
			name: "valid config",
			modifyConfig: func(cfg *Config) {
				cfg.Redis = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
			},
			expectError: nil,
		},
		{
			name: "missing redis client",
			modifyConfig: func(cfg *Config) {
				cfg.Redis = nil
			},
			expectError: ErrRedisClientRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modifyConfig(cfg)

			err := cfg.Validate()
			if err != tt.expectError {
				t.Errorf("Expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

