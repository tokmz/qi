# Request Package

HTTP 客户端封装，提供链式调用、泛型响应解析、重试、拦截器、OpenTelemetry 追踪等能力。

## 功能特性

- 链式 API 构建请求（Header、Query、Body、Auth、Timeout）
- 泛型响应解析 `Do[T]()` / `DoList[T]()`
- 指数退避重试（带抖动），支持自定义重试条件
- 拦截器链（BeforeRequest / AfterResponse）
- OpenTelemetry 追踪（Client Span + Header 传播）
- 文件上传（multipart/form-data）
- 可插拔日志接口，默认不记录
- 零外部日志依赖（不依赖 zap / logrus 等）

## 快速开始

```go
import "github.com/tokmz/qi/pkg/request"

client := request.New(
    request.WithBaseURL("https://api.example.com"),
    request.WithTimeout(10 * time.Second),
)

// GET
resp, err := client.Get("/users").
    SetQuery("page", "1").
    SetBearerToken("xxx").
    Do()

// POST + 泛型解析
user, err := request.Do[User](
    client.Post("/users").SetBody(&CreateUserReq{Name: "test"}),
)

// 列表解析
users, err := request.DoList[User](client.Get("/users"))
```

## 配置选项

```go
client := request.New(
    request.WithBaseURL("https://api.example.com"),
    request.WithTimeout(10 * time.Second),
    request.WithHeader("X-App", "myapp/1.0"),
    request.WithHeaders(map[string]string{"X-A": "1", "X-B": "2"}),
    request.WithMaxIdleConns(100),
    request.WithMaxIdleConnsPerHost(10),
    request.WithMaxConnsPerHost(100),
    request.WithIdleConnTimeout(90 * time.Second),
    request.WithInsecureSkipVerify(false),
    request.WithTransport(customTransport),
)
```

## 重试

```go
client := request.New(
    request.WithBaseURL("https://api.example.com"),
    request.WithRetry(&request.RetryConfig{
        MaxAttempts:  3,                      // 最大重试次数
        InitialDelay: 100 * time.Millisecond, // 初始退避
        MaxDelay:     5 * time.Second,        // 最大退避
        Multiplier:   2.0,                    // 退避倍数
        RetryIf: func(resp *http.Response, err error) bool {
            // 自定义重试条件（默认：网络错误或 5xx）
            return err != nil || resp.StatusCode >= 500
        },
    }),
)

// 也可在请求级覆盖
resp, err := client.Get("/flaky").
    SetRetry(&request.RetryConfig{MaxAttempts: 5}).
    Do()
```

重试使用指数退避 + ±25% 抖动。`SetBody` 设置的 JSON body 支持重试重放，`SetRawBody` 的 `io.Reader` 不支持。

## 拦截器

```go
// 自定义拦截器
type Interceptor interface {
    BeforeRequest(ctx context.Context, req *http.Request) error
    AfterResponse(ctx context.Context, resp *request.Response) error
}

// 内置：日志拦截器（需传入 Logger 实现）
client := request.New(
    request.WithInterceptor(request.NewLoggingInterceptor(myLogger)),
)

// 内置：认证拦截器（动态 token）
client := request.New(
    request.WithInterceptor(request.NewAuthInterceptor(func() string {
        return getTokenFromCache()
    })),
)
```

## 日志

包定义了最小日志接口，默认不记录任何日志：

```go
type Logger interface {
    InfoContext(ctx context.Context, msg string, keysAndValues ...any)
    ErrorContext(ctx context.Context, msg string, keysAndValues ...any)
}
```

传入实现即可启用：

```go
client := request.New(
    request.WithLogger(myLogger),
)
```

日志在请求失败和读取响应体失败时自动记录。

## OpenTelemetry 追踪

```go
client := request.New(
    request.WithTracing(true),
)
```

启用后：
- 每次请求创建 `SpanKindClient` Span（名称 `HTTP GET` 等）
- 自动注入 W3C TraceContext 到请求头
- Span 记录 method、url、status_code 属性
- 错误时设置 Span 状态为 Error

## 文件上传

```go
resp, err := client.Post("/upload").
    SetFile("avatar", "/path/to/file.png").
    SetFormData(map[string]string{"name": "test"}).
    Do()

// 多文件
resp, err := client.Post("/upload").
    SetFiles(map[string]string{
        "file1": "/path/to/a.png",
        "file2": "/path/to/b.png",
    }).
    Do()
```

## 认证

```go
// Bearer Token
resp, err := client.Get("/secure").SetBearerToken("mytoken").Do()

// Basic Auth
resp, err := client.Get("/auth").SetBasicAuth("admin", "secret").Do()
```

## 表单数据

```go
resp, err := client.Post("/form").
    SetFormData(map[string]string{
        "name": "test",
        "age":  "18",
    }).
    Do()
```

## 通用请求构建

```go
resp, err := client.R(ctx).
    SetMethod(http.MethodPost).
    SetURL("/custom").
    SetBody(payload).
    Do()
```

## 响应处理

```go
resp, err := client.Get("/data").Do()

resp.StatusCode    // HTTP 状态码
resp.Headers       // http.Header
resp.Body          // []byte
resp.Duration      // 请求耗时
resp.IsSuccess()   // 2xx
resp.IsError()     // 4xx/5xx
resp.String()      // Body 字符串
resp.Unmarshal(&v) // JSON 反序列化
```

## 错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| 4001 | 500 | 请求失败 |
| 4002 | 504 | 请求超时 |
| 4003 | 500 | 序列化失败 |
| 4004 | 500 | 反序列化失败 |
| 4005 | 502 | 重试次数已用尽 |
| 4006 | 400 | 无效的 URL |

```go
import "github.com/tokmz/qi/pkg/errors"

if errors.Is(err, request.ErrTimeout) {
    // 处理超时
}
if errors.Is(err, request.ErrMaxRetry) {
    // 重试用尽
}
```

## 文件结构

```
pkg/request/
├── logger.go       // Logger 日志接口
├── errors.go       // 错误定义（4000 段错误码）
├── config.go       // Config 配置、Option 函数
├── retry.go        // 重试策略、指数退避
├── interceptor.go  // Interceptor 接口、内置拦截器
├── transport.go    // tracingTransport（OTel header 传播）
├── multipart.go    // 文件上传、multipart 构建
├── response.go     // Response 包装、Do[T]() / DoList[T]()
├── request.go      // Request 链式构建器
├── client.go       // Client 核心
└── request_test.go // 单元测试
```
