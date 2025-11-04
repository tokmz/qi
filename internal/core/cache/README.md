# Cache - 高性能分布式缓存

基于 Redis v9 和 Singleflight 的高性能分布式缓存管理包，提供完整的缓存策略、防击穿机制、预热功能和并发安全的缓存操作。

## 功能特性

### 核心特性

- ✅ **Redis v9**: 基于最新的 Redis v9 客户端
- ✅ **防缓存击穿**: 使用 singleflight 防止缓存击穿
- ✅ **防缓存穿透**: 支持空值缓存和布隆过滤器
- ✅ **防缓存雪崩**: 随机过期时间、缓存预热
- ✅ **Cache-Aside 模式**: 应用层控制缓存维护
- ✅ **多种序列化**: 支持 JSON、MessagePack、Gob
- ✅ **缓存预热**: 系统启动时预加载热点数据
- ✅ **主动失效**: 数据变更时主动删除缓存
- ✅ **批量操作**: Pipeline 批量读写优化
- ✅ **统计监控**: 完整的缓存命中率统计
- ✅ **并发安全**: 所有操作线程安全
- ✅ **类型安全**: 泛型支持和类型断言
- ✅ **高质量代码**: 清晰的结构，完整的中文注释

### 技术栈

- [Redis v9](https://github.com/redis/go-redis) - Redis 客户端
- [Singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) - 防缓存击穿
- [MessagePack](https://github.com/vmihailenco/msgpack) - 高效序列化（可选）

## 目录结构

```
cache/
├── config.go          # 配置定义
├── errors.go          # 错误定义
├── types.go           # 类型定义
├── cache.go           # 缓存管理器核心实现
├── serializer.go      # 序列化器
├── logger.go          # 日志接口和实现
├── helper.go          # 辅助函数
├── example_test.go    # 示例代码
├── cache_test.go      # 单元测试
└── README.md          # 文档（本文件）
```

## 快速开始

### 1. 基础使用

```go
package main

import (
    "context"
    "log"
    "time"
    "qi/internal/core/cache"
    
    "github.com/redis/go-redis/v9"
)

func main() {
    // 创建 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
    })

    // 创建配置
    cfg := cache.DefaultConfig()
    cfg.Redis = rdb
    cfg.DefaultExpiration = 5 * time.Minute

    // 创建缓存管理器
    manager, err := cache.New(cfg, &cache.DefaultLogger{})
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    ctx := context.Background()

    // 设置缓存
    err = manager.Set(ctx, "user:123", map[string]interface{}{
        "id":   123,
        "name": "Alice",
        "age":  25,
    }, 10*time.Minute)
    if err != nil {
        log.Fatal(err)
    }

    // 获取缓存
    var user map[string]interface{}
    err = manager.Get(ctx, "user:123", &user)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("User: %+v", user)

    // 删除缓存
    err = manager.Delete(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. 防缓存击穿（Singleflight）

```go
// GetUser 获取用户信息（自动防击穿）
func GetUser(manager *cache.Manager, userID int64) (*User, error) {
    ctx := context.Background()
    key := fmt.Sprintf("user:%d", userID)
    
    // 使用 GetOrLoad 自动处理缓存击穿
    var user User
    err := manager.GetOrLoad(ctx, key, &user, func() (interface{}, error) {
        // 这个函数在缓存未命中时被调用
        // singleflight 确保同一时间只有一个请求查询数据库
        return loadUserFromDB(userID)
    }, 10*time.Minute)
    
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}

func loadUserFromDB(userID int64) (*User, error) {
    // 从数据库加载用户
    user := &User{
        ID:   userID,
        Name: "Alice",
        Age:  25,
    }
    return user, nil
}
```

### 3. 缓存预热

```go
// 系统启动时预热缓存
func warmupCache(manager *cache.Manager) error {
    ctx := context.Background()
    
    // 预热热点数据
    hotUsers := []int64{1, 2, 3, 4, 5}
    
    for _, userID := range hotUsers {
        user, err := loadUserFromDB(userID)
        if err != nil {
            continue
        }
        
        key := fmt.Sprintf("user:%d", userID)
        manager.Set(ctx, key, user, 1*time.Hour)
    }
    
    return nil
}

// 使用 Warmup 方法批量预热
func warmupCacheBatch(manager *cache.Manager) error {
    items := []cache.WarmupItem{
        {
            Key:   "user:1",
            Value: &User{ID: 1, Name: "Alice"},
            TTL:   1 * time.Hour,
        },
        {
            Key:   "user:2",
            Value: &User{ID: 2, Name: "Bob"},
            TTL:   1 * time.Hour,
        },
    }
    
    return manager.Warmup(context.Background(), items)
}
```

### 4. 批量操作

```go
// 批量获取
func batchGet(manager *cache.Manager, userIDs []int64) (map[int64]*User, error) {
    ctx := context.Background()
    
    // 构建键列表
    keys := make([]string, len(userIDs))
    for i, id := range userIDs {
        keys[i] = fmt.Sprintf("user:%d", id)
    }
    
    // 批量获取
    results, err := manager.GetMulti(ctx, keys)
    if err != nil {
        return nil, err
    }
    
    // 解析结果
    users := make(map[int64]*User)
    for key, val := range results {
        var user User
        if err := cache.Unmarshal(val, &user); err != nil {
            continue
        }
        users[user.ID] = &user
    }
    
    return users, nil
}

// 批量设置
func batchSet(manager *cache.Manager, users []*User) error {
    ctx := context.Background()
    
    items := make(map[string]interface{})
    for _, user := range users {
        key := fmt.Sprintf("user:%d", user.ID)
        items[key] = user
    }
    
    return manager.SetMulti(ctx, items, 10*time.Minute)
}
```

## 缓存策略

### 1. Cache-Aside（旁路缓存）

这是最常用的缓存策略，由应用层控制缓存的读写。

```go
// 读取数据
func getData(manager *cache.Manager, key string) (interface{}, error) {
    ctx := context.Background()
    
    // 1. 先查缓存
    var data interface{}
    err := manager.Get(ctx, key, &data)
    if err == nil {
        return data, nil // 缓存命中
    }
    
    if err != cache.ErrCacheMiss {
        // 缓存错误，记录日志但继续查数据库
        log.Printf("Cache error: %v", err)
    }
    
    // 2. 缓存未命中，查询数据库
    data, err = loadFromDB(key)
    if err != nil {
        return nil, err
    }
    
    // 3. 更新缓存（异步）
    go func() {
        manager.Set(context.Background(), key, data, 10*time.Minute)
    }()
    
    return data, nil
}

// 更新数据
func updateData(manager *cache.Manager, key string, data interface{}) error {
    ctx := context.Background()
    
    // 1. 更新数据库
    if err := updateDB(key, data); err != nil {
        return err
    }
    
    // 2. 删除缓存（而不是更新）
    // 下次读取时会重新加载最新数据
    if err := manager.Delete(ctx, key); err != nil {
        log.Printf("Failed to delete cache: %v", err)
    }
    
    return nil
}
```

### 2. 防缓存穿透

```go
// 方法1: 缓存空值
func getDataWithNullCache(manager *cache.Manager, key string) (interface{}, error) {
    ctx := context.Background()
    
    // 使用 GetOrLoad，即使数据库返回 nil 也会缓存
    var data interface{}
    err := manager.GetOrLoad(ctx, key, &data, func() (interface{}, error) {
        data, err := loadFromDB(key)
        if err != nil {
            return nil, err
        }
        if data == nil {
            // 数据库也没有，缓存一个空对象，设置较短的过期时间
            return &emptyObject{}, nil
        }
        return data, nil
    }, 5*time.Minute)
    
    return data, err
}

// 方法2: 使用布隆过滤器（需要额外实现）
func getDataWithBloomFilter(manager *cache.Manager, bf *BloomFilter, key string) (interface{}, error) {
    // 先检查布隆过滤器
    if !bf.Exists(key) {
        return nil, cache.ErrNotFound
    }
    
    // 正常的缓存逻辑
    return getData(manager, key)
}
```

### 3. 防缓存雪崩

```go
// 设置随机过期时间
func setWithRandomExpiration(manager *cache.Manager, key string, value interface{}, baseTTL time.Duration) error {
    ctx := context.Background()
    
    // 在基础 TTL 上增加随机时间（±20%）
    randomTTL := manager.RandomExpiration(baseTTL, 0.2)
    
    return manager.Set(ctx, key, value, randomTTL)
}

// 批量设置时使用不同的过期时间
func batchSetWithRandomTTL(manager *cache.Manager, items map[string]interface{}) error {
    ctx := context.Background()
    
    for key, value := range items {
        ttl := manager.RandomExpiration(10*time.Minute, 0.2)
        if err := manager.Set(ctx, key, value, ttl); err != nil {
            log.Printf("Failed to set %s: %v", key, err)
        }
    }
    
    return nil
}
```

## 配置说明

### 完整配置示例

```yaml
cache:
  # Redis 配置
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
    pool_size: 10
    min_idle_conns: 5
    max_retries: 3
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
  
  # 缓存配置
  default_expiration: 5m      # 默认过期时间
  cleanup_interval: 10m       # 清理间隔
  
  # 序列化器: json/msgpack/gob
  serializer: json
  
  # 键前缀
  key_prefix: "qi:"
  
  # 空值缓存
  null_cache:
    enabled: true
    expiration: 1m            # 空值缓存时间
  
  # 统计
  stats:
    enabled: true
    report_interval: 1m
```

### Config 结构体

```go
type Config struct {
    // Redis 客户端
    Redis *redis.Client
    
    // 默认过期时间
    DefaultExpiration time.Duration
    
    // 清理间隔
    CleanupInterval time.Duration
    
    // 序列化器类型
    Serializer SerializerType
    
    // 键前缀
    KeyPrefix string
    
    // 空值缓存配置
    NullCache NullCacheConfig
    
    // 统计配置
    Stats StatsConfig
}

type NullCacheConfig struct {
    // 是否启用空值缓存
    Enabled bool
    
    // 空值过期时间
    Expiration time.Duration
}

type StatsConfig struct {
    // 是否启用统计
    Enabled bool
    
    // 统计上报间隔
    ReportInterval time.Duration
}
```

## API 文档

### 缓存管理器

```go
// 创建缓存管理器
New(cfg *Config, logger Logger) (*Manager, error)

// 初始化全局缓存管理器
InitGlobal(cfg *Config, logger Logger) error

// 获取全局缓存管理器
GetGlobal() *Manager

// 关闭缓存管理器
Close() error
```

### 基础操作

```go
// 设置缓存
Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

// 获取缓存
Get(ctx context.Context, key string, dest interface{}) error

// 删除缓存
Delete(ctx context.Context, key string) error

// 检查键是否存在
Exists(ctx context.Context, key string) (bool, error)

// 设置过期时间
Expire(ctx context.Context, key string, ttl time.Duration) error

// 获取剩余过期时间
TTL(ctx context.Context, key string) (time.Duration, error)
```

### 高级操作

```go
// 获取或加载（防击穿）
GetOrLoad(ctx context.Context, key string, dest interface{}, 
    loader LoaderFunc, ttl time.Duration) error

// 批量获取
GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)

// 批量设置
SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

// 批量删除
DeleteMulti(ctx context.Context, keys []string) error

// 模式匹配删除
DeletePattern(ctx context.Context, pattern string) error

// 缓存预热
Warmup(ctx context.Context, items []WarmupItem) error

// 自增
Incr(ctx context.Context, key string) (int64, error)

// 自减
Decr(ctx context.Context, key string) (int64, error)

// 增加指定值
IncrBy(ctx context.Context, key string, value int64) (int64, error)

// 获取原始字节数据
GetRaw(ctx context.Context, key string) ([]byte, error)

// 检查 Redis 连接
Ping(ctx context.Context) error

// 生成随机过期时间（防缓存雪崩）
RandomExpiration(baseTTL time.Duration, jitter float64) time.Duration

// 获取配置（只读）
GetConfig() *Config

// 清空所有缓存（危险操作）
FlushAll(ctx context.Context) error

// 健康检查
Health(ctx context.Context) error
```

### 统计监控

```go
// 获取统计信息
GetStats() *Stats

// 重置统计
ResetStats()

// 获取命中率
GetHitRate() float64
```

### Serializer 接口

```go
type Serializer interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}

