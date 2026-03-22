# Qi

基于 [Gin](https://github.com/gin-gonic/gin) 的 Go Web 框架，提供统一响应封装、业务错误系统、泛型请求绑定和自动 OpenAPI 3.0 文档生成。

```
go get github.com/tokmz/qi
```

> 要求 Go 1.25+

## 快速开始

```go
package main

import "github.com/tokmz/qi"

func main() {
    app := qi.New(
        qi.WithAddr(":8080"),
        qi.WithOpenAPI(&qi.OpenAPIConfig{
            Title:       "My API",
            Version:     "1.0.0",
            Description: "示例 API",
        }),
    )

    app.GET("/ping", func(c *qi.Context) {
        c.OK("pong")
    })

    app.Run()
}
```

启动后访问 `http://127.0.0.1:8080/docs/` 查看 Swagger UI。

## 核心特性

- **统一响应封装** — 所有响应走同一 JSON 结构，自动填充 `trace_id`
- **业务错误系统** — 预定义错误码，支持不可变克隆链
- **泛型请求绑定** — `Bind` / `BindR` 自动完成请求绑定 + 响应包装，请求路径零反射
- **OpenAPI 3.0 自动生成** — 基于类型反射，注册路由时同步生成文档
- **优雅关闭** — 监听系统信号，可配置超时

## 响应

所有响应通过 `Response` 结构体封装，格式如下：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "trace_id": "abc123"
}
```

`trace_id` 自动从 `c.Set("trace_id", "xxx")` 读取，通常在中间件中注入。

```go
c.OK(data)                          // code=0, message="success"
c.OK(data, "创建成功")                // code=0, 自定义 message
c.Fail(qi.ErrNotFound)              // 自动提取 code / status / message
c.FailWithCode(400, 1001, "参数错误") // 完全自定义
c.Page(total, list)                  // 分页: {"total": N, "list": [...]}
```

## 路由注册

支持三种方式注册路由：

```go
// 1. 基础方式 — HandlerFunc
app.GET("/ping", func(c *qi.Context) {
    c.OK("pong")
})

// 2. 路由分组 + 中间件
v1 := app.Group("/api/v1")
v1.Use(authMiddleware())
v1.GET("/users", listUsers)

// 3. 链式 API（配合 OpenAPI 文档）
v1.API().
    GET("/users", listUsers).
    Summary("获取用户列表").
    Tags("用户").
    Response([]User{}).
    Done()
