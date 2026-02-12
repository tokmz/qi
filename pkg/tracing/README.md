# Qi 框架链路追踪包

基于 OpenTelemetry 的链路追踪集成，提供开箱即用的分布式追踪能力。

## 特性

- **零侵入集成**：通过中间件自动追踪 HTTP 请求
- **日志关联**：自动注入 TraceID 和 SpanID 到日志
- **多后端支持**：支持 OTLP、Stdout、Noop 导出器
- **灵活采样**：支持多种采样策略（Always/Never/Ratio/ParentBased）
- **性能优化**：异步批量导出，生产环境开销 <1%
- **标准兼容**：遵循 W3C Trace Context 标准

## 快速开始

### 1. 初始化 TracerProvider

```go
import "qi/pkg/tracing"

tp, err := tracing.NewTracerProvider(&tracing.Config{
    ServiceName:      "my-service",
    ServiceVersion:   "1.0.0",
    Environment:      "production",
    ExporterType:     "otlp",
    ExporterEndpoint: "http://localhost:4318",
    SamplingRate:     0.1, // 10% 采样
    Enabled:          true,
})
if err != nil {
    panic(err)
}
defer tracing.Shutdown(context.Background())
```

### 2. 注册中间件

```go
engine := qi.New()

// 注册追踪中间件
engine.Use(tracing.Middleware(
    tracing.WithFilter(func(c *qi.Context) bool {
        // 过滤健康检查
        return c.Request().URL.Path != "/health"
    }),
))
```

### 3. 手动创建 Span

```go
func processOrder(ctx context.Context, order *Order) error {
    // 创建 Span
    ctx, span := tracing.StartSpan(ctx, "process-order")
    defer span.End()

    // 添加属性
    tracing.SetAttributes(span, map[string]interface{}{
        "order.id":     order.ID,
        "order.amount": order.Amount,
    })

    // 业务逻辑
    if err := validateOrder(ctx, order); err != nil {
        tracing.RecordError(span, err)
        return err
    }

    // 添加事件
    tracing.AddEvent(span, "order.validated", map[string]interface{}{
        "validator": "rule-engine",
    })

    return nil
}
```

## 配置说明

### Config 结构

```go
type Config struct {
    // 基础配置
    ServiceName      string            // 服务名称（必填）
    ServiceVersion   string            // 服务版本
    Environment      string            // 环境标识（dev/staging/prod）
    Enabled          bool              // 是否启用（默认 true）

    // 导出器配置
    ExporterType     string            // 导出器类型：otlp/stdout/noop
    ExporterEndpoint string            // 导出器端点
    ExporterHeaders  map[string]string // 导出器请求头（用于认证）

    // 采样配置
    SamplingRate     float64           // 采样率（0.0-1.0）
    SamplingType     string            // 采样类型：always/never/ratio/parent_based

    // 资源属性
    ResourceAttributes map[string]string // 自定义标签

    // 批处理配置
    BatchTimeout       time.Duration     // 批量导出超时（默认 5s）
    MaxExportBatchSize int               // 最大批量大小（默认 512）
    MaxQueueSize       int               // 最大队列大小（默认 2048）
}
```

### 环境变量支持

遵循 OpenTelemetry 标准环境变量：

```bash
# 服务信息
OTEL_SERVICE_NAME=my-service
OTEL_SERVICE_VERSION=1.0.0
OTEL_RESOURCE_ATTRIBUTES=environment=production,region=us-west-1

# 导出器配置
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_HEADERS=authorization=Bearer token123

# 采样配置
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

## 中间件选项

### WithTracerName

设置 Tracer 名称（默认 "qi.http"）：

```go
tracing.Middleware(
    tracing.WithTracerName("my-custom-tracer"),
)
```

### WithSpanNameFormatter

自定义 Span 名称格式：

```go
tracing.Middleware(
    tracing.WithSpanNameFormatter(func(c *qi.Context) string {
        return fmt.Sprintf("[%s] %s", c.Request().Method, c.Request().URL.Path)
    }),
)
```

### WithFilter

过滤不需要追踪的请求：

```go
tracing.Middleware(
    tracing.WithFilter(func(c *qi.Context) bool {
        // 跳过健康检查和静态资源
        path := c.Request().URL.Path
        return path != "/health" && !strings.HasPrefix(path, "/static/")
    }),
)
```

## 采样策略

### Always（全量采集）

适用于开发/测试环境：

```go
SamplingRate: 1.0
SamplingType: "always"
```

### Never（禁用追踪）

完全禁用追踪：

```go
SamplingRate: 0.0
SamplingType: "never"
```

### Ratio（按比例采样）

适用于高流量生产环境：

```go
SamplingRate: 0.01  // 1% 采样
SamplingType: "ratio"
```

### ParentBased（继承上游决策）

适用于分布式系统：

```go
SamplingRate: 0.1   // 10% 采样
SamplingType: "parent_based"
```

## 导出器

### OTLP（推荐）

支持 Jaeger、Tempo、云厂商等多种后端：

```go
ExporterType:     "otlp"
ExporterEndpoint: "http://localhost:4318"
```

### Stdout（开发调试）

输出到标准输出，便于本地调试：

```go
ExporterType: "stdout"
```

### Noop（禁用导出）

不导出任何数据，用于性能测试：

```go
ExporterType: "noop"
```

## 最佳实践

### 1. TraceID 传播

框架自动处理 TraceID 传播，无需手动操作：

```go
func handler(c *qi.Context) {
    // 获取带 TraceContext 的 context.Context
    ctx := c.RequestContext()

    // 传递给 Service 层
    result, err := userService.GetUser(ctx, userID)
}
```

### 2. 日志关联

日志自动关联 TraceID 和 SpanID：

```go
log.InfoContext(ctx, "Processing order")
// 输出: {"level":"info","ts":"...","msg":"Processing order","trace_id":"...","span_id":"..."}
```

### 3. 日志关联

日志自动关联 TraceID 和 SpanID：

```go
log.InfoContext(ctx, "Processing order")
// 输出: {"level":"info","ts":"...","msg":"Processing order","trace_id":"...","span_id":"..."}
```

### 4. 异步任务追踪

手动传递 context：

```go
go func(ctx context.Context) {
    ctx, span := tracing.StartSpan(ctx, "async-task")
    defer span.End()

    // 异步任务逻辑
    processTask(ctx)
}(ctx)
```

### 5. 错误记录

使用 `RecordError` 记录错误：

```go
if err != nil {
    tracing.RecordError(span, err)
    return err
}
```

## 性能指标

| 场景 | 延迟增加 | CPU 增加 | 内存增加 |
|------|---------|---------|---------|
| 采样率 100% | <0.5ms | <2% | <10MB |
| 采样率 10% | <0.1ms | <0.5% | <5MB |
| 采样率 0% | <0.01ms | <0.1% | <1MB |

## 故障排查

### 1. Span 未导出

检查导出器配置和网络连接：

```bash
# 测试 OTLP 端点
curl http://localhost:4318/v1/traces
```

### 2. TraceID 不一致

确保使用 `c.RequestContext()` 获取 context：

```go
// ✅ 正确
ctx := c.RequestContext()
db.WithContext(ctx).Find(&users)

// ❌ 错误
db.Find(&users) // 缺少 context
```

### 3. 性能问题

降低采样率或调整批处理配置：

```go
SamplingRate:       0.01  // 降低到 1%
BatchTimeout:       10 * time.Second
MaxExportBatchSize: 256
```

## 完整示例

参考 `example/tracing/main.go` 查看完整示例代码。

## 相关文档

- [设计文档](DESIGN.md)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
