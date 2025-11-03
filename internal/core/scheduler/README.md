# Scheduler - 定时任务调度

基于 Cron v3 的定时任务调度包，支持标准 cron 表达式和秒级任务，提供完整的任务管理、监控和统计功能。

## 功能特性

- ✅ **标准 Cron 表达式**: 支持 5 字段和 6 字段格式
- ✅ **秒级任务**: 支持秒级精度的定时任务
- ✅ **任务管理**: 注册、注销、启动、停止任务
- ✅ **并发控制**: 防止同一任务重复执行
- ✅ **Panic 恢复**: 自动捕获任务执行中的 panic
- ✅ **超时控制**: 任务执行超时自动取消
- ✅ **重试机制**: 任务失败自动重试
- ✅ **统计监控**: 任务执行统计和监控
- ✅ **灵活配置**: 支持配置文件和代码配置
- ✅ **完整日志**: 任务执行日志记录
- ✅ **高质量代码**: 清晰的结构，完整的中文注释

## 目录结构

```
scheduler/
├── config.go          # 配置定义
├── errors.go          # 错误定义
├── job.go             # 任务定义和统计
├── scheduler.go       # 调度器核心实现
├── logger.go          # 日志接口和实现
├── wrapper.go         # 任务包装器
├── helper.go          # 辅助函数
├── example_test.go    # 示例代码
└── README.md          # 文档（本文件）
```

## 快速开始

### 1. 基础使用

```go
package main

import (
    "context"
    "log"
    "qi/internal/core/scheduler"
)

func main() {
    // 创建配置
    cfg := scheduler.DefaultConfig()
    cfg.Enabled = true
    cfg.WithSeconds = true
    cfg.Timezone = "Asia/Shanghai"

    // 创建调度器
    logger := &scheduler.DefaultLogger{}
    sched, err := scheduler.New(cfg, logger)
    if err != nil {
        log.Fatal(err)
    }

    // 注册任务
    sched.RegisterJobFunc("my-task", func(ctx context.Context) error {
        log.Println("Task executed!")
        return nil
    })

    // 添加任务到调度器
    sched.AddJob("my-task", "*/5 * * * * *", nil) // 每5秒执行

    // 启动调度器
    if err := sched.Start(); err != nil {
        log.Fatal(err)
    }

    // 等待...
    select {}
}
```

### 2. 使用配置文件

```yaml
# config.yaml
cron:
  enabled: true
  with_seconds: true
  timezone: "Asia/Shanghai"
  default_timeout: 30m
  skip_if_still_running: true
  recover_panic: true
  retry_count: 3
  retry_interval: 5s
  
  jobs:
    - name: clean_expired_tokens
      spec: "0 0 2 * * *"      # 每天凌晨2点
      enabled: true
      description: "清理过期的 Token"
      timeout: 10m
      
    - name: update_statistics
      spec: "0 */30 * * * *"   # 每30分钟
      enabled: true
      description: "更新统计数据"
```

```go
// 从配置初始化
cfg := loadConfigFromFile("config.yaml")
sched, _ := scheduler.New(cfg, logger)

// 注册任务处理器
sched.RegisterJobFunc("clean_expired_tokens", cleanExpiredTokens)
sched.RegisterJobFunc("update_statistics", updateStatistics)

// 启动（会自动加载配置中的任务）
sched.Start()
```

### 3. 任务定义

```go
// 方式1: 实现 Job 接口
type MyJob struct {
    name string
}

func (j *MyJob) Execute(ctx context.Context) error {
    log.Printf("Executing job: %s", j.name)
    return nil
}

// 注册
sched.RegisterJob("my-job", &MyJob{name: "test"})

// 方式2: 使用函数
sched.RegisterJobFunc("my-func-job", func(ctx context.Context) error {
    log.Println("Function job executed")
    return nil
})

// 方式3: 使用 JobFunc 类型
var myJob scheduler.JobFunc = func(ctx context.Context) error {
    log.Println("JobFunc executed")
    return nil
}
sched.RegisterJob("job-func", myJob)
```

### 4. 动态管理任务

