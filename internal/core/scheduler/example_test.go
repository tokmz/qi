package scheduler_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"qi/internal/core/scheduler"
)

// Example_basic 基础使用示例
func Example_basic() {
	// 1. 创建配置
	cfg := scheduler.DefaultConfig()
	cfg.Enabled = true
	cfg.WithSeconds = true
	cfg.Timezone = "Asia/Shanghai"

	// 2. 创建调度器
	logger := &scheduler.DefaultLogger{}
	sched, err := scheduler.New(cfg, logger)
	if err != nil {
		log.Fatal(err)
	}

	// 3. 注册任务
	sched.RegisterJobFunc("task1", func(ctx context.Context) error {
		log.Println("Task 1 executed!")
		return nil
	})

	// 4. 添加任务到调度器（每5秒执行一次）
	sched.AddJob("task1", "*/5 * * * * *", nil)

	// 5. 启动调度器
	if err := sched.Start(); err != nil {
		log.Fatal(err)
	}

	// 6. 等待一段时间
	time.Sleep(20 * time.Second)

	// 7. 停止调度器
	sched.Stop()

	fmt.Println("Scheduler stopped")
}

// Example_configFile 使用配置文件示例
func Example_configFile() {
	// 创建配置
	cfg := &scheduler.Config{
		Enabled:            true,
		WithSeconds:        true,
		Timezone:           "Asia/Shanghai",
		DefaultTimeout:     10 * time.Minute,
		SkipIfStillRunning: true,
		RecoverPanic:       true,
		RetryCount:         3,
		RetryInterval:      5 * time.Second,
		Jobs: []scheduler.JobConfig{
			{
				Name:        "clean_expired_tokens",
				Spec:        "0 0 2 * * *", // 每天凌晨2点
				Enabled:     true,
				Description: "清理过期的 Token",
				Timeout:     10 * time.Minute,
			},
			{
				Name:        "update_statistics",
				Spec:        "0 */30 * * * *", // 每30分钟
				Enabled:     true,
				Description: "更新统计数据",
			},
		},
	}

	// 创建调度器
	logger := &scheduler.DefaultLogger{}
	sched, _ := scheduler.New(cfg, logger)

	// 注册任务处理器
	sched.RegisterJobFunc("clean_expired_tokens", func(ctx context.Context) error {
		log.Println("Cleaning expired tokens...")
		// 清理逻辑
		return nil
	})

	sched.RegisterJobFunc("update_statistics", func(ctx context.Context) error {
		log.Println("Updating statistics...")
		// 统计逻辑
		return nil
	})

	// 启动（会自动加载配置中的任务）
	sched.Start()
	defer sched.Stop()

	// 等待...
	time.Sleep(5 * time.Second)
}

// CleanupJob 自定义任务结构（实现 Job 接口）
type CleanupJob struct {
	name string
}

// Execute 实现 scheduler.Job 接口
func (j *CleanupJob) Execute(ctx context.Context) error {
	log.Printf("Executing cleanup job: %s", j.name)
	// 清理逻辑
	time.Sleep(1 * time.Second)
	return nil
}

// Example_jobInterface 实现 Job 接口示例
func Example_jobInterface() {
	// 创建调度器
	cfg := scheduler.DefaultConfig()
	sched, _ := scheduler.New(cfg, nil) // 不使用日志

	// 创建任务实例
	job := &CleanupJob{name: "expired-data"}

	// 注册任务
	sched.RegisterJob("cleanup", job)

	// 添加到调度器
	sched.AddJob("cleanup", "*/10 * * * * *", nil) // 每10秒执行

	// 启动
	sched.Start()
	defer sched.Stop()

	time.Sleep(30 * time.Second)
}

// Example_dynamicManagement 动态管理任务示例
func Example_dynamicManagement() {
	cfg := scheduler.DefaultConfig()
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})
	sched.Start()
	defer sched.Stop()

	// 添加任务
	sched.AddJob("dynamic-job", "*/5 * * * * *", scheduler.JobFunc(func(ctx context.Context) error {
		log.Println("Dynamic job executed")
		return nil
	}))

	// 等待一段时间
	time.Sleep(15 * time.Second)

	// 移除任务
	sched.RemoveJob("dynamic-job")
	log.Println("Job removed")

	// 再等待一段时间（任务不会再执行）
	time.Sleep(10 * time.Second)
}

// Example_statistics 任务统计示例
func Example_statistics() {
	cfg := scheduler.DefaultConfig()
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})

	// 注册任务
	sched.RegisterJobFunc("stat-job", func(ctx context.Context) error {
		// 模拟工作
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	sched.AddJob("stat-job", "*/2 * * * * *", nil)
	sched.Start()
	defer sched.Stop()

	// 等待任务执行几次
	time.Sleep(10 * time.Second)

	// 获取统计信息
	stats, _ := sched.GetJobStats("stat-job")
	fmt.Printf("Job: %s\n", stats.JobName)
	fmt.Printf("Total: %d, Success: %d, Failed: %d\n",
		stats.TotalCount, stats.SuccessCount, stats.FailureCount)
	fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate()*100)
	fmt.Printf("Avg Duration: %v\n", stats.AvgDuration)
	fmt.Printf("Max Duration: %v\n", stats.MaxDuration)
	fmt.Printf("Min Duration: %v\n", stats.MinDuration)
}

