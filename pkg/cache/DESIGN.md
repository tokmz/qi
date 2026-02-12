# Qi 框架缓存集成架构设计

## 1. 设计目标

为 Qi 框架设计一个**简洁、开箱即用、零侵入**的缓存集成方案，符合框架的核心设计理念：

- **简洁易用**：API 设计直观，开发者无需深入了解底层实现
- **开箱即用**：提供合理的默认配置，快速上手
- **零侵入**：不强制依赖，可选集成
- **类型安全**：利用 Go 泛型提供类型安全的 API
- **统一错误处理**：与 `pkg/errors` 集成
- **Context 传递**：支持超时控制、链路追踪

## 2. 技术选型

### 2.1 缓存后端

| 后端 | 优势 | 劣势 | 适用场景 |
|------|------|------|----------|
| **Redis** | 生产级、分布式、持久化、丰富数据结构 | 需要外部服务 | 生产环境、分布式系统 |
| **Memory** | 零依赖、高性能、简单 | 单机、无持久化 | 开发环境、单机应用 |

**推荐方案**：
- **生产环境**：Redis（使用 `go-redis/redis/v9`）
- **开发/测试**：Memory（使用 `patrickmn/go-cache`）

### 2.2 序列化方案

| 方案 | 优势 | 劣势 | 推荐度 |
|------|------|------|--------|
| **JSON** | 可读性强、跨语言 | 性能一般、体积大 | ⭐⭐⭐⭐ |
| **Msgpack** | 高性能、体积小 | 不可读 | ⭐⭐⭐⭐⭐ |
| **Gob** | Go 原生、高性能 | 仅限 Go | ⭐⭐⭐ |

**推荐方案**：默认使用 **JSON**（兼容性好），支持自定义序列化器。

## 3. 架构设计

### 3.1 核心接口

```go
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
```

### 3.2 配置系统（Options 模式）

参考 `pkg/orm` 和 `pkg/logger` 的设计：

```go
// Config 缓存配置
type Config struct {
    // 驱动类型
    Driver DriverType // redis, memory

    // Redis 配置
    Redis *RedisConfig

    // Memory 配置
    Memory *MemoryConfig

    // 序列化器
    Serializer Serializer

    // 键前缀（避免冲突）
    KeyPrefix string

    // 默认 TTL
 TL time.Duration
}

// RedisConfig Redis 配置
type RedisConfig struct {
    Addr         string        // 地址（单机）
    Addrs        []string      // 地址列表（集群/哨兵）
    Mode         RedisMode     // standalone, cluster, sentinel
    Password     string        // 密码
    DB           int           // 数据库编号
    PoolSize     int           // 连接池大小
    MinIdleConns int           // 最小空闲连接
    MaxRetries   int           // 最大重试次数
    DialTimeout  time.Duration // 连接超时
    ReadTimeout  time.Duration // 读超时
    WriteTimeout time.Duration // 写超时

    // 哨兵模式配置
    MasterName string // 主节点名称
}

// MemoryConfig 内存缓存配置
type MemoryConfig struct {
    DefaultExpiration time.Duration // 默认过期时间
    CleanupInterval   time.Duration // 清理间隔
    MaxEntries        int           // 最大条目数（0 表示无限制）
}

// Option 配置选项
type Option func(*Config)

// 示例 Options
func WithRedis(cfg *RedisConfig) Option
func WithMemory(cfg *MemoryConfig) Option
func WithSerializer(s Serializer) Option
func WithKeyPrefix(prefix string) Option
func WithDefaultTTL(ttl time.Duration) Option
```

### 3.3 驱动实现

#### Redis 驱动

```go
// redisCache Redis 缓存实现
type redisCache struct {
    client     redis.UniversalClient // 支持单机/集群/哨兵
    serializer Serializer
    keyPrefix  string
    defaultTTL time.Duration
}

// 支持三种模式：
// 1. Standalone（单机）
// 2. Cluster（集群）
// 3. Sentinel（哨兵）
```

#### Memory 驱动

```go
// memoryCache 内存缓存实现
type memoryCache struct {
    cache      *cache.Cache // patrickmn/go-cache
    serializer Serializer
    keyPrefix  string
    defaultTTL time.Duration
}
```

