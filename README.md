# Qi - 基于 Gin 的 Go Web 框架

Qi 是一个基于 Gin 的轻量级 Web 框架，提供统一的响应格式、自动参数绑定、泛型路由支持和优雅关机功能。

## 特性

- 🚀 **基于 Gin** - 继承 Gin 的高性能和稳定性
- 📦 **统一响应** - 标准化的 JSON 响应格式
- 🔄 **自动绑定** - 根据 Content-Type 和 HTTP 方法自动绑定请求参数
- 🎯 **泛型路由** - 使用 Go 泛型简化路由处理
- 🛡️ **错误处理** - 统一的错误码和 HTTP 状态码映射
- 🔍 **链路追踪** - 内置 TraceID 支持，OpenTelemetry 集成
- ⚙️ **Options 模式** - 灵活的配置方式
- 🛑 **优雅关机** - 支持优雅关机和生命周期回调
- 🔒 **封装设计** - Context 包装器提供清晰的 API 边界
- 🛠️ **内置 Recovery** - 默认启用 panic 恢复机制，防止服务崩溃
- 🌍 **国际化** - 内置 i18n 支持，JSON 翻译文件、变量替换、复数形式、懒加载
- 🔧 **丰富中间件** - CORS、限流、Gzip 压缩、超时控制、链路追踪

## 安装

```bash
go get github.com/tokmz/qi@latest
```

## 快速开始

### 基础用法

```go
package main

import "github.com/tokmz/qi"

func main() {
    // 创建 Engine（New() 默认包含 Recovery，Default() 额外添加 Logger）
    engine := qi.Default()
    r := engine.Router()

    // 基础路由
    r.GET("/ping", func(c *qi.Context) {
        c.Success("pong")
    })

    // 手动绑定参数（绑定失败时自动响应错误）
    r.POST("/user", func(c *qi.Context) {
        var req CreateUserReq
        if err := c.BindJSON(&req); err != nil {
            return  // 绑定失败已自动响应错误，直接 return 即可
        }
        c.Success(&UserResp{ID: 1, Name: req.Name})
    })

    // 启动服务器（支持优雅关机）
    engine.Run(":8080")
}
```

### 使用 Options 配置

```go
import (
    "time"
    "qi"
    "github.com/gin-gonic/gin"
)

func main() {
    // 使用 Options 模式配置
    engine := qi.New(
        qi.WithMode(gin.ReleaseMode),
        qi.WithAddr(":8080"),
        qi.WithReadTimeout(15 * time.Second),
        qi.WithWriteTimeout(15 * time.Second),
        qi.WithShutdownTimeout(30 * time.Second),
        qi.WithBeforeShutdown(func() {
            log.Println("清理资源...")
        }),
        qi.WithAfterShutdown(func() {
            log.Println("关机完成")
        }),
        qi.WithTrustedProxies("127.0.0.1"),
    )

    r := engine.Router()
    r.GET("/ping", func(c *qi.Context) {
        c.Success("pong")
    })

    // 启动服务器
    if err := engine.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 高级泛型路由

```go
// 有请求有响应
qi.Handle[CreateUserReq, UserResp](r.POST, "/user",
    func(c *qi.Context, req *CreateUserReq) (*UserResp, error) {
        // 自动绑定 req，自动处理响应
        return &UserResp{ID: 1, Name: req.Name}, nil
    })

// 有请求无响应
qi.Handle0[DeleteUserReq](r.DELETE, "/user/:id",
    func(c *qi.Context, req *DeleteUserReq) error {
        // 自动绑定 URI 参数
        return deleteUser(req.ID)
    })

// 无请求有响应
qi.HandleOnly[InfoResp](r.GET, "/info",
    func(c *qi.Context) (*InfoResp, error) {
        return &InfoResp{Version: "1.0.0"}, nil
    })

// 泛型路由支持中间件（单个或多个）
qi.Handle[CreateUserReq, UserResp](r.POST, "/admin/user",
    createUserHandler,
    authMiddleware,      // 第一个中间件
    adminMiddleware,     // 第二个中间件
)
```

### 路由组和中间件

```go
engine := qi.Default()
r := engine.Router()