// 内置序列化器
// - JSONSerializer: JSON 序列化器（默认）
// - GobSerializer: Gob 序列化器
// - MsgPackSerializer: MessagePack 序列化器（需要引入依赖）
```

### Logger 接口

```go
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}

// 提供的实现
// - NoopLogger: 无操作日志（默认）
// - DefaultLogger: 标准输出日志
// - DebugLogger: 调试日志
```

### 错误定义

```go
var (
    // 配置错误
    ErrInvalidConfig       = errors.New("invalid cache config")
    ErrRedisClientRequired = errors.New("redis client is required")
    
    // 缓存操作错误
    ErrCacheMiss             = errors.New("cache miss")
    ErrKeyNotFound           = errors.New("key not found")
    ErrSerializationFailed   = errors.New("serialization failed")
    ErrDeserializationFailed = errors.New("deserialization failed")
    ErrInvalidSerializer     = errors.New("invalid serializer type")
    ErrNilValue              = errors.New("nil value")
    ErrInvalidTTL            = errors.New("invalid TTL")
    
    // 管理器状态错误
    ErrManagerNotInitialized = errors.New("cache manager not initialized")
    ErrManagerAlreadyClosed  = errors.New("cache manager already closed")
    
    // 其他错误
    ErrLoaderFuncRequired = errors.New("loader function is required")
    ErrNotFound           = errors.New("data not found")
)
```

### 辅助类型

```go
// 序列化器类型
type SerializerType string

