# Logger 包架构设计

## 1. 设计目标

基于 uber-go/zap 构建高性能日志系统，提供：
- **零分配日志** - 生产环境零内存分配
- **结构化日志** - JSON/Console 双格式支持
- **灵活输出** - 控制台/文件/轮转多目标
- **深度集成** - 与 Qi 框架无缝集成（TraceID、错误处理、中间件）
- **可扩展性** - 自定义 Encoder、多输出、动态级别、Hook

## 2. 日志级别分类

### 级别定义
```go
const (
    DebugLevel  // 调试信息（开发环境）
    InfoLevel   // 常规信息（默认级别）
    WarnLevel   // 警告信息（需要关注但不影响运行）
    ErrorLevel  // 错误信息（影响功能但不致命）
    DPanicLevel // 开发环境 panic（生产环境记录错误）
    PanicLevel  // 记录后 panic
    FatalLevel  // 记录后退出程序
)
```

### 使用场景
- **Debug** - 变量值、函数调用、详细流程（仅开发环境）
- **Info** - 请求日志、业务操作、状态变更
- **Warn** - 降级处理、重试操作、配置缺失
- **Error** - 数据库错误、外部 API 失败、业务异常
- **DPanic** - 不应该发生的错误（开发环境立即发现）
- **Panic** - 严重错误需要中断当前请求
- **Fatal** - 无法恢复的错误（如配置错误、依赖不可用）

## 3. 日志格式设计

### JSON 格式（生产环境推荐）
```json
{
  "level": "info",
  "ts": "2026-02-11T10:30:45.123+08:00",
  "caller": "main.go:42",
  "msg": "用户登录成功",
  "trace_id": "trace-1234567890",
  "uid": 12345,
  "method": "POST",
  "path": "/api/v1/login",
  "status": 200,
  "latency": "15ms",
  "error": "database connection timeout"
}
```

### Console 格式（开发环境推荐）
```
2026-02-11T10:30:45.123+08:00  INFO  main.go:42  用户登录成功
    trace_id=trace-1234567890
    uid=12345
    method=POST
    path=/api/v1/login
    status=200
    latency=15ms
```

### 字段说明
- **level** - 日志级别
- **ts** - 时间戳（ISO8601 格式）
- **caller** - 调用位置（文件:行号）
- **msg** - 日志消息
- **trace_id** - 链路追踪 ID（从 qi.Context 自动提取）
- **uid** - 用户 ID（从 qi.Context 自动提取）
- **自定义字段** - 业务相关字段

## 4. 输出目标配置

### 支持的输出目标
1. **控制台输出** - stdout/stderr
2. **文件输出** - 单文件写入
3. **文件轮转** - 按大小/时间自动轮转（基于 lumberjack）

### 轮转策略
```go
type RotateConfig struct {
    Filename   string // 日志文件路径
    MaxSize    int    // 单文件最大大小（MB，默认 100MB）
    MaxAge     int    // 文件保留天数（默认 30 天）
    MaxBackups int    // 最多保留文件数（默认 10 个）
    LocalTime  bool   // 使用本地时间（默认 true）
    Compress   bool   // 是否压缩（默认 false）
}
```

### 多输出组合
- 开发环境：Console → stdout
- 生产环境：JSON → 文件轮转
- 混合模式：Info → 文件，Error → stderr + 文件

## 5. 性能优化策略

### 零分配优化
```go
// ✅ 零分配写法
logger.Info("用户登录",
    zap.String("username", username),
    zap.Int64("uid", uid),
)

// ❌ 避免使用（会产生分配）
logger.Infof("用户 %s 登录，UID: %d", username, uid)
```

### 异步写入
- 使用 `zapcore.WriteSyncer` 包装 Writer
- 缓冲区大小可配置（默认 256KB）
- 定期 Sync（默认 1 秒）