// 定义中间件
func traceMiddleware(c *qi.Context) {
    traceID := c.GetHeader("X-Trace-ID")
    if traceID == "" {
        traceID = generateTraceID()
    }
    qi.SetContextTraceID(c, traceID)
    c.Header("X-Trace-ID", traceID)
    c.Next()
}

// 全局中间件
engine.Use(traceMiddleware)

// 路由组中间件
v1 := r.Group("/api/v1")
v1.Use(authMiddleware)

qi.Handle[LoginReq, TokenResp](v1.POST, "/login", loginHandler)

// 单个路由使用中间件（不需要路由组）
qi.Handle[CreateUserReq, UserResp](
    r.POST,
    "/admin/user",
    createUserHandler,
    authMiddleware,      // 认证中间件
    adminMiddleware,     // 管理员中间件
)

// 基础路由也支持中间件
r.GET("/admin/dashboard", dashboardHandler, authMiddleware, adminMiddleware)

// 中间件执行顺序
v1 := r.Group("/api/v1")
v1.Use(middleware1)  // 第一个执行

qi.Handle[Req, Resp](
    v1.POST,
    "/user",
    handler,
    middleware2,  // 第二个执行
    middleware3,  // 第三个执行
)
// handler 最后执行
```

## 配置选项

### 可用的 Options

```go
// 服务器配置
qi.WithMode(gin.ReleaseMode)           // 运行模式
qi.WithAddr(":8080")                   // 监听地址
qi.WithReadTimeout(10 * time.Second)   // 读取超时
qi.WithWriteTimeout(10 * time.Second)  // 写入超时
qi.WithIdleTimeout(60 * time.Second)   // 空闲超时
qi.WithMaxHeaderBytes(1 << 20)         // 最大请求头（1MB）

// 关机配置
qi.WithShutdownTimeout(10 * time.Second)  // 关机超时
qi.WithBeforeShutdown(func() {})          // 关机前回调
qi.WithAfterShutdown(func() {})           // 关机后回调

// 其他配置
qi.WithTrustedProxies("127.0.0.1")        // 信任的代理
qi.WithMaxMultipartMemory(32 << 20)       // Multipart 内存（32MB）
qi.WithI18n(&i18n.Config{...})            // 国际化配置（nil 不启用）
```

### 默认配置

```go
Mode:               gin.DebugMode
Addr:               ":8080"
ReadTimeout:        10s
WriteTimeout:       10s
IdleTimeout:        60s
MaxHeaderBytes:     1MB
ShutdownTimeout:    10s
MaxMultipartMemory: 32MB
```

## 自动绑定策略

Qi 会根据 HTTP 方法和 Content-Type 自动选择绑定策略：

- **GET/DELETE** → `ShouldBindQuery` + `ShouldBindUri`
- **POST/PUT/PATCH** → `ShouldBind`（根据 Content-Type 自动选择）+ `ShouldBindUri`
  - `application/json` → JSON
  - `application/xml` → XML
  - `application/x-www-form-urlencoded` → Form
  - `multipart/form-data` → Multipart Form
- **其他方法** → `ShouldBind`（自动检测）

### 绑定方法

所有绑定方法在失败时会**自动响应错误**，用户只需判断 `err != nil` 并 `return`：

```go
// BindJSON - 绑定 JSON 请求体
if err := c.BindJSON(&req); err != nil {
    return  // 已自动响应 400 错误
}

// BindQuery - 绑定 URL 查询参数
if err := c.BindQuery(&req); err != nil {
    return  // 已自动响应 400 错误
}

// BindURI - 绑定路径参数
if err := c.BindURI(&req); err != nil {
    return  // 已自动响应 400 错误
}

// BindHeader - 绑定请求头
if err := c.BindHeader(&req); err != nil {
    return  // 已自动响应 400 错误
}

// Bind - 根据 Content-Type 自动选择
if err := c.Bind(&req); err != nil {
    return  // 已自动响应 400 错误
}
```

### 示例

```go
// JSON 请求
type CreateUserReq struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

// Form 请求
type LoginReq struct {
    Username string `form:"username" binding:"required"`
    Password string `form:"password" binding:"required"`
}

// 文件上传
type UploadReq struct {
    File *multipart.FileHeader `form:"file" binding:"required"`
}

