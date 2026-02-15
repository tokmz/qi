# Qi Middleware

Qi 框架中间件集合。

## 中间件列表

| 中间件 | 文件 | 说明 |
|--------|------|------|
| Tracing | tracing.go | OpenTelemetry 链路追踪 |
| CORS | cors.go | 跨域资源共享 |
| RateLimiter | ratelimit.go | 令牌桶限流 |
| Timeout | timeout.go | 请求超时控制 |
| Gzip | gzip.go | 响应压缩 |
| I18n | i18n.go | 国际化语言识别 |

> Logger 和 Recovery 内置在 qi 核心包中，`qi.New()` 默认启用 Recovery，`qi.Default()` 额外启用 Logger。

## 推荐注册顺序

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
// 6. I18n（业务相关）
e.Use(middleware.I18n(translator))
```

Tracing 应放在最前面，因为：
- 创建根 Span，后续所有中间件和业务处理都在 Span 时间范围内
- 设置 TraceID 到 `qi.Context`，Logger 中间件需要用到
- 注入带 Span 的 context，后续中间件可创建子 Span

## Tracing 链路追踪中间件

基于 OpenTelemetry 的链路追踪中间件，自动提取/注入 TraceContext，创建 HTTP Server Span。OTel 自动生成 TraceID 和 SpanID。

### 使用

```go
import "qi/middleware"

// 默认配置
engine.Use(middleware.Tracing())

// 自定义配置
engine.Use(middleware.Tracing(&middleware.TracingConfig{
    TracerName: "my-service",
    ExcludePaths: []string{"/health", "/ping"},
}))

// 自定义 Span 名称
engine.Use(middleware.Tracing(&middleware.TracingConfig{
    SpanNameFormatter: func(c *qi.Context) string {
        return fmt.Sprintf("HTTP %s %s", c.Request().Method, c.FullPath())
    },
}))

// 过滤不需要追踪的请求
engine.Use(middleware.Tracing(&middleware.TracingConfig{
    Filter: func(c *qi.Context) bool {
        return !strings.HasPrefix(c.Request().URL.Path, "/internal")
    },
}))
```

### 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| TracerName | string | `qi.http` | Tracer 名称 |
| SpanNameFormatter | func | `METHOD PATH` | 自定义 Span 名称格式 |
| Filter | func | nil | 过滤函数，返回 true 追踪 |
| ExcludePaths | []string | nil | 排除的路径 |

### Span 属性

自动记录以下属性：
- `http.request.method` - 请求方法
- `http.route` - 路由模板
- `url.path` - 请求路径
- `server.address` - 服务地址
- `user_agent.original` - User-Agent
- `http.client_ip` - 客户端 IP
- `http.response.status_code` - 响应状态码

### 注意事项

- 需要先初始化 OTel Provider（`TracerProvider` + `TextMapPropagator`）
- 每次请求时获取 tracer，避免 Provider 后初始化导致使用 noop
- TraceID 自动同步到 `qi.Context`，可通过 `qi.GetContextTraceID(c)` 获取
- 状态码 >= 500 时 Span 标记为 Error

## CORS 中间件

跨域资源共享中间件，自动处理 OPTIONS 预检请求，支持通配符源匹配。

### 使用

```go
import "qi/middleware"

// 允许所有源（开发环境）
engine.Use(middleware.CORS())

// 指定允许的源
engine.Use(middleware.CORS(&middleware.CORSConfig{
    AllowOrigins:     []string{"https://example.com", "https://*.example.com"},
    AllowCredentials: true,
}))

