# pkg/cache

基于 Redis 和内存的多级缓存封装，提供统一接口、LRU 淘汰、防缓存三大问题（穿透/击穿/雪崩）和 OpenTelemetry 链路追踪。

## 特性

- **多驱动**：内存（LRU）、Redis（单机/哨兵/集群）、多级（L1 内存 + L2 Redis）
- **单一入口**：`New(*Config)` 统一创建，驱动通过 `Driver` 字段切换
- **防缓存穿透**：Bloom filter（线程安全）+ 独立空值标记 key
- **防缓存击穿**：`GetOrSet` 内置 `singleflight` 合并并发请求
- **防缓存雪崩**：Redis TTL ±10% 随机抖动
- **分布式锁**：Redis SET NX + Lua 原子解锁，指数退避重试
- **链路追踪**：OpenTelemetry 装饰器，自动为每个操作创建 span
- **序列化**：内置 JSON / GOB，可自定义

## 快速开始

### 内存缓存（LRU）

```go
import "github.com/tokmz/qi/pkg/cache"

c, err := cache.New(&cache.Config{
    Driver: cache.DriverMemory,
    Memory: &cache.MemoryConfig{
        MaxSize:         10_000,        // LRU 最大条目数
        CleanupInterval: time.Minute,   // 后台清理间隔
    },
    DefaultTTL: 10 * time.Minute,
})

// 写入
c.Set(ctx, "user:1", user, time.Hour)

// 读取
var u User
c.Get(ctx, "user:1", &u)

// 删除
c.Del(ctx, "user:1")
```

### Redis 缓存

```go
// 单机
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverRedis,
    KeyPrefix: "app:",
    Redis: &cache.RedisConfig{
        Addr:     "127.0.0.1:6379",
        Password: "secret",
        DB:       0,
    },
})

// 哨兵模式
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverRedis,
    KeyPrefix: "app:",
    Redis: &cache.RedisConfig{
        Addrs:  []string{"sentinel1:26379", "sentinel2:26379"},
        Master: "mymaster",
    },
})

// 集群模式
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverRedis,
    KeyPrefix: "app:",
    Redis: &cache.RedisConfig{
        Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
    },
})
```

### 多级缓存（L1 内存 + L2 Redis）

```go
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverMultiLevel,
    KeyPrefix: "app:",
    Memory: &cache.MemoryConfig{
        MaxSize:         1_000,
        CleanupInterval: time.Minute,
    },
    Redis: &cache.RedisConfig{
        Addr: "127.0.0.1:6379",
    },
})
// Get：L1 命中直接返回；L1 miss → 查 L2 → 自动回填 L1（TTL 取 L2 实际剩余的 20%）
// Set：同时写 L1 + L2
// Del：同时删 L1 + L2
```

### 防缓存穿透

```go
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverRedis,
    KeyPrefix: "app:",
    Redis:     &cache.RedisConfig{Addr: "127.0.0.1:6379"},
    Penetration: &cache.PenetrationConfig{
        EnableBloom: true,    // 启用 Bloom filter
        BloomN:      100_000, // 预期最大 key 数量
        BloomFP:     0.01,    // 误判率 1%
        NullTTL:     60 * time.Second, // 空值标记缓存时长
    },
})

// GetOrSet：缓存未命中时调用 fn 加载并回写
err = c.GetOrSet(ctx, "user:999", &u, time.Hour, func() (any, error) {
    user, err := db.FindUser(999)
    if err == sql.ErrNoRows {
        return nil, cache.ErrNotFound // 触发空值标记（NullTTL 内不再查库）
    }
    return user, err
})
// bloom filter 不影响 GetOrSet 的回调执行，仅在 Get/Exists/MGet 中拦截确定不存在的 key
```

### 链路追踪

```go
// 需外部提前初始化 OTel TracerProvider
c, err := cache.New(&cache.Config{
    Driver:         cache.DriverRedis,
    KeyPrefix:      "app:",
    Redis:          &cache.RedisConfig{Addr: "127.0.0.1:6379"},
    TracingEnabled: true, // 自动为每个操作创建 OTel span
})
// span 命名：cache.Get / cache.Set / cache.Del 等
// 自动记录 attributes：cache.key、cache.hit、cache.ttl、cache.key_count
```

### 完整配置示例