// URI 参数
type GetUserReq struct {
    ID int64 `uri:"id" binding:"required,min=1"`
}
```

## 响应格式

### 标准响应

```json
{
  "code": 200,
  "data": {...},
  "message": "success",
  "trace_id": "xxx"
}
```

### 响应方法

```go
// 成功响应
c.Success(data)
c.SuccessWithMessage(data, "操作成功")
c.Nil()  // 无数据响应

// 失败响应
c.Fail(code, message)
c.RespondError(err)

// 分页响应
c.Page(users, 100)
```

### 分页响应

```go
// 方式 1：使用 Context.Page（推荐）
r.GET("/users", func(c *qi.Context) {
    users := []User{...}
    c.Page(users, 100)
})

// 方式 2：使用 NewPageResp
r.GET("/users", func(c *qi.Context) {
    users := []User{...}
    resp := qi.NewPageResp(users, 100)
    c.Success(resp)
})

// 方式 3：使用 PageData
r.GET("/users", func(c *qi.Context) {
    users := []User{...}
    resp := qi.PageData(users, 100)
    c.JSON(200, resp)
})
```

响应格式：
```json
{
  "code": 200,
  "data": {
    "list": [...],
    "total": 100
  },
  "message": "success"
}
```

## 错误处理

```go
import "qi/pkg/errors"

// 使用预定义错误
return nil, errors.ErrBadRequest.WithMessage("用户名不能为空")

// 自定义错误
return nil, errors.New(2001, 403, "禁止访问", nil)
```

### 内置错误码

- `ErrServer` - 服务器错误 (1000, HTTP 500)
- `ErrBadRequest` - 请求错误 (1001, HTTP 400)
- `ErrUnauthorized` - 未授权 (1002, HTTP 401)
- `ErrForbidden` - 禁止访问 (1003, HTTP 403)
- `ErrNotFound` - 资源不存在 (1004, HTTP 404)

## 优雅关机

Qi 内置优雅关机支持，自动监听 `SIGINT` 和 `SIGTERM` 信号。

```go
engine := qi.New(
    qi.WithShutdownTimeout(30 * time.Second),
    qi.WithBeforeShutdown(func() {
        log.Println("关闭数据库连接...")
        db.Close()
    }),
    qi.WithAfterShutdown(func() {
        log.Println("清理完成")
    }),
)

// Run 会阻塞直到收到关机信号
if err := engine.Run(":8080"); err != nil {
    log.Fatal(err)
}
```

### 手动关机

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := engine.Shutdown(ctx); err != nil {
    log.Printf("关机失败: %v", err)
}
```

## 上下文辅助方法

```go
// TraceID
qi.SetContextTraceID(c, "trace-123")
traceID := qi.GetContextTraceID(c)

// 用户 UID
qi.SetContextUid(c, 12345)
uid := qi.GetContextUid(c)

// 语言
qi.SetContextLanguage(c, "zh-CN")
lang := qi.GetContextLanguage(c)
```

## 静态文件服务

```go
r.Static("/static", "./public")
r.StaticFile("/favicon.ico", "./public/favicon.ico")
```

## HTTPS 支持

```go
// 启动 HTTPS 服务器
if err := engine.RunTLS(":443", "cert.pem", "key.pem"); err != nil {
    log.Fatal(err)
}
```

## 国际化 (i18n)

Qi 内置国际化支持，通过 `WithI18n` 配置即可启用。框架自动初始化翻译器并注册语言检测中间件，Context 上直接调用 `T()`/`Tn()`。

### 创建翻译文件

```
locales/
├── zh-CN.json
└── en-US.json
```

```json
// locales/zh-CN.json
{
    "hello": "你好 {{.Name}}",
    "user": {
        "login": "登录",
        "logout": "退出登录"
    }
}
```

### 启用 i18n

```go
import "github.com/tokmz/qi/pkg/i18n"

engine := qi.New(
    qi.WithI18n(&i18n.Config{
        Dir:             "./locales",
        DefaultLanguage: "zh-CN",
        Languages:       []string{"zh-CN", "en-US"},
    }),
)
```

框架会自动：
1. 初始化翻译器
2. 注册语言检测中间件（优先级：`Query(lang)` > `X-Language` Header > `Accept-Language` Header > 默认语言）

### 在路由中使用翻译

