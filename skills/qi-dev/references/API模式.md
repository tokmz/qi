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
c.Header("X-Request-Id", "abc")      // 设置响应头

// 上下文传值
c.Set("uid", "user-123")
uid, ok := c.Get("uid")              // ok=false 时 uid=nil
uid := c.MustGet("uid")              // 不存在时 panic

// 响应
c.OK(data)                           // code=0, status=200
c.OK(data, "操作成功")               // 自定义 message
c.Page(total, list)                  // 分页：{total, list}
c.Fail(err)                          // 提取 errors.Error 的 code/status
c.FailWithCode(code, status, msg)    // 完全自定义（参数顺序：code, status, msg）

// 逃生舱（访问 gin 原生功能，如 SSE、流式响应等）
c.Gin().ShouldBindHeader(&h)
```

## pkg/cache 使用

```go
import "github.com/tokmz/qi/pkg/cache"

// 初始化（memory 驱动）
c, err := cache.New(&cache.Config{
    Driver:     cache.DriverMemory,
    DefaultTTL: 5 * time.Minute,
    Memory: &cache.MemoryConfig{
        MaxSize:         1000,
        CleanupInterval: time.Minute,
    },
})

// Redis 驱动（需 KeyPrefix 才能 Flush）
c, err := cache.New(&cache.Config{
    Driver:     cache.DriverRedis,
    KeyPrefix:  "myapp:",
    DefaultTTL: 10 * time.Minute,
    Redis: &cache.RedisConfig{
        Addr:     "127.0.0.1:6379",
        Password: "",
        DB:       0,
    },
})

// 多级缓存（L1 内存 + L2 Redis）
c, err := cache.New(&cache.Config{
    Driver:     cache.DriverMultiLevel,
    KeyPrefix:  "myapp:",
    DefaultTTL: 10 * time.Minute,
    Memory:     &cache.MemoryConfig{MaxSize: 5_000},
    Redis:      &cache.RedisConfig{Addr: "127.0.0.1:6379"},
    Penetration: &cache.PenetrationConfig{
        EnableBloom: true,
        BloomN:      100_000,
        NullTTL:     60 * time.Second,
    },
})
```

### 基础操作

```go
ctx := context.Background()

// Set/Get/Del
err = c.Set(ctx, "key", value, 0)              // TTL=0 使用默认
var val MyType
err = c.Get(ctx, "key", &val)                  // 反序列化到 val，未命中返回 cache.ErrNotFound
err = c.Del(ctx, "key")                        // 注意：方法名是 Del，不是 Delete
err = c.Del(ctx, "key1", "key2")               // 批量删除

// 存在性、过期
exists, err := c.Exists(ctx, "key")
err = c.Expire(ctx, "key", 30*time.Minute)
ttl, err := c.TTL(ctx, "key")

// 批量
rawMap, err := c.MGet(ctx, []string{"k1", "k2"})
err = c.MSet(ctx, map[string]any{"k1": v1, "k2": v2}, time.Hour)

// 计数器
n, err := c.Incr(ctx, "counter")
n, err := c.IncrBy(ctx, "counter", 10)
n, err := c.DecrBy(ctx, "counter", 5)

// GetOrSet（singleflight 防击穿）
var user User
err = c.GetOrSet(ctx, "user:1", &user, time.Hour, func() (any, error) {
    return db.FindUser(1)                      // fn 签名：func() (any, error)，无 ctx 参数
})
// 注意：bloom filter 不影响 GetOrSet 的回调执行。bloom 仅拦截 Get/Exists/MGet 的纯读操作。
// fn 返回 cache.ErrNotFound 时自动写入空值标记（__null__:<key>），NullTTL 内不再查库。

// Flush（Redis 驱动必须配置 KeyPrefix，否则拒绝执行）
err = c.Flush(ctx)

// 序列化：默认 JSONSerializer，可选 GOBSerializer 或自定义
c, err := cache.New(&cache.Config{
    Driver:     cache.DriverMemory,
    Serializer: cache.GOBSerializer{},
})
```

### 分布式锁

```go
import "github.com/tokmz/qi/pkg/cache"

locker, err := cache.NewLocker(&cache.RedisConfig{
    Addr: "127.0.0.1:6379",
}, "lock:")

ctx := context.Background()

