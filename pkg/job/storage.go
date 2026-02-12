package job

import (
	"context"
)

// Storage 存储接口
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