```go
r.GET("/hello", func(c *qi.Context) {
    msg := c.T("hello", "Name", "Alice")
    c.Success(msg)
})

// 泛型路由
qi.Handle[HelloReq, HelloResp](r.POST, "/hello",
    func(c *qi.Context, req *HelloReq) (*HelloResp, error) {
        msg := c.T("hello", "Name", req.Name)
        return &HelloResp{Message: msg}, nil
    })
```

### 复数形式

```go
// 翻译文件: {"item_one": "{{.Count}} item", "item_other": "{{.Count}} items"}
c.Tn("item_one", "item_other", 1)  // "1 item"
c.Tn("item_one", "item_other", 5)  // "5 items"
```

### 获取翻译器实例

如需直接操作翻译器（如预加载、检查 key 是否存在），可通过 `engine.Translator()` 获取：

```go
t := engine.Translator()
t.Preload("ja-JP")
t.HasKey("hello")
```

### 语言回退

当请求的语言中找不到翻译键时，自动回退到默认语言。如果默认语言也找不到，返回 key 本身。

## 中间件

Qi 提供丰富的内置中间件，分为核心中间件和扩展中间件。

### 核心中间件（qi 包内置）

- **Recovery** - panic 恢复，`qi.New()` 默认启用
- **Logger** - 请求日志，`qi.Default()` 默认启用

### 扩展中间件（middleware 包）

```go
import "qi/middleware"
```

| 中间件 | 说明 |
|--------|------|
| `middleware.Tracing()` | OpenTelemetry 链路追踪 |
| `middleware.CORS()` | 跨域资源共享 |
| `middleware.RateLimiter()` | 令牌桶限流 |
| `middleware.Timeout()` | 请求超时控制 |
| `middleware.Gzip()` | 响应压缩 |

> **注意：** i18n 中间件已内置到框架中，通过 `qi.WithI18n()` 配置即可自动注册，无需手动添加。

### 推荐注册顺序

```go
e := qi.Default() // 内置 Recovery + Logger

// 1. 链路追踪（最先，创建根 Span + 生成 TraceID）
e.Use(middleware.Tracing())
// 2. CORS（在业务逻辑之前处理跨域预检）
e.Use(middleware.CORS())
// 3. 限流（在业务处理之前拦截超限请求）
e.Use(middleware.RateLimiter())
// 4. 超时控制
e.Use(middleware.Timeout())
// 5. Gzip 压缩
e.Use(middleware.Gzip())
// i18n 中间件通过 WithI18n 配置自动注册，无需手动添加
```

详细配置请参考 [middleware/README.md](middleware/README.md)。

## 注意事项

### Gin Mode 全局状态

`gin.SetMode()` 是全局操作，建议在程序启动时只创建一个 Engine 实例：

```go
// ✅ 推荐：单例模式
func main() {
    engine := qi.New(qi.WithMode(gin.ReleaseMode))
    setupRoutes(engine)
    engine.Run(":8080")
}

// ❌ 避免：同一进程多个 Engine
func main() {
    engine1 := qi.New(qi.WithMode(gin.ReleaseMode))
    engine2 := qi.New(qi.WithMode(gin.DebugMode))  // 可能影响 engine1
}
```

### Context 包装器

Qi 使用私有字段封装 `gin.Context`，提供清晰的 API 边界。如果需要在测试中创建 `qi.Context` 实例，请使用公开的构造函数：

```go
// ✅ 测试中创建 Context
import (
    "testing"
    "github.com/gin-gonic/gin"
    "qi"
)

func TestHandler(t *testing.T) {
    ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
    c := qi.NewContext(ginCtx)  // 使用公开的构造函数
    // 进行测试...
}

// ❌ 避免：直接构造（编译错误）
c := &qi.Context{ctx: ginCtx}  // ctx 是私有字段，无法访问
```

### Recovery 中间件

`qi.New()` 默认包含 Recovery 中间件（使用 qi 统一响应格式），防止 panic 导致服务崩溃。`qi.Default()` 在此基础上额外添加了 Logger 中间件：

```go
// New() - 仅包含 Recovery
engine := qi.New()

// Default() - 包含 Recovery + Logger
engine := qi.Default()
```

## API 参考

### Engine API

#### 创建 Engine

```go
// New 创建一个新的 Engine 实例（包含 Recovery 中间件）
func New(opts ...Option) *Engine

// Default 创建带有 Logger + Recovery 中间件的 Engine
func Default(opts ...Option) *Engine
```