### 3.4 序列化器实现

```go
// JSONSerializer JSON 序列化器（默认）
type JSONSerializer struct{}

func (s *JSONSerializer) Marshal(v any) ([]byte, error) {
    return json.Marshal(v)
}

func (s *JSONSerializer) Unmarshal(data []byte, v any) error {
    return json.Unmarshal(data, v)
}

// MsgpackSerializer Msgpack 序列化器（e MsgpackSerializer struct{}

// GobSerializer Gob 序列化器（Go 原生）
type GobSerializer struct{}
```

### 3.5 错误处理

与 `pkg/errors` 集成：

```go
// 预定义错误
var (
    ErrCacheNotFound    = errors.New(3001, 404, "cache key not found", nil)
    ErrCacheExpired     = errors.New(3002, 404, "cache key expired", nil)
    ErrCacheConnection  = errors.New(3003, 500, "cache connection failed", nil)
    ErrCacheSerialization = errors.New(3004, 500, "cache serialization failed", nil)
    ErrCacheInvalidConfig = errors.New(3005, 500, "cache invalid config", nil)
)
```

### 3.6 Context 集成

支持 Qi 框架的 Contex机制：

```go
// 从 qi.Context 提取 context.Context
ctx := c.RequestContext()

// 自动传递 TraceID、UID
cache.Get(ctx, "user:123", &user)

// 支持超时控制
ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()
cache.Set(ctx, "key", value, 10*time.Minute)
```

## 4. API 设计

### 4.1 基础 API

```go
// 创建缓存实例
cache, err := cache.New(&cache.Config{
    Driver: cache.DriverRedis,
    Redis: &cache.RedisConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    },
    KeyPrefix:  "myapp:",
    DefaultTTL: 10 * time.Minute,
})

// 使用 Options 模式
cache, err := cache.NewWithOptions(
    cache.WithRedis(&cache.RedisConfig{
        Addr: "localhost:6379",
    }),
    cache.WithKeyPrefix("myapp:"),
    cache.WithDefaultTTL(10 * time.Minute),
)

// 基础操作
ctx := context.Background()

// Set
err := cache.Set(ctx, "user:123", user, 10*time.Minute)

// Get
var user User
err := cache.Get(ctx, "user:123", &user)
if errors.Is(err, cache.ErrCacheNotFound) {
    // 缓存未命中
}

// Delete
err := cache.Delete(ctx, "user:123")

// Exists
exists, err := cache.Exists(ctx, "user:123")
```

### 4.2 泛型 API（类型安全）

```go
// GetTyped 泛型 Get（类型安全）
func GetTyped[T any](ctx context.Context, c Cache, key string) (T, error)

// SetTyped 泛型 Set
func SetTyped[T any](ctx context.Context, c Cache, key string, value T, ttl time.Duration) error

// 使用示例
user, err := cache.GetTyped[User](ctx, cache, "user:123")
err = cache.SetTyped(ctx, cache, "user:123", user, 10*time.Minute)
```

### 4.3 Remember 模式（缓存装饰器）

```go
// Remember 缓存装饰器（Cache-Aside 模式）
func Remember[T any](ctx context.Context, c Cache, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
    // 1. 尝试从缓存获取
    var result T
    err := c.Get(ctx, key, &result)
    if err == nil {
        return result, nil
    }

    // 2. 缓存未命中，执行回调函数
    result, err = fn()
    if err != nil {
        return result, err
    }

    // 3. 写入缓存
    _ = c.Set(ctx, key, result, ttl)
    return result, nil
}

// 使用user, err := cache.Remember(ctx, cache, "user:123", 10*time.Minute, func() (User, error) {
    return userService.GetByID(ctx, 123)
})
```

### 4.4 标签管理（Tag-based Invalidation）

```go
// TaggedCache 支持标签的缓存
type TaggedCache interface {
    Cache
    Tags(tags ...string) TaggedCache
    Flush() error // 清空当前标签下的所有缓存
}

// 使用示例
taggedCache := cache.Tags("users", "posts")
taggedCache.Set(ctx, "user:123", user, 10*time.Minute)
taggedCache.Set(ctx, "post:456", post, 10*time.Minute)

// 清空所有带 "users" 标签的缓存
cache.Tags("users").Flush()
```

### 4.5 批量操作

```go
// MGet 批量获取
keys := []string{"user:1", "user:2", "user:3"}
var users []User
err := cache.MGet(ctx, keys, &users)

// MSet 批量设置
items := map[string]any{
    "user:1": user1,
    "user:2": user2,
    "user:3": user3,
}
err := cache.MSet(ctx, items, 10*time.Minute)
```

### 4.6 原子操作

```go
// 计数器
count, err := cache.Incr(ctx, "page_views")
count, err := cache.IncrBy(ctx, "page_views", 10)
count, err := cache.Decr(ctx, "stock:123")
```

## 5. 缓存策略

### 5.1 Cache-Aside（旁路缓存）

**推荐使用**，适合读多写少场景：

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

### 5.2 Write-Through（写穿透）

适合写多读多场景：

```go
func UpdateUser(ctx context.Context, user *User) error {
    // 1. 更新数据库
    err := db.UpdateUser(ctx, user)
    if err != nil {
        return err
    }

    // 2. 更新缓存
    key := fmt.Sprintf("user:%d", user.ID)
    _ = cache.Set(ctx, key, user, 10*time.Minute)
    return nil
}
```

### 5.3 Write-Behind（写回）

适合高并发写场景（需要异步队列支持，暂不实现）。

## 6. 性能优化

### 6.1 连接池

Redis 驱动使用连接池：

```go
&RedisConfig{
    PoolSize:     100,  // 连接池大小
    MinIdleConns: 10,   // 最小空闲连接
}
```

### 6.2 Pipeline（批量操作）

```go
// Redis Pipeline 支持
pipe := redisClient.Pipeline()
pipe.Set(ctx, "key1", "value1", 0)
pipe.Set(ctx, "key2", "value2", 0)
_, err := pipe.Exec(ctx)
```

### 6.3 序列化优化

- 默认使用 JSON（兼容性）
- 高性能场景使用 Msgpack（体积小 30-50%）
- Go 内部使用 Gob（最快）

### 6.4 缓存预热

```go
// Warmup 缓存预热
func Warmup(ctx context.Context, cache Cache) error {
    users, err := db.GetAllUsers(ctx)
    if err != nil {
        return err
    }

    for _, user := range users {
        key := fmt.Sprintf("user:%d", user.ID)
        _ = cache.Set(ctx, key, user, 1*time.Hour)
    }
    return nil
}
```

## 7. 数据一致性

### 7.1 缓存更新策略

| 策略 | 优势 | 劣势 | 推荐度 |
|------|------|------|--------|
| **先删缓存，再更新数据库** | 简单 | 可能读到旧数据 | ⭐⭐⭐ |
| **先更新数据库，再删缓存** | 数据一致性好 | 删除失败导致脏数据 | ⭐⭐⭐⭐⭐ |
| **先更新数据库，再更新缓存** | 实时性好 | 并发问题 | ⭐⭐⭐ |

**推荐方案**：**先更新数据库，再删除缓存**（Cache-Aside）。

### 7.2 缓存穿透（Cache Penetration）

**问题**：查询不存在的数据，导致每次都查数据库。

**解决方案**：
1. **缓存空值**（推荐）
2. **布隆过滤器**（高级）

```go
// 缓存空值
func GetUser(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    var user User
    err := cache.Get(ctx, key, &user)
    if err == nil {
        if user.ID == 0 {
            return nil, ErrUserNotFound // 空值标记
        }
        return &user, nil
    }

    user, err = db.GetUser(ctx, id)
    if err == sql.ErrNoRows {
        // 缓存空值（短 TTL）
        _ = cache.Set(ctx, key, User{}, 1*time.Minute)
        return nil, ErrUserNotFound
    }

    _ = cache.Set(ctx, key, user, 10*time.Minute)
    return &user, nil
}
```

### 7.3 缓存击穿（Cache Breakdown）

**问题**：热点数据过期，大量请求同时查数据库。

**解决方案**：**分布式锁**（singleflight）

