# Tracing - 分布式链路追踪

基于 OpenTelemetry 的分布式链路追踪包，提供完整的 trace 功能，支持多种导出器，配置灵活，代码质量高。

## 功能特性

- ✅ **完整的 OpenTelemetry 支持**: 基于官方 SDK，符合标准规范
- ✅ **W3C Trace Context**: 完整支持 W3C Trace Context 标准
- ✅ **多种导出器**: 支持 OTLP (HTTP/gRPC)、Jaeger、Stdout
- ✅ **灵活配置**: 采样率、批量导出、超时等可配置
- ✅ **Context 传递**: 通过 Context 传递，对业务代码无侵入
- ✅ **中间件支持**: 提供 Gin 等框架的中间件
- ✅ **丰富的辅助函数**: Span 操作、属性设置等
- ✅ **高质量代码**: 清晰的结构，完整的中文注释

## 目录结构

```
tracing/
├── config.go          # 配置定义
├── errors.go          # 错误定义
├── tracer.go          # Tracer 核心实现
├── exporter.go        # 导出器实现
├── span.go            # Span 辅助函数
├── context.go         # Context 传递
├── middleware.go      # 中间件实现
└── README.md          # 文档（本文件）
```

## 快速开始

### 1. 基础使用

```go
package main

import (
    "context"
    "log"
    "qi/internal/core/tracing"
)

func main() {
    // 创建配置
    cfg := &tracing.Config{
        Enabled:        true,
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        Sampler: tracing.SamplerConfig{
            Type:  "parent_based",
            Ratio: 1.0,
        },
        Exporter: tracing.ExporterConfig{
            Type: "otlp",
            OTLP: tracing.OTLPConfig{
                Endpoint: "localhost:4318",
                Insecure: true,
                Protocol: "http",
            },
        },
    }

    // 初始化全局 Tracer
    if err := tracing.InitGlobal(cfg); err != nil {
        log.Fatal(err)
    }

    // 获取全局 Tracer
    tracer := tracing.GetGlobal()
    defer tracer.Shutdown(context.Background())

    // 使用示例
    ctx := context.Background()
    ctx, span := tracing.StartSpan(ctx, "my-operation")
    defer tracing.EndSpan(span)

    // 业务逻辑
    doSomething(ctx)
}

func doSomething(ctx context.Context) {
    // 创建子 span
    ctx, span := tracing.StartSpan(ctx, "sub-operation")
    defer tracing.EndSpan(span)

    // 添加属性
    tracing.SetAttributes(ctx,
        tracing.UserIDKey.String("user-123"),
        tracing.UserNameKey.String("Alice"),
    )

    // 业务逻辑...
}
```

### 2. Gin 中间件使用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "qi/internal/core/tracing"
)

func main() {
    // 初始化 Tracer
    cfg := tracing.DefaultConfig()
    tracing.InitGlobal(cfg)

    // 创建 Gin 应用
    r := gin.Default()

    // 使用链路追踪中间件
    r.Use(tracing.GinMiddleware("my-service"))

    // 定义路由
    r.GET("/users/:id", func(c *gin.Context) {
        // 在处理函数中创建子 span
        ctx, span := tracing.StartSpan(c.Request.Context(), "getUserHandler")
        defer tracing.EndSpan(span)

        // 获取参数
        userID := c.Param("id")

        // 添加属性
        tracing.SetAttributes(ctx, tracing.UserIDKey.String(userID))

        // 调用服务层
        user, err := getUserFromDB(ctx, userID)
        if err != nil {
            tracing.RecordError(ctx, err)
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        c.JSON(200, user)
    })

    r.Run(":8080")
}

func getUserFromDB(ctx context.Context, userID string) (*User, error) {
    // 创建数据库查询的 span
    ctx, span := tracing.StartSpan(ctx, "db.query.users",
        tracing.WithSpanKind(trace.SpanKindClient),
        tracing.WithAttributes(
            tracing.DBAttributes("mysql", "mydb", "SELECT", "users", "SELECT * FROM users WHERE id = ?")...,
        ),
    )
    defer tracing.EndSpan(span)

    // 执行数据库查询...
    return &User{}, nil
}
```

### 3. 跨服务调用

```go
// 客户端：发送 HTTP 请求
func callRemoteService(ctx context.Context) error {
    // 创建 span
    ctx, span := tracing.StartSpan(ctx, "http.client.call",
        tracing.WithSpanKind(trace.SpanKindClient),
    )
    defer tracing.EndSpan(span)

    // 创建 HTTP 请求
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://remote-service/api", nil)

    // 注入 trace context 到请求头
    headers := make(map[string]string)
    tracing.InjectHTTPHeaders(ctx, headers)
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    // 发送请求
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        tracing.RecordError(ctx, err)
        return err
    }
    defer resp.Body.Close()

    return nil
}