#### Engine 方法

```go
// Use 注册全局中间件
func (e *Engine) Use(middlewares ...HandlerFunc)

// Group 创建路由组
func (e *Engine) Group(path string, middlewares ...HandlerFunc) *RouterGroup

// Router 返回根路由组
func (e *Engine) Router() *RouterGroup

// Translator 返回 i18n 翻译器实例（未启用 i18n 时返回 nil）
func (e *Engine) Translator() i18n.Translator

// Run 启动 HTTP 服务器（支持优雅关机）
func (e *Engine) Run(addr ...string) error

// RunTLS 启动 HTTPS 服务器（支持优雅关机）
func (e *Engine) RunTLS(addr, certFile, keyFile string) error

// Shutdown 手动关闭服务器
func (e *Engine) Shutdown(ctx context.Context) error
```

### RouterGroup API

#### 路由组管理

```go
// Group 创建子路由组
func (rg *RouterGroup) Group(path string, middlewares ...HandlerFunc) *RouterGroup

// Use 注册中间件到路由组
func (rg *RouterGroup) Use(middlewares ...HandlerFunc)
```

#### 基础路由方法

```go
// GET 注册 GET 路由
func (rg *RouterGroup) GET(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// POST 注册 POST 路由
func (rg *RouterGroup) POST(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// PUT 注册 PUT 路由
func (rg *RouterGroup) PUT(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// DELETE 注册 DELETE 路由
func (rg *RouterGroup) DELETE(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// PATCH 注册 PATCH 路由
func (rg *RouterGroup) PATCH(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// HEAD 注册 HEAD 路由
func (rg *RouterGroup) HEAD(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// OPTIONS 注册 OPTIONS 路由
func (rg *RouterGroup) OPTIONS(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// Any 注册所有 HTTP 方法的路由
func (rg *RouterGroup) Any(path string, handler HandlerFunc, middlewares ...HandlerFunc)
```

#### 静态文件服务

```go
// Static 注册静态文件目录服务
func (rg *RouterGroup) Static(relativePath, root string)

// StaticFile 注册单个静态文件服务
func (rg *RouterGroup) StaticFile(relativePath, filepath string)

// StaticFS 注册静态文件系统服务
func (rg *RouterGroup) StaticFS(relativePath string, fs http.FileSystem)
```

#### 泛型路由方法

```go
// Handle 有请求参数，有响应数据（自动绑定 + 自动响应）
le[Req any, Resp any](
    register RouteRegister,
    path string,
    handler func(*Context, *Req) (*Resp, error),
    middlewares ...HandlerFunc,
)

// Handle0 有请求参数，无响应数据（自动绑定 + 自动响应）
func Handle0[Req any](
    register RouteRegister,
    path string,
    handler func(*Context, *Req) error,
    middlewares ...HandlerFunc,
)

// HandleOnly 无请求参数，有响应数据（自动响应）
func HandleOnly[Resp any](
    register RouteRegister,
    path string,
    handler func(*Context) (*Resp, error),
    middlewares ...HandlerFunc,
)
```

### Context API

#### 请求信息获取

```go
// Request 返回底层的 *http.Request
func (c *Context) Request() *http.Request

// Writer 返回底层的 http.ResponseWriter
func (c *Context) Writer() gin.ResponseWriter

// Param 获取路径参数
func (c *Context) Param(key string) string

// FullPath 获取路由模板路径（如 /users/:id）
func (c *Context) FullPath() string

// Query 获取 URL 查询参数
func (c *Context) Query(key string) string

// DefaultQuery 获取 URL 查询参数（带默认值）
func (c *Context) DefaultQuery(key, defaultValue string) string

// GetQuery 获取 URL 查询参数（返回是否存在）
func (c *Context) GetQuery(key string) (string, bool)

// PostForm 获取 POST 表单参数
func (c *Context) PostForm(key string) string
// DefaultPostForm 获取 POST 表单参数（带默认值）
func (c *Context) DefaultPostForm(key, defaultValue string) string

// GetPostForm 获取 POST 表单参数（返回是否存在）
func (c *Context) GetPostForm(key string) (string, bool)

// ClientIP 获取客户端 IP
func (c *Context) ClientIP() string

// ContentType 获取 Content-Type
func (c *Context) ContentType() string

// GetHeader 获取请求头
func (c *Context) GetHeader(key string) string
```