### 采样策略
```go
type SamplingConfig struct {
    Initial    int // 每秒前 N 条日志必定记录
    Thereafter int // 之后每 M 条记录 1 条
}

// 示例：每秒前 100 条全记录，之后每 100 条记录 1 条
// 可防止日志风暴导致性能下降
```

### 条件编译
```go
// 开发环境启用 Caller、Stacktrace
// 生产环境禁用 Caller（减少性能开销）
```

## 6. 可扩展性设计

### 自定义 Encoder
```go
// 允许用户自定义日志格式
type Encodruct {
    MessageKey    string // 消息字段名（默认 "msg"）
    LevelKey      string // 级别字段名（默认 "level"）
    TimeKey       string // 时间字段名（默认 "ts"）
    CallerKey     string // 调用位置字段名（默认 "caller"）
    StacktraceKey string // 堆栈字段名（默认 "stacktrace"）
    LineEnding    string // 行结束符（默认 "\n"）
    EncodeLevel   func   // 级别编码器
    EncodeTime    func   // 时间编码器
    EncodeCaller  func   // 调用位置编码器
}
```

### 多输出支持
```go
// 同时输出到多个目标
logger := New(
    WithConsoleOutput(),
    WithFileOutput("app.log"),
    WithRotateOutput(rotateConfig),
)
```

### 动态级别调整
```go
// 运行时调整日志级别（用于线上调试）
logger.SetLevel(DebugLevel)
```

### Hook 机制
```go
// 日志写入前/后执行自定义逻辑
type Hook interface {
    OnWrite(entry zapcore.Entry, fields []zapcore.Field) error
}

// 示例：错误日志发送告警
type AlertHook struct{}
func (h *AlertHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) error {
    if entry.Level >= ErrorLevel {
        sendAlert(entry.Message)
    }
    return nil
}
```

## 7. 接口规范

### Logger 接口
```go
type Logger interface {
    // 基础日志方法
    Debug(msg string, fields ...zap.Field)
    Info(msg string, fields ...zap.Field)
    Warn(msg string, fields ...zap.Field)
    Error(msg string, fields ...zap.Field)
    DPanic(msg string, fields ...zap.Field)
    Panic(msg string, fields ...zap.Field)
    Fatal(msg string, fields ...zap.Field)

    // 带 context.Context 的日志方法（自动提取 TraceID、UID）
    // 支持标准 context.Context，适用于 Service 层
    DebugContext(ctx context.Context, msg string, fields ...zap.Field)
    InfoContext(ctx context.Context, msg string, fields ...zap.Field)
    WarnContext(ctx context.Context, msg string, fields ...zap.Field)
    ErrorContext(ctx context.Context, msg string, fields ...zap.Field)

    // 工具方法
    With(fields ...zap.Field) Logger           // 创建子 Logger
    WithContext(ctx context.Context) Logger    // 创建带 Context 的子 Logger
    Sync() error                               // 刷新缓冲区
    SetLevel(level Level)                      // 动态调整级别
}
```

### Con
```go
type Config struct {
    // 基础配置
    Level      Level  // 日志级别（默认 InfoLevel）
    Format     Format // 日志格式（json/console，默认 json）

    // 输出配置
    Console    bool          // 是否输出到控制台（默认 true）
    File       string        // 文件路径（空则不输出到文件）
    Rotate     *RotateConfig // 轮转配置（nil 则不轮转）

    // 性能配置
    Sampling   *SamplingConfig // 采样配置（nil 则不采样）
    BufferSize int             // 缓冲区大小（默认 256KB）

    // 功能配置
    EnableCaller     bool // 是否记录调用位置（默认 true）
    EnableStacktrace bool // 是否记录堆栈（Error 及以上，默认 true）

    // 扩展配置
    EncoderConfig *EncoderConfig // 自定义 Encoder 配置
    Hooks         []Hook         // Hook 列表
}
```