// 服务端：接收 HTTP 请求
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 从请求头提取 trace context
    headers := make(map[string]string)
    for key, values := range r.Header {
        if len(values) > 0 {
            headers[key] = values[0]
        }
    }
    ctx := tracing.ExtractHTTPHeaders(r.Context(), headers)

    // 创建 span
    ctx, span := tracing.StartSpan(ctx, "handleRequest",
        tracing.WithSpanKind(trace.SpanKindServer),
    )
    defer tracing.EndSpan(span)

    // 处理请求...
}
```

## 配置说明

### 完整配置示例

```yaml
tracing:
  # 是否启用链路追踪
  enabled: true
  
  # 服务名称
  service_name: "qi-service"
  
  # 服务版本
  service_version: "1.0.0"
  
  # 部署环境
  environment: "production"
  
  # 采样器配置
  sampler:
    # 采样类型: always_on, always_off, trace_id_ratio, parent_based
    type: "parent_based"
    # 采样率 (0.0 - 1.0)
    ratio: 1.0
  
  # 导出器配置
  exporter:
    # 导出器类型: otlp, jaeger, zipkin, stdout
    type: "otlp"
    
    # OTLP 配置
    otlp:
      endpoint: "localhost:4318"
      insecure: true
      protocol: "http"  # http 或 grpc
      timeout: 10s
      compression: "gzip"  # gzip 或 none
      headers:
        Authorization: "Bearer token"
    
    # Jaeger 配置
    jaeger:
      agent_endpoint: "localhost:6831"
      collector_endpoint: "http://localhost:14268/api/traces"
    
    # Stdout 配置（调试用）
    stdout:
      pretty_print: true
  
  # 批处理器配置
  batch_span_processor:
    max_queue_size: 2048
    max_export_batch_size: 512
    schedule_delay: 5s
    export_timeout: 30s
  
  # 自定义资源属性
  resource_attributes:
    deployment.environment: "production"
    service.namespace: "qi"
```

### 采样策略

1. **always_on**: 100% 采样，记录所有 trace
2. **always_off**: 0% 采样，不记录任何 trace
3. **trace_id_ratio**: 按比例采样，ratio 为 0.1 表示 10% 采样率
4. **parent_based**: 根据父 span 决定是否采样（推荐）

### 导出器类型

1. **OTLP (推荐)**: 
   - 支持 HTTP 和 gRPC 协议
   - 支持多种后端（Jaeger、Zipkin、云服务等）
   
2. **Jaeger**:
   - 直接导出到 Jaeger
   - 支持 Agent 和 Collector 模式
   
3. **Stdout**:
   - 输出到标准输出
   - 用于开发调试

## API 文档

### Tracer 管理

```go
// 初始化全局 Tracer
InitGlobal(cfg *Config) error

// 获取全局 Tracer
GetGlobal() *Tracer

// 创建新的 Tracer 实例
New(cfg *Config) (*Tracer, error)

// 关闭 Tracer
Shutdown(ctx context.Context) error

// 强制刷新
ForceFlush(ctx context.Context) error
```

### Span 操作

```go
// 开始一个新的 span
StartSpan(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, trace.Span)

// 结束 span
EndSpan(span trace.Span)

// 记录错误
RecordError(ctx context.Context, err error)

// 设置 span 状态
SetSpanStatus(ctx context.Context, code codes.Code, description string)

// 添加事件
AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue)

// 设置属性
SetAttributes(ctx context.Context, attrs ...attribute.KeyValue)

// Span 包装器
SpanWrapper(ctx context.Context, spanName string, fn func(context.Context) error, opts ...SpanOption) error

// 异步 Span 包装器
AsyncSpanWrapper(ctx context.Context, spanName string, fn func(context.Context), opts ...SpanOption)
```

### Context 传递

```go
// 注入 trace context
InjectContext(ctx context.Context, carrier propagation.TextMapCarrier)

// 提取 trace context
ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context

// HTTP 专用方法
InjectHTTPHeaders(ctx context.Context, headers map[string]string)
ExtractHTTPHeaders(ctx context.Context, headers map[string]string) context.Context

// gRPC 专用方法
InjectGRPCMetadata(ctx context.Context, md map[string]string)
ExtractGRPCMetadata(ctx context.Context, md map[string]string) context.Context
```

### 属性辅助函数

```go
// HTTP 属性
HTTPAttributes(method, url, route string, statusCode int) []attribute.KeyValue

// 数据库属性
DBAttributes(system, name, operation, table, statement string) []attribute.KeyValue

// 缓存属性
CacheAttributes(system, key string, hit bool) []attribute.KeyValue

// 用户属性
UserAttributes(userID, userName string) []attribute.KeyValue

// 错误属性
ErrorAttributes(err error) []attribute.KeyValue
```

### 中间件

```go
// Gin 中间件（基础）
GinMiddleware(serviceName string) gin.HandlerFunc

// Gin 中间件（高级配置）
GinMiddlewareWithConfig(config GinMiddlewareConfig) gin.HandlerFunc

