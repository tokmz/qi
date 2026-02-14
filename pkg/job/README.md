# Job 任务调度包

Qi 框架的任务调度功能包，提供内存存储和基于 GORM 的持久化存储两种后端支持。

## 功能特性

- **多种任务类型**: Cron 表达式、一次性任务、间隔任务
- **双重存储后端**: 内存存储（开发/测试）、GORM 持久化存储（生产）
- **任务管理**: 添加、删除、暂停、恢复、手动触发
- **执行历史**: 记录每次执行的开始/结束时间、输出、错误信息
- **重试机制**: 支持自动重试和最大重试次数配置
- **并发控制**: 可配置的并发执行任务数
- **批量更新**: 可选的批量更新优化，减少数据库 I/O
- **LRU 缓存**: 热点任务缓存，支持 singleflight 防击穿
- **链路追踪**: OpenTelemetry 集成，覆盖任务执行、批量更新、缓存查询全链路
- **性能指标**: 内置 Metrics 统计

## 快速开始

### 1. 创建调度器

```go
import (
    "context"
    "qi/pkg/job"
    "qi/pkg/job/storage"
)

// 创建内存存储
memStorage := storage.NewMemoryStorage()
defer memStorage.Close()

// 创建调度器
scheduler := job.NewScheduler(memStorage, nil)
```

### 2. 注册任务处理器

```go
// 使用函数式处理器
scheduler.RegisterHandlerFunc("myHandler", func(ctx context.Context, payload string) (string, error) {
    return "执行成功", nil
})

// 或实现 Handler 接口
type MyHandler struct{}

func (h *MyHandler) Execute(ctx context.Context, payload string) (string, error) {
    return "执行成功", nil
}

scheduler.RegisterHandler("myHandler2", &MyHandler{})
```

### 3. 添加任务

```go
ctx := context.Background()

// Cron 任务（每5秒执行一次）
cronJob := &job.Job{
    Name:        "定时清理任务",
    Description: "每5秒清理一次缓存",
    Cron:        "*/5 * * * * *",
    Type:        job.JobTypeCron,
    HandlerName: "myHandler",
    Payload:     `{"action": "cleanup"}`,
    MaxRetry:    3,
}

if err := scheduler.AddJob(ctx, cronJob); err != nil {
    // 处理错误
}
```

### 4. 启动调度器

```go
if err := scheduler.Start(ctx); err != nil {
    // 处理错误
}
defer scheduler.Stop(ctx)
```

## API 参考

### Job 类型

```go
type Job struct {
    ID          string       // 任务ID（自动生成）
    Name        string       // 任务名称
    Description string       // 任务描述
    Cron        string       // Cron 表达式
    Interval    Duration     // 间隔时间（用于 interval 类型）
    Type        JobType      // 任务类型：cron, once, interval
    Status      JobStatus    // 任务状态：pending, running, paused, completed, failed
    HandlerName string       // 处理器名称
    Payload     string       // JSON 格式的参数
    NextRunAt   *time.Time   // 下次执行时间
    LastRunAt   *time.Time   // 上次执行时间
    LastResult  string       // 上次执行结果
    RetryCount  int          // 当前重试次数
    MaxRetry    int          // 最大重试次数
    CreatedAt   time.Time    // 创建时间
    UpdatedAt   time.Time    // 更新时间
}
```

### Run 类型

```go
type Run struct {
    ID        string      // 执行记录ID
    JobID     string      // 关联的任务ID
    Status    RunStatus   // 执行状态：running, success, failed
    StartAt   time.Time   // 开始时间
    EndAt     *time.Time  // 结束时间
    Output    string      // 执行输出
    Error     string      // 错误信息
    Duration  int64       // 执行耗时（毫秒）
    TraceID   string      // 链路追踪ID
    CreatedAt time.Time   // 创建时间
}
```

### Storage 接口

```go
type Storage interface {
    // 任务 CRUD
    CreateJob(ctx context.Context, job *Job) error
    GetJob(ctx context.Context, id string) (*Job, error)
    UpdateJob(ctx context.Context, job *Job) error
    DeleteJob(ctx context.Context, id string) error
    ListJobs(ctx context.Context, status JobStatus) ([]*Job, error)

    // 执行历史
    CreateRun(ctx context.Context, run *Run) error
    UpdateRun(ctx context.Context, run *Run) error
    GetRuns(ctx context.Context, jobID string, limit int) ([]*Run, error)

    // 统计
    GetJobRunCount(ctx context.Context, jobID string) (int64, error)

    // 生命周期
    Close() error
    Ping(ctx context.Context) error
}
```