```go
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverMultiLevel,
    KeyPrefix: "app:",
    DefaultTTL: time.Hour,
    Serializer: cache.JSONSerializer{}, // 默认，可改为 cache.GOBSerializer{}

    Memory: &cache.MemoryConfig{
        MaxSize:         5_000,
        CleanupInterval: time.Minute,
    },
    Redis: &cache.RedisConfig{
        Addr:          "127.0.0.1:6379",
        Password:      "secret",
        PoolSize:      100,
        MinIdleConns:  10,
        DialTimeout:   5 * time.Second,
        ReadTimeout:   3 * time.Second,
        WriteTimeout:  3 * time.Second,
        DisableJitter: false, // 开启 TTL ±10% 抖动防雪崩
    },
    Penetration: &cache.PenetrationConfig{
        EnableBloom: true,
        BloomN:      100_000,
        BloomFP:     0.01,
        NullTTL:     60 * time.Second,
    },
    TracingEnabled: true,
})
```

## 接口说明

```go
type Cache interface {
    Get(ctx, key string, dest any) error
    Set(ctx, key string, value any, ttl time.Duration) error
    Del(ctx, keys ...string) error
    Exists(ctx, key string) (bool, error)
    Expire(ctx, key string, ttl time.Duration) error
    TTL(ctx, key string) (time.Duration, error)

    MGet(ctx, keys []string) (map[string][]byte, error)
    MSet(ctx, kvs map[string]any, ttl time.Duration) error

    // 内置 singleflight，防击穿
    GetOrSet(ctx, key string, dest any, ttl time.Duration, fn func() (any, error)) error

    Incr(ctx, key string) (int64, error)
    IncrBy(ctx, key string, delta int64) (int64, error)
    DecrBy(ctx, key string, delta int64) (int64, error)

    Flush(ctx) error  // Redis 需配置 KeyPrefix，否则拒绝执行
    Close() error
}
```

## 分布式锁

```go
locker, err := cache.NewLocker(&cache.RedisConfig{
    Addr: "127.0.0.1:6379",
}, "app:")

// 阻塞获取锁（指数退避重试，ctx 取消时退出）
unlock, err := locker.Lock(ctx, "order:create:123", 10*time.Second)
if err != nil {
    return err
}
defer unlock()

// 非阻塞尝试获取锁
ok, unlock, err := locker.TryLock(ctx, "order:create:123", 10*time.Second)
if !ok {
    return errors.New("获取锁失败")
}
defer unlock()
```

## 错误处理

```go
var u User
err := c.Get(ctx, "user:1", &u)
if errors.Is(err, cache.ErrNotFound) {
    // key 不存在
}
```

| 错误 | 含义 |
|------|------|
| `ErrNotFound` | key 不存在（含空值标记拦截） |
| `ErrNilValue` | 写入值为 nil |
| `ErrNotSupported` | 当前驱动不支持该操作 |
| `ErrLockNotHeld` | 解锁时 token 不匹配 |

## 三大缓存问题

| 问题 | 解决方案 | 触发方式 |
|------|---------|----------|
| 缓存穿透（key 不存在） | Bloom filter（读操作拦截）+ 空值标记 key（阻止重复查库） | 配置 `Penetration` |
| 缓存击穿（热 key 过期） | `singleflight` 合并并发 fn | `GetOrSet` 自动生效 |
| 缓存雪崩（批量过期） | TTL ±10% 随机抖动 | Redis 驱动自动生效 |

> Bloom filter 仅作用于读操作（`Get`/`Exists`/`MGet`），不会阻止 `GetOrSet` 的回调执行。这是因为 `GetOrSet` 的语义是"未命中则加载"，bloom 拦截会导致新 key 永远无法写入缓存。

## 注意事项

- `Flush` 在 Redis 驱动下要求配置 `KeyPrefix`，否则返回错误（防止误清整个 DB）
- Bloom filter 仅驻内存，服务重启后需重建；重启期间防穿透退化为仅空值标记
- 多级缓存中 L1 TTL = L2 实际剩余 TTL × 20%（最低 1 秒），L1 过期后自动从 L2 回填
- 链路追踪需外部提前初始化 `otel.SetTracerProvider`
- `GetOrSet` 的 `fn` 若返回 `ErrNotFound`，会自动写入空值标记；其他错误不写入
- Bloom filter 仅拦截 `Get`/`Exists`/`MGet`，`GetOrSet` 不受 bloom 影响（否则新 key 回调永远不执行）