```

支持的 HTTP 方法：`GET` `POST` `PUT` `PATCH` `DELETE` `HEAD` `OPTIONS`，以及 `Any`（注册全部方法）。

## 泛型请求绑定

`Bind` 和 `BindR` 通过泛型函数自动完成请求绑定、OpenAPI 类型推导和响应包装，注册后请求路径上零反射开销。

### Bind — 有请求体

函数签名：`func(*Context, *Req) (*Resp, error)`

```go
type CreateUserReq struct {
    Name  string `json:"name"  binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func CreateUser(c *qi.Context, req *CreateUserReq) (*User, error) {
    return &User{ID: "1", Name: req.Name, Email: req.Email}, nil
}

api.API().
    POST("/users", qi.Bind(CreateUser)).
    Summary("创建用户").
    Tags("用户").
    Done()
```

- POST / PUT / PATCH：自动绑定请求体（JSON）
- GET / DELETE 等：自动绑定 Query 参数
- 字段含 `uri` tag 时额外调用 `BindURI` 绑定路径参数

### BindR — 无请求体

函数签名：`func(*Context) (*Resp, error)`

```go
func GetUser(c *qi.Context) (*User, error) {
    id := c.Param("id")
    if id == "" {
        return nil, qi.ErrMissingParams.WithMessage("id is required")
    }
    return &User{ID: id, Name: "Alice"}, nil
}

api.API().
    GET("/users/:id", qi.BindR(GetUser)).
    Summary("获取用户详情").
    Tags("用户").
    Done()
```

## 业务错误

### 预定义错误

| 变量 | Code | HTTP 状态 | 说明 |
|------|------|-----------|------|
| `ErrServer` | 1000 | 500 | 服务器错误 |
| `ErrBadRequest` | 1001 | 400 | 请求参数错误 |
| `ErrUnauthorized` | 1002 | 401 | 未授权 |
| `ErrForbidden` | 1003 | 403 | 禁止访问 |
| `ErrNotFound` | 1004 | 404 | 资源不存在 |
| `ErrConflict` | 1005 | 409 | 资源冲突 |
| `ErrTooManyRequests` | 1006 | 429 | 请求过于频繁 |
| `ErrInvalidParams` | 1100 | 400 | 参数无效 |
| `ErrMissingParams` | 1101 | 400 | 缺少参数 |
| `ErrInvalidFormat` | 1102 | 400 | 格式错误 |
| `ErrOutOfRange` | 1103 | 400 | 超出范围 |

### 使用方式

预定义错误是哨兵值，链式方法每次返回新实例，不会污染原始错误：

```go
// 直接返回
return nil, qi.ErrNotFound

// 覆盖消息
return nil, qi.ErrNotFound.WithMessage("用户不存在")

// 格式化消息
return nil, qi.ErrNotFound.WithMessagef("用户 %s 不存在", id)

// 附加原始错误（用于日志）
return nil, qi.ErrServer.WithErr(dbErr)

// 自定义 HTTP 状态码
return nil, qi.ErrBadRequest.WithStatus(422)
```

### 自定义业务错误

```go
import "github.com/tokmz/qi/pkg/errors"

var (
    ErrUserNotFound  = errors.NewWithStatus(2001, 404, "用户不存在")
    ErrPasswordWrong = errors.NewWithStatus(2002, 401, "密码错误")
    ErrUserDisabled  = errors.NewWithStatus(2003, 403, "账号已禁用")
)
```

在 handler 中使用标准 `errors.Is` 判断：

```go
if errors.Is(err, ErrUserNotFound) {
    // ...
}
```

## OpenAPI 文档

### 基础配置

```go
app := qi.New(
    qi.WithOpenAPI(&qi.OpenAPIConfig{
        Title:       "My API",
        Version:     "1.0.0",
        Description: "API 描述",
        // Path:     "/docs",       // 默认 /docs
        // SwaggerUI: true,         // 默认开启
    }),
)
```

启动后：
- Swagger UI：`http://127.0.0.1:8080/docs/`
- OpenAPI JSON：`http://127.0.0.1:8080/docs/openapi.json`

### RouteBuilder 链式方法

```go
api.API().
    POST("/users", qi.Bind(CreateUser)).
    Summary("创建用户").           // 接口摘要
    Description("创建一个新用户").  // 详细描述
    Tags("用户").                  // 分组标签
    Request(&CreateUserReq{}).    // 显式指定请求类型（Bind 自动推导可省略）
    Response(&User{}).            // 显式指定响应类型（Bind 自动推导可省略）
    Deprecated().                 // 标记为已废弃
    Done()                        // 完成注册，必须调用
```

类型推导优先级：显式 `.Request()` / `.Response()` > `Bind` / `BindR` 自动推导。

## 中间件

```go
// 全局中间件
app.Use(Logger(), Recovery())

// 分组中间件
v1 := app.Group("/api/v1")
v1.Use(AuthRequired())

// 路由级中间件
v1.GET("/admin", AdminOnly(), adminHandler)

// 中间件示例
func Logger() qi.HandlerFunc {
    return func(c *qi.Context) {
        start := time.Now()
        c.Next()
        log.Printf("%s %s %v", c.Request().Method, c.Request().URL.Path, time.Since(start))
    }
}

func AuthRequired() qi.HandlerFunc {
    return func(c *qi.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.Fail(qi.ErrUnauthorized)
            c.Abort()
            return
        }
        c.Set("user_id", parseToken(token))
        c.Next()
    }
}
```

## Context API

### 请求

```go
// 路径参数
id := c.Param("id")                  // /users/:id

// Query 参数
page := c.Query("page")              // ?page=1
page := c.DefaultQuery("page", "1")  // 带默认值

// 表单 / Body
name := c.PostForm("name")

// 请求头
token := c.GetHeader("Authorization")

// 手动绑定
var req MyReq
c.Bind(&req)       // JSON body（POST/PUT/PATCH）
c.BindQuery(&req)  // Query 参数
c.BindURI(&req)    // 路径参数

// 客户端 IP
ip := c.ClientIP()
```

### 上下文值

```go
c.Set("user_id", 123)      // 写入（gin context）
val, ok := c.Get("user_id") // 读取

// 类型化读取
uid := c.GetString("user_id")
count := c.GetInt("count")
```

### 逃生舱

需要 gin 原生能力时：

```go
gc := c.Gin()  // 获取底层 *gin.Context
```

## Engine 配置

```go
app := qi.New(
    qi.WithAddr(":8080"),              // 监听地址，默认 :8080
    qi.WithMode("release"),            // 运行模式：debug / release / test
    qi.WithOpenAPI(&qi.OpenAPIConfig{
        Title:   "My API",
        Version: "1.0.0",
    }),
    func(cfg *qi.Config) {             // 直接操作 Config
        cfg.ReadTimeout       = 30 * time.Second
        cfg.WriteTimeout      = 30 * time.Second
        cfg.ShutdownTimeout   = 10 * time.Second
    },
)
```

**Config 字段：**

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `Addr` | `:8080` | 监听地址 |
| `Mode` | `debug` | 运行模式 |
| `ReadTimeout` | 10s | 读取超时 |
| `WriteTimeout` | 10s | 写入超时 |
| `IdleTimeout` | 60s | 空闲超时 |
| `ReadHeaderTimeout` | 5s | 读取请求头超时 |
| `MaxHeaderBytes` | 1 MiB | 最大请求头字节数 |
| `ShutdownTimeout` | 5s | 优雅关闭超时 |

## 完整示例

```go
package main

import (
    "log"
    "time"

    "github.com/tokmz/qi"
    "github.com/tokmz/qi/pkg/errors"
)

// 自定义业务错误
var ErrUserNotFound = errors.NewWithStatus(2001, 404, "用户不存在")

type CreateUserReq struct {
    Name  string `json:"name"  binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    app := qi.New(
        qi.WithAddr(":8080"),
        qi.WithMode("release"),
        qi.WithOpenAPI(&qi.OpenAPIConfig{
            Title:   "User API",
            Version: "1.0.0",
        }),
    )

    // 全局中间件
    app.Use(traceMiddleware())

    // 路由注册
    api := app.Group("/api/v1")

    api.API().GET("/users", qi.BindR(listUsers)).Summary("列出用户").Tags("用户").Done()
    api.API().GET("/users/:id", qi.BindR(getUser)).Summary("获取用户").Tags("用户").Done()
    api.API().POST("/users", qi.Bind(createUser)).Summary("创建用户").Tags("用户").Done()

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}