### BatchStorage 可选接口

Storage 实现可选择性实现此接口以获得批量更新性能优化。`BatchUpdater` 会在初始化时通过类型断言自动检测。

```go
type BatchStorage interface {
    BatchUpdateJobs(ctx context.Context, jobs []*Job) error
    BatchUpdateRuns(ctx context.Context, runs []*Run) error
}
```

`MemoryStorage` 和 `GormStorage` 均已实现此接口。`GormStorage` 使用单事务提交，N 条数据 = 1 次网络往返。

### Scheduler 接口

```go
type Scheduler interface {
    // 任务管理
    AddJob(ctx context.Context, job *Job) error
    RemoveJob(ctx context.Context, id string) error
    PauseJob(ctx context.Context, id string) error
    ResumeJob(ctx context.Context, id string) error
    TriggerJob(ctx context.Context, id string) error

    // 查询
    GetJob(ctx context.Context, id string) (*Job, error)
    ListJobs(ctx context.Context) ([]*Job, error)
    GetRuns(ctx context.Context, jobID string, limit int) ([]*Run, error)

    // 生命周期
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsStarted() bool

    // 处理器注册
    RegisterHandler(name string, handler Handler)
    RegisterHandlerFunc(name string, fn func(ctx context.Context, payload string) (string, error))
}
```

## 配置选项

```go
scheduler := job.NewScheduler(storage,
    job.WithConcurrentRuns(10),                        // 并发执行数（默认5）
    job.WithJobTimeout(10*time.Minute),                // 任务超时时间（默认5分钟）
    job.WithRetryDelay(5*time.Second),                 // 重试间隔（默认5秒）
    job.WithAutoStart(true),                           // 是否自动启动
    job.WithLogger(myLogger),                          // 日志器
    job.WithEnableBatchUpdate(true),                   // 启用批量更新
    job.WithBatchSize(20),                             // 批量大小（默认10）
    job.WithBatchFlushInterval(500*time.Millisecond),  // 批量刷新间隔（默认1秒）
    job.WithEnableCache(true),                         // 启用 LRU 缓存
    job.WithCacheCapacity(200),                        // 缓存容量（默认100）
    job.WithCacheTTL(10*time.Minute),                  // 缓存 TTL（默认5分钟）
    job.WithCacheCleanupInterval(2*time.Minute),       // 缓存清理间隔（默认1分钟）
)
```

## GORM 持久化存储

推荐使用 `qi/pkg/orm` 创建 `*gorm.DB`，获得完整的连接池、预编译语句、链路追踪等能力，然后传入 `NewGormStorage`：

```go
import (
    "qi/pkg/orm"
    "qi/pkg/job"
    "qi/pkg/job/storage"
)

// 使用 orm 包创建 DB（含连接池、慢查询日志等）
db, err := orm.New(&orm.Config{
    Type: orm.MySQL,
    DSN:  "user:password@tcp(localhost:3306)/database?charset=utf8mb4",
    MaxIdleConns:    10,
    MaxOpenConns:    100,
    PrepareStmt:     true,
    SlowThreshold:   200 * time.Millisecond,
})
if err != nil {
    panic(err)
}

// 注册 DB 链路追踪插件（可选）
db.Use(orm.NewTracingPlugin())

// 创建 GORM 存储
gormStorage, err := storage.NewGormStorage(db,
    storage.WithTablePrefix("myapp_"),
)
if err != nil {
    panic(err)
}

// 创建调度器
scheduler := job.NewScheduler(gormStorage,
    job.WithEnableBatchUpdate(true),
)
```

### 存储配置选项

```go
storage.WithTablePrefix("myapp_")          // 表名前缀
storage.WithJobTableName("custom_jobs")    // 自定义任务表名
storage.WithRunTableName("custom_runs")    // 自定义执行记录表名
```

## 日志器

### 内置日志器

```go
// 标准库日志
job.WithLogger(&job.StdLogger{})

// 空日志（不输出）
job.WithLogger(&job.NopLogger{})
```

### 自定义日志器