// Example_timeout 超时控制示例
func Example_timeout() {
	cfg := scheduler.DefaultConfig()
	cfg.DefaultTimeout = 2 * time.Second
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})

	// 注册一个会超时的任务
	sched.RegisterJobFunc("timeout-job", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			// 任务需要5秒，但超时设置为2秒
			log.Println("Job completed")
			return nil
		case <-ctx.Done():
			// 超时
			log.Println("Job timeout!")
			return ctx.Err()
		}
	})

	sched.AddJob("timeout-job", "*/10 * * * * *", nil)
	sched.Start()
	defer sched.Stop()

	time.Sleep(15 * time.Second)

	// 查看统计
	stats, _ := sched.GetJobStats("timeout-job")
	fmt.Printf("Failed: %d (due to timeout)\n", stats.FailureCount)
}

// Example_retry 重试机制示例
func Example_retry() {
	cfg := scheduler.DefaultConfig()
	cfg.RetryCount = 3
	cfg.RetryInterval = 1 * time.Second
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})

	// 注册一个会失败的任务
	attempt := 0
	sched.RegisterJobFunc("retry-job", func(ctx context.Context) error {
		attempt++
		log.Printf("Attempt %d", attempt)

		if attempt < 3 {
			return fmt.Errorf("task failed")
		}

		log.Println("Task succeeded on attempt 3")
		return nil
	})

	sched.AddJob("retry-job", "*/10 * * * * *", nil)
	sched.Start()
	defer sched.Stop()

	time.Sleep(15 * time.Second)
}

// Example_helperFunctions 辅助函数使用示例
func Example_helperFunctions() {
	cfg := scheduler.DefaultConfig()
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})

	// 使用辅助函数生成 Cron 表达式
	sched.RegisterJobFunc("every-5-seconds", func(ctx context.Context) error {
		log.Println("Every 5 seconds")
		return nil
	})
	sched.AddJob("every-5-seconds", scheduler.Every.Seconds(5), nil)

	sched.RegisterJobFunc("every-10-minutes", func(ctx context.Context) error {
		log.Println("Every 10 minutes")
		return nil
	})
	sched.AddJob("every-10-minutes", scheduler.Every.Minutes(10), nil)

	sched.RegisterJobFunc("daily-at-14-30", func(ctx context.Context) error {
		log.Println("Daily at 14:30")
		return nil
	})
	sched.AddJob("daily-at-14-30", scheduler.AtTime.DailyAt(14, 30), nil)

	sched.RegisterJobFunc("weekly-monday-9am", func(ctx context.Context) error {
		log.Println("Weekly Monday 9:00")
		return nil
	})
	sched.AddJob("weekly-monday-9am", scheduler.AtTime.WeeklyAt(1, 9, 0), nil)

	// 使用预定义表达式
	sched.RegisterJobFunc("every-hour", func(ctx context.Context) error {
		log.Println("Every hour")
		return nil
	})
	sched.AddJob("every-hour", scheduler.Predefined.EveryHour, nil)

	sched.Start()
	defer sched.Stop()

	time.Sleep(20 * time.Second)
}

// Example_errorHandling 错误处理示例
func Example_errorHandling() {
	cfg := scheduler.DefaultConfig()
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})

	// 正常任务
	sched.RegisterJobFunc("success-job", func(ctx context.Context) error {
		log.Println("Success job executed")
		return nil
	})

	// 失败任务
	sched.RegisterJobFunc("fail-job", func(ctx context.Context) error {
		log.Println("Fail job executed")
		return fmt.Errorf("something went wrong")
	})

	// Panic 任务（会被恢复）
	sched.RegisterJobFunc("panic-job", func(ctx context.Context) error {
		log.Println("Panic job executed")
		panic("oh no!")
	})

	sched.AddJob("success-job", "*/5 * * * * *", nil)
	sched.AddJob("fail-job", "*/5 * * * * *", nil)
	sched.AddJob("panic-job", "*/5 * * * * *", nil)

	sched.Start()
	defer sched.Stop()

	time.Sleep(15 * time.Second)

	// 查看统计
	allStats := sched.GetAllJobStats()
	for name, stats := range allStats {
		fmt.Printf("Job: %s, Success: %d, Failed: %d\n",
			name, stats.SuccessCount, stats.FailureCount)
	}
}

// Example_globalScheduler 使用全局调度器示例
func Example_globalScheduler() {
	// 初始化全局调度器
	cfg := scheduler.DefaultConfig()
	scheduler.InitGlobal(cfg, &scheduler.DefaultLogger{})

	// 获取全局调度器
	sched := scheduler.GetGlobal()

	// 注册任务
	sched.RegisterJobFunc("global-job", func(ctx context.Context) error {
		log.Println("Global job executed")
		return nil
	})

	sched.AddJob("global-job", "*/5 * * * * *", nil)
	sched.Start()
	defer sched.Stop()

	time.Sleep(15 * time.Second)
}

// Example_contextUsage Context 使用示例
func Example_contextUsage() {
	cfg := scheduler.DefaultConfig()
	sched, _ := scheduler.New(cfg, &scheduler.DefaultLogger{})

	sched.RegisterJobFunc("context-job", func(ctx context.Context) error {
		// 检查 context 是否取消
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 执行长时间操作，定期检查 context
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				log.Println("Job cancelled")
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
				log.Printf("Working... %d/10", i+1)
			}
		}

		log.Println("Job completed")
		return nil
	})

	sched.AddJob("context-job", "*/10 * * * * *", nil)
	sched.Start()
	defer sched.Stop()

	time.Sleep(30 * time.Second)
}
