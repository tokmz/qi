# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指导。

## 项目概述

**qi** (`github.com/tokmz/qi`) 是一个 Go HTTP 框架，基于 gin-gonic/gin 构建，提供统一的 Context、响应封装、业务错误系统、OpenAPI 3.0 文档生成、请求日志、链路追踪、多级缓存、数据库封装等生产级能力。注释和文档主要使用中文。

## 构建和测试命令

```bash
# 构建所有包
go build ./...

# 运行所有测试
go test ./...

# 运行单个测试
go test ./pkg/errors/ -run TestErrorCloneSafety -v

# 运行带竞态检测的测试
go test ./... -race

# 运行带覆盖率的测试
go test ./... -cover
```

## 架构

```
qi (根包)                ← 公共 API: Engine, Context, Router, HandlerFunc, Response
├── engine.go            Engine + Config + 优雅关闭
├── context.go           Context 封装 + 绑定 + 响应
├── handler.go           HandlerFunc + gin 适配器
├── router.go            RouterGroup + routerStore
├── binding.go           Bind/BindR 泛型请求绑定
├── openapi.go           OpenAPIConfig + RouteBuilder
├── tracing.go           TracingConfig 类型别名 + WithTracing option
├── logger.go            LoggerConfig + WithLogger option
├── response.go          Response 统一响应结构体
├── errors.go            预定义业务错误（code 1000–1103）
├── internal/
│   ├── openapi/         私有 OpenAPI 3.0.3 文档生成器（反射 + 约束解析）
│   ├── tracing/         OTel TracerProvider 初始化 + HTTP 追踪中间件（gin.HandlerFunc）
│   └── logging/         请求日志中间件（gin.HandlerFunc，fmt.Fprintf 直接输出）
├── pkg/
│   ├── errors/          业务错误类型（Code + Message + HTTP Status，不可变克隆链）
│   ├── logger/          zap 日志封装（多输出、采样、日志轮转）
│   ├── config/          viper 配置管理（多格式、热重载、远程配置）
│   ├── database/        GORM 封装（读写分离、连接池、zap 日志、OTel 追踪）
│   ├── cache/           多级缓存（内存 LRU + Redis、防穿透/击穿/雪崩、分布式锁）
│   └── middleware/      i18n 中间件（Accept-Language 协商）
├── utils/
│   ├── strings/         字符串操作、大小写转换、验证
│   ├── array/           通用切片操作（Go 泛型）
│   ├── convert/         类型转换
│   ├── datetime/        时间格式化、解析、范围计算
│   ├── pointer/         通用指针辅助函数
│   └── regexp/          LRU 缓存正则表达式池
└── examples/
    └── basic/           完整示例（日志、追踪、OpenAPI、鉴权、分页、错误）
```

**依赖流向:**
- 根包 `qi` → `internal/tracing` + `internal/logging` + `internal/openapi` + `pkg/errors` + `gin`
- `internal/*` → 只依赖第三方库，不依赖根包（避免循环依赖）
- `pkg/*` → 各自独立，不互相依赖，不依赖根包
- `utils/*` → 仅使用标准库

## 核心设计模式

