---
name: "qi-framework"
description: "Qi framework expert for Go web development. Invoke when working with qi framework code, creating routes, middleware, error handling, i18n, OpenAPI docs, or pkg/* packages."
---

# Qi Framework Skill

Qi 是一个基于 Gin 的轻量级 Go Web 框架，提供统一响应格式、自动参数绑定、泛型路由支持和优雅关机功能。

## 核心架构

```
HTTP Request → Engine → Gin Engine → Middleware Chain → Wrapper Layer → Handler → Response
```

| 组件 | 文件 | 职责 |
|------|------|------|
| Engine | `engine.go` | http.Server 包装，生命周期管理 |
| Context | `context.go` | gin.Context 包装，统一响应 |
| Router | `router.go` | 路由注册，泛型路由处理 |
| Response | `response.go` | 统一 JSON 响应格式 |

## 快速开始

```go
engine := qi.Default(qi.WithOpenAPI(&openapi.Config{Title: "API", Version: "1.0.0"}))
r := engine.Router()

// 泛型路由（推荐）
qi.POST[CreateReq, UserResp](r, "/users", handler,
    openapi.Doc(openapi.Summary("创建用户"), openapi.Tags("Users")))

// 基础路由
r.GET("/ping", func(c *qi.Context) { c.Success("pong") })

engine.Run(":8080")
```

## API 参考

### Engine

| 方法 | 说明 |
|------|------|
| `qi.New(opts ...Option)` | 创建 Engine（含 Recovery） |
| `qi.Default(opts ...Option)` | 创建 Engine（含 Recovery + Logger） |
| `engine.Use(mw ...HandlerFunc)` | 注册全局中间件 |
| `engine.Group(path, mw ...)` | 创建路由组 |
| `engine.Router()` | 获取根路由组 |
| `engine.Translator()` | 获取 i18n 翻译器 |
| `engine.Run(addr ...)` | 启动 HTTP 服务器 |
| `engine.RunTLS(addr, cert, key)` | 启动 HTTPS 服务器 |
| `engine.Shutdown(ctx)` | 手动关闭服务器 |

### RouterGroup

| 方法 | 说明 |
|------|------|
| `rg.Group(path, mw ...)` | 创建子路由组 |
| `rg.Use(mw ...)` | 注册中间件 |
| `rg.SetTag(name, desc)` | 设置默认 OpenAPI tag |
| `rg.SetSecurity(schemes ...)` | 设置默认认证 |
| `rg.DocRoute(method, path, doc)` | 手动注册文档 |
| `rg.GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS/Any(path, handler, mw ...)` | 基础路由 |
| `rg.Static/StaticFile/StaticFS(...)` | 静态文件服务 |

### 泛型路由（自动收集 OpenAPI）

| 类型 | 方法 | 说明 |
|------|------|------|
| 有请求有响应 | `qi.GET/POST/PUT/PATCH/DELETE[Req, Resp](rg, path, handler, doc, mw ...)` | 自动绑定 + 响应 |
| 有请求无响应 | `qi.GET0/POST0/PUT0/PATCH0/DELETE0[Req](rg, path, handler, doc, mw ...)` | 自动绑定，返回 Nil |
| 无请求有响应 | `qi.GETOnly/POSTOnly[Resp](rg, path, handler, doc, mw ...)` | 无绑定，自动响应 |

底层函数（不收集 OpenAPI）：`qi.Handle/Handle0/HandleOnly`

### Context

**请求方法**

| 方法 | 说明 |
|------|------|
| `c.Request()` / `c.Writer()` | 获取底层 http.Request / ResponseWriter |
| `c.ClientIP()` / `c.ContentType()` / `c.FullPath()` | 基础信息 |
| `c.Param(key)` / `c.Query(key)` / `c.PostForm(key)` | 参数获取 |
| `c.GetHeader(key)` / `c.Header(key, value)` | 请求头/响应头 |

**绑定方法**

| 方法 | 自动响应错误 | 说明 |
|------|-------------|------|
| `c.Bind/BindJSON/BindQuery/BindURI/BindHeader(obj)` | ✅ | 推荐 |
| `c.ShouldBind/ShouldBindJSON/ShouldBindQuery/ShouldBindUri/ShouldBindHeader(obj)` | ❌ | 手动处理 |

