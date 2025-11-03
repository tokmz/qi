package scheduler

import (
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/robfig/cron/v3"
)

// newRecoverWrapper 创建 panic 恢复包装器
func newRecoverWrapper(logger Logger) cron.JobWrapper {
	return func(job cron.Job) cron.Job {
		return cron.FuncJob(func() {
			defer func() {
				if r := recover(); r != nil {
					if logger != nil {
						stack := string(debug.Stack())
						logger.Error(fmt.Sprintf("Job panic recovered: %v\nStack trace:\n%s", r, stack))
					}
				}
			}()
			job.Run()
		})
	}
}

// newSkipIfStillRunningWrapper 创建跳过仍在运行的任务的包装器
func newSkipIfStillRunningWrapper(logger Logger) cron.JobWrapper {
	return func(job cron.Job) cron.Job {
		var mu sync.Mutex
		return cron.FuncJob(func() {
			if !mu.TryLock() {
				if logger != nil {
					logger.Warn("Job still running, skipping this execution")
				}
				return
			}
			defer mu.Unlock()
			job.Run()
		})
	}
}

// newDelayIfStillRunningWrapper 创建延迟执行的包装器
// 如果任务仍在运行，等待其完成后再执行
func newDelayIfStillRunningWrapper(logger Logger) cron.JobWrapper {
	return func(job cron.Job) cron.Job {
		var mu sync.Mutex
		return cron.FuncJob(func() {
			if logger != nil {
				logger.Debug("Waiting for previous job to complete")
			}
			mu.Lock()
			defer mu.Unlock()
			job.Run()
		})
	}
}

// JobWrapper 自定义任务包装器类型
type JobWrapper func(job cron.Job) cron.Job

// WithTimeout 创建带超时的任务包装器
func WithTimeout(logger Logger) JobWrapper {
	return func(job cron.Job) cron.Job {
		return cron.FuncJob(func() {
			// 超时逻辑在 scheduler.wrapJob 中实现
			job.Run()
		})
	}
}

// WithRetry 创建带重试的任务包装器
func WithRetry(maxRetries int, logger Logger) JobWrapper {
	return func(job cron.Job) cron.Job {
		return cron.FuncJob(func() {
			// 重试逻辑在 scheduler.wrapJob 中实现
			job.Run()
		})
	}
}

// WithLogging 创建带日志的任务包装器
func WithLogging(logger Logger) JobWrapper {
	return func(job cron.Job) cron.Job {
		return cron.FuncJob(func() {
			if logger != nil {
				logger.Info("Job started")
			}
			job.Run()
			if logger != nil {
				logger.Info("Job completed")
			}
		})
	}
}