const (
    SerializerJSON    SerializerType = "json"     // JSON 序列化器
    SerializerMsgPack SerializerType = "msgpack"  // MessagePack 序列化器
    SerializerGob     SerializerType = "gob"      // Gob 序列化器
)

// 数据加载函数
type LoaderFunc func() (interface{}, error)

// 预热项
type WarmupItem struct {
    // 缓存键
    Key string
    
    // 缓存值
    Value interface{}
    
    // 过期时间
    TTL time.Duration
}
```

### Stats 结构

```go
type Stats struct {
    // 总请求数
    Requests int64 `json:"requests"`
    
    // 命中数
    Hits int64 `json:"hits"`
    
    // 未命中数
    Misses int64 `json:"misses"`
    
    // 命中率
    HitRate float64 `json:"hit_rate"`
    
    // 设置次数
    Sets int64 `json:"sets"`
    
    // 删除次数
    Deletes int64 `json:"deletes"`
    
    // 错误次数
    Errors int64 `json:"errors"`
    
    // 加载函数调用次数
    LoaderCalls int64 `json:"loader_calls"`
    
    // Singleflight 命中次数
    SingleflightHits int64 `json:"singleflight_hits"`
}
```

## 使用场景

### 1. 用户信息缓存

```go
type UserCache struct {
    cache *cache.Manager
}