**响应方法**

| 方法 | 说明 |
|------|------|
| `c.Success(data)` | 成功响应 |
| `c.SuccessWithMessage(data, msg)` | 带消息的成功响应 |
| `c.Nil()` | 无数据响应 |
| `c.Fail(code, msg)` | 失败响应 |
| `c.RespondError(err)` | 自动处理 *errors.Error |
| `c.Page(list, total)` | 分页响应 |
| `c.JSON(code, obj)` | 原始 JSON |

**上下文存储**：`c.Set/Get/GetString/GetInt/GetInt64/GetBool(key)`

**流程控制**：`c.Next()` / `c.Abort()` / `c.AbortWithStatus(code)` / `c.IsAborted()`

**标准库 Context**：`c.RequestContext()` 返回含 TraceID/UID/Language 的 context.Context

**i18n**：`c.T(key, args...)` / `c.Tn(one, other, count, args...)`

### Response

```go
qi.NewResponse(code, data, msg)    // 创建响应
qi.Success(data)                   // 成功响应
qi.Fail(code, msg)                 // 失败响应
qi.NewPageResp(list, total)        // 分页数据
qi.PageData(list, total)           // 分页响应
resp.WithTraceID(traceID)          // 设置 TraceID
```

### 配置选项

```go
qi.New(
    qi.WithMode(mode),                    // debug/release/test
    qi.WithAddr(addr),                    // 监听地址
    qi.WithReadTimeout/WriteTimeout/IdleTimeout(d),
    qi.WithShutdownTimeout(d),
    qi.WithBeforeShutdown/AfterShutdown(fn),
    qi.WithTrustedProxies(proxies...),
    qi.WithMaxMultipartMemory(size),
    qi.WithI18n(cfg),                     // nil 不启用
    qi.WithOpenAPI(cfg),                  // nil 不启用
)
```

### 辅助函数

```go
// Gin Context
qi.SetContextTraceID/GetContextTraceID(c, traceID)
qi.SetContextUid/GetContextUid(c, uid)
qi.SetContextLanguage/GetContextLanguage(c, lang)

// 标准 context.Context
qi.GetTraceIDFromContext/GetUidFromContext/GetLanguageFromContext(ctx)
```

## 自动绑定策略

| HTTP 方法 | 绑定方式 |
|-----------|----------|
| GET/DELETE | URI + Query |
| POST/PUT/PATCH | Body (自动检测 Content-Type) + URI |

**重要**：`Bind*()` 失败时自动响应 400 错误，只需 `return`。

## 响应格式

```json
// 标准响应
{"code": 200, "data": {...}, "message": "success", "trace_id": "xxx"}

// 分页响应
{"code": 200, "data": {"list": [...], "total": 100}, "message": "success"}
```

## 错误处理

```go
import "github.com/tokmz/qi/pkg/errors"

// 预定义错误
errors.ErrServer(1000,500) / ErrBadRequest(1001,400) / ErrUnauthorized(1002,401) / ErrForbidden(1003,403) / ErrNotFound(1004,404)

// 使用方式
return nil, errors.ErrBadRequest.WithMessage("用户名不能为空")
return nil, errors.New(2001, "禁止访问", 403)
return nil, errors.ErrServer.WithError(err)
```

## 中间件

**执行顺序**：全局 → 路由组 → 路由级 → Handler

```go
func authMiddleware(c *qi.Context) {
    if c.GetHeader("Authorization") == "" {
        c.RespondError(errors.ErrUnauthorized)
        return
    }
    c.Next()
}

// 使用：engine.Use(mw) / rg.Use(mw) / r.GET(path, handler, mw1, mw2)
```

## i18n

```go
qi.WithI18n(&i18n.Config{
    Dir: "./locales", DefaultLanguage: "zh-CN", Languages: []string{"zh-CN", "en-US"},
})

// 语言检测：Query(lang) > X-Language > Accept-Language > 默认
// 使用：c.T("key", "Name", "World") / c.Tn("one", "other", count)
```

## OpenAPI