#### 参数绑定方法（自动响应错误）

```go
// Bind 自动绑定并验证请求参数（根据 Content-Type 自动选择）
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) Bind(obj any) error

// BindJSON 绑定 JSON 请求体
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindJSON(obj any) error

// BindQuery 绑定 URL 查询参数
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindQuery(obj any) error

// BindURI 绑定路径参数
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindURI(obj any) error

// BindHeader 绑定请求头
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindHeader(obj any) error
```

#### 参数绑定方法（不自动响应错误）

```go
// ShouldBind 绑定请求参数（不自动响应错误）
func (c *Context) ShouldBind(obj any) error

// ShouldBindJSON 绑定 JSON 请求体（不自动响应错误）
func (c *Context) ShouldBindJSON(obj any) error

// ShouldBindQuery 绑定 URL 查询参数（不自动响应错误）
func (c *Context) ShouldBindQuery(obj any) error

// ShouldBindUri 绑定路径参数（不自动响应错误）
func (c *Context) ShouldBindUri(obj any) error

// ShouldBindHeader 绑定请求头（不自动响应错误）
func (c *Context) ShouldBindHeader(obj any) error
```

#### 响应方法

```go
// Success 成功响应
func (c *Context) Success(data any)

// SuccessWithMessage 成功响应（自定义消息）
func (c *Context) SuccessWithMessage(data any, message string)

// Nil 成功响应（无数据）
func (c *Context) Nil()

// Fail 失败响应
func (c *Context) Fail(code int, message string)

// RespondError 错误响应
func (c *Context) RespondError(err error)

// Page 分页响应
func (c *Context) Page(list any, total uint64)

// JSON 发送 JSON 响应
func (c *Context) JSON(code int, obj any)
```

#### 国际化方法

```go
// T 获取翻译（支持变量替换），未启用 i18n 时返回 key
func (c *Context) T(key string, args ...any) string

// Tn 获取翻译（支持复数形式），未启用 i18n 时返回 key
func (c *Context) Tn(key, plural string, n int, args ...any) string
```

#### 响应头设置

```go
// Header 设置响应头
func (c *Context) Header(key, value string)
```

#### 上下文键值对操作

```go
// Set 设置上下文键值对
func (c *Context) Set(key string, value any)

// Get 获取上下文键值对
func (c *Context) Get(key string) (any, bool)

// GetString 获取字符串类型的上下文值
func (c *Context) GetString(key string) string

// GetInt 获取整数类型的上下文值
func (c *Context) GetInt(key string) int

// GetInt64 获取 int64 类型的上下文值
func (c *Context) GetInt64(key string) int64

// GetUint 获取 uint 类型的上下文值
func (c *Context) GetUint(key string) uint

// GetUint64 获取 uint64 类型的上下文值
func (c *Context) GetUint64(key string) uint64

// GetFloat64 获取 float64 类型的上下文值
func (c *Context) GetFloat64(key string) float64

// GetBool 获取布尔类型的上下文值
func (c *Context) GetBool(key string) bool
```

#### 中间件控制

```go
// Next 执行下一个中间件或处理函数
func (c *Context) Next()

// Abort 中止请求处理
func (c *Context) Abort()

// AbortWithStatus 中止请求并设置状态码
func (c *Context) AbortWithStatus(code int)

// AbortWithStatusJSON 中止请求并返回 JSON
func (c *Context) AbortWithStatusJSON(code int, jsonObj any)

// IsAborted 检查请求是否已中止
func (c *Context) IsAborted() bool
```

#### Context 传递

```go
// RequestContext 返回标准库 context.Context，用于传递给 Service 层
// 自动将 TraceID、UID、Language 注入到 context.Context
func (c *Context) RequestContext() context.Context

// SetRequestContext 更新 Request 的 Context（用于中间件注入 SpanContext）
func (c *Context) SetRequestContext(ctx context.Context)
```

### 配置选项 API

