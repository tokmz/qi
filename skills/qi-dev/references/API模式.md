# qi 框架 API 模式速查

## Context 高频方法

```go
// 路径参数
c.Param("id")                        // /users/:id → "123"

// 查询参数
c.Query("page")                       // ?page=1 → "1"
c.DefaultQuery("size", "20")          // 缺省返回 "20"
c.QueryArray("ids")                   // ?ids=1&ids=2 → ["1","2"]

// Header
c.GetHeader("Authorization")         // → "Bearer xxx"
c.SetHeader("X-Request-Id", "abc")

// 上下文传值
c.Set("uid", "user-123")
uid, ok := c.Get("uid")              // ok=false 时 uid=nil
uid := c.MustGet("uid")              // 不存在时 panic

// 响应
c.OK(data)                           // code=0, status=200
c.OK(data, "操作成功")               // 自定义 message
c.Page(total, list)                  // 分页：{total, list}
c.Fail(err)                          // 提取 errors.Error 的 code/status

// 逃生舱（访问 gin 原生功能）
c.Gin().ShouldBindHeader(&h)
```

## pkg/cache 使用

```go
import "github.com/tokmz/qi/pkg/cache"

// 初始化（memory 驱动）
c, err := cache.New(&cache.Config{
    Driver:   cache.DriverMemory,
    Capacity: 1000,
    TTL:      5 * time.Minute,
})

// Redis 驱动（需 KeyPrefix 才能 Flush）
c, err := cache.New(&cache.Config{
    Driver:    cache.DriverRedis,
    KeyPrefix: "myapp:",
    Redis: &cache.RedisConfig{
        Addr:     "127.0.0.1:6379",
        Password: "",
        DB:       0,
    },
    TTL: 10 * time.Minute,
})

// 使用
ctx := context.Background()
err = c.Set(ctx, "key", value, 0)   // TTL=0 使用默认
val, err := c.Get(ctx, "key")       // cache.ErrNotFound
err = c.Delete(ctx, "key")
err = c.Flush(ctx)                  // Redis 必须有 KeyPrefix

// GetOrSet（singleflight 防击穿）
val, err := c.GetOrSet(ctx, "key", func(ctx context.Context) (any, error) {
    return fetchFromDB(ctx)
}, 0)
```

## pkg/database 使用

```go
import "github.com/tokmz/qi/pkg/database"

db, err := database.New(&database.Config{
    DSN:            "user:pass@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4",
    MaxOpenConns:   20,
    MaxIdleConns:   5,
    ZapLogger:      zapLogger,     // 统一日志
    TracingEnabled: true,          // 开启 OTel 追踪
    ReadWriteSplit: &database.ReadWriteSplitConfig{
        Replicas: []string{
            "replica1:3306",
            "replica2:3306",
        },
    },
})

// 获取 *gorm.DB
gormDB := db.DB()
var users []User
gormDB.Find(&users)
```

## pkg/logger 使用

```go
import "github.com/tokmz/qi/pkg/logger"

log, err := logger.New(&logger.Config{
    Level:      "info",         // debug/info/warn/error
    Encoding:   "json",        // json/console
    OutputPath: "stdout",      // stdout 或文件路径
})

log.Info("服务启动", zap.String("addr", ":8080"))
log.Error("请求失败", zap.Error(err))
```

## 分布式锁

```go
import "github.com/tokmz/qi/pkg/cache"

locker := cache.NewLocker(&cache.RedisConfig{
    Addr: "127.0.0.1:6379",
}, "lock:")

ctx := context.Background()
unlock, err := locker.Lock(ctx, "order:123", 30*time.Second)
if err != nil {
    return nil, qi.ErrTooManyRequests.WithMessage("请勿重复提交")
}
defer unlock()
// 临界区代码...
```