package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qi/pkg/job"
	"qi/pkg/job/storage"
)

// ExampleHandler 示例任务处理器
type ExampleHandler struct{}

// Execute 实现 Handler 接口
func (h *ExampleHandler) Execute(ctx context.Context, payload string) (string, error) {
	fmt.Printf("执行任务，参数: %s，时间: %s\n", payload, time.Now().Format("2006-01-02 15:04:05"))
	return "执行成功", nil
}

func main() {
	ctx := context.Background()

	// 1. 创建内存存储
	memStorage := storage.NewMemoryStorage()
	defer memStorage.Close()

	// 2. 创建调度器配置
	config := job.DefaultConfig()
	config.ConcurrentRuns = 3
	config.RetryDelay = time.Second * 5

	// 3. 创建调度器
	scheduler := job.NewScheduler(memStorage, config)

	// 4. 注册任务处理器
	scheduler.RegisterHandler("example", &ExampleHandler{})
	scheduler.RegisterHandlerFunc("print", func(ctx context.Context, payload string) (string, error) {
		return fmt.Sprintf("打印任务完成: %s", payload), nil
	})

	// 5. 添加 Cron 任务
	cronJob := &job.Job{
		Name:        "定时任务示例",
		Description: "每5秒执行一次的定时任务",
		Cron:        "*/5 * * * * *", // 每5秒
		Type:        job.JobTypeCron,
		HandlerName: "example",
		Payload:     `{"message": "hello"}`,
		MaxRetry:    3,
	}

	if err := scheduler.AddJob(ctx, cronJob); err != nil {
		log.Fatalf("添加Cron任务失败: %v", err)
	}
	fmt.Printf("添加Cron任务成功，ID: %s\n", cronJob.ID)

	// 6. 添加一次性任务
	onceJob := &job.Job{
		Name:        "一次性任务示例",
		Description: "立即执行的一次性任务",
		Type:        job.JobTypeOnce,
		HandlerName: "example",
		Payload:     `{"type": "once"}`,
	}

	if err := scheduler.AddJob(ctx, onceJob); err != nil {
		log.Fatalf("添加一次性任务失败: %v", err)
	}
	fmt.Printf("添加一次性任务成功，ID: %s\n", onceJob.ID)

	// 7. 启动调度器
	fmt.Println("启动调度器...")
	if err := scheduler.Start(ctx); err != nil {
		log.Fatalf("启动调度器失败: %v", err)
	}

	// 8. 列出所有任务
	jobs, err := scheduler.ListJobs(ctx)
	if err != nil {
		log.Fatalf("列出任务失败: %v", err)
	}
	fmt.Printf("\n当前任务数量: %d\n", len(jobs))
	for _, j := range jobs {
		fmt.Printf("  - ID: %s, Name: %s, Status: %s\n", j.ID, j.Name, j.Status)
	}

	// 9. 等待一段时间观察任务执行
	fmt.Println("\n等待10秒观察任务执行...")
	time.Sleep(10 * time.Second)

	// 10. 查看执行记录
	fmt.Println("\n查看执行记录...")
	for _, j := range jobs {
		runs, err := scheduler.GetRuns(ctx, j.ID, 5)
		if err != nil {
			log.Printf("获取任务%s的执行记录失败: %v", j.ID, err)
			continue
		}
		fmt.Printf("任务 %s 的执行记录:\n", j.Name)
		for _, run := range runs {
			status := "成功"
			if run.Status == job.RunStatusFailed {
				status = fmt.Sprintf("失败: %s", run.Error)
			}
			fmt.Printf("  - 时间: %s, 状态: %s, 耗时: %dms\n",
				run.StartAt.Format("15:04:05"), status, run.Duration)
		}
	}

	// 11. 手动触发任务
	fmt.Println("\n手动触发任务...")
	if err := scheduler.TriggerJob(ctx, cronJob.ID); err != nil {
		log.Printf("触发任务失败: %v", err)
	}

	// 12. 暂停任务
	if err := scheduler.PauseJob(ctx, cronJob.ID); err != nil {
		log.Printf("暂停任务失败: %v", err)
	}
	fmt.Println("任务已暂停")

	// 等待
	time.Sleep(2 * time.Second)

	// 13. 恢复任务
	if err := scheduler.ResumeJob(ctx, cronJob.ID); err != nil {
		log.Printf("恢复任务失败: %v", err)
	}
	fmt.Println("任务已恢复")

	// 14. 停止调度器
	fmt.Println("\n停止调度器...")
	if err := scheduler.Stop(ctx); err != nil {
		log.Printf("停止调度器失败: %v", err)
	}

	fmt.Println("示例程序结束")
}