// 阻塞锁（指数退避 + 抖动）
unlock, err := locker.Lock(ctx, "order:123", 30*time.Second)
if err != nil {
    return nil, qi.ErrTooManyRequests.WithMessage("请勿重复提交")
}
defer unlock()

// 非阻塞尝试锁
ok, unlock, err := locker.TryLock(ctx, "order:123", 30*time.Second)
if !ok {
    // 锁被占用或出错
}
```

## pkg/database 使用

```go
import "github.com/tokmz/qi/pkg/database"

// MySQL（默认）
db, err := database.New(&database.Config{
    Type:           database.MySQL,
    DSN:            "user:pass@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4",
    MaxOpenConns:   20,
    MaxIdleConns:   5,
    ZapLogger:      zapLogger,     // 统一日志
    TracingEnabled: true,          // 开启 OTel 追踪
})

// PostgreSQL
db, err := database.New(&database.Config{
    Type: database.Postgres,                       // 注意：Postgres，不是 PostgresSQL
    DSN:  "host=localhost user=postgres password=pass dbname=mydb port=5432 sslmode=disable",
})

// 读写分离
db, err := database.New(&database.Config{
    Type: database.MySQL,
    DSN:  "user:pass@tcp(primary:3306)/mydb",
    ReadWriteSplit: &database.ReadWriteSplitConfig{
        Replicas: []string{
            "user:pass@tcp(replica1:3306)/mydb",
            "user:pass@tcp(replica2:3306)/mydb",
        },
        Policy: "round_robin",                  // random / round_robin
    },
})

// 使用（返回 *gorm.DB）
var users []User
db.Find(&users)
```

支持的数据库类型：`database.MySQL`、`database.Postgres`、`database.SQLite`、`database.SQLServer`

## pkg/logger 使用

```go
import "github.com/tokmz/qi/pkg/logger"

// 生产环境
log, err := logger.New(&logger.Config{
    Level:   logger.InfoLevel,
    Format:  logger.JSONFormat,
    Console: true,
})

// 开发环境
log, err := logger.New(&logger.Config{
    Level:   logger.DebugLevel,
    Format:  logger.ConsoleFormat,
    Console: true,
})

// 文件轮转
log, err := logger.New(&logger.Config{
    Level:  logger.InfoLevel,
    Format: logger.JSONFormat,
    Console: true,
    Rotate: &logger.RotateConfig{
        Filename:   "./logs/app.log",
        MaxSize:    100,    // MB
        MaxAge:     30,     // 天
        MaxBackups: 10,
        Compress:   true,
    },
})

// 使用（返回 logger.Logger 接口，底层 zap）
log.Info("服务启动", zap.String("addr", ":8080"))
log.Error("请求失败", zap.Error(err))

// 带 Context（自动提取 trace_id / span_id / uid）
log.InfoContext(ctx, "订单创建", zap.String("order_id", "ORD-001"))

// 关闭（刷新缓冲区 + 关闭文件句柄）
defer log.Close()
```

## pkg/config 使用

```go
import "github.com/tokmz/qi/pkg/config"

// 初始化（函数式 Option 模式，非 *Config）
cfg := config.New(
    config.WithConfigFile("config.yaml"),
    config.WithConfigPaths("./conf", "/etc/myapp"),
    config.WithEnvPrefix("APP"),
    config.WithAutoWatch(true),                  // 自动监听文件变更
    config.WithOnChange(func() { log.Info("配置已变更") }),
)
// New() 不返回 error，Load() 才会
if err := cfg.Load(); err != nil {
    panic(err)
}
defer cfg.Close()

// 读取
dsn := cfg.GetString("database.dsn")
port := cfg.GetInt("server.port")
debug := cfg.GetBool("debug")
timeout := cfg.GetDuration("timeout")

// 泛型读取
port := config.Get[int](cfg, "server.port")

// 热重载（需在 New 时传入 WithAutoWatch(true) 或手动调用）
cfg.StartWatch()
cfg.StopWatch()

// 保护模式（防止关键配置被覆盖）
cfg.SetProtected(true)

// 高级方法
subCfg := cfg.Sub("database")        // 子配置（只读轻量实例）
var dbCfg DBConfig
cfg.Unmarshal(&dbCfg)                // 全量反序列化
cfg.UnmarshalKey("database", &dbCfg) // 指定 key 反序列化
```
