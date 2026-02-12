# Qi 框架缓存集成

Qi 框架的缓存集成提供了简洁、开箱即用、零侵入的缓存解决方案，支持 Redis 和 Memory 两种驱动。

## 特性

- **简洁易用**：直观的 API 设计，`Get/Set/Delete` 三个核心方法
- **开箱即用**：提供合理的默认配置，快速上手
- **零侵入**：可选集成，不强制依赖
- **类型安全**：利用 Go 泛型提供类型安全的 API
- **统一错误处理**：与 `pkg/errors` 集成
- **Context 传递**：支持超时控制、链路追踪
- **多驱动支持**：Redis（生产环境）、Memory（开发/测试）

## 快速开始

### 安装依赖

```bash
go get github.com/redis/go-redis/v9
go get github.com/patrickmn/go-cache
```

### 基础使用

```go
package main

import (
    "context"
    "time"
    "qi/pkg/cache"
)

func main() {
    // 创建内存缓存
    c, err := cache.NewWithOptions(
        cache.WithMemory(&cache.MemoryConfig{
            DefaultExpiration: 10 * time.Minute,
            CleanupInterval:   5 * time.Minute,
        }),
        cache.WithKeyPrefix("myapp:"),
    )
    if err != nil {
        panic(err)
    }
    defer c.Close()

    ctx := context.Background()

    // 设置缓存
    user := User{ID: 123, Name: "Alice"}
    err = c.Set(ctx, "user:123", user, 10*time.Minute)

    // 获取缓存
    var cachedUser User
    err = c.Get(ctx, "user:123", &cachedUser)

    // 删除缓存
    err = c.Delete(ctx, "user:123")
}
```

## 配置选项

### Memory 驱动（开发/测试）

```go
c, err := cache.NewWithOptions(
    cache.WithMemory(&cache.MemoryConfig{
        DefaultExpiration: 10 * time.Minute, // 默认过期时间
        CleanupInterval:   5 * time.Minute,  // 清理间隔
        MaxEntries:        0,                 // 最大条目数（0 表示无限制）
    }),
    cache.WithKeyPrefix("myapp:"),
    cache.WithDefaultTTL(10 * time.Minute),
)
```

### Redis 驱动（生产环境）

#### 单机模式

```go
c, err := cache.NewWithOptions(
    cache.WithRedis(&cache.RedisConfig{
        Addr:         "localhost:6379",
        Password:     "",
        DB:           0,
        PoolSize:     100,
        MinIdleConns: 10,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    }),
    cache.WithKeyPrefix("myapp:"),
)
```

#### 集群模式

```go
c, err := cache.NewWithOptions(
    cache.WithRedis(&cache.RedisConfig{
        Mode:  cache.RedisCluster,
        Addrs: []string{
            "localhost:7000",
            "localhost:7001",
            "localhost:7002",
        },
        Password: "",
    }),
)
```

#### 哨兵模式

```go
c, err := cache.NewWithOptions(
    cache.WithRedis(&cache.RedisConfig{
        Mode:       cache.RedisSentinel,
        MasterName: "mymaster",
        Addrs: []string{
            "localhost:26379",
            "localhost:26380",
            "localhost:26381",
        },
        Password: "",
    }),
)
```

## API 文档

### 基础操作

```go
// Get 获取缓存
err := c.Get(ctx, "key", &value)

// Set 设置缓存
err := c.Set(ctx, "key", value, 10*time.Minute)

// Delete 删除缓存
err := c.Delete(ctx, "key1", "key2", "key3")

// Exists 检查键是否存在
exists, err := c.Exists(ctx, "key")
```

### 批量操作

```go
// MGet 批量获取
var users []User
keys := []string{"user:1", "user:2", "user:3"}
err := c.MGet(ctx, keys, &users)

// MSet 批量设置（Pipeline，高性能）
items := map[string]any{
    "user:1": user1,
    "user:2": user2,
    "user:3": user3,
}
err := c.MSet(ctx, items, 10*time.Minute)

// MSetTx 批量设置（事务，原子性）
// 使用 Redis Watch 机制，确保所有操作原子性执行
err := c.MSetTx(ctx, items, 10*time.Minute)
```

### TTL 管理

```go
// TTL 获取剩余生存时间
ttl, err := c.TTL(ctx, "key")

// Expire 设置过期时间
err := c.Expire(ctx, "key", 5*time.Minute)
```

