package job

import (
	"time"

	"github.com/robfig/cron/v3"
)

// 常量定义
const (
	MaxJobNameLength     = 128
	MaxHandlerNameLength = 128
	MaxPayloadLength     = 65535
	DefaultRunLimit      = 10
)

// JobType 任务类型
type JobType string

const (
	JobTypeCron     JobType = "cron"     // Cron 表达式调度
	JobTypeOnce     JobType = "once"     // 一次性任务
	JobTypeInterval JobType = "interval" // 间隔任务
)

// JobStatus 任务状态
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"   // 等待执行
	JobStatusRunning   JobStatus = "running"   // 执行中
	JobStatusPaused    JobStatus = "paused"    // 暂停
	JobStatusCompleted JobStatus = "completed" // 已完成
	JobStatusFailed    JobStatus = "failed"    // 失败
)

// RunStatus 执行记录状态
type RunStatus string

const (
	RunStatusSuccess RunStatus = "success" // 成功
	RunStatusFailed  RunStatus = "failed"  // 失败
	RunStatusRunning RunStatus = "running" // 执行中
)

// Job 任务定义
type Job struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Cron        string        `json:"cron"`                  // Cron 表达式
	Interval    Duration      `json:"interval"`              // 间隔时间（用于 interval 类型）
	Type        JobType       `json:"type"`                  // 任务类型
	Status      JobStatus  `json:"status"`                // 任务状态
	HandlerName string     `json:"handler_name"`          // 处理器名称
	Payload     string     `json:"payload"`               // JSON 参数
	NextRunAt   *time.Time `json:"next_run_at,omitempty"` // 下次执行时间
	LastRunAt   *time.Time `json:"last_run_at,omitempty"` // 上次执行时间
	LastResult  string     `json:"last_result,omitempty"` // 上次执行结果
	RetryCount  int        `json:"retry_count"`           // 当前重试次数
	MaxRetry    int        `json:"max_retry"`             // 最大重试次数
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Clone 浅拷贝 Job 对象（优化版：字符串共享底层数据）
// 注意：Go 的字符串是不可变的，多个变量可以安全地共享同一个字符串
func (j *Job) Clone() *Job {
	job := &Job{
		ID:          j.ID,
		Name:        j.Name,
		Description: j.Description,
		Cron:        j.Cron,
		Interval:    j.Interval,
		Type:        j.Type,
		Status:      j.Status,
		HandlerName: j.HandlerName,
		Payload:     j.Payload,    // 字符串共享底层数据，无需深拷贝
		LastResult:  j.LastResult, // 字符串共享底层数据，无需深拷贝
		RetryCount:  j.RetryCount,
		MaxRetry:    j.MaxRetry,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}
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

// Run 执行记录
type Run struct {
	ID        string     `json:"id"`
	JobID     string     `json:"job_id"`
	Status    RunStatus  `json:"status"`
	StartAt   time.Time  `json:"start_at"`
	EndAt     *time.Time `json:"end_at,omitempty"`
	Output    string     `json:"output,omitempty"`
	Error     string     `json:"error,omitempty"`
	Duration  int64      `json:"duration"` // 毫秒
	TraceID   string     `json:"trace_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// JobData 用于存储和传输的 Job 数据（不包含指针）
type JobData struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Cron        string        `json:"cron"`
	Interval    Duration      `json:"interval"`
	Type        JobType       `json:"type"`
	Status      JobStatus `json:"status"`
	HandlerName string    `json:"handler_name"`
	Payload     string    `json:"payload"`
	NextRunAt   time.Time `json:"next_run_at,omitempty"`
	LastRunAt   time.Time `json:"last_run_at,omitempty"`
	LastResult  string    `json:"last_result,omitempty"`
	RetryCount  int       `json:"retry_count"`
	MaxRetry    int       `json:"max_retry"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToData 将 Job 转换为 JobData
func (j *Job) ToData() JobData {
	data := JobData{
		ID:          j.ID,
		Name:        j.Name,
		Description: j.Description,
		Cron:        j.Cron,
		Interval:    j.Interval,
		Type:        j.Type,
		Status:      j.Status,
		HandlerName: j.HandlerName,
		Payload:     j.Payload,
		RetryCount:  j.RetryCount,
		MaxRetry:    j.MaxRetry,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}
	if j.NextRunAt != nil {
		next := *j.NextRunAt
		data.NextRunAt = next
	}
	if j.LastRunAt != nil {
		last := *j.LastRunAt
		data.LastRunAt = last
	}
	return data
}

// FromData 从 JobData 创建 Job
func FromData(data JobData) *Job {
	job := &Job{
		ID:          data.ID,
		Name:        data.Name,
		Description: data.Description,
		Cron:        data.Cron,
		Interval:    data.Interval,
		Type:        data.Type,
		Status:      data.Status,
		HandlerName: data.HandlerName,
		Payload:     data.Payload,
		RetryCount:  data.RetryCount,
		MaxRetry:    data.MaxRetry,
		CreatedAt:   data.CreatedAt,
		UpdatedAt:   data.UpdatedAt,
	}
	if !data.NextRunAt.IsZero() {
		job.NextRunAt = &data.NextRunAt
	}
	if !data.LastRunAt.IsZero() {
		job.LastRunAt = &data.LastRunAt
	}
	return job
}

// Validate 验证任务参数
func (j *Job) Validate() error {
	if j.Name == "" {
		return NewError(ErrCodeInvalidJobName, "job name is required", ErrInvalidJobName)
	}
	if len(j.Name) > MaxJobNameLength {
		return NewError(ErrCodeInvalidJobName, "job name too long (max 128)", ErrInvalidJobName)
	}
	if j.HandlerName == "" {
		return NewError(ErrCodeHandlerNotFound, "handler name is required", ErrHandlerNotFound)
	}
	if len(j.HandlerName) > MaxHandlerNameLength {
		return NewError(ErrCodeHandlerNotFound, "handler name too long (max 128)", ErrHandlerNotFound)
	}
	if j.Type == "" {
		return NewError(ErrCodeInvalidCron, "job type is required", ErrInvalidCronExpression)
	}
	if j.Type == JobTypeCron && j.Cron == "" {
		return NewError(ErrCodeInvalidCron, "cron expression is required for cron job", ErrInvalidCronExpression)
	}
	// 验证 Cron 表达式格式
	if j.Type == JobTypeCron && j.Cron != "" {
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(j.Cron); err != nil {
			return NewError(ErrCodeInvalidCron, "invalid cron expression format", err)
		}
	}
	if j.Type == JobTypeInterval && j.Interval <= 0 {
		return NewError(ErrCodeInvalidCron, "interval is required for interval job", nil)
	}
	if len(j.Payload) > MaxPayloadLength {
		return NewError(ErrCodeInvalidPayload, "payload too long (max 65535)", ErrInvalidPayload)
	}
	if j.MaxRetry < 0 {
		return NewError(ErrCodeExecutionFailed, "max retry cannot be negative", nil)
	}
	return nil
}
