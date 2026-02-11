# Qi - 基于 Gin 的 Go Web 框架

Qi 是一个基于 Gin 的轻量级 Web 框架，提供统一的响应格式、自动参数绑定、泛型路由支持和优雅关机功能。

## 特性

- 🚀 **基于 Gin** - 继承 Gin 的高性能和稳定性
- 📦 **统一响应** - 标准化的 JSON 响应格式
- 🔄 **自动绑定** - 根据 Content-Type 和 HTTP 方法自动绑定请求参数
- 🎯 **泛型路由** - 使用 Go 泛型简化路由处理
- 🛡️ **错误处理** - 统一的错误码和 HTTP 状态码映射
- 🔍 **链路追踪** - 内置 TraceID 支持
- ⚙️ **Options 模式** - 灵活的配置方式
- 🛑 **优雅关机** - 支持优雅关机和生命周期回调
- 🔒 **封装设计** - Context 包装器提供清晰的 API 边界
- 🛠️ **内置 Recovery** - 默认启用 panic 恢复机制，防止服务崩溃

## 快速开始

### 基础用法

```go
package main

import "qi"

func main() {
    // 创建 Engine（New() 默认包含 Recovery，Default() 额外添加 Logger）
    engine := qi.Default()
    r := engine.RouterGroup()

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

    r := engine.RouterGroup()
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
r := engine.RouterGroup()

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

`qi.New()` 默认包含 `gin.Recovery()` 中间件，防止 panic 导致服务崩溃。`qi.Default()` 在此基础上额外添加了 `gin.Logger()` 中间件：

```go
// New() - 仅包含 Recovery
engine := qi.New()

// Default() - 包含 Recovery + Logger
engine := qi.Default()
```

## License

MIT
