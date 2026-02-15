# Qi Middleware

Qi 框架中间件集合。

## 中间件列表

| 中间件 | 文件 | 说明 |
|--------|------|------|
| I18n | i18n.go | 国际化语言识别 |
| RateLimiter | ratelimit.go | 令牌桶限流 |

> 日志中间件内置在 qi 核心包中（`qi.Logger()`），`qi.Default()` 默认启用。

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

## 文件结构

```
middleware/
├── README.md       # 本文档
├── i18n.go         # 国际化中间件
└── ratelimit.go    # 限流中间件
```