func NewUserCache(manager *cache.Manager) *UserCache {
    return &UserCache{cache: manager}
}

func (uc *UserCache) GetUser(ctx context.Context, userID int64) (*User, error) {
    key := fmt.Sprintf("user:%d", userID)
    
    var user User
    err := uc.cache.GetOrLoad(ctx, key, &user, func() (interface{}, error) {
        // 从数据库加载
        return db.GetUser(userID)
    }, 10*time.Minute)
    
    return &user, err
}

func (uc *UserCache) UpdateUser(ctx context.Context, user *User) error {
    // 更新数据库
    if err := db.UpdateUser(user); err != nil {
        return err
    }
    
    // 删除缓存
    key := fmt.Sprintf("user:%d", user.ID)
    return uc.cache.Delete(ctx, key)
}

func (uc *UserCache) DeleteUser(ctx context.Context, userID int64) error {
    // 删除数据库
    if err := db.DeleteUser(userID); err != nil {
        return err
    }
    
    // 删除缓存
    key := fmt.Sprintf("user:%d", userID)
    return uc.cache.Delete(ctx, key)
}
```

### 2. 列表查询缓存

```go
func GetProjectList(ctx context.Context, manager *cache.Manager, page, pageSize int) ([]*Project, error) {
    key := fmt.Sprintf("projects:list:%d:%d", page, pageSize)
    
    var projects []*Project
    err := manager.GetOrLoad(ctx, key, &projects, func() (interface{}, error) {
        return db.GetProjects(page, pageSize)
    }, 5*time.Minute)
    
    return projects, err
}

