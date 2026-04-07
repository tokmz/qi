# Qi

基于 [Gin](https://github.com/gin-gonic/gin) 的 Go Web 框架，提供统一响应封装、业务错误系统、泛型请求绑定、自动 OpenAPI 3.0 文档生成、请求日志、链路追踪等生产级能力。

```
go get github.com/tokmz/qi
```

> 要求 Go 1.25+

## 快速开始

```go
package main

import (
    "github.com/tokmz/qi"
)

func main() {
    app := qi.New(
        qi.WithAddr(":8080"),
        qi.WithOpenAPI(&qi.OpenAPIConfig{
            Title:     "My API",
            Version:   "1.0.0",
            SwaggerUI: "/docs",
        }),
    )

    app.GET("/ping", func(c *qi.Context) {
        c.OK("pong")
    })

    app.Run()
}
```

启动后访问 `http://127.0.0.1:8080/docs/` 查看 Swagger UI。

完整示例见 [`examples/basic/main.go`](examples/basic/main.go)。

---

## 核心特性

| 特性 | 说明 |
|------|------|
| **统一响应封装** | 所有响应走同一 JSON 结构，自动填充 `trace_id` |
| **业务错误系统** | 预定义错误码，不可变克隆链，Code + HTTP Status 分离 |
| **泛型请求绑定** | `Bind` / `BindR` / `BindE` / `BindRE` 自动完成请求绑定 + 响应包装，请求路径零反射 |
| **OpenAPI 3.0** | 基于类型反射，注册路由时同步生成文档，内置 Swagger UI |
| **请求日志** | 基于 zap，记录方法/路径/状态码/耗时/IP/trace_id |
| **链路追踪** | 集成 OpenTelemetry，支持 OTLP gRPC/HTTP，自动注入 `trace_id` |
| **多级缓存** | 内存 LRU + Redis，防穿透/击穿/雪崩，分布式锁 |
| **数据库** | GORM 封装，读写分离，连接池，zap 日志接入 |
| **优雅关闭** | 监听系统信号，flush span 后关闭 HTTP server |

---

## Engine 配置

```go
app := qi.New(
    qi.WithAddr(":8080"),                  // 监听地址（默认 :8080）
    qi.WithMode("release"),                // 运行模式：debug / release / test
    qi.WithLogger(&qi.LoggerConfig{...}),  // 请求日志
    qi.WithTracing(&qi.TracingConfig{...}),// 链路追踪
    qi.WithOpenAPI(&qi.OpenAPIConfig{...}),// OpenAPI 文档
)
```

---

## 响应

统一 JSON 格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

`trace_id` 由 tracing 中间件自动注入，也可手动 `c.Set("trace_id", "xxx")`。

```go
c.OK(data)                           // code=0, message="success"
c.OK(data, "创建成功")                 // code=0, 自定义 message
c.Fail(qi.ErrNotFound)               // 自动提取 code / status / message
c.FailWithCode(400, 1001, "参数错误")  // 完全自定义
c.Page(total, list)                   // 分页响应
```

---

## 路由注册

```go
// 基础方式
app.GET("/ping", func(c *qi.Context) { c.OK("pong") })

// 路由分组 + 中间件
v1 := app.Group("/api/v1")
v1.Use(authMiddleware())
v1.GET("/users", listUsers)

// 链式 API（同步生成 OpenAPI 文档）
v1.API().
    POST("/users", createUser).
    Summary("创建用户").
    Tags("用户").
    Done()
```

支持：`GET` `POST` `PUT` `PATCH` `DELETE` `HEAD` `OPTIONS` `Any`

---

## 路由元信息

通过 `RouteBuilder` 注册的路由，元信息（Summary、Tags 等）在运行时可被中间件查询，适用于操作日志、权限注解等场景。

```go
// 注册路由时声明元信息
v1.API().
    POST("/users", qi.Bind(createUser)).
    Summary("创建用户").
    Tags("用户").
    OperationID("createUser").
    Done()

// 中间件中查询
func OperationLogMiddleware(e *qi.Engine) qi.HandlerFunc {
    return func(c *qi.Context) {
        c.Next()
        meta := e.RouteMeta(c.Request().Method, c.FullPath())
        if meta != nil {
            log.Printf("操作：%s tags=%v", meta.Summary, meta.Tags)
        }
    }
}
```

> 直接用 `app.GET/POST` 注册的路由，`meta.Summary` 降级为 handler 函数名，其余字段为零值。

---

## 泛型绑定

```go
// Bind[Req, Resp]：自动绑定请求 + 包装响应
app.POST("/users", qi.Bind[CreateUserReq, User](createUser))

func createUser(c *qi.Context, req *CreateUserReq) (*User, error) {
    return &User{Name: req.Name}, nil
    // 返回 error 自动调用 c.Fail()
}

// BindR[Resp]：无请求体，只包装响应
app.GET("/users", qi.BindR[[]User](listUsers))

// BindE[Req]：有请求体，无响应体（如 DELETE）
app.DELETE("/users/:id", qi.BindE[DeleteUserReq](deleteUser))

func deleteUser(c *qi.Context, req *DeleteUserReq) error {
    return service.Delete(req.ID)
}

// BindRE：无请求体，无响应体
app.POST("/cache/flush", qi.BindRE(clearCache))
```

---

## 请求日志

```go
app := qi.New(
    qi.WithLogger(&qi.LoggerConfig{
        SkipPaths: []string{"/ping", "/health"}, // 跳过的路径
        // Output 默认 os.Stdout，可指定其他 io.Writer
    }),
)
```

输出格式：
```
[QI] 2026/03/23 - 17:50:29 |  200 |       917ns |       127.0.0.1 | GET     "/api/v1/users" 4bf92f35
```

- `≥500` 走 `logger.Error`，`≥400` 走 `logger.Warn`，其余走 `logger.Info`
- 有 `trace_id` 时自动追加在末尾（与 tracing 中间件联动）
- Logger 的 `Sync()` 由调用方管理，框架不介入

---

## 链路追踪

```go
app := qi.New(
    qi.WithTracing(&qi.TracingConfig{
        ServiceName: "user-service",
        Exporter:    qi.TracingExporterOTLPGRPC,
        Endpoint:    "otel-collector:4317",
        Insecure:    true,
        SampleRate:  0.1,
        SkipPaths:   []string{"/ping", "/health"},
    }),
)
// ✅ 自动初始化 OTel TracerProvider
// ✅ 自动注册追踪中间件
// ✅ trace_id 自动填充响应 JSON 和日志
// ✅ 优雅关闭时自动 flush span
```

导出器：

| 常量 | 说明 |
|------|------|
| `TracingExporterNoop` | 禁用（默认） |
| `TracingExporterStdout` | 控制台输出（调试） |
| `TracingExporterOTLPGRPC` | gRPC，端口 4317 |
| `TracingExporterOTLPHTTP` | HTTP，支持 `http://` / `https://` 前缀 |

业务层开启子 span：
```go
import "go.opentelemetry.io/otel"

ctx, span := otel.Tracer("my-service").Start(ctx, "CreateOrder")
defer span.End()
```

---

## 业务错误

```go
import "github.com/tokmz/qi/pkg/errors"

// 定义哨兵错误
var ErrUserNotFound = errors.NewWithStatus(2001, 404, "user not found")

// 克隆链（不污染原始哨兵）
return nil, ErrUserNotFound.
    WithErr(dbErr).
    WithMessage("用户 ID 不存在")
```

框架预定义错误：

| 变量 | Code | Status |
|------|------|--------|
| `ErrServer` | 1000 | 500 |
| `ErrBadRequest` | 1001 | 400 |
| `ErrUnauthorized` | 1002 | 401 |
| `ErrForbidden` | 1003 | 403 |
| `ErrNotFound` | 1004 | 404 |
| `ErrConflict` | 1005 | 409 |
| `ErrTooManyRequests` | 1006 | 429 |
| `ErrInvalidParams` | 1100 | 500 |
| `ErrMissingParams` | 1101 | 500 |
| `ErrInvalidFormat` | 1102 | 500 |
| `ErrOutOfRange` | 1103 | 500 |

---

## 缓存

```go
import "github.com/tokmz/qi/pkg/cache"

c, err := cache.New(&cache.Config{
    Driver:    cache.DriverMultiLevel,
    KeyPrefix: "app:",
    Memory:    &cache.MemoryConfig{MaxSize: 5_000},
    Redis:     &cache.RedisConfig{Addr: "127.0.0.1:6379"},
    Penetration: &cache.PenetrationConfig{
        EnableBloom: true,
        BloomN:      100_000,
        NullTTL:     60 * time.Second,
    },
    TracingEnabled: true,
})

c.Set(ctx, "user:1", user, time.Hour)
var u User
c.Get(ctx, "user:1", &u)

// 防击穿
c.GetOrSet(ctx, "user:1", &u, time.Hour, func() (any, error) {
    return db.FindUser(1)
})

// 分布式锁
locker, _ := cache.NewLocker(&cache.RedisConfig{Addr: "127.0.0.1:6379"}, "app:")
unlock, _ := locker.Lock(ctx, "order:create", 10*time.Second)
defer unlock()
```

| 问题 | 方案 |
|------|------|
| 缓存穿透 | Bloom filter + 空值标记 key |
| 缓存击穿 | `GetOrSet` 内置 singleflight |
| 缓存雪崩 | TTL ±10% 随机抖动 |

---

## 数据库

```go
import "github.com/tokmz/qi/pkg/database"

db, err := database.New(&database.Config{
    Type:           database.MySQL,
    DSN:            "user:pass@tcp(localhost:3306)/app?parseTime=True",
    ZapLogger:      zapLogger,
    TracingEnabled: true,
    ReadWriteSplit: &database.ReadWriteSplitConfig{
        Replicas: []string{"user:pass@tcp(replica:3306)/app?parseTime=True"},
        Policy:   "round_robin",
    },
})
```

---

## 项目结构

```
qi/
├── engine.go              Engine、Config、服务启动与优雅关闭
├── context.go             Context 封装、绑定、响应方法
├── handler.go             HandlerFunc 类型定义、gin 适配器
├── router.go              RouterGroup、routerStore、路由注册
├── binding.go             Bind/BindR/BindE/BindRE 泛型请求绑定
├── openapi.go             OpenAPIConfig、RouteBuilder、OpenAPI 集成
├── tracing.go             TracingConfig 类型别名、WithTracing option
├── logger.go              LoggerConfig、WithLogger option
├── response.go            Response 统一响应结构体
├── errors.go              预定义业务错误
├── internal/
│   ├── openapi/           OpenAPI 3.0.3 文档生成器
│   ├── tracing/           OTel TracerProvider 初始化、HTTP 追踪中间件
│   └── logging/           请求日志中间件
├── pkg/
│   ├── errors/            业务错误类型（可独立使用）
│   ├── logger/            zap 日志封装
│   ├── config/            viper 配置管理
│   ├── database/          GORM 封装，读写分离，链路追踪
│   ├── cache/             多级缓存，防穿透/击穿/雪崩，分布式锁
│   └── middleware/        i18n 等中间件
├── utils/
│   ├── strings/           字符串操作、大小写转换
│   ├── array/             泛型切片操作
│   ├── convert/           类型转换
│   ├── datetime/          时间格式化、解析
│   ├── pointer/           指针辅助函数
│   └── regexp/            LRU 缓存正则表达式池
└── examples/
    └── basic/             完整示例（日志、追踪、OpenAPI、鉴权、分页）
```

---

## 许可证

MIT