func traceMiddleware() qi.HandlerFunc {
    return func(c *qi.Context) {
        c.Set("trace_id", generateTraceID())
        c.Next()
    }
}

func listUsers(c *qi.Context) (*[]User, error) {
    users := []User{
        {ID: "1", Name: "Alice", Email: "alice@example.com"},
    }
    return &users, nil
}

func getUser(c *qi.Context) (*User, error) {
    id := c.Param("id")
    if id != "1" {
        return nil, ErrUserNotFound
    }
    return &User{ID: id, Name: "Alice", Email: "alice@example.com"}, nil
}

func createUser(c *qi.Context, req *CreateUserReq) (*User, error) {
    return &User{ID: "2", Name: req.Name, Email: req.Email}, nil
}

func generateTraceID() string {
    return time.Now().Format("20060102150405.000")
}
```

## 项目结构

```
qi/
├── engine.go          Engine、Config、服务启动与优雅关闭
├── context.go         Context 封装、绑定、响应方法
├── handler.go         HandlerFunc 类型定义、gin 适配器
├── router.go          RouterGroup、routerStore、路由注册
├── binding.go         Bind/BindR 泛型请求绑定
├── openapi.go         OpenAPIConfig、RouteBuilder、OpenAPI 集成
├── response.go        Response 统一响应结构体
├── errors.go          预定义业务错误
├── pkg/errors/        业务错误类型（可独立使用）
├── internal/openapi/  OpenAPI 3.0.3 文档生成器（反射 + 约束解析）
├── utils/             独立工具包（仅依赖标准库）
│   ├── strings/       字符串操作、大小写转换
│   ├── array/         泛型切片操作
│   ├── convert/       类型转换
│   ├── datetime/      时间格式化、解析
│   ├── pointer/       指针辅助函数
│   └── regexp/        LRU 缓存正则表达式池
└── examples/
    └── basic/         基础示例
```

## 许可证

MIT