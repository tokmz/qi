# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指导。

## 项目概述

**qi** (`github.com/tokmz/qi`) 是一个 Go HTTP 框架，基于 gin-gonic/gin 构建，提供统一的 Context、响应封装、业务错误系统以及内部 OpenAPI 3.0 文档生成器。注释和文档主要使用中文。

## 构建和测试命令

```bash
# 构建所有包
go build ./...

# 运行所有测试
go test ./...

# 运行单个测试
go test ./pkg/errors/ -run TestErrorCloneSafety -v

# 运行带覆盖率的测试
go test ./... -cover
```

## 架构

```
qi (根包)                ← 公共 API: Engine, Context, Router, HandlerFunc, Response
├── pkg/errors/          ← 公共可复用的业务错误类型 (Code + Message + HTTP status)
├── internal/openapi/    ← 私有 OpenAPI 3.0.3 文档生成器 (基于反射，已接入根包 RouteBuilder)
└── utils/               ← 独立工具包 (无内部依赖)
    ├── strings/         ← 字符串操作、大小写转换、验证
    ├── array/           ← 通用切片操作 (Go 泛型)
    ├── convert/         ← 类型转换 (string↔int↔float↔bool↔bytes, base64, hex)
    ├── datetime/         ← 时间格式化、解析、范围计算、相对时间显示
    ├── pointer/         ← 通用指针辅助函数 (Of, Get, Coalesce)
    └── regexp/          ← LRU 缓存的正则表达式池与预定义模式
```

**依赖流向:** 根包 `qi` → `pkg/errors` + `gin`。`internal/openapi` → `goccy/go-yaml`。`utils/*` → 仅使用标准库。

## 核心设计模式

- **包装器/适配器模式**: `qi.Context` 包装 `gin.Context`；`toGinHandler`/`toGinHandlers` 桥接 `qi.HandlerFunc` 到 `gin.HandlerFunc`。通过 `Context.Gin()` 提供逃生舱。
- **函数式选项 + 结构体配置**: `qi.New(opts ...Option)` 使用 `type Option func(*Config)`；OpenAPI 配置使用 `*OpenAPIConfig` 结构体（`WithOpenAPI(&OpenAPIConfig{...})`），内部 `openapi.New()` 仍使用函数式选项。便捷 Option：`WithAddr()`、`WithMode()`。
- **不可变克隆链**: `errors.Error.WithErr()`、`.WithStatus()`、`.WithMessage()` 返回新实例——`errors.go` 中的哨兵错误不得被修改。
- **统一响应封装**: 所有响应通过 `qi.Response` (`code`, `message`, `data`, `trace_id`) 发送。内部统一走 `Context.respond()` 方法，自动从 `gin.Context.Get("trace_id")` 填充 `trace_id`。公共方法：`OK(data, msg...)`、`Fail(err)`、`FailWithCode()`、`Page()`。
- **泛型 Handler 自动绑定**: `qi.Bind[Req, Resp]` 和 `qi.BindR[Resp]` 通过泛型函数将 handler 签名作为契约，自动完成请求绑定 + OpenAPI 类型推导 + 响应包装。`BoundHandler` 携带 `reflect.Type` 元信息，注册时提取，请求路径上无反射调用。
- **优雅关闭**: `Engine.Run()` 在 goroutine 中启动 HTTP，监听操作系统信号，然后使用可配置的超时调用 `server.Shutdown()`。

## 核心类型

| 类型 | 文件 | 用途 |
|------|------|------|
| `Engine` | `engine.go` | 服务器入口；包装 gin.Engine + http.Server |
| `Context` | `context.go` | 请求上下文；包装 gin.Context，实现 `context.Context` |
| `HandlerFunc` | `handler.go` | `func(*Context)` — 框架处理器签名 |
| `BoundHandler` | `bind.go` | 携带请求/响应 `reflect.Type` 元信息的包装 handler |
| `RouterGroup` | `router.go` | 路由分组，支持前缀和中间件继承 |
| `Response` | `response.go` | 统一 JSON 响应结构体 |
| `OpenAPIConfig` | `openapi.go` | OpenAPI 文档配置（Title/Version/Path/SwaggerUI 等） |
| `errors.Error` | `pkg/errors/error.go` | 业务错误，包含 Code、Message、HTTP 状态和错误链 |

## 约定

- 路由注册立即发生在 `Engine.handle()` 中——路由在注册时同时写入 gin 和内部 `routerStore`，而不是延迟处理。
- `routerStore` 仅用于内省 (`Engine.Routes()` 返回快照)。
- `errors.go` 中的错误使用业务代码 (1000–1103) 映射到 HTTP 状态。
- 测试仅使用标准 `testing` 包（不使用 testify 或其他框架）。
- Go 1.25+，在 `utils/array`、`utils/pointer` 和 `bind.go` 中使用泛型。
- `RouteBuilder` 的 HTTP 方法（GET/POST 等）接受 `handler any`，支持三种类型：`BoundHandler`、`HandlerFunc`、`func(*Context)`。
- `Bind`/`BindR` 的类型推导优先级：显式 `.Body()`/`.Query()` > `.Request()` > `boundRequest`（Bind 推导）；显式 `.Response()` > `boundResponse`（Bind 推导）。
- `typeHasTag` 在注册时预计算 URI tag 存在性，闭包捕获结果，请求路径上无反射开销。