# Job 任务调度包

Qi 框架的任务调度功能包，提供内存存储和基于 GORM 的持久化存储两种后端支持。

## 功能特性

- **多种任务类型**: Cron 表达式、一次性任务、间隔任务
- **双重存储后端**: 内存存储（开发/测试）、GORM 持久化存储（生产）
- **任务管理**: 添加、删除、暂停、恢复、手动触发
- **执行历史**: 记录每次执行的开始/结束时间、输出、错误信息
- **重试机制**: 支持自动重试和最大重试次数配置
- **并发控制**: 可配置的并发执行任务数
- **链路追踪**: OpenTelemetry 集成，自动记录任务执行追踪

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

### JobType 常量

```go
job.JobTypeCron     // Cron 表达式调度
job.JobTypeOnce     // 一次性任务
job.JobTypeInterval // 间隔任务
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
    GetJobRunCount(ctx context.Context, jobID string) (int64, error)

    // 生命周期
    Close() error
    Ping(ctx context.Context) error
}
```

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

### Config 结构

```go
config := &job.Config{
    StorageType:    job.StorageTypeMemory,        // 存储类型
    ConcurrentRuns: 5,                            // 并发执行数（默认5）
    JobTimeout:     5 * time.Minute,             // 任务超时时间
    RetryDelay:     5 * time.Second,             // 重试间隔
    AutoStart:      false,                        // 是否自动启动
    Logger:         &job.StdLogger{},             // 日志器
}
```

### 使用配置选项

```go
scheduler := job.NewScheduler(storage, job.WithConcurrentRuns(10))
scheduler := job.NewScheduler(storage, job.WithJobTimeout(10*time.Minute))
scheduler := job.NewScheduler(storage, job.WithLogger(zapLogger))
```

## GORM 持久化存储

### 基本用法

```go
import (
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "qi/pkg/job"
    "qi/pkg/job/storage"
)

// 创建 GORM 数据库连接
db, err := gorm.Open(mysql.Open("user:password@tcp(localhost:3306)/database?charset=utf8mb4"), nil)
if err != nil {
    panic(err)
}

// 创建 GORM 存储，支持表前缀配置
gormStorage, err := storage.NewGormStorage(db,
    storage.WithTablePrefix("myapp_"),
)
if err != nil {
    panic(err)
}

// 创建调度器
scheduler := job.NewScheduler(gormStorage, nil)
```

### 配置选项

```go
// 设置表名前缀
storage.WithTablePrefix("myapp_")

// 自定义表名
storage.WithJobTableName("custom_jobs")
storage.WithRunTableName("custom_runs")
```

## 日志器

### 内置日志器

```go
// 标准库日志
scheduler := job.NewScheduler(storage, job.WithLogger(&job.StdLogger{}))

// Zap 日志适配器
zapLogger, _ := zap.NewDevelopment()
scheduler := job.NewScheduler(storage, job.WithLogger(job.NewZapLogger(zapLogger)))

// 空日志（不输出）
scheduler := job.NewScheduler(storage, job.WithLogger(&job.NopLogger{}))
```

### 自定义日志器

实现 `job.Logger` 接口：

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...any) {}
func (l *MyLogger) Info(msg string, args ...any) {}
func (l *MyLogger) Warn(msg string, args ...any) {}
func (l *MyLogger) Error(msg string, args ...any) {}

scheduler := job.NewScheduler(storage, job.WithLogger(&MyLogger{}))
```

## 链路追踪

任务调度支持 OpenTelemetry 链路追踪，自动记录：

- 任务开始/结束事件
- 执行耗时和结果
- 错误和重试信息
- TraceID 关联执行记录

### 初始化追踪

```go
import "qi/pkg/tracing"

// 创建追踪提供者
tp, err := tracing.NewTracerProvider(&tracing.Config{
    Enabled:      true,
    ExporterType: "jaeger", // 或 "otlp", "noop"
    ServiceName:  "my-job-scheduler",
})
defer tracing.Shutdown(context.Background())
```

### 追踪属性

每个任务执行会生成包含以下属性的 span：

| 属性 | 说明 |
|------|------|
| `job.id` | 任务ID |
| `job.name` | 任务名称 |
| `job.type` | 任务类型 |
| `job.handler` | 处理器名称 |
| `job.status` | 执行状态 |
| `job.retry_count` | 重试次数 |
| `run.id` | 执行记录ID |
| `run.duration_ms` | 执行耗时 |
| `run.status` | 执行结果状态 |
| `trace_id` | 链路追踪ID（存入Run记录） |

## Cron 表达式

支持 6 位格式（带秒）：

```
秒 分 时 日 月 周
```

示例：

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

if err == job.ErrJobNotFound {
    // 任务不存在
} else if err == job.ErrJobAlreadyExists {
    // 任务已存在
} else if err == job.ErrJobPaused {
    // 任务已暂停
} else if err == job.ErrJobRunning {
    // 任务正在运行
} else if err == job.ErrSchedulerAlreadyStarted {
    // 调度器已启动
} else if err == job.ErrSchedulerNotStarted {
    // 调度器未启动
} else if err == job.ErrHandlerNotFound {
    // 处理器不存在
} else if err == job.ErrStorageClosed {
    // 存储已关闭
}
```

## 完整示例

参见 [examples/main.go](examples/main.go)

## 项目结构

```
pkg/job/
├── config.go          # 配置结构和选项
├── errors.go          # 错误定义和错误码
├── handler.go         # 处理器接口
├── logger.go          # 日志器接口和实现
├── scheduler.go       # 调度器核心实现
├── storage.go         # 存储接口定义
├── types.go           # Job/Run 数据类型
├── zap_logger.go      # Zap 日志适配器
├── storage/
│   ├── memory.go      # 内存存储实现
│   └── gorm.go        # GORM 持久化存储实现
└── examples/
    └── main.go        # 使用示例
```

## 依赖

- `github.com/robfig/cron/v3` - Cron 表达式解析
- `github.com/google/uuid` - UUID 生成
- `gorm.io/gorm` - ORM 框架
- `gorm.io/driver/mysql` - MySQL 驱动
- `gorm.io/driver/postgres` - PostgreSQL 驱动
- `gorm.io/driver/sqlite` - SQLite 驱动
- `go.opentelemetry.io/otel` - OpenTelemetry 追踪
- `go.uber.org/zap` - 高性能日志（可选）
