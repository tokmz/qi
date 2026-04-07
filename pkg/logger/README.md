# pkg/logger

基于 [uber-go/zap](https://github.com/uber-go/zap) 构建的高性能结构化日志包，提供 JSON/Console 双格式、多输出目标、文件轮转、采样、动态级别调整、OpenTelemetry 集成和 Hook 扩展机制。

---

## 文件结构

```
pkg/logger/
├── logger.go     ← Logger 接口、实现、全局工厂函数
├── config.go     ← Config 配置结构体
├── options.go    ← 函数式 Option
├── level.go      ← Level 类型与枚举
├── format.go     ← Format 类型（json / console）
├── rotate.go     ← RotateConfig 文件轮转配置
├── sampling.go   ← SamplingConfig 采样配置
├── hook.go       ← Hook 接口
└── logger_test.go
```

---

## 快速上手

```go
import "github.com/tokmz/qi/pkg/logger"

// 开发环境（Console 格式 + Debug 级别）
log := logger.Default()

// 生产环境（JSON 格式 + Info 级别）
log, err := logger.NewProduction()

// 自定义
log, err := logger.NewWithOptions(
    logger.WithLevel(logger.InfoLevel),
    logger.WithFormat(logger.JSONFormat),
    logger.WithConsoleOutput(),
    logger.WithFileOutput("./logs/app.log"),
)
```

---

## 日志级别

| 级别 | 常量 | 使用场景 |
|------|------|----------|
| -1 | `DebugLevel` | 变量值、详细流程（仅开发环境） |
| 0 | `InfoLevel` | 请求日志、业务操作（**默认**） |
| 1 | `WarnLevel` | 降级处理、重试、配置缺失 |
| 2 | `ErrorLevel` | 数据库错误、外部 API 失败 |
| 3 | `DPanicLevel` | 不应发生的错误（开发环境 panic） |
| 4 | `PanicLevel` | 严重错误，记录后 panic |
| 5 | `FatalLevel` | 无法恢复的错误，记录后退出 |

---

## 日志格式

### JSON 格式（生产环境推荐）

```json
{
  "level": "info",
  "ts": "2026-03-22T10:30:45.123+08:00",
  "caller": "service/order.go:42",
  "msg": "订单创建成功",
  "trace_id": "abc123",
  "uid": 10086,
  "order_id": "ORD-20260322-001"
}
```

### Console 格式（开发环境推荐）

```
2026-03-22T10:30:45.123+08:00  INFO  service/order.go:42  订单创建成功
    trace_id=abc123  uid=10086  order_id=ORD-20260322-001
```

---

## API 参考

### Logger 接口

```go
// 基础日志方法
log.Debug(msg string, fields ...zap.Field)
log.Info(msg string, fields ...zap.Field)
log.Warn(msg string, fields ...zap.Field)
log.Error(msg string, fields ...zap.Field)
log.DPanic(msg string, fields ...zap.Field)
log.Panic(msg string, fields ...zap.Field)
log.Fatal(msg string, fields ...zap.Field)

// 带 Context 方法（自动提取 trace_id、span_id、uid）
log.DebugContext(ctx, msg, fields...)
log.InfoContext(ctx, msg, fields...)
log.WarnContext(ctx, msg, fields...)
log.ErrorContext(ctx, msg, fields...)

// 工具方法
log.With(fields ...zap.Field) Logger       // 创建携带固定字段的子 Logger
log.WithContext(ctx context.Context) Logger // 从 context 提取字段创建子 Logger
log.SetLevel(level Level)                  // 动态调整级别（原子操作）
log.Level() Level                          // 获取当前级别
log.Sync() error                           // 刷新缓冲区（程序退出前调用）
log.Close() error                          // 刷新缓冲区并关闭文件句柄（推荐替代 Sync）
```

### Option

| Option | 说明 | 默认值 |
|--------|------|--------|
| `WithLevel(level)` | 日志级别 | `InfoLevel` |
| `WithFormat(format)` | 输出格式 | `JSONFormat` |
| `WithConsoleOutput()` | 启用控制台输出 | 无文件配置时自动启用 |
| `WithFileOutput(path)` | 输出到文件 | 不输出 |
| `WithRotateOutput(cfg)` | 文件轮转输出 | 不轮转 |
| `WithSampling(cfg)` | 采样配置 | 不采样 |
| `WithBufferSize(n)` | 缓冲区大小（字节） | `262144`（256KB） |
| `WithCaller(bool)` | 记录调用位置 | `true` |
| `WithStacktrace(bool)` | Error+ 记录堆栈 | `true` |
| `WithEncoderConfig(cfg)` | 自定义 Encoder 配置 | zap 默认 |
| `WithHook(hook)` | 添加 Hook | 无 |

---

## Context 集成

`XXXContext` 方法从标准 `context.Context` 中自动提取以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `trace_id` | `string` | 请求链路 ID |
| `span_id` | `string` | OpenTelemetry Span ID |
| `uid` | `int64` | 用户 ID |

向 context 写入时必须使用包导出的 key 函数：

```go
ctx = context.WithValue(ctx, logger.ContextKeyTraceID(), "abc123")
ctx = context.WithValue(ctx, logger.ContextKeyUID(), int64(10086))

// Service 层使用
log.InfoContext(ctx, "订单创建成功", zap.String("order_id", "ORD-001"))
```

---

## 输出目标

### 多目标同时输出

```go
log, err := logger.NewWithOptions(
    logger.WithConsoleOutput(),
    logger.WithRotateOutput(&logger.RotateConfig{
        Filename:   "./logs/app.log",
        MaxSize:    100,  // MB
        MaxAge:     30,   // 天
        MaxBackups: 10,   // 文件数
        Compress:   true,
    }),
)
```

### RotateConfig 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Filename` | string | — | 日志文件路径（必填） |
| `MaxSize` | int | 100 | 单文件最大 MB |
| `MaxAge` | int | 30 | 文件保留天数 |
| `MaxBackups` | int | 10 | 最多保留文件数 |
| `LocalTime` | bool | true | 使用本地时间 |
| `Compress` | bool | false | 压缩历史文件 |

---

## 采样

高流量场景下防止日志量爆炸：

```go
log, err := logger.NewWithOptions(
    logger.WithLevel(logger.InfoLevel),
    logger.WithSampling(&logger.SamplingConfig{
        Initial:    100, // 每秒前 100 条必定记录
        Thereafter: 100, // 之后每 100 条记录 1 条
    }),
)
```

---

## 子 Logger

```go
// 携带固定字段（适合模块级别的 Logger）
orderLog := log.With(
    zap.String("module", "order"),
    zap.String("version", "v2"),
)
orderLog.Info("创建订单", zap.String("id", "ORD-001"))
// 输出中始终带有 module=order version=v2

// 从 context 提取 trace_id / uid 创建子 Logger（适合请求维度）
reqLog := log.WithContext(ctx)
reqLog.Info("处理请求")
```

---

## 动态调整级别

运行时无需重启即可调整日志级别：

```go
// 临时开启 Debug 排查问题
log.SetLevel(logger.DebugLevel)

// 恢复 Info 级别
log.SetLevel(logger.InfoLevel)
```

`With()` 创建的子 Logger 与父 Logger 共享同一个 `atomic.Value`，`SetLevel` 对所有子 Logger 同时生效。

---

## Hook

Hook 在每条日志写入时触发，可用于告警上报、日志转发等场景：

```go
type AlertHook struct{}

func (h *AlertHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) error {
    if entry.Level >= zapcore.ErrorLevel {
        // 发送告警到钉钉、Slack 等
        alert.Send(entry.Message)
    }
    return nil
}

log, err := logger.NewWithOptions(
    logger.WithHook(&AlertHook{}),
)
```

---

## 程序退出处理

```go
log, err := logger.NewProduction()
if err != nil {
    panic(err)
}
defer log.Close() // 刷新缓冲区 + 关闭文件句柄（推荐）
```

> `Close` 内部会先调用 `Sync` 再关闭文件句柄，可完全替代 `Sync`。如果使用了文件输出（`WithFileOutput`），务必使用 `Close` 避免文件句柄泄漏。

---

## 预置工厂函数

```go
// 开发环境：Debug 级别 + Console 格式 + Caller + Stacktrace
log := logger.Default()

// 开发环境（同上，返回 error）
log, err := logger.NewDevelopment()

// 生产环境：Info 级别 + JSON 格式 + 禁用 Caller
log, err := logger.NewProduction()
```