### 初始化方法
```go
// New 创建 Logger（使用 Config）
func New(config *Config) (Logger, error)

// NewWithOptions 创建 Logger（使用 Options 模式）
func NewWithOptions(opts ...Option) (Logger, error)

// Default 创建默认 Logger（开发环境配置）
func Default() Logger

// NewProduction 创建生产环境 Logger
func NewProduction() Logger

// NewDevelopment 创建开发环境 Logger
func NewDevelopment() Logger
```

### Options 模式
```go
type Option func(*Config)

func WithLevel(level Level) Option
func WithFormat(format Format) Option
func WithConsoleOutput() Option
func WithFileOutput(filename string) Option
func WithRotateOutput(config *RotateConfig) Option
func WithSampling(config *SamplingConfig) Option
func WithCaller(enable bool) Option
func WithStacktrace(enable bool) Option
func WithHook(hook Hook) Option
```

## 8. 与 Qi 框架集成

### Context 传递设计

**核心思路**：使用标准 `context.Context` 传递 TraceID/UID，兼容 Service 层和 Handler 层。

```go
// 在 qi/helper.go 中添加
const (
    ContextTraceIDKey = "trace_id"
    ContextUidKey     = "uid"
    ContextLoggerKey  = "logger"
)

// 从 qi.Context 提取 context.Context
func (c *Context) Context() context.Context {
    // 将 qi.Context 的值注入到 context.Context
    ctx := c.Request.Context()
    if traceID := GetContextTraceID(c); traceID != "" {
        ctx = context.WithValue(ctx, ContextTraceIDKey, traceID)
    }
    if uid := GetContextUid(c); uid != 0 {
        ctx = context.WithValue(ctx, ContextUidKey, uid)
    }
    return ctx
}

// 从 context.Context 提取 TraceID
func GetTraceIDFromContext(ctx context.Context) string {
    if traceID, ok := ctx.Value(ContextTraceIDKey).(string); ok {
        return traceID
    }
    return ""
}

// 从 context.Context 提取 UID
func GetUidFromContext(ctx context.Context) int64 {
    if uid, ok := ctx.Value(ContextUidKey).(int64); ok {
        return uid
    }
    return 0
}
```

### TraceID 自动注入
```go
// 中间件自动设置 TraceID 并传递 context.Context
func LoggerMiddleware(logger Logger) qi.HandlerFunc {
    return func(c *qi.Context) {
        start := time.Now()
        path := c.Request.URL.Path

        // 将 qi.Context 的值注入到 Request.Context
        ctx := c.Context()
        c.Request = c.Request.WithContext(ctx)

        c.Next()

        // 请求日志自动包含 TraceID
        logger.InfoContext(ctx, "HTTP Request",
            zap.String("method", c.Request.Method),
            zap.String("path", path),
            zap.Int("status", c.Writer.Status()),
            zap.Duration("latency", time.Since(start)),
        )
    }
}
```

### Service 层使用
```go
// Service 层接收 context.Context，自动包含 TraceID/UID
type UserService struct {
    logger logger.Logger
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    // 自动提取 TraceID/UID
    s.logger.InfoContext(ctx, "查询用户", zap.Int64("id", id))

    // 调用 Repository 时继续传递 context
    return s.repo.GetUser(ctx, id)
}

// Handler 层调用 Service
func getUserHandler(c *qi.Context, req *GetUserReq) (*UserResp, error) {
    // 从 qi.Context 提取 context.Context
    ctx := c.Context()

    // 传递给 Service 层
    user, err := userService.GetUser(ctx, req.ID)
    if err != nil {
        return nil, err
    }
    return &UserResp{User: user}, nil
}
```

### 错误日志集成
```go
// 自动记录 qi/pkg/errors.Error
func (c *Context) RespondError(err error) {
    var bizErr *errors.Error
    if errors.As(err, &bizErr) {
        ctx := c.Context()
        logger.ErrorContext(ctx, "业务错误",
            zap.Int("code", bizErr.Code),
            zap.Int("http_code", bizErr.HttpCode),
            zap.String("message", bizErr.Message),
            zap.Error(bizErr.Err),
        )
    }
    // ... 原有逻辑
}
```