```go
import "golang.org/x/sync/singleflight"

var g singleflight.Group

func GetUser(ctx context.Context, id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // singleflight 保证同一时刻只有一个请求查数据库
    v, err, _ := g.Do(key, func() (interface{}, erro    var user User
        err := cache.Get(ctx, key, &user)
        if err == nil {
            return &user, nil
        }

        user, err = db.GetUser(ctx, id)
        if err != nil {
            return nil, err
        }

        _ = cache.Set(ctx, key, user, 10*time.Minute)
        return &user, nil
    })

    if err != nil {
        return nil, err
    }
    return v.(*User), nil
}
```

### 7.4 缓存雪崩（Cache Avalanche）

**问题**：大量缓存同时过期，导致数据库压力激增。

**解决方案**：
1. **随机 TTL**（推荐）
2. **永不过期 + 异步更新**

```go
// 随机 TTL
func randomTTL(base time.Duration) time.Duration {
    jer := time.Duration(rand.Intn(300)) * time.Second // 0-5分钟
    return base + jitter
}

cache.Set(ctx, key, user, randomTTL(10*time.Minute))
```

## 8. 安全性

### 8.1 键命名规范

```go
// 使用前缀避免冲突
cache.WithKeyPrefix("myapp:")

// 键命名规范：<prefix>:<resource>:<id>
"myapp:user:123"
"myapp:post:456"
"myapp:session:abc123"
```

### 8.2 敏感数据加密

```go
// EncryptedSerializer 加密序列化器
type EncryptedSerializer struct {
    inner Serializer
    key   []byte // AES 密钥
}

func (s *EncryptedSerializer) Marshal(v any) ([]byte, error) {
    data, err := s.inner.Marshal(v)
    if err != nil {
        return nil, err
    }
    return encrypt(data, s.key)
}
```

### 8.3 访问控制

```go
// Redis 密码认证
&RedisConfig{
    Password: os.Getenv("REDIS_PASSWORD"),
}

// Redis ACL（Redis 6.0+）
&RedisConfig{
    Username: "myapp",
    Password: "secret",
}
```

## 9. 监控与调试

### 9.1 链路追踪集成

与 `pkg/tracing` 集成：

```go
import "go.opentelemetry.io/otel"

func (c *redisCache) Get(ctx context.Context, key string, value any) error {
tracer := otel.Tracer("qi.cache")
    ctx, span := tracer.Start(ctx, "cache.Get",
        trace.WithAttributes(
            attribute.Se.key", key),
            attribute.String("cache.driver", "redis"),
        ),
    )
    defer span.End()

    // 执行缓存操作
    err := c.client.Get(ctx, c.buildKey(key)).Bytes()
    if err != nil {
        span.RecordError(err)
        return err
    }

    span.SetAttributes(attribute.Bool("cache.hit", true))
    return nil
}
```

### 9.2 日志集成

与 `pkg/logger` 集成：

```go
import "qi/pkg/logger"

func (c *redisCache) Get(ctx context.Context, key string, value any) error {
    start := time.Now()
    err := c.client.Get(ctx, c.buildKey(key)).Bytes()
    duration := time.Since(start)

    logger.InfoContext(ctx, "cache operation",
        zap.String("operation", "get"),
        zap.String("key", key),
        zap.Duration("duration", duration),
        zap.Bool("hit", err == nil),
    )

    return err
}
```

### 9.3 性能指标

```go
// CacheStats 缓存统计
type CacheStats struct {
    Hits        int64 // 命中次数
    Misses      int64 // 未命中次数
    Sets        int64 // 写入次数
    Deletes     int64 // 删除次数
    HitRate     float64 // 命中率
    AvgLatency  time.Duration // 平均延迟
}

func (c *Cache) Stats() *CacheStats
```

## 10. 使用示例

### 10.1 基础使用

```go
package main

import (
    "context"
    "time"
    "qi/pkg/cache"
)

func main() {
    // 创建 Redis 缓存
    c, err := cache.NewWithOptions(
        cache.WithRedis(&cache.RedisConfig{
            Addr: "localhost:6379",
        }),
        cache.WithKeyPrefix("myapp:"),
        cache.WithDefaultTTL(10 * time.Minute),
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

### 10.2 与 Qi 框架集成

```go
package main

