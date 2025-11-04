package cache_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qi/internal/core/cache"

	"github.com/redis/go-redis/v9"
)

// User 示例用户结构
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// Example_basic 基础使用示例
func Example_basic() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, err := cache.New(cfg, &cache.NoopLogger{})
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 设置缓存
	user := &User{ID: 123, Name: "Alice", Age: 25}
	err = manager.Set(ctx, "user:123", user, 10*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	// 获取缓存
	var cachedUser User
	err = manager.Get(ctx, "user:123", &cachedUser)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s, Age: %d\n", cachedUser.Name, cachedUser.Age)
}

// Example_防缓存击穿
func Example_singleflight() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, _ := cache.New(cfg, &cache.NoopLogger{})
	defer manager.Close()

	ctx := context.Background()

	// 模拟并发请求同一个key
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var user User
			manager.GetOrLoad(ctx, "user:999", &user, func() (interface{}, error) {
				// 这个函数只会被执行一次
				time.Sleep(100 * time.Millisecond) // 模拟慢查询
				return &User{ID: 999, Name: "Bob", Age: 30}, nil
			}, 10*time.Minute)
		}()
	}
	wg.Wait()

	stats := manager.GetStats()
	fmt.Printf("Loader calls: %d\n", stats.LoaderCalls)
	fmt.Printf("Singleflight hits: %d\n", stats.SingleflightHits)
}

// Example_批量操作
func Example_batch() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, _ := cache.New(cfg, &cache.NoopLogger{})
	defer manager.Close()

	ctx := context.Background()

	// 批量设置
	items := map[string]interface{}{
		"user:1": &User{ID: 1, Name: "Alice", Age: 25},
		"user:2": &User{ID: 2, Name: "Bob", Age: 30},
		"user:3": &User{ID: 3, Name: "Charlie", Age: 35},
	}
	manager.SetMulti(ctx, items, 10*time.Minute)

	// 批量获取
	keys := []string{"user:1", "user:2", "user:3"}
	results, _ := manager.GetMulti(ctx, keys)

	fmt.Printf("Fetched %d items\n", len(results))
}

// Example_缓存预热
func Example_warmup() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, _ := cache.New(cfg, &cache.NoopLogger{})
	defer manager.Close()

	ctx := context.Background()

	// 预热数据
	items := []cache.WarmupItem{
		{Key: "user:1", Value: &User{ID: 1, Name: "Alice", Age: 25}, TTL: 1 * time.Hour},
		{Key: "user:2", Value: &User{ID: 2, Name: "Bob", Age: 30}, TTL: 1 * time.Hour},
	}

	manager.Warmup(ctx, items)

	fmt.Println("Cache warmed up")
}

// Example_计数器
func Example_counter() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, _ := cache.New(cfg, &cache.NoopLogger{})
	defer manager.Close()

	ctx := context.Background()

	// 增加浏览量
	count, _ := manager.Incr(ctx, "article:123:views")
	fmt.Printf("View count: %d\n", count)

	// 增加指定数量
	newCount, _ := manager.IncrBy(ctx, "article:123:views", 10)
	fmt.Printf("New count: %d\n", newCount)
}

// Example_统计信息
func Example_stats() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, _ := cache.New(cfg, &cache.NoopLogger{})
	defer manager.Close()

	ctx := context.Background()

	// 执行一些操作
	manager.Set(ctx, "key1", "value1", time.Minute)
	manager.Get(ctx, "key1", nil)
	manager.Get(ctx, "key2", nil)

	// 获取统计
	stats := manager.GetStats()
	fmt.Printf("Requests: %d\n", stats.Requests)
	fmt.Printf("Hits: %d\n", stats.Hits)
	fmt.Printf("Misses: %d\n", stats.Misses)
	fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate*100)
}

// Example_tracing 链路追踪示例
func Example_tracing() {
	// 注意：实际使用时需要先初始化 tracing.InitGlobal(cfg)
	// 这里仅演示 cache 包如何与 tracing 集成
	
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	cfg := cache.DefaultConfig()
	cfg.Redis = rdb

	manager, _ := cache.New(cfg, &cache.NoopLogger{})
	defer manager.Close()

	ctx := context.Background()

	// cache 操作会自动创建 span 并记录以下信息：
	// - cache.key: 缓存键
	// - cache.operation: 操作类型
	// - cache.hit: 是否命中（Get 操作）
	// - cache.singleflight_hit: 是否命中 singleflight（GetOrLoad 操作）
	
	// Set 操作会创建 span: cache.Set
	manager.Set(ctx, "user:123", &User{ID: 123, Name: "Alice", Age: 25}, 10*time.Minute)
	
	// Get 操作会创建 span: cache.Get，并记录 cache.hit=true
	var user User
	manager.Get(ctx, "user:123", &user)
	
	// GetOrLoad 操作会创建 span: cache.GetOrLoad
	// 如果缓存未命中，会记录 cache.loaded_from_source=true
	manager.GetOrLoad(ctx, "user:456", &user, func() (interface{}, error) {
		// 从数据库加载（可以在这里创建子 span）
		return &User{ID: 456, Name: "Bob", Age: 30}, nil
	}, 10*time.Minute)

	fmt.Println("Cache operations with tracing completed")
	// Output: Cache operations with tracing completed
}