## 9. 使用示例

### 基础使用
```go
// 创建 Logger
logger := logger.NewProduction()
defer logger.Sync()

// 基础日志
logger.Info("服务启动", zap.String("addr", ":8080"))

// 结构化日志
logger.Info("用户登录",
    zap.String("username", "alice"),
    zap.Int64("uid", 12345),
    zap.String("ip", "192.168.1.1"),
)

// 错误日志
logger.Error("数据库连接失败",
    zap.Error(err),
    zap.String("dsn", dsn),
)
```

### 与 Qi 框架集成
```go
// 创建 Logger
logger := logger.NewProduction()

// 注册中间件
engine := qi.Default()
engine.Use(logger.Middleware())

// 在 Handler 中使用
r.GET("/user/:id", func(c *qi.Context) {
    // 从 qi.Context 提取 context.Context
    ctx := c.Context()

    // 自动包含 TraceID/UID
    logger.InfoContext(ctx, "查询用户", zap.Int64("id", id))

    // 或使用 Context Logger
    ctxLogger := logger.WithContext(ctx)
    ctxLogger.Info("查询用户", zap.Int64("id", id))

    // 传递给 Service 层
    user, err := userService.GetUser(ctx, id)
})
```

### 高级配置
```go
// 开发环境
logger := logger.NewWithOptions(
    logger.WithLevel(logger.DebugLevel),
    logger.WithFormat(logger.ConsoleFormat),
    logger.WithConsoleOutput(),
    logger.WithCaller(true),
)

// 生产环境
logger := logger.NewWithOptions(
    logger.WithLevel(logger.InfoL,
    logger.WithFormat(logger.JSONFormat),
    logger.WithRotateOutput(&logger.RotateConfig{
        Filename:   "/var/log/app.log",
        MaxSize:    100,
        MaxAge:     30,
        MaxBackups: 10,
        Compress:   true,
    }),
    logger.WithSampling(&logger.SamplingConfig{
        Initial:    100,
        Thereafter: 100,
    }),
    logger.WithCaller(false), // 生产环境禁用 Caller
)

// 混合输出
logger := logger.NewWithOptions(
    logger.WithConsoleOutput(),           // Info → stdout
    logger.WithFileOutput("app.log"),     // All → 文件
    logger.WithRotateOutput(rotateConfig), // All → 轮转文件
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

### Hook 示例
```go
// 错误告警 Hook
type AlertHook struct {
    alerter Alerter
}

func (h *AlertHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) error {
    if entry.Level >= logger.ErrorLevel {
        h.alerter.Send(entry.Message)
    }
    return nil
}gger := logger.NewWithOptions(
    logger.WithHook(&AlertHook{alerter: alerter}),
)
```

## 10. 最佳实践

### 日志级别选择
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

### 字段命名规范
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

### 性能优化
```go
// ✅ 零分配写法
logger.Info("用户操作",
    zap.String("action", action),
    zap.Int64("uid", uid),
)

// ❌ 避免字符串拼接
logger.Info(fmt.Sprintf("用户 %d 执行 %s", uid, action)) // 会产生分配

// ✅ 条件日志
if logger.Level() <= DebugLevel {
    logger.Debug("详细信息", zap.Any("data", expensiveOperation()))
}

// ❌ 无条件计算
logger.Debug("详细信息", zap.Any("data", expensiveOperation())) // 即使不记录也会计算
```

### 错误处理
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

### Context 传递
```go
// ✅ 使用 context.Context 自动传递（适用于 Service 层）
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    // 自动提取 TraceID/UID
    s.logger.InfoContext(ctx, "查询用户", zap.Int64("id", id))
    return s.repo.GetUser(ctx, id)
}

// Handler 层调用
func getUserHandler(c *qi.Context, req *GetUserReq) (*UserResp, error) {
    ctx := c.Context() // 从 qi.Context 提取 context.Context
    user, err := userService.GetUser(ctx, req.ID)
    return &UserResp{User: user}, err
}