import (
    "qi"
    "qi/pkg/cache"
)

func main() {
    // 创建缓存实例
    c, _ := cache.NewWithOptions(
        cache.WithRedis(&cache.RedisConfig{
     Addr: "localhost:6379",
        }),
    )

    // 创建 Qi Engine
    engine := qi.Default()
    r := engine.RouterGroup()

    // 在路由中使用缓存
    r.GET("/user/:id", func(ctx *qi.Context) {
        id := ctx.Param("id")
        key := "user:" + id

        // 从缓存获取
        var user User
        err := c.Get(ctx.RequestContext(), key, &user)
        if err == nil {
            ctx.Success(user)
            return
        }

        // 缓存未命中，从数据库查询
        user, err = db.GetUser(ctx.RequestContext(), id)
        if err != nil {
            ctx.RespondError(err)
            return
        }

        // 写入缓存
        _ = c.Set(ctx.RequestContext(), key, user, 10*time.Minute)
        ctx.Success(user)
    })

    engine.Run(":8080")
}
```

### 10.3 Remember 模式

```go
r.GET("/user/:id", func(ctx *qi.Context) {
    id := ctx.Param("id")
    key := "user:" + id

    // Remember 模式（自动处理缓存逻辑）
    user, err := cache.Remember(ctx.RequestContext(), c, key, 10*time.Minute, func() (User, error) {
        return db.GetUser(ctx.RequestContext(), id)
    })

    if err != nil {
        ctx.RespondError(err)
        return
    }

    ctx.Success(user)
})
```

## 11. 实施路线图

### Phase 1: 核心功能（MVP）
- [ ] 定义 `Cache` 接口
- [ ] 实现 Redis 驱动（单机模式）
- [ ] 实现 Memory 驱动
- [ ] JSON 序列化器
- [ ] 基础错误处理
- [ ] 配置系统（Config + Options）
- [ ] 单元测试

### Phase 2: 高级功能
- [ ] 泛型 API（`GetTyped`, `SetTyped`）
- [ ] Remember 模式
- [ ] 批量操作（`MGet`, `MSet`）
- [ ] 原子操作（`Incr`, `Decr`）
- [ ] Msgpack 序列化器
- [ ] Redis 集群/哨兵支持

### Phase 3: 企业级功能
- [ ] 标签管理（Tag-based Invalidation）
- [ ] 链路追踪集成（`pkg/tracing`）
- [ ] 日志集成（`pkg/logger`）
- [ ] 性能指标（Stats）
- [ ] 分布式锁（singleflight）
- [ ] 缓存预热

### Phase 4: 文档与示例
- [ ] API 文档
- [ ] 使用示例
- [ ] 最佳实践指南
- [ ] 性能测试报告

## 12. 依赖清单

```go
// go.mod
require (
    github.com/redis/go-redis/v9 v9.5.1        // Redis 客户端
    github.com/patrickmn/go-cache v2.1.0+incompatible // 内存缓存
    github.com/vmihailenco/msgpack/v5 v5.4.1   // Msgpack 序列化（可选）
    golang.org/x/sync v0.6.0                   // singleflight（可选）
)
```

## 13. 总结

本设计方案完全符合 Qi 框架的设计理念：

✅ **简洁易用**：API 设计直观，`Get/Set/Delete` 三个核心方法
✅ **开箱即用**：提供 `DefaultConfig()`，快速上手
✅ **零侵入**：可选集成，不强制依赖
✅ **类型安全**：泛型 API 提供编译时类型检查
✅ **统一错误处理**：与 `pkg/errors` 集成
✅ **Context 传递**：支持超时控制、链路追踪
✅ **Options 模式**：与 `pkg/orm`、`pkg/logger` 保持一致

**推荐实施顺序**：Phase 1 → Phase 2 → Phase 3 → Phase 4

**预计工作量**：
- Phase 1（MVP）：3-5 天
- Phase 2（高级功能）：2-3 天
- Phase 3（企业级）：3-5 天
- Phase 4（文档）：1-2 天

**总计**：9-15 天（根据团队规模和经验调整）