```go
// WithMode 设置运行模式
func WithMode(mode string) Option

// WithAddr 设置监听地址
func WithAddr(addr string) Option

// WithReadTimeout 设置读取超时
func WithReadTimeout(timeout time.Duration) Option

// WithWriteTimeout 设置写入超时
func WithWriteTimeout(timeout time.Duration) Option

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout(timeout time.Duration) Option

// WithMaxHeaderBytes 设置最大请求头字节数
func WithMaxHeaderBytes(size int) Option

// WithShutdownTimeout 设置关机超时时间
func WithShutdownTimeout(timeout time.Duration) Option

// WithBeforeShutdown 设置关机前回调
func WithBeforeShutdown(fn func()) Option

// WithAfterShutdown 设置关机后回调
func WithAfterShutdown(fn func()) Option

// WithTrustedProxies 设置信任的代理
func WithTrustedProxies(proxies ...string) Option

// WithMaxMultipartMemory 设置最大 multipart 内存
func WithMaxMultipartMemory(size int64) Option

// WithI18n 设置国际化配置
func WithI18n(cfg *i18n.Config) Option
```

### 响应结构 API

```go
// Response 统一响应结构
type Response struct {
    Code    int    `json:"code"`             // 业务状态码
    Data    any    `json:"data"`               // 响应数据
    Message string `json:"message"`            // 响应消息
    TraceID string `json:"trace_id,omitempty"` // 追踪ID（可选）
}

// NewResponse 创建响应
func NewResponse(code int, data any, message string) *Response

// WithTraceID 设置追踪ID
func (r *Response) WithTraceID(traceID string) *Response

// Success 创建成功响应
func Success(data any) *Response

// SuccessWithMessage 创建成功响应（自定义消息）
func SuccessWithMessage(data any, message string) *Response

// Fail 创建失败响应
func Fail(code int, message string) *Response

// PageResp 分页响应结构
type PageResp struct {
    List  any    `json:"list"`  // 数据列表
    Total uint64 `json:"total"` // 总数
}

// NewPageResp 创建分页响应
func NewPageResp(list any, total uint64) *PageResp

// PageData 分页数据包装器
func PageData(list any, total uint64) *Response
```

### 上下文辅助函数 API

```go
// GetContextTraceID 获取上下文链路追踪 trace_id
func GetContextTraceID(ctx *Context) string

// SetContextTraceID 设置上下文链路追踪 trace_id
func SetContextTraceID(ctx *Context, traceID string)

// GetContextUid 获取上下文用户 uid
func GetContextUid(ctx *Context) int64

// SetContextUid 设置上下文用户 uid
func SetContextUid(ctx *Context, uid int64)

// GetContextLanguage 获取上下文用户语言
func GetContextLanguage(ctx *Context) string

// SetContextLanguage 设置上下文用户语言
func SetContextLanguage(ctx *Context, language string)

// GetTraceIDFromContext 从标准库 context.Context 获取 TraceID
func GetTraceIDFromContext(ctx context.Context) string

// GetUidFromContext 从标准库 context.Context 获取 UID
func GetUidFromContext(ctx context.Context) int64

// GetLanguageFromContext 从标准库 context.Context 获取 Language
func GetLanguageFromContext(ctx context.Context) string
```

### 上下文常量

```go
const (
    // ContextTraceIDKey 链路追踪 trace_id 键（用于 Gin Context）
    ContextTraceIDKey = "trace_id"

    // ContextUidKey 用户 uid 键（用于 Gin Context）
    ContextUidKey = "uid"

    // ContextLanguageKey 用户语言键（用于 Gin Context）
    ContextLanguageKey = "language"
)
```

### 错误处理 API

详见 `pkg/errors/` 包文档。

```go
// Error 自定义错误类型
type Error struct {
    Code     int    // 业务错误码
    HttpCode int    // HTTP 状态码
    Message  string // 错误消息
    Err      error  // 原始错误
}

// New 创建自定义错误
func New(code int, httpCode int, message string, err error) *Error

// WithMessage 设置错误消息
func (e *Error) WithMessage(message string) *Error

// WithEor 包装原始错误
func (e *Error) WithError(err error) *Error

// 预定义错误
var (
    ErrServer       = New(1000, 500, "服务器错误", nil)
    ErrBadRequest   = New(1001, 400, "请求参数错误", nil)
    ErrUnauthorized = New(1002, 401, "未授权", nil)
    ErrForbidden    = New(1003, 403, "禁止访问", nil)
    ErrNotFound     = New(1004, 404, "资源不存在", nil)
)
```

## License

MIT