// ❌ 手动传递 TraceID（不推荐）
func (s *UserService) GetUser(traceID string, id int64) (*User, error) {
    logger.Info("查询用户",
        zap.String("trace_id", traceID), // 手动传递，容易遗漏
        zap.Int64("id", id),
    )
    return s.repo.GetUser(id)
}
```

## 11. 性能指标

### 目标性能
- **零分配** - 生产环境日志记录零内存分配
- **高吞吐** - 单核 > 1M logs/sec（JSON 格式）
- **低延迟** - P99 < 1μs（异步模式）
- **低开销** - CPU < 5%（正常负载）

### Benchmark 基准
```go
// 预期 Benchmark 结果
BenchmarkLogger/JSON-8              1000000    1200 ns/op    0 B/op    0 allocs/op
BenchmarkLogger/Console-8            800000    1500 ns/op    0 allocs/op
BenchmarkLogger/WithFields-8         900000    1300 ns/op    0 B/op    0 allocs/op
BenchmarkLogger/WithContext-8        850000    1400 ns/op    0 B/op    0 allocs/op
```

## 12. 实现优先级

### P0 - 核心功能（必须实现）
- [x] Logger 接口定义
- [ ] Config 配置结构
- [ ] 基础日志方法（Debug/Info/Warn/Error）
- [ ] JSON/Console 格式支持
- [ ] 控制台/文件输出
- [ ] 文件轮转（基于 lumberjack）
- [ ] Options 模式初始化

### P1 - 框架集成（高优先级）
- [ ] InfoContext/ErrorContext 等方法（从 context.Context 提取 TraceID/UID）
- [ ] LoggerMiddleware（请求日志，注入 context.Context）
- [ ] qi.Context.Context() 方法（提取 context.Context）
- [ ] Context 辅助方法（GetTraceIDFromContext/GetUidFromContext）
- [ ] 错误日志集成（qi/pkg/errors.Error）

### P2 - 性能优化（中优先级）
- [ ] 零分配优化
- [ ] 异步写入
- [ ] 采样策略
- [ ] Benchmark3 - 扩展功能（低优先级）
- [ ] 自定义 Encoder
- [ ] Hook 机制
- [ ] 动态级别调整
- [ ] 多输出组合

## 13. 依赖项

```go
require (
    go.uber.org/zap v1.27.0           // 核心日志库
    gopkg.in/natefinch/lumberjack.v2  // 文件轮转
)
```

## 14. 文件结构

```
pkg/logger/
├── DESIGN.md           # 架构设计文档（本文件）
├── README.md           # 使用文档
├── logger.go           # Logger 接口和实现
├── config.go           # Config 配置结构
├── options.go          # Options 模式
├── level.go            # 日志级别定义
├── format.go           # 日志格式定义
├── rotate.go           # 文件轮转配置
├── sampling.go         # 采样配置
├── hook.go             # Hook 接口
├── middleware.go       # Qi 中间件
├──t.go          # Context 集成
├── logger_test.go      # 单元测试
├── benchmark_test.go   # 性能测试
└── examples/           # 使用示例
    ├── basic.go
    ├── production.go
    └── middleware.go
```

## 15. 后续规划

### v1.0.0 - 核心功能
- 基础日志功能
- JSON/Console 格式
- 文件轮转
- Qi 框架集成

### v1.1.0 - 性能优化
- 零分配优化
- 异步写入
- 采样策略

### v1.2.0 - 扩展功能
- 自定义 Encoder
- Hook 机制
- 动态级别调整

### v2.0.0 - 高级特性
- 分布式追踪集成（OpenTelemetry）
- 日志聚合支持（Elasticsearch、Loki）
- 云原生支持（Kubernetes、Docker）

---

**设计完成时间**: 2026-02-11
**设计者**: Architect
**审核状态**: 待审核
