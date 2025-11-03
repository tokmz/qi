package scheduler

import (
	"context"
	"time"
)

// Job 任务接口
type Job interface {
	// Execute 执行任务
	Execute(ctx context.Context) error
}

// JobFunc 任务函数类型
type JobFunc func(ctx context.Context) error

// Execute 实现 Job 接口
func (f JobFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// JobHandler 任务处理器，包装用户的任务函数
type JobHandler struct {
	name    string
	handler Job
	config  *JobConfig
}

// NewJobHandler 创建任务处理器
func NewJobHandler(name string, handler Job, config *JobConfig) *JobHandler {
	return &JobHandler{
		name:    name,
		handler: handler,
		config:  config,
	}
}

// Name 获取任务名称
func (j *JobHandler) Name() string {
	return j.name
}

// Config 获取任务配置
func (j *JobHandler) Config() *JobConfig {
	return j.config
}

// Execute 执行任务
func (j *JobHandler) Execute(ctx context.Context) error {
	return j.handler.Execute(ctx)
}

// JobResult 任务执行结果
type JobResult struct {
	// 任务名称
	JobName string

	// 开始时间
	StartTime time.Time

	// 结束时间
	EndTime time.Time

	// 执行时长
	Duration time.Duration

	// 是否成功
	Success bool

	// 错误信息
	Error error

	// 重试次数
	RetryCount int
}

// NewJobResult 创建任务执行结果
func NewJobResult(jobName string) *JobResult {
	return &JobResult{
		JobName:   jobName,
		StartTime: time.Now(),
		Success:   true,
	}
}

// Finish 完成任务执行
func (r *JobResult) Finish(err error) {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime)
	if err != nil {
		r.Success = false
		r.Error = err
	}
}

// JobStats 任务统计信息
type JobStats struct {
	// 任务名称
	JobName string

	// 总执行次数
	TotalCount int64

	// 成功次数
	SuccessCount int64

	// 失败次数
	FailureCount int64

	// 最后执行时间
	LastExecuteTime time.Time

	// 最后成功时间
	LastSuccessTime time.Time

	// 最后失败时间
	LastFailureTime time.Time

	// 平均执行时长
	AvgDuration time.Duration

	// 最大执行时长
	MaxDuration time.Duration

	// 最小执行时长
	MinDuration time.Duration
}

// NewJobStats 创建任务统计信息
func NewJobStats(jobName string) *JobStats {
	return &JobStats{
		JobName:     jobName,
		MinDuration: time.Duration(1<<63 - 1), // 最大值
	}
}

// Update 更新统计信息
func (s *JobStats) Update(result *JobResult) {
	s.TotalCount++
	s.LastExecuteTime = result.EndTime

	if result.Success {
		s.SuccessCount++
		s.LastSuccessTime = result.EndTime
	} else {
		s.FailureCount++
		s.LastFailureTime = result.EndTime
	}

	// 更新执行时长统计
	if result.Duration > s.MaxDuration {
		s.MaxDuration = result.Duration
	}
	if result.Duration < s.MinDuration {
		s.MinDuration = result.Duration
	}

	// 计算平均执行时长
	s.AvgDuration = time.Duration(
		(int64(s.AvgDuration)*int64(s.TotalCount-1) + int64(result.Duration)) / int64(s.TotalCount),
	)
}

// SuccessRate 计算成功率
func (s *JobStats) SuccessRate() float64 {
	if s.TotalCount == 0 {
		return 0
	}
	return float64(s.SuccessCount) / float64(s.TotalCount)
}

// FailureRate 计算失败率
func (s *JobStats) FailureRate() float64 {
	if s.TotalCount == 0 {
		return 0
	}
	return float64(s.FailureCount) / float64(s.TotalCount)
}