// 项目变更时失效缓存
func InvalidateProjectCache(ctx context.Context, manager *cache.Manager) error {
    // 删除所有项目列表缓存
    return manager.DeletePattern(ctx, "projects:list:*")
}
```

### 3. 计数器缓存

```go
// 文章浏览量
func IncrementViewCount(ctx context.Context, manager *cache.Manager, articleID int64) error {
    key := fmt.Sprintf("article:%d:views", articleID)
    
    // Redis 自增
    count, err := manager.Incr(ctx, key)
    if err != nil {
        return err
    }
    
    // 每100次写回数据库
    if count%100 == 0 {
        go func() {
            db.UpdateViewCount(articleID, count)
        }()
    }
    
    return nil
}

// 获取浏览量
func GetViewCount(ctx context.Context, manager *cache.Manager, articleID int64) (int64, error) {
    key := fmt.Sprintf("article:%d:views", articleID)
    
    var count int64
    err := manager.Get(ctx, key, &count)
    if err == cache.ErrCacheMiss {
        // 从数据库加载
        count, err = db.GetViewCount(articleID)
        if err != nil {
            return 0, err
        }
        // 设置缓存
        manager.Set(ctx, key, count, 1*time.Hour)
    }
    
    return count, nil
}
```

### 4. Session 缓存

```go
type SessionCache struct {
    cache *cache.Manager
}

func (sc *SessionCache) Set(ctx context.Context, sessionID string, data map[string]interface{}) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return sc.cache.Set(ctx, key, data, 30*time.Minute)
}

func (sc *SessionCache) Get(ctx context.Context, sessionID string) (map[string]interface{}, error) {
    key := fmt.Sprintf("session:%s", sessionID)
    
    var data map[string]interface{}
    err := sc.cache.Get(ctx, key, &data)
    if err != nil {
        return nil, err
    }
    
    // 刷新过期时间
    sc.cache.Expire(ctx, key, 30*time.Minute)
    
    return data, nil
}

func (sc *SessionCache) Delete(ctx context.Context, sessionID string) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return sc.cache.Delete(ctx, key)
}
```

## 最佳实践

### 1. 键命名规范

```go
// 推荐格式: {业务模块}:{对象类型}:{ID}:{属性}
"user:profile:123"           // 用户资料
"user:settings:123"          // 用户设置
"project:info:456"           // 项目信息
"project:members:456"        // 项目成员列表
"article:content:789"        // 文章内容
"article:views:789"          // 文章浏览数

// 列表缓存
"users:list:page:1:size:20"  // 用户列表
"projects:list:status:active" // 项目列表
```

### 2. 合理设置过期时间

```go
// 热点数据: 较长时间
manager.Set(ctx, "hot:user:123", user, 1*time.Hour)

// 普通数据: 中等时间
manager.Set(ctx, "user:123", user, 10*time.Minute)

// 临时数据: 较短时间
manager.Set(ctx, "verify:code:123", code, 5*time.Minute)

// 计数器: 根据写回策略
manager.Set(ctx, "counter:views:123", count, 24*time.Hour)
```

### 3. 错误处理

```go
func getData(manager *cache.Manager, key string) (interface{}, error) {
    ctx := context.Background()
    
    var data interface{}
    err := manager.Get(ctx, key, &data)
    
    switch err {
    case nil:
        // 缓存命中
        return data, nil
        
    case cache.ErrCacheMiss:
        // 缓存未命中，查询数据库
        return loadFromDB(key)
        
    default:
        // 缓存错误，降级到数据库
        log.Printf("Cache error: %v", err)
        return loadFromDB(key)
    }
}
```

### 4. 并发安全

```go
// 使用 GetOrLoad 自动处理并发
func getConcurrently(manager *cache.Manager, key string) (interface{}, error) {
    var data interface{}
    err := manager.GetOrLoad(context.Background(), key, &data, func() (interface{}, error) {
        // singleflight 确保只有一个请求执行这个函数
        time.Sleep(100 * time.Millisecond) // 模拟慢查询
        return loadFromDB(key)
    }, 10*time.Minute)
    
    return data, err
}

// 多个 goroutine 同时调用，只会有一个查询数据库
func testConcurrent(manager *cache.Manager) {
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            getConcurrently(manager, "same:key")
        }()
    }
    wg.Wait()
}
```

### 5. 监控和告警

```go
// 定期检查缓存状态
func monitorCache(manager *cache.Manager) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := manager.GetStats()
        
        // 检查命中率
        if stats.HitRate < 0.8 {
            log.Printf("Warning: Low cache hit rate: %.2f%%", stats.HitRate*100)
        }
        
        // 检查错误率
        errorRate := float64(stats.Errors) / float64(stats.Requests)
        if errorRate > 0.01 {
            log.Printf("Warning: High error rate: %.2f%%", errorRate*100)
        }
        
        // 上报监控指标
        reportMetrics(stats)
    }
}
```

## 性能优化

### 1. 使用 Pipeline 批量操作

```go
// 批量设置（使用 Pipeline）
func batchSetOptimized(manager *cache.Manager, items map[string]interface{}) error {
    return manager.SetMulti(context.Background(), items, 10*time.Minute)
}

// 批量获取（使用 Pipeline）
func batchGetOptimized(manager *cache.Manager, keys []string) (map[string]interface{}, error) {
    results, err := manager.GetMulti(context.Background(), keys)
    if err != nil {
        return nil, err
    }
    
    data := make(map[string]interface{})
    for key, val := range results {
        var obj interface{}
        if err := cache.Unmarshal(val, &obj); err == nil {
            data[key] = obj
        }
    }
    
    return data, nil
}
```

### 2. 序列化优化

```go
// 使用 MessagePack 替代 JSON（更快、更小）
cfg := cache.DefaultConfig()
cfg.Serializer = cache.SerializerMsgPack

manager, _ := cache.New(cfg, nil)
```

### 3. 连接池优化

```go
rdb := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     10,              // 连接池大小
    MinIdleConns: 5,               // 最小空闲连接
    MaxRetries:   3,               // 最大重试次数
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

## 故障排查

### 1. 缓存命中率低

```go
// 检查统计信息
stats := manager.GetStats()
log.Printf("Hit rate: %.2f%%", stats.HitRate*100)
log.Printf("Hits: %d, Misses: %d", stats.Hits, stats.Misses)

// 可能的原因：
// 1. TTL 设置太短
// 2. 缓存key设计不合理
// 3. 缓存空间不足
// 4. 数据变更频繁
```

### 2. Redis 连接问题

```go
// 健康检查
func checkRedis(manager *cache.Manager) error {
    ctx := context.Background()
    return manager.Ping(ctx)
}

// 检查连接池状态
stats := rdb.PoolStats()
log.Printf("Hits: %d, Misses: %d, Timeouts: %d", 
    stats.Hits, stats.Misses, stats.Timeouts)
```

### 3. 序列化错误

```go
// 确保类型一致
var user User
err := manager.Get(ctx, "user:123", &user)
if err != nil {
    log.Printf("Failed to unmarshal: %v", err)
    // 检查缓存中存储的原始数据
    raw, _ := manager.GetRaw(ctx, "user:123")
    log.Printf("Raw data: %s", raw)
}
```

## 依赖

在 `go.mod` 中添加：

```go
require (
    github.com/redis/go-redis/v9 v9.7.0
    golang.org/x/sync v0.16.0
    github.com/vmihailenco/msgpack/v5 v5.4.1  // 可选
)
```

## 参考资源

- [Redis 官方文档](https://redis.io/docs/)
- [go-redis 文档](https://redis.uptrace.dev/)
- [Singleflight 文档](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [缓存设计模式](https://docs.microsoft.com/zh-cn/azure/architecture/patterns/cache-aside)

## 许可证

MIT License