实现 `job.Logger` 接口：

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...any) {}
func (l *MyLogger) Info(msg string, args ...any)  {}
func (l *MyLogger) Warn(msg string, args ...any)  {}
func (l *MyLogger) Error(msg string, args ...any) {}
```

## 链路追踪

任务调度全链路支持 OpenTelemetry 追踪：

### 追踪覆盖范围

| 组件 | Span 名称 | 说明 |
|------|-----------|------|
| 任务执行 | `job.execute` | 每次任务执行的完整生命周期 |
| 批量更新 | `batch.flush.jobs` / `batch.flush.runs` | 批量写入 DB，通过 Link 关联原始任务 trace |
| DB 操作 | `gorm.Query` / `gorm.Create` 等 | 需注册 `orm.TracingPlugin`，自动记录 SQL 层 span |
| 缓存穿透 | 继承调用方 span context | 缓存未命中时 DB 查询保持 trace 连续性 |

### 初始化追踪

```go
import "qi/pkg/tracing"

tp, err := tracing.NewTracerProvider(&tracing.Config{
    Enabled:      true,
    ExporterType: "jaeger", // 或 "otlp", "noop"
    ServiceName:  "my-job-scheduler",
})
defer tracing.Shutdown(context.Background())
```

### 任务执行 Span 属性

| 属性 | 说明 |
|------|------|
| `job.id` | 任务ID |
| `job.name` | 任务名称 |
| `job.handler` | 处理器名称 |
| `job.retry_count` | 重试次数 |
| `run.duration_ms` | 执行耗时（毫秒） |
| `run.status` | 执行结果状态 |

### 追踪链路示意

```
job.execute (root span)
├── gorm.Query   (UpdateJob - 状态改为 running)
├── gorm.Create  (CreateRun)
├── handler_executing (event)
└── batch.flush.jobs (link) ──→ gorm.Update (事务内批量写入)
    batch.flush.runs (link) ──→ gorm.Update
```

## Cron 表达式

支持 6 位格式（带秒）：

```
秒 分 时 日 月 周
```

| 表达式 | 说明 |
|--------|------|
| `* * * * * *` | 每秒执行 |
| `*/5 * * * * *` | 每5秒执行 |
| `0 * * * * *` | 每分钟执行 |
| `0 0 * * * *` | 每小时执行 |
| `0 0 0 * * *` | 每天零点执行 |
| `0 0 0 * * 1` | 每周一零点执行 |

## 错误处理

```go
import "qi/pkg/job"

switch err {
case job.ErrJobNotFound:          // 任务不存在
case job.ErrJobAlreadyExists:     // 任务已存在
case job.ErrJobPaused:            // 任务已暂停
case job.ErrJobRunning:           // 任务正在运行
case job.ErrSchedulerAlreadyStarted: // 调度器已启动
case job.ErrSchedulerNotStarted:  // 调度器未启动
case job.ErrHandlerNotFound:      // 处理器不存在
case job.ErrStorageClosed:        // 存储已关闭
}
```

## 完整示例

参见 [examples/main.go](examples/main.go)

## 项目结构

```
pkg/job/
├── batch.go           # 批量更新器（支持 trace context 传递）
├── cache.go           # LRU 缓存（singleflight + trace 穿透）
├── config.go          # 配置结构和选项
├── constants.go       # 常量定义
├── duration.go        # Duration 类型
├── errors.go          # 错误定义和错误码
├── handler.go         # 处理器接口
├── heap.go            # 优先队列（任务调度）
├── logger.go          # 日志器接口和实现
├── metrics.go         # 性能指标统计
├── pool.go            # 对象池
├── scheduler.go       # 调度器核心实现
├── storage.go         # Storage / BatchStorage 接口定义
├── types.go           # Job/Run 数据类型
├── storage/
│   ├── gorm.go        # GORM 持久化存储（实现 BatchStorage）
│   └── memory.go      # 内存存储（实现 BatchStorage）
└── examples/
    └── main.go        # 使用示例
```

## 依赖

- `github.com/robfig/cron/v3` - Cron 表达式解析
- `github.com/google/uuid` - UUID 生成
- `go.opentelemetry.io/otel` - OpenTelemetry 追踪
- `golang.org/x/sync` - singleflight（缓存防击穿）
- `gorm.io/gorm` - ORM 框架（存储后端）