### 原子操作

```go
// Incr 自增
count, err := c.Incr(ctx, "counter")

// Decr 自减
count, err := c.Decr(ctx, "counter")

// IncrBy 增加指定值
count, err := c.IncrBy(ctx, "counter", 10)
```

### 泛型 API（类型安全）

```go
// GetTyped 泛型 Get
user, err := cache.GetTyped[User](ctx, c, "user:123")

// SetTyped 泛型 Set
err = cache.SetTyped(ctx, c, "user:123", user, 10*time.Minute)
```

### Remember 模式（缓存装饰器）

```go
// Remember 自动处理缓存逻辑
user, err := cache.Remember(ctx, c, "user:123", 10*time.Minute, func() (User, error) {
    // 缓存未命中时执行此函数
    return userService.GetByID(ctx, 123)
})
```

### 缓存击穿防护

热点数据过期瞬间，大量并发请求穿透到数据库。

```go
// 方式1：创建 SingleflightCache（推荐）
sf := cache.NewSingleflightCache(c)

// 使用 RememberWithLock（防击穿）
user, err := cache.RememberWithLock(ctx, sf, "user:123", 10*time.Minute, func() (User, error) {
    return userService.GetByID(ctx, 123)
})

// 或直接使用 Do 方法
user, err := sf.Do(ctx, "user:123", 10*time.Minute, func() (any, error) {
    return userService.GetByID(ctx, 123)
})

// 强制刷新缓存
sf.Forget("user:123")
```

**原理：** 使用 singleflight.Group 确保同一 key 的多个并发请求只执行一次回调函数，所有请求返回相同结果。

| 场景 | 推荐方式 |
|------|----------|
| 非热点数据 | `Remember` |
| 热点数据 | `RememberWithLock` |
| 需要精细控制 | `NewSingleflight` + `Do` |

## 与 Qi 框架集成

```go
package main

import (
    "qi"
    "qi/pkg/cache"
)

func main() {
    // 创建缓存实例
    c, _ := cache.NewWithOptions(
        cache.WithMemory(cache.DefaultMemoryConfig()),
    )

    // 创建 Qi Engine
    engine := qi.Default()
    r := engine.RouterGroup()

    // 在路由中使用缓存
    r.GET("/user/:id", func(ctx *qi.Context) {
        id := ctx.Param("id")
        key := "user:" + id

        // Remember 模式
        user, err := cache.Remember(ctx.RequestContext(), c, key, 10*time.Minute, func() (*User, error) {
            return db.GetUser(ctx.RequestContext(), id)
        })

        if err != nil {
            ctx.RespondError(err)
            return
        }

        ctx.Success(user)
    })

    engine.Run(":8080")
}
```

## 缓存策略

### Cache-Aside（旁路缓存）

推荐使用，适合读多写少场景：

```go
// 读取
func GetUser(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // 1. 尝试从缓存读取
    var user User
    err := cache.Get(ctx, key, &user)
    if err == nil {
        return &user, nil
    }

    // 2. 缓存未命中，从数据库读取
    user, err = db.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. 写入缓存
    _ = cache.Set(ctx, key, user, 10*time.Minute)
    return &user, nil
}

// 更新
func UpdateUser(ctx context.Context, user *User) error {
    // 1. 更新数据库
    err := db.UpdateUser(ctx, user)
    if err != nil {
        return err
    }

    // 2. 删除缓存（下次读取时重新加载）
    key := fmt.Sprintf("user:%d", user.ID)
    _ = cache.Delete(ctx, key)
    return nil
}
```

## 错误处理

```go
import "qi/pkg/errors"

// 预定义错误
var (
    ErrCacheNotFound      = errors.New(3001, 404, "cache key not found", nil)
    ErrCacheExpired       = errors.New(3002, 404, "cache key expired", nil)
    ErrCacheConnection    = errors.New(3003, 500, "cache connection failed", nil)
    ErrCacheSerialization = errors.New(3004, 500, "cache serialization failed", nil)
    ErrCacheInvalidConfig = errors.New(3005, 500, "cache invalid config", nil)
    ErrCacheOperation     = errors.New(3006, 500, "cache operation failed", nil)
)

// 使用示例
err := c.Get(ctx, "key", &value)
if errors.Is(err, cache.ErrCacheNotFound) {
    // 缓存未命中
}
```

## 性能优化

### 连接池配置

```go
&cache.RedisConfig{
    PoolSize:     100,  // 连接池大小
    MinIdleConns: 10,   // 最小空闲连接
}
```

### 批量操作

使用 `MGet` 和 `MSet` 减少网络往返：

```go
// 批量获取（一次网络请求）
var users []User
err := c.MGet(ctx, []string{"user:1", "user:2", "user:3"}, &users)
```

### 随机 TTL（防止缓存雪崩）

```go
func randomTTL(base time.Duration) time.Duration {
    jitter := time.Duration(rand.Intn(300)) * time.Second // 0-5分钟
    return base + jitter
}

cache.Set(ctx, key, user, randomTTL(10*time.Minute))
```

## 完整示例

查看 `example/cache/main.go` 获取完整的使用示例，包括：

- 基础缓存使用
- Remember 模式
- 泛型 API
- 批量操作
- 计数器（原子操作）
- TTL 管理
- 健康检查

运行示例：

```bash
go run example/cache/main.go
```

## 测试

运行单元测试：

```bash
go test ./pkg/cache/ -v
```

## 链路追踪集成

缓存操作支持 OpenTelemetry 链路追踪，自动记录所有缓存操作的 Span。

### 快速开始

```go
import (
    "qi"
    "qi/pkg/cache"
    "qi/pkg/tracing"
)

func main() {
    // 1. 初始化链路追踪
    _, err := tracing.NewTracerProvider(&tracing.Config{
        Enabled:      true,
        ServiceName:  "my-service",
        ExporterType: "console", // 或 "otlp", "jaeger" 等
    })
    if err != nil {
        panic(err)
    }

    // 2. 创建缓存实例
    c, _ := cache.NewWithOptions(
        cache.WithRedis(cache.DefaultRedisConfig()),
    )

    // 3. 启用链路追踪（装饰器模式）
    tracedCache := cache.NewTracing(c)

    // 使用 tracedCache 进行缓存操作
    ctx := context.Background()
    err = tracedCache.Get(ctx, "user:123", &user)
}
```

### 追踪属性

所有缓存操作都会记录以下属性：

| 属性 | 说明 | 示例 |
|------|------|------|
| `cache.key` | 缓存键 | `"user:123"` |
| `cache.operation` | 操作类型 | `"cache.Get"`, `"cache.Set"` |
| `cache.hit` | 是否命中 | `true` / `false` |
| `cache.miss` | 是否未命中 | `true` / `false` |
| `cache.duration_ms` | 操作耗时(ms) | `15` |
| `cache.ttl_seconds` | TTL 秒数 | `600` |
| `cache.keys_count` | 键数量 | `10` |
| `cache.items_count` | 项数量 | `5` |
| `cache.incr_value` | 自增/自减值 | `1` |

### 与 Qi 框架集成

```go
engine := qi.Default()

// 创建带追踪的缓存
c := cache.NewTracing(cache.NewWithOptions(
    cache.WithRedis(cache.DefaultRedisConfig()),
))

// 在中间件中使用
engine.Use(func(ctx *qi.Context) {
    cctx := ctx.RequestContext()
    // 缓存操作会自动携带当前 TraceID
    _ = c.Get(cctx, "key", &value)
    ctx.Next()
})
```

### 与 ORM 追踪结合

缓存和数据库操作会自动关联到同一 Trace：

```go
// 同一 TraceID 下：
// 1. 缓存查询（未命中）
// 2. 数据库查询
// 3. 缓存写入
```

## 最佳实践

1. **使用键前缀**：避免键冲突
   ```go
   cache.WithKeyPrefix("myapp:")
   ```

2. **合理设置 TTL**：根据数据更新频率设置过期时间
   ```go
   cache.Set(ctx, key, value, 10*time.Minute)
   ```

3. **使用 Remember 模式**：简化缓存逻辑
   ```go
   user, err := cache.Remember(ctx, c, key, ttl, fetchFromDB)
   ```

4. **批量操作优化**：减少网络往返
   ```go
   c.MGet(ctx, keys, &values)
   ```

5. **错误处理**：正确处理缓存未命中
   ```go
   if errors.Is(err, cache.ErrCacheNotFound) {
       // 从数据库加载
   }
   ```

## 许可证

MIT License