// 完整配置
engine.Use(middleware.CORS(&middleware.CORSConfig{
    AllowOrigins:     []string{"https://app.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"X-Total-Count"},
    AllowCredentials: true,
    MaxAge:           24 * time.Hour,
}))
```

### 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| AllowOrigins | []string | `["*"]` | 允许的源，支持通配符 |
| AllowMethods | []string | GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS | 允许的方法 |
| AllowHeaders | []string | Origin, Content-Type, Accept, Authorization, X-Requested-With | 允许的请求头 |
| ExposeHeaders | []string | nil | 允许前端访问的响应头 |
| AllowCredentials | bool | `false` | 是否允许携带凭证 |
| MaxAge | time.Duration | `12h` | 预检请求缓存时间 |

> 注意：`AllowCredentials` 为 `true` 时，`AllowOrigins` 不能为 `["*"]`，需指定具体源。

## I18n 中间件

从请求中识别语言（Query > Cookie > Accept-Language），设置到 `qi.Context` 和 `request context` 中。

### 使用

```go
import (
    "qi/middleware"
    "qi/pkg/i18n"
)

trans, _ := i18n.NewWithOptions(
    i18n.WithDir("./locales"),
    i18n.WithDefaultLanguage("zh-CN"),
    i18n.WithLanguages("zh-CN", "en-US"),
)

// 默认配置
engine.Use(middleware.I18n(trans))

// 自定义配置
engine.Use(middleware.I18n(trans, &middleware.I18nConfig{
    QueryKey:     "lang",
    CookieKey:    "language",
    HeaderKey:    "Accept-Language",
    SetCookie:    true,
    CookieMaxAge: 86400 * 30,
}))
```

### 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| QueryKey | string | `lang` | URL 查询参数名 |
| CookieKey | string | `language` | Cookie 名 |
| HeaderKey | string | `Accept-Language` | Header 名 |
| SetCookie | bool | `false` | 是否将语言写入 Cookie |
| CookieMaxAge | int | `2592000` (30天) | Cookie 过期时间（秒） |

### 语言识别优先级

1. URL 查询参数（`?lang=en-US`）
2. Cookie（`language=en-US`）
3. Accept-Language Header（`zh-CN,zh;q=0.9,en;q=0.8`）
4. 翻译器默认语言

### 在路由中获取翻译

```go
r.GET("/hello", func(c *qi.Context) {
    // 通过 request context 获取当前语言的翻译
    msg := trans.T(c.RequestContext(), "hello", "Name", "Alice")
    c.Success(msg)
})
```

## RateLimiter 限流中间件

基于令牌桶算法，按 key（默认客户端 IP）进行限流。支持突发流量、自定义 key、过期桶自动清理。

### 使用

```go
import "qi/middleware"

// 默认配置（100 req/s）
engine.Use(middleware.RateLimiter())

// 自定义配置
engine.Use(middleware.RateLimiter(&middleware.RateLimiterConfig{
    RequestsPerSecond: 50,
    Burst:             100,
}))

// 按用户 ID 限流
engine.Use(middleware.RateLimiter(&middleware.RateLimiterConfig{
    RequestsPerSecond: 10,
    KeyFunc: func(c *qi.Context) string {
        return fmt.Sprintf("user:%d", qi.GetContextUid(c))
    },
}))

// 排除健康检查路径
engine.Use(middleware.RateLimiter(&middleware.RateLimiterConfig{
    ExcludePaths: []string{"/health", "/ping"},
}))
```

### 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| RequestsPerSecond | float64 | `100` | 每秒允许的请求数 |
| Burst | int | `100` | 突发容量 |
| KeyFunc | func | 客户端 IP | 自定义限流 key |
| SkipFunc | func | nil | 跳过限流的判断函数 |
| ExcludePaths | []string | nil | 排除的路径 |
| Logger | logger.Logger | Development | 日志实例 |
| CleanupInterval | time.Duration | `10m` | 过期桶清理间隔 |
| BucketExpiry | time.Duration | `30m` | 桶过期时间 |

### 限流响应

超限时返回 HTTP 429：

```json
{
    "code": 429,
    "message": "too many requests"
}
```

## Timeout 超时中间件

通过 `context.WithTimeout` 控制请求处理超时，防止慢请求长时间占用资源。

### 使用

```go
import "qi/middleware"

// 默认配置（30 秒超时）
engine.Use(middleware.Timeout())

// 自定义超时时间
engine.Use(middleware.Timeout(&middleware.TimeoutConfig{
    Timeout: 10 * time.Second,
}))

// 排除长耗时路径（如文件上传）
engine.Use(middleware.Timeout(&middleware.TimeoutConfig{
    Timeout:      30 * time.Second,
    ExcludePaths: []string{"/upload", "/export"},
}))
```

### 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Timeout | time.Duration | `30s` | 请求超时时间 |
| TimeoutMessage | string | `request timeout` | 超时响应消息 |
| SkipFunc | func | nil | 跳过超时控制的判断函数 |
| ExcludePaths | []string | nil | 排除的路径 |

### 超时响应

超时时返回 HTTP 408：

```json
{
    "code": 408,
    "message": "request timeout"
}
```

## Gzip 压缩中间件

对支持 gzip 的客户端自动压缩响应，减少传输体积。使用 `sync.Pool` 复用 gzip.Writer，支持最小长度阈值。

### 使用

```go
import "qi/middleware"

// 默认配置
engine.Use(middleware.Gzip())

// 自定义压缩级别
engine.Use(middleware.Gzip(&middleware.GzipConfig{
    Level: gzip.BestSpeed,
}))

// 排除静态资源和图片
engine.Use(middleware.Gzip(&middleware.GzipConfig{
    ExcludeExtensions: []string{".png", ".jpg", ".gif", ".webp"},
    ExcludePaths:      []string{"/static"},
}))

// 调整最小压缩长度
engine.Use(middleware.Gzip(&middleware.GzipConfig{
    MinLength: 1024, // 小于 1KB 不压缩
}))
```

### 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Level | int | `gzip.DefaultCompression` | 压缩级别 |
| MinLength | int | `256` | 最小压缩长度（字节） |
| ExcludePaths | []string | nil | 排除的路径 |
| ExcludeExtensions | []string | nil | 排除的文件扩展名 |

## 文件结构

```
middleware/
├── README.md       # 本文档
├── tracing.go      # OpenTelemetry 链路追踪中间件
├── cors.go         # CORS 跨域中间件
├── timeout.go      # 请求超时中间件
├── gzip.go         # Gzip 压缩中间件
├── i18n.go         # 国际化中间件
└── ratelimit.go    # 限流中间件
```
