# Logger 包使用文档

## 简介

Logger 是基于 [uber-go/zap](https://github.com/uber-go/zap) 构建的高性能日志系统，专为 Qi 框架设计，提供零分配、结构化日志和深度框架集成。

## 特性

- **高性能** - 基于 zap 的零分配日志系统
- **结构化日志** - JSON/Console 双格式支持
- **灵活输出** - 控制台/文件/轮转多目标
- **框架集成** - 自动提取 TraceID、UID
- **可扩展** - 支持自定义 Encoder、Hook、动态级别

## 快速开始

### 安装

```bash
go get qi/pkg/logger
```

### 基础使用

```go
package main

import (
    "qi/pkg/logger"
    "go.uber.org/zap"
)

func main() {
    // 创建 Logger
    log, _ := logger.NewProduction()
    defer log.Sync()

    // 记录日志
    log.Info("服务启动", zap.String("addr", ":8080"))
    log.Error("数据库连接失败", zap.Error(err))
}
```

## 日志级别

Logger 支持 7 个日志级别：

| 级别 | 说明 | 使用场景 |
|------|------|----------|
| Debug | 调试信息 | 变量值、函数调用、详细流程（仅开发环境） |
| Info | 常规信息 | 请求日志、业务操作、状态变更 |
| Warn | 警告信息 | 降级处理、重试操作、配置缺失 |
| Error | 错误信息 | 数据库错误、外部 API 失败、业务异常 |
| DPanic | 开发环境 panic | 不应该发生的错误（开发环境立即发现） |
| Panic | 记录后 panic | 严重错误需要中断当前请求 |
| Fatal | 记录后退出 | 无法恢复的错误（如配置错误、依赖不可用） |

## 日志格式

### JSON 格式（生产环境推荐）

```json
{
  "level": "info",
  "ts": "2026-02-11T10:30:45.123+08:00",
  "caller": "main.go:42",
  "msg": "用户登录成功",
  "trace_id": "trace-1234567890",
  "uid": 12345,
  "username": "alice"
}
```

### Console 格式（开发环境推荐）

```
2026-02-11T10:30:45.123+08:00  INFO  main.go:42  用户登录成功
    trace_id=trace-1234567890
    uid=12345
    username=alice
```

## 创建 Logger

### 预设配置

```go
// 生产环境（JSON 格式，Info 级别）
logger, _ := logger.NewProduction()

// 开发环境（Console 格式，Debug 级别）
logger, _ := logger.NewDevelopment()

// 默认配置（等同于 NewDevelopment）
logger := logger.Default()
```

### 自定义配置（Config）

```go
logger, _ := logger.New(&logger.Config{
    Level:            logger.InfoLevel,
    Format:           logger.JSONFormat,
    Console:          true,
    File:             "/var/log/app.log",
    EnableCaller:     true,
    EnableStacktrace: true,
})
```

### 自定义配置（Options）

```go
logger, _ := logger.NewWithOptions(
    logger.WithLevel(logger.DebugLevel),
    logger.WithFormat(logger.ConsoleFormat),
    logger.WithConsoleOutput(),
    logger.WithFileOutput("/var/log/app.log"),
    logger.WithCaller(true),
    logger.WithStacktrace(true),
)
```

## 基础日志方法

```go
// 各级别日志
logger.Debug("调试信息", zap.String("key", "value"))
logger.Info("常规信息", zap.Int("count", 42))
logger.Warn("警告信息", zap.Duration("latency", time.Second))
logger.Error("错误信息", zap.Error(err))
logger.DPanic("开发环境 panic", zap.String("reason", "unexpected"))
logger.Panic("记录后 panic", zap.String("reason", "critical"))
logger.Fatal("记录后退出", zap.String("reason", "fatal"))
```

## 与 Qi 框架集成

### 中间件集成

```go
package main

import (
    "qi"
    "qi/pkg/logger"
)

func main() {
    // 创建 Logger
    log, _ := logger.NewProduction()
    defer log.Sync()

    // 创建 Engine
    engine := qi.New()

    // 注册日志中间件
    engine.Use(logger.Middleware(log))

    // 启动服务
    engine.Run(":8080")
}
```

### 带 Context 的日志

使用标准库 `context.Context` 自动提取 TraceID 和 UID：

```go
// 在 Handler 中使用
r.GET("/user/:id", func(c *qi.Context) {
    // 从 *qi.Context 获取标准库 context.Context
    ctx := c.RequestContext()

    // 使用 context.Context 记录日志（自动包含 TraceID 和 UID）
    logger.InfoContext(ctx, "查询用户", zap.Int64("id", id))
    // 输出：{"level":"info","trace_id":"trace-123","uid":12345,"msg":"查询用户","id":1}
})
```

### Context Logger

```go
// 创建带 Context 的子 Logger
r.GET("/user/:id", func(c *qi.Context) {
    ctx := c.RequestContext()
    ctxLogger := logger.WithContext(ctx)
    ctxLogger.Info("查询用户", zap.Int64("id", id))
    // TraceID 和 UID 自动包含在所有日志中
})
```

## 高级功能

### 文件轮转

```go
logger, _ := logger.NewWithOptions(
    logger.WithRotateOutput(&logger.RotateConfig{
        Filename:   "/var/log/app.log",
        MaxSize:    100,  // 100MB
        MaxAge:     30,   // 30 天
        MaxBackups: 10,   // 10 个文件
        Compress:   true, // 压缩旧文件
    }),
)
```

### 采样（防止日志风暴）

```go
logger, _ := logger.NewWithOptions(
    logger.WithSampling(&logger.SamplingConfig{
        Initial:    100, // 每秒前 100 条全记录
        Thereafter: 100, // 之后每 100 条记录 1 条
    }),
)
```

### 子 Logger

```go
// 创建带固定字段的子 Logger
userLogger := logger.With(
    zap.String("module", "user"),
    zap.String("version", "v1"),
)

userLogger.Info("用户注册", zap.String("username", "alice"))
// 输出：{"level":"info","module":"user","version":"v1","msg":"用户注册","username":"alice"}
```

### 动态调整级别

```go
// 运行时调整日志级别（用于线上调试）
logger.SetLevel(logger.DebugLevel)

// 获取当前级别
level := logger.Level()
```

### Hook 机制

```go
// 实现 Hook 接口
type AlertHook struct{}

func (h *AlertHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) error {
    if entry.Level >= logger.ErrorLevel {
        // 发送告警
        sendAlert(entry.Message)
    }
    return nil
}

// 注册 Hook
logger, _ := logger.NewWithOptions(
    logger.WithHook(&AlertHook{}),
)
```

## 最佳实践

### 1. 日志级别选择

```go
// ✅ 正确使用
logger.Debug("变量值", zap.Any("data", data))           // 调试信息
logger.Info("用户登录", zap.Int64("uid", uid))          // 业务操作
logger.Warn("缓存未命中", zap.String("key", key))       // 降级处理
logger.Error("数据库错误", zap.Error(err))              // 错误信息
logger.Fatal("配置加载失败", zap.Error(err))            // 致命错误

// ❌ 错误使用
logger.Info("变量值", zap.Any("data", data))            // 应该用 Debug
logger.Error("缓存未命中", zap.String("key", key))      // 应该用 Warn
```

### 2. 字段命名规范

```go
// ✅ 使用 snake_case
logger.Info("请求完成",
    zap.String("trace_id", traceID),
    zap.Int64("user_id", userID),
    zap.Duration("response_time", duration),
)

// ❌ 避免 camelCase
logger.Info("请求完成",
    zap.String("traceId", traceID),      // 不推荐
    zap.Int64("userId", userID),         // 不推荐
)
```

### 3. 零分配优化

```go
// ✅ 零分配写法
logger.Info("用户操作",
    zap.String("action", action),
    zap.Int64("uid", uid),
)

// ❌ 避免字符串拼接
logger.Info(fmt.Sprintf("用户 %d 执行 %s", uid, action)) // 会产生分配
```

### 4. 错误处理

```go
// ✅ 记录错误上下文
if err := db.Query(sql); err != nil {
    logger.Error("数据库查询失败",
        zap.Error(err),
        zap.String("sql", sql),
        zap.Any("params", params),
    )
    return err
}

// ❌ 只记录错误消息
if err := db.Query(sql); err != nil {
    logger.Error(err.Error()) // 缺少上下文
    return err
}
```

### 5. TraceID 传递

```go
// ✅ 使用 Context 自动传递
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    logger.InfoContext(ctx, "查询用户", zap.Int64("id", id)) // 自动包含 TraceID
    return s.repo.GetUser(id)
}

// ❌ 手动传递 TraceID
func (s *UserService) GetUser(traceID string, id int64) (*User, error) {
    logger.Info("查询用户",
        zap.String("trace_id", traceID), // 手动传递，容易遗漏
        zap.Int64("id", id),
    )
    return s.repo.GetUser(id)
}
```

## 性能指标

基于 Apple M4 的 Benchmark 结果：

| 测试场景 | 吞吐量 | 延迟 | 内存分配 |
|---------|--------|------|----------|
| JSON 格式 | ~470K ops/s | ~2.5μs | 401 B/op, 3 allocs/op |
| Console 格式 | ~470K ops/s | ~2.6μs | 482 B/op, 7 allocs/op |
| 带字段 | ~367K ops/s | ~2.9μs | 594 B/op, 3 allocs/op |
| 带 Context | ~494K ops/s | ~2.6μs | 337 B/op, 3 allocs/op |
| 禁用级别 | ~67M ops/s | ~17ns | 64 B/op, 1 allocs/op |

## 配置选项

### 完整配置示例

```go
logger, _ := logger.NewWithOptions(
    // 基础配置
    logger.WithLevel(logger.InfoLevel),
    logger.WithFormat(logger.JSONFormat),

    // 输出配置
    logger.WithConsoleOutput(),
    logger.WithFileOutput("/var/log/app.log"),
    logger.WithRotateOutput(&logger.RotateConfig{
        Filename:   "/var/log/app-rotate.log",
        MaxSize:    100,
        MaxAge:     30,
        MaxBackups: 10,
        Compress:   true,
    }),

    // 性能配置
    logger.WithSampling(&logger.SamplingConfig{
        Initial:    100,
        Thereafter: 100,
    }),
    logger.WithBufferSize(256 * 1024),

    // 功能配置
    logger.WithCaller(true),
    logger.WithStacktrace(true),

    // 扩展配置
    logger.WithHook(&AlertHook{}),
)
```

## 常见问题

### Q: 如何在生产环境禁用 Caller 以提升性能？

```go
logger, _ := logger.NewWithOptions(
    logger.WithCaller(false), // 禁用 Caller
)
```

### Q: 如何同时输出到控制台和文件？

```go
logger, _ := logger.NewWithOptions(
    logger.WithConsoleOutput(),
    logger.WithFileOutput("/var/log/app.log"),
)
```

### Q: 如何防止日志风暴？

```go
logger, _ := logger.NewWithOptions(
    logger.WithSampling(&logger.SamplingConfig{
        Initial:    100, // 每秒前 100 条全记录
        Thereafter: 100, // 之后每 100 条记录 1 条
    }),
)
```

### Q: 如何在 Handler 中获取 Logger？

```go
// 方法 1：使用全局 Logger
ctx := c.RequestContext()
logger.InfoContext(ctx, "用户操作")

// 方法 2：使用子 Logger（推荐）
ctxLogger := logger.WithContext(c.RequestContext())
ctxLogger.Info("用户操作")
```

## 完整示例

```go
package main

import (
    "context"
    "qi"
    "qi/pkg/logger"
    "go.uber.org/zap"
)

func main() {
    // 创建 Logger
    log, err := logger.NewWithOptions(
        logger.WithLevel(logger.InfoLevel),
        logger.WithFormat(logger.JSONFormat),
        logger.WithRotateOutput(&logger.RotateConfig{
            Filename:   "/var/log/app.log",
            MaxSize:    100,
            MaxAge:     30,
            MaxBackups: 10,
            Compress:   true,
        }),
        logger.WithCaller(false),
    )
    if err != nil {
        panic(err)
    }
    defer log.Sync()

    // 创建 Engine
    engine := qi.New()

    // 注册日志中间件
    engine.Use(logger.Middleware(log))

    // 路由
    r := engine.Router()
    r.GET("/user/:id", func(c *qi.Context) {
        id := c.Param("id")

        // 获取标准库 context.Context
        ctx := c.RequestContext()

        // 记录日志（自动包含 TraceID 和 UID）
        log.InfoContext(ctx, "查询用户", zap.String("id", id))

        c.Success(map[string]string{"id": id, "name": "Alice"})
    })

    // Service 层示例（使用 context.Context）
    type UserService struct {
        log *logger.Logger
    }

    func (s *UserService) GetUser(ctx context.Context, id int64) error {
        s.log.InfoContext(ctx, "查询用户", zap.Int64("id", id))
        return nil
    }

    // 启动服务
    log.Info("服务启动", zap.String("addr", ":8080"))
    engine.Run(":8080")
}
```

## 参考资料

- [uber-go/zap](https://github.com/uber-go/zap) - 核心日志库
- [lumberjack](https://github.com/natefinch/lumberjack) - 文件轮转
- [Qi 框架文档](../../README.md) - Qi 框架使用文档