```go
qi.WithOpenAPI(&openapi.Config{
    Title: "API", Version: "1.0.0", Path: "/openapi.json", SwaggerUI: "/docs",
})

// 文档选项
openapi.Doc(
    openapi.Summary("摘要"),
    openapi.Tags("Users"),
    openapi.Security("BearerAuth"),
    openapi.NoSecurity(),
)

// 路由组继承
v1.SetTag("V1", "描述")
v1.SetSecurity("BearerAuth")
```

## pkg 子包

### pkg/cache

```go
// 初始化
cache, err := cache.New(cfg *cache.Config)
cache, err := cache.NewWithOptions(opts ...cache.Option)

// 配置字段
&cache.Config{
    Driver     DriverType    // redis/memory
    Redis      *RedisConfig  // Redis 配置
    Memory     *MemoryConfig // Memory 配置
    Serializer Serializer    // 序列化器
    KeyPrefix  string        // 键前缀
    DefaultTTL time.Duration // 默认 TTL
}

// Redis 配置
&cache.RedisConfig{
    Addr, Addrs []string, Mode RedisMode, Username, Password, DB int,
    PoolSize, MinIdleConns, MaxRetries int,
    DialTimeout, ReadTimeout, WriteTimeout time.Duration,
    MasterName string, // 哨兵模式
}

// Memory 配置
&cache.MemoryConfig{
    DefaultExpiration, CleanupInterval time.Duration,
    MaxEntries int,
}

// 配置选项
cache.WithRedis(cfg), cache.WithMemory(cfg), cache.WithSerializer(s),
cache.WithKeyPrefix(prefix), cache.WithDefaultTTL(ttl)
```

### pkg/request

```go
// 初始化
client := request.New(opts ...request.Option)
client := request.NewWithConfig(cfg *request.Config)

// 配置字段
&request.Config{
    BaseURL             string
    Timeout             time.Duration
    Headers             map[string]string
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    MaxConnsPerHost     int
    IdleConnTimeout     time.Duration
    Retry               *RetryConfig
    Interceptors        []Interceptor
    Logger              Logger
    EnableTracing       bool
    InsecureSkipVerify  bool
    Transport           http.RoundTripper
}

// 配置选项
request.WithBaseURL(url), request.WithTimeout(d),
request.WithHeader/Headers(k, v), request.WithRetry(cfg),
request.WithInterceptor(i), request.WithLogger(l),
request.WithTracing(bool), request.WithInsecureSkipVerify(bool),
request.WithTransport(t), request.WithMaxIdleConns/PerHost(n)

// 链式调用
resp, err := client.Get("/users").SetQuery("page", "1").Do()
user, err := request.Do[User](client.Post("/users").SetBody(&req))
users, err := request.DoList[User](client.Get("/users"))

// 认证
client.Get("/secure").SetBearerToken("token").Do()
client.Get("/auth").SetBasicAuth("user", "pass").Do()

// 文件上传
client.Post("/upload").SetFile("avatar", "/path/file.png").Do()
```

### pkg/job

```go
// 初始化
scheduler := job.NewScheduler(storage job.Storage, opts ...job.Option)
scheduler.Start() / scheduler.Stop()

// 配置字段
&job.Config{
    Logger               Logger
    ConcurrentRuns       int           // 并发数
    JobTimeout           time.Duration // 任务超时
    RetryDelay           time.Duration // 重试间隔
    AutoStart            bool
    EnableBatchUpdate    bool
    BatchSize            int
    BatchFlushInterval   time.Duration
    EnableCache          bool
    CacheCapacity        int
    CacheTTL             time.Duration
    CacheCleanupInterval time.Duration
}

// 配置选项
job.WithConcurrentRuns(n), job.WithJobTimeout(d), job.WithRetryDelay(d),
job.WithAutoStart(bool), job.WithLogger(l),
job.WithEnableBatchUpdate(bool), job.WithBatchSize(n),
job.WithEnableCache(bool), job.WithCacheCapacity(n), job.WithCacheTTL(d)

// 注册任务
scheduler.Register(name, cronExpr, handler, opts ...JobOption)
scheduler.RegisterWithID(id, name, cronExpr, handler, opts ...JobOption)
```

### pkg/logger