```go
// 添加任务
sched.AddJob("dynamic-job", "*/10 * * * * *", scheduler.JobFunc(func(ctx context.Context) error {
    log.Println("Dynamic job")
    return nil
}))

// 移除任务
sched.RemoveJob("dynamic-job")

// 注销任务
sched.UnregisterJob("dynamic-job")

// 获取任务统计
stats, _ := sched.GetJobStats("my-task")
fmt.Printf("Total: %d, Success: %d, Failed: %d\n", 
    stats.TotalCount, stats.SuccessCount, stats.FailureCount)

// 获取所有统计
allStats := sched.GetAllJobStats()
for name, stats := range allStats {
    fmt.Printf("Job: %s, Success Rate: %.2f%%\n", 
        name, stats.SuccessRate()*100)
}
```

## Cron 表达式

### 格式说明

#### 5 字段格式（标准）
```
分 时 日 月 周
│ │ │ │ │
│ │ │ │ └─ 星期几 (0-6, 0=Sunday)
│ │ │ └─── 月份 (1-12)
│ │ └───── 日期 (1-31)
│ └─────── 小时 (0-23)
└───────── 分钟 (0-59)
```

#### 6 字段格式（带秒）
```
秒 分 时 日 月 周
│  │ │ │ │ │
│  │ │ │ │ └─ 星期几 (0-6, 0=Sunday)
│  │ │ │ └─── 月份 (1-12)
│  │ │ └───── 日期 (1-31)
│  │ └─────── 小时 (0-23)
│  └───────── 分钟 (0-59)
└─────────── 秒 (0-59)
```

### 特殊字符

| 字符 | 含义 | 示例 |
|------|------|------|
| `*` | 任意值 | `* * * * *` = 每分钟 |
| `/` | 间隔 | `*/5 * * * *` = 每5分钟 |
| `,` | 列举 | `0,15,30,45 * * * *` = 每小时的0,15,30,45分 |
| `-` | 范围 | `0 9-17 * * *` = 9点到17点每小时 |
| `?` | 不指定 | 日和周字段可用 |

### 常用表达式示例

```go
// 5 字段格式
"0 * * * *"        // 每小时
"0 0 * * *"        // 每天凌晨
"0 2 * * *"        // 每天凌晨2点
"0 9 * * 1"        // 每周一早上9点
"0 3 1 * *"        // 每月1号凌晨3点
"0 8 * * 1-5"      // 工作日早上8点
"*/30 * * * *"     // 每30分钟
"0 */2 * * *"      // 每2小时
"0 9-17 * * 1-5"   // 工作日的9-17点每小时

// 6 字段格式（带秒）
"* * * * * *"      // 每秒
"*/5 * * * * *"    // 每5秒
"0 * * * * *"      // 每分钟第0秒
"0 0 * * * *"      // 每小时第0分0秒
"0 0 2 * * *"      // 每天凌晨2点（精确到秒）
"0 */30 * * * *"   // 每30分钟（精确到秒）
```

### 使用辅助函数生成表达式

```go
// 每N秒/分钟/小时
cron.Every.Seconds(5)    // "*/5 * * * * *" - 每5秒
cron.Every.Minutes(10)   // "0 */10 * * *"  - 每10分钟
cron.Every.Hours(2)      // "0 0 */2 * *"   - 每2小时
cron.Every.Days(1)       // "0 0 0 */1 * *" - 每1天

// 在指定时间执行
cron.AtTime.DailyAt(14, 30)      // "0 30 14 * * *" - 每天14:30
cron.AtTime.WeeklyAt(1, 9, 0)    // "0 0 9 * * 1"   - 每周一9:00
cron.AtTime.MonthlyAt(1, 0, 0)   // "0 0 0 1 * *"   - 每月1号0:00

// 预定义表达式
cron.Predefined.EveryMinute      // "0 * * * *"
cron.Predefined.EveryHour        // "0 0 * * *"
cron.Predefined.EveryDay         // "0 0 0 * *"
```

## 配置说明

### 调度器配置

```go
type Config struct {
    // 是否启用定时任务
    Enabled bool
    
    // 是否启用秒字段
    WithSeconds bool
    
    // 时区
    Timezone string
    
    // 默认超时时间
    DefaultTimeout time.Duration
    
    // 跳过仍在运行的任务
    SkipIfStillRunning bool
    
    // 启用 panic 恢复
    RecoverPanic bool
    
    // 重试次数
    RetryCount int
    
    // 重试间隔
    RetryInterval time.Duration
    
    // 任务列表
    Jobs []JobConfig
}
```