- **包装器/适配器模式**: `qi.Context` 包装 `gin.Context`；`toGinHandler`/`toGinHandlers` 桥接 `qi.HandlerFunc` 到 `gin.HandlerFunc`。通过 `Context.Gin()` 提供逃生舱。
- **函数式选项 + 结构体配置**: `qi.New(opts ...Option)` 使用 `type Option func(*Config)`；各功能配置使用 `*XxxConfig` 结构体（`WithXxx(&XxxConfig{...})`）。
- **internal 包类型别名重导出**: `internal/tracing.Config` 通过根包 `tracing.go` 的类型别名 `type TracingConfig = itrace.Config` 重导出，使用户可用 `qi.TracingConfig` 而不依赖 internal 包。与 `OpenAPIConfig` 模式一致。
- **不可变克隆链**: `errors.Error.WithErr()`、`.WithStatus()`、`.WithMessage()` 返回新实例——`errors.go` 中的哨兵错误不得被修改。
- **统一响应封装**: 所有响应通过 `qi.Response` (`code`, `message`, `data`, `trace_id`) 发送。内部统一走 `Context.respond()` 方法，自动从 `gin.Context.Get("trace_id")` 填充 `trace_id`。
- **泛型 Handler 自动绑定**: `qi.Bind[Req, Resp]` 和 `qi.BindR[Resp]` 通过泛型函数将 handler 签名作为契约，自动完成请求绑定 + OpenAPI 类型推导 + 响应包装。请求路径上无反射调用。
- **Engine 自动注册中间件**: `WithLogger`/`WithTracing` 配置后，`New()` 自动注册对应 gin 中间件，顺序为：Recovery → Logger → Tracing。
- **优雅关闭**: `Engine.Run()` 监听操作系统信号，先调用 `tracingShutdown`（flush span）再调用 `server.Shutdown()`。

## 核心类型

| 类型 | 文件 | 用途 |
|------|------|------|
| `Engine` | `engine.go` | 服务器入口；包装 gin.Engine + http.Server |
| `Config` | `engine.go` | Engine 配置，含 openAPIConfig/tracingConfig/loggerConfig |
| `Context` | `context.go` | 请求上下文；包装 gin.Context，实现 `context.Context` |
| `HandlerFunc` | `handler.go` | `func(*Context)` — 框架处理器签名 |
| `BoundHandler` | `binding.go` | 携带请求/响应 `reflect.Type` 元信息的包装 handler |
| `RouterGroup` | `router.go` | 路由分组，支持前缀和中间件继承 |
| `Response` | `response.go` | 统一 JSON 响应结构体 |
| `OpenAPIConfig` | `openapi.go` | OpenAPI 文档配置 |
| `TracingConfig` | `tracing.go` | `= internal/tracing.Config` 类型别名 |
| `LoggerConfig` | `logger.go` | 请求日志配置（Output io.Writer + SkipPaths） |
| `errors.Error` | `pkg/errors/error.go` | 业务错误，包含 Code、Message、HTTP 状态和错误链 |

## 约定

- 路由注册立即发生在 `Engine.handle()` 中——路由在注册时同时写入 gin 和内部 `routerStore`，而不是延迟处理。
- `routerStore` 仅用于内省 (`Engine.Routes()` 返回快照)。
- `errors.go` 中的错误使用业务代码 (1000–1103) 映射到 HTTP 状态。
- 测试仅使用标准 `testing` 包（不使用 testify 或其他框架）。
- Go 1.25+，在 `utils/array`、`utils/pointer` 和 `binding.go` 中使用泛型。
- `RouteBuilder` 的 HTTP 方法（GET/POST 等）接受 `handler any`，支持三种类型：`BoundHandler`、`HandlerFunc`、`func(*Context)`。
- `internal/logging` 中间件直接用 `fmt.Fprintf` 写 `io.Writer`（默认 stdout），不依赖 zap，避免与 zap 格式重叠。
- `internal/tracing` 中间件返回 `gin.HandlerFunc`（非 `qi.HandlerFunc`），避免与根包循环依赖。
- OTLP HTTP exporter 根据 `Endpoint` 前缀自动判断 TLS：`http://` → 明文，`https://` → TLS。
- `pkg/cache` 的 `Flush` 在 Redis 驱动下要求配置 `KeyPrefix`，否则拒绝执行（防止误清整个 DB）。
- `pkg/cache` 测试需要本地 Redis（127.0.0.1:6379），使用 `t.Skipf` 在连接失败时跳过。
- `Bind`/`BindR` 的类型推导优先级：显式 `.Body()`/`.Query()` > `.Request()` > `boundRequest`（Bind 推导）；显式 `.Response()` > `boundResponse`（Bind 推导）。