```go
// 初始化
logger, err := logger.New(cfg *logger.Config)

// 配置字段
&logger.Config{
    Level             Level              // 日志级别
    Format            Format             // json/console
    Console           bool               // 控制台输出
    File              string             // 文件路径
    Rotate            *RotateConfig      // 轮转配置
    Sampling          *SamplingConfig    // 采样配置
    BufferSize        int                // 缓冲区大小
    EnableCaller      *bool              // 调用位置
    EnableStacktrace  *bool              // 堆栈
    EncoderConfig     *zapcore.EncoderConfig
    Hooks             []Hook
}

// 轮转配置
&logger.RotateConfig{
    MaxSize, MaxBackups, MaxAge int, Compress bool,
}

// 使用
logger.Info/Debug/Warn/Error(msg, fields...)
logger.With(fields...).Info(msg)
logger.Sync()
```

### pkg/tracing

```go
// 初始化
provider, err := tracing.NewProvider(cfg *tracing.Config)

// 配置字段
&tracing.Config{
    ServiceName        string
    ServiceVersion     string
    Environment        string
    ExporterType       string            // otlp/stdout/noop
    ExporterEndpoint   string
    ExporterHeaders    map[string]string
    Insecure           bool
    SamplingRate       float64
    SamplingType       string            // always/never/ratio/parent_based
    Enabled            bool
    ResourceAttributes map[string]string
    BatchTimeout       time.Duration
    MaxExportBatchSize int
    MaxQueueSize       int
}

// 使用
ctx = tracing.ContextWithSpan(ctx, span)
tracer := tracing.Tracer("name")
```

### pkg/orm

```go
// 初始化
db, err := orm.New(cfg *orm.Config)

// 配置字段
&orm.Config{
    Type                   DBType  // mysql/postgres/sqlite/sqlserver
    DSN                    string
    MaxIdleConns           int
    MaxOpenConns           int
    ConnMaxLifetime        time.Duration
    ConnMaxIdleTime        time.Duration
    SkipDefaultTransaction bool
    PrepareStmt            bool
    DisableAutomaticPing   bool
    LogLevel               int
    SlowThreshold          time.Duration
    Colorful               bool
    TablePrefix            string
    SingularTable          bool
    DryRun                 bool
    ReadWriteSplit         *ReadWriteSplitConfig
}

// 读写分离配置
&orm.ReadWriteSplitConfig{
    Sources         []string        // 从库 DSN
    Policy          string          // random/round_robin
    MaxIdleConns    *int
    MaxOpenConns    *int
    ConnMaxLifetime *time.Duration
    ConnMaxIdleTime *time.Duration
}
```

### pkg/config

```go
// 初始化
cfg, err := config.Load(path string, opts ...config.Option)

// 配置选项
config.WithEnvPrefix(prefix), config.WithWatch(bool)

// 使用
cfg.GetString/GetInt/GetBool(key)
cfg.Unmarshal(&struct)
cfg.Watch(callback)
```

### pkg/i18n

```go
// 初始化
t, err := i18n.New(cfg *i18n.Config)

// 配置字段
&i18n.Config{
    Dir             string
    DefaultLanguage string
    Languages       []string
}

// 使用
t.Tr(lang, key, args...)
t.TrPlural(lang, one, other, count, args...)
```

### pkg/errors

```go
// 创建错误
errors.New(code, message, httpCode)
errors.ErrXxx.WithMessage(msg)
errors.ErrXxx.WithError(err)
errors.ErrXxx.Clone()

// 检查
errors.Is(err, target)
errors.As(err, &target)
```

### pkg/openapi

```go
// 文档选项
openapi.Doc(
    openapi.Summary("摘要"),
    openapi.Desc("描述"),
    openapi.Tags("Users"),
    openapi.Security("BearerAuth"),
    openapi.NoSecurity(),
    openapi.Deprecated(),
    openapi.RequestType(MyReq{}),
    openapi.ResponseType(MyResp{}),
)
```

## 开发命令

```bash
go build ./...      # 构建
go test ./...       # 测试
go run example/main.go  # 运行示例
```

## 代码规范

1. 使用泛型路由减少样板代码
2. `Bind*()` 失败直接 `return`
3. 使用 `pkg/errors` 统一错误码
4. 中间件变长参数：`r.GET(path, handler, mw1, mw2)`
5. Options 模式配置

## 模块路径

`github.com/tokmz/qi` (Go 1.25+)