### 任务配置

```go
type JobConfig struct {
    // 任务名称
    Name string
    
    // Cron 表达式
    Spec string
    
    // 是否启用
    Enabled bool
    
    // 描述
    Description string
    
    // 超时时间（覆盖默认）
    Timeout time.Duration
    
    // 重试次数（覆盖默认）
    RetryCount int
    
    // 任务参数
    Params map[string]interface{}
}
```

## API 文档

### 调度器管理

```go
// 创建调度器
New(cfg *Config, logger Logger) (*Scheduler, error)

// 初始化全局调度器
InitGlobal(cfg *Config, logger Logger) error

// 获取全局调度器
GetGlobal() *Scheduler

// 启动调度器
Start() error

// 停止调度器
Stop() error

// 检查是否运行
IsRunning() bool
```

### 任务管理

```go
// 注册任务
RegisterJob(name string, handler Job) error

// 注册任务函数
RegisterJobFunc(name string, fn JobFunc) error

// 注销任务
UnregisterJob(name string) error

// 添加任务到调度器
AddJob(name, spec string, handler Job) error

// 从调度器移除任务
RemoveJob(name string) error
```

### 统计监控

```go
// 获取任务统计
GetJobStats(name string) (*JobStats, error)

// 获取所有统计
GetAllJobStats() map[string]*JobStats

// 获取任务条目
GetEntry(name string) *cron.Entry

// 获取所有条目
GetEntries() []cron.Entry
```

### 任务统计信息

```go
type JobStats struct {
    JobName         string        // 任务名称
    TotalCount      int64         // 总执行次数
    SuccessCount    int64         // 成功次数
    FailureCount    int64         // 失败次数
    LastExecuteTime time.Time     // 最后执行时间
    LastSuccessTime time.Time     // 最后成功时间
    LastFailureTime time.Time     // 最后失败时间
    AvgDuration     time.Duration // 平均执行时长
    MaxDuration     time.Duration // 最大执行时长
    MinDuration     time.Duration // 最小执行时长
}

// 计算成功率
SuccessRate() float64

// 计算失败率
FailureRate() float64
```

## 最佳实践

### 1. 任务命名规范

```go
// 推荐格式: <模块>_<操作>_<对象>
"user_clean_expired_tokens"     // 用户模块-清理-过期Token
"order_update_statistics"       // 订单模块-更新-统计数据
"system_backup_database"        // 系统模块-备份-数据库
"notify_send_reminders"         // 通知模块-发送-提醒
```

### 2. 错误处理

```go
sched.RegisterJobFunc("my-task", func(ctx context.Context) error {
    // 检查 context 是否取消
    if ctx.Err() != nil {
        return ctx.Err()
    }
    
    // 执行任务
    if err := doSomething(); err != nil {
        // 返回错误，触发重试
        return fmt.Errorf("task failed: %w", err)
    }
    
    return nil
})
```

### 3. 超时控制

```go
// 方式1: 使用配置
cfg := cron.DefaultConfig()
cfg.DefaultTimeout = 10 * time.Minute  // 全局默认超时

// 方式2: 单独设置任务超时
jobConfig := &cron.JobConfig{
    Name:    "long-task",
    Timeout: 30 * time.Minute,  // 该任务30分钟超时
}

// 方式3: 在任务中使用 context
scheduler.RegisterJobFunc("my-task", func(ctx context.Context) error {
    select {
    case <-time.After(5 * time.Second):
        // 工作完成
        return nil
    case <-ctx.Done():
        // 超时或取消
        return ctx.Err()
    }
})
```

### 4. 并发控制

```go
// 启用跳过仍在运行的任务
cfg.SkipIfStillRunning = true

// 或使用延迟执行（等待上一次执行完成）
// 在创建调度器时使用自定义包装器
```

### 5. 日志集成

```go
// 自定义日志实现
type MyLogger struct {
    zapLogger *zap.Logger
}

func (l *MyLogger) Info(msg string) {
    l.zapLogger.Info(msg)
}

func (l *MyLogger) Error(msg string) {
    l.zapLogger.Error(msg)
}

// 使用自定义日志
logger := &MyLogger{zapLogger: zap.L()}
scheduler, _ := cron.New(cfg, logger)
```

### 6. 监控和告警

```go
// 定期检查任务状态
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := scheduler.GetAllJobStats()
        for name, stat := range stats {
            // 检查失败率
            if stat.FailureRate() > 0.1 { // 失败率超过10%
                // 发送告警
                sendAlert(fmt.Sprintf("Job %s failure rate: %.2f%%", 
                    name, stat.FailureRate()*100))
            }
            
            // 检查最后执行时间
            if time.Since(stat.LastExecuteTime) > time.Hour {
                // 任务超过1小时未执行
                sendAlert(fmt.Sprintf("Job %s not executed for 1 hour", name))
            }
        }
    }
}()
```

## 完整示例

### 示例1: Web 应用集成

```go
package main

import (
    "context"
    "log"
    "qi/internal/core/scheduler"
    "time"
)

func main() {
    // 初始化调度器
    cfg := scheduler.DefaultConfig()
    sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})
    
    // 注册清理任务
    scheduler.RegisterJobFunc("clean_expired_tokens", func(ctx context.Context) error {
        log.Println("Cleaning expired tokens...")
        // 清理逻辑
        return nil
    })
    
    // 注册统计任务
    scheduler.RegisterJobFunc("update_statistics", func(ctx context.Context) error {
        log.Println("Updating statistics...")
        // 统计逻辑
        return nil
    })
    
    // 注册备份任务
    scheduler.RegisterJobFunc("backup_database", func(ctx context.Context) error {
        log.Println("Backing up database...")
        // 备份逻辑
        return nil
    })
    
    // 启动调度器
    if err := scheduler.Start(); err != nil {
        log.Fatal(err)
    }
    defer scheduler.Stop()
    
    // 启动 Web 服务器
    // ...
}
```

### 示例2: 动态任务管理

```go
// 添加临时任务
func addTemporaryJob(scheduler *cron.Scheduler) {
    scheduler.AddJob("temp-job", "*/5 * * * * *", cron.JobFunc(func(ctx context.Context) error {
        log.Println("Temporary job executed")
        return nil
    }))
    
    // 30秒后移除
    time.AfterFunc(30*time.Second, func() {
        scheduler.RemoveJob("temp-job")
        log.Println("Temporary job removed")
    })
}
```

### 示例3: 任务监控面板

```go
// HTTP 接口查看任务状态
func handleJobStats(w http.ResponseWriter, r *http.Request) {
    scheduler := cron.GetGlobal()
    stats := scheduler.GetAllJobStats()
    
    json.NewEncoder(w).Encode(stats)
}

// 查看任务列表
func handleJobList(w http.ResponseWriter, r *http.Request) {
    scheduler := cron.GetGlobal()
    entries := scheduler.GetEntries()
    
    type JobInfo struct {
        ID      int
        Next    time.Time
        Prev    time.Time
    }
    
    var jobs []JobInfo
    for _, entry := range entries {
        jobs = append(jobs, JobInfo{
            ID:   int(entry.ID),
            Next: entry.Next,
            Prev: entry.Prev,
        })
    }
    
    json.NewEncoder(w).Encode(jobs)
}
```

## 故障排查

### 1. 任务未执行

检查项：
- 调度器是否已启动
- 任务是否已注册
- 任务配置是否启用
- Cron 表达式是否正确
- 时区设置是否正确

```go
// 调试
log.Printf("Is running: %v", scheduler.IsRunning())
log.Printf("Entries count: %d", len(scheduler.GetEntries()))
```

### 2. 时区问题

```go
// 确保时区正确
cfg.Timezone = "Asia/Shanghai"

// 或使用 UTC
cfg.Timezone = "UTC"
```

### 3. 任务执行慢

```go
// 查看执行时长
stats, _ := scheduler.GetJobStats("slow-job")
log.Printf("Avg duration: %v", stats.AvgDuration)
log.Printf("Max duration: %v", stats.MaxDuration)

// 考虑增加超时时间或优化任务逻辑
```

## 许可证

MIT License

