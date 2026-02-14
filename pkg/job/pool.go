// pool.go 提供 Job/Run 对象池，供外部调用方使用以减少内存分配。
// 调度器内部使用 Job.Clone() 进行对象复制。
package job

import (
	"sync"
)

// JobPool Job 对象池，减少内存分配和 GC 压力
var JobPool = sync.Pool{
	New: func() any {
		return &Job{}
	},
}

// RunPool Run 对象池
var RunPool = sync.Pool{
	New: func() any {
		return &Run{}
	},
}

// AcquireJob 从对象池获取 Job 对象
func AcquireJob() *Job {
	return JobPool.Get().(*Job)
}

// ReleaseJob 释放 Job 对象到对象池
func ReleaseJob(job *Job) {
	if job == nil {
		return
	}

	// 使用零值初始化，避免遗漏字段
	*job = Job{}

	JobPool.Put(job)
}

// AcquireRun 从对象池获取 Run 对象
func AcquireRun() *Run {
	return RunPool.Get().(*Run)
}

// ReleaseRun 释放 Run 对象到对象池
func ReleaseRun(run *Run) {
	if run == nil {
		return
	}

	// 使用零值初始化，避免遗漏字段
	*run = Run{}

	RunPool.Put(run)
}

// CloneFromPool 从对象池克隆 Job 对象（优化版）
func CloneFromPool(j *Job) *Job {
	job := AcquireJob()

	job.ID = j.ID
	job.Name = j.Name
	job.Description = j.Description
	job.Cron = j.Cron
	job.Interval = j.Interval
	job.Type = j.Type
	job.Status = j.Status
	job.HandlerName = j.HandlerName
	job.Payload = j.Payload
	job.LastResult = j.LastResult
	job.RetryCount = j.RetryCount
	job.MaxRetry = j.MaxRetry
	job.CreatedAt = j.CreatedAt
	job.UpdatedAt = j.UpdatedAt

	if j.NextRunAt != nil {
		next := *j.NextRunAt
		job.NextRunAt = &next
	}
	if j.LastRunAt != nil {
		last := *j.LastRunAt
		job.LastRunAt = &last
	}

	return job
}