// 从 Gin Context 获取 Trace ID
GetTraceIDFromGin(c *gin.Context) string

// 从 Gin Context 开始新 span
StartSpanFromGin(c *gin.Context, spanName string, opts ...SpanOption) (gin.Context, trace.Span)
```

## 最佳实践

### 1. Span 命名规范

```go
// 格式: <操作类型>.<资源>.<动作>
// 示例:
"http.server.request"           // HTTP 服务端请求
"http.client.call"              // HTTP 客户端调用
"db.mysql.query"                // MySQL 查询
"db.redis.get"                  // Redis 获取
"mq.kafka.produce"              // Kafka 生产消息
"service.user.create"           // 用户服务创建操作
```

### 2. 合理使用属性

```go
// 推荐：使用预定义的属性键
tracing.SetAttributes(ctx,
    tracing.UserIDKey.String("123"),
    tracing.HTTPMethodKey.String("GET"),
)

// 不推荐：使用字符串直接定义
span.SetAttributes(
    attribute.String("user_id", "123"),
    attribute.String("method", "GET"),
)
```

### 3. 错误处理

```go
func doSomething(ctx context.Context) error {
    ctx, span := tracing.StartSpan(ctx, "doSomething")
    defer tracing.EndSpan(span)

    if err := operation(); err != nil {
        // 记录错误到 span
        tracing.RecordError(ctx, err)
        return err
    }

    return nil
}
```

### 4. 使用 SpanWrapper

```go
// 简化 span 创建和错误处理
err := tracing.SpanWrapper(ctx, "operation", func(ctx context.Context) error {
    // 业务逻辑
    return doWork(ctx)
})
```

### 5. 异步操作

```go
// 异步操作需要传递 context
go func() {
    ctx, span := tracing.StartSpan(ctx, "async-operation")
    defer tracing.EndSpan(span)
    
    // 异步逻辑
}()

// 或使用 AsyncSpanWrapper
tracing.AsyncSpanWrapper(ctx, "async-operation", func(ctx context.Context) {
    // 异步逻辑
})
```

## 性能优化

### 1. 采样率配置

```yaml
# 生产环境可以降低采样率
sampler:
  type: "trace_id_ratio"
  ratio: 0.1  # 10% 采样率
```

### 2. 批处理优化

```yaml
batch_span_processor:
  max_queue_size: 2048          # 增大队列
  max_export_batch_size: 512    # 增大批次
  schedule_delay: 5s            # 适当延迟
  export_timeout: 30s           # 合理超时
```

### 3. 避免过度追踪

```go
// 跳过不重要的路径
func defaultSkipper(c *gin.Context) bool {
    path := c.Request.URL.Path
    return path == "/health" || path == "/metrics"
}
```

## 故障排查

### 1. Trace 未显示

检查项：
- Tracer 是否已初始化
- 配置是否正确
- 采样率是否过低
- 导出器连接是否正常

```go
// 调试：使用 stdout 导出器
cfg.Exporter.Type = "stdout"
cfg.Exporter.Stdout.PrettyPrint = true
```

### 2. Context 传递问题

```go
// 确保 context 正确传递
func handler(c *gin.Context) {
    ctx := c.Request.Context()  // ✅ 正确
    // ctx := context.Background()  // ❌ 错误，会丢失 trace
    
    doSomething(ctx)
}
```

### 3. 内存泄漏

```go
// 确保 span 总是被关闭
ctx, span := tracing.StartSpan(ctx, "operation")
defer tracing.EndSpan(span)  // 使用 defer 确保执行
```

## 集成示例

### 与 GORM 集成

```go
import "gorm.io/plugin/opentelemetry/tracing"

func initDB() *gorm.DB {
    db, _ := gorm.Open(mysql.Open(dsn))
    
    // 注册 OpenTelemetry 插件
    db.Use(tracing.NewPlugin())
    
    return db
}
```

### 与 Redis 集成

```go
func getFromRedis(ctx context.Context, key string) (string, error) {
    ctx, span := tracing.StartSpan(ctx, "redis.get",
        tracing.WithSpanKind(trace.SpanKindClient),
        tracing.WithAttributes(
            tracing.CacheAttributes("redis", key, false)...,
        ),
    )
    defer tracing.EndSpan(span)

    val, err := rdb.Get(ctx, key).Result()
    if err == nil {
        tracing.SetAttributes(ctx, tracing.CacheHitKey.Bool(true))
    }

    return val, err
}
```

## 监控与可视化

### Jaeger UI

1. 启动 Jaeger:
```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

2. 访问 UI: http://localhost:16686

### Zipkin UI

1. 启动 Zipkin:
```bash
docker run -d --name zipkin \
  -p 9411:9411 \
  openzipkin/zipkin
```

2. 访问 UI: http://localhost:9411

## 参考资源

- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/instrumentation/go/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [OTLP 规范](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)

## 许可证

MIT License

