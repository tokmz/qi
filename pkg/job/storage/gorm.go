package storage

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"github.com/tokmz/qi/pkg/job"
)

// GormStorage GORM 持久化存储实现
type GormStorage struct {
	db          *gorm.DB
	tablePrefix string
	jobTable    string
	runTable    string
	closed      atomic.Bool
}

// JobModel Job 数据模型
type JobModel struct {
	ID          string        `gorm:"primaryKey;size:64" json:"id"`
	Name        string        `gorm:"size:128;index" json:"name"`
	Description string        `gorm:"size:512" json:"description"`
	Cron        string        `gorm:"size:64" json:"cron"`
	Interval    int64         `gorm:"default:0" json:"interval"`
	Type        job.JobType   `gorm:"size:32;index:idx_type_status" json:"type"`
	Status      job.JobStatus `gorm:"size:32;index:idx_type_status,index:idx_status_next_run" json:"status"`
	HandlerName string        `gorm:"size:128" json:"handler_name"`
	Payload     string        `gorm:"type:text" json:"payload"`
	NextRunAt   *time.Time    `gorm:"index:idx_status_next_run" json:"next_run_at"`
	LastRunAt   *time.Time    `gorm:"index" json:"last_run_at"`
	LastResult  string        `gorm:"type:text" json:"last_result"`
	RetryCount  int           `gorm:"default:0" json:"retry_count"`
	MaxRetry    int           `gorm:"default:3" json:"max_retry"`
	CreatedAt   time.Time     `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

// RunModel Run 数据模型
type RunModel struct {
	ID        string        `gorm:"primaryKey;size:64" json:"id"`
	JobID     string        `gorm:"index:idx_job_created;size:64" json:"job_id"`
	Status    job.RunStatus `gorm:"size:32;index" json:"status"`
	StartAt   time.Time     `json:"start_at"`
	EndAt     *time.Time    `json:"end_at"`
	Output    string        `gorm:"type:text" json:"output"`
	Error     string        `gorm:"type:text" json:"error"`
	Duration  int64         `gorm:"default:0;index" json:"duration"`
	TraceID   string        `gorm:"size:64;index" json:"trace_id"`
	CreatedAt time.Time     `gorm:"autoCreateTime;index:idx_job_created" json:"created_at"`
}

// NewGormStorage 创建 GORM 存储（外部传入 DB，不负责关闭）
func NewGormStorage(db *gorm.DB, opts ...Option) (*GormStorage, error) {
	s := &GormStorage{
		db:          db,
		tablePrefix: "",
		jobTable:    "qi_jobs",
		runTable:    "qi_runs",
	}

	for _, opt := range opts {
		opt(s)
	}

	// 配置表名
	if s.tablePrefix != "" {
		s.jobTable = s.tablePrefix + "jobs"
		s.runTable = s.tablePrefix + "runs"
	}

	// 迁移表结构
	if err := db.Table(s.jobTable).AutoMigrate(&JobModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate jobs: %w", err)
	}
	if err := db.Table(s.runTable).AutoMigrate(&RunModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate runs: %w", err)
	}

	return s, nil
}

// Option GORM 存储选项
type Option func(*GormStorage)

// WithTablePrefix 设置表名前缀
func WithTablePrefix(prefix string) Option {
	return func(s *GormStorage) {
		s.tablePrefix = prefix
	}
}

// WithJobTableName 设置任务表名
func WithJobTableName(name string) Option {
	return func(s *GormStorage) {
		s.jobTable = name
	}
}

// WithRunTableName 设置执行记录表名
func WithRunTableName(name string) Option {
	return func(s *GormStorage) {
		s.runTable = name
	}
}

// toJobModel 转换为数据模型
func toJobModel(j *job.Job) *JobModel {
	model := &JobModel{
		ID:          j.ID,
		Name:        j.Name,
		Description: j.Description,
		Cron:        j.Cron,
		Interval:    int64(j.Interval.Unwrap()),
		Type:        j.Type,
		Status:      j.Status,
		HandlerName: j.HandlerName,
		Payload:     j.Payload,
		LastResult:  j.LastResult,
		RetryCount:  j.RetryCount,
		MaxRetry:    j.MaxRetry,
	}

	if j.NextRunAt != nil {
		model.NextRunAt = j.NextRunAt
	}
	if j.LastRunAt != nil {
		model.LastRunAt = j.LastRunAt
	}

	return model
}

// toJob 从数据模型转换
func toJob(m *JobModel) *job.Job {
	return &job.Job{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Cron:        m.Cron,
		Interval:    job.Duration(m.Interval),
		Type:        m.Type,
		Status:      m.Status,
		HandlerName: m.HandlerName,
		Payload:     m.Payload,
		NextRunAt:   m.NextRunAt,
		LastRunAt:   m.LastRunAt,
		LastResult:  m.LastResult,
		RetryCount:  m.RetryCount,
		MaxRetry:    m.MaxRetry,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// toRunModel 转换为数据模型
func toRunModel(r *job.Run) *RunModel {
	return &RunModel{
		ID:       r.ID,
		JobID:    r.JobID,
		Status:   r.Status,
		StartAt:  r.StartAt,
		EndAt:    r.EndAt,
		Output:   r.Output,
		Error:    r.Error,
		Duration: r.Duration,
		TraceID:  r.TraceID,
	}
}

// toRun 从数据模型转换
func toRun(m *RunModel) *job.Run {
	return &job.Run{
		ID:       m.ID,
		JobID:    m.JobID,
		Status:   m.Status,
		StartAt:  m.StartAt,
		EndAt:    m.EndAt,
		Output:   m.Output,
		Error:    m.Error,
		Duration: m.Duration,
		TraceID:  m.TraceID,
	}
}

// CreateJob 创建任务
func (s *GormStorage) CreateJob(ctx context.Context, j *job.Job) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	// 检查是否存在
	var count int64
	if err := s.db.WithContext(ctx).Table(s.jobTable).Where("id = ?", j.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return job.ErrJobAlreadyExists
	}

	model := toJobModel(j)
	return s.db.WithContext(ctx).Table(s.jobTable).Create(model).Error
}

// GetJob 获取任务
func (s *GormStorage) GetJob(ctx context.Context, id string) (*job.Job, error) {
	if s.closed.Load() {
		return nil, job.ErrStorageClosed
	}

	var model JobModel
	err := s.db.WithContext(ctx).Table(s.jobTable).Where("id = ?", id).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, job.ErrJobNotFound
		}
		return nil, err
	}

	return toJob(&model), nil
}

// UpdateJob 更新任务
func (s *GormStorage) UpdateJob(ctx context.Context, j *job.Job) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	model := toJobModel(j)
	return s.db.WithContext(ctx).Table(s.jobTable).Save(model).Error
}

// DeleteJob 删除任务
func (s *GormStorage) DeleteJob(ctx context.Context, id string) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table(s.jobTable).Where("id = ?", id).Delete(&JobModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return job.ErrJobNotFound
		}

		if err := tx.Table(s.runTable).Where("job_id = ?", id).Delete(&RunModel{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// ListJobs 列出任务
func (s *GormStorage) ListJobs(ctx context.Context, status job.JobStatus) ([]*job.Job, error) {
	if s.closed.Load() {
		return nil, job.ErrStorageClosed
	}

	var models []JobModel
	query := s.db.WithContext(ctx).Table(s.jobTable)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&models).Error
	if err != nil {
		return nil, err
	}

	jobs := make([]*job.Job, len(models))
	for i, m := range models {
		jobs[i] = toJob(&m)
	}

	return jobs, nil
}

// CreateRun 创建执行记录
func (s *GormStorage) CreateRun(ctx context.Context, r *job.Run) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	model := toRunModel(r)
	return s.db.WithContext(ctx).Table(s.runTable).Create(model).Error
}

// UpdateRun 更新执行记录
func (s *GormStorage) UpdateRun(ctx context.Context, r *job.Run) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	model := toRunModel(r)
	return s.db.WithContext(ctx).Table(s.runTable).Save(model).Error
}

// GetRuns 获取执行记录
func (s *GormStorage) GetRuns(ctx context.Context, jobID string, limit int) ([]*job.Run, error) {
	if s.closed.Load() {
		return nil, job.ErrStorageClosed
	}

	if limit <= 0 {
		limit = job.DefaultRunLimit
	}

	var models []RunModel
	err := s.db.WithContext(ctx).Table(s.runTable).
		Where("job_id = ?", jobID).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	runs := make([]*job.Run, len(models))
	for i, m := range models {
		runs[i] = toRun(&m)
	}

	return runs, nil
}

// GetJobRunCount 获取任务执行次数
func (s *GormStorage) GetJobRunCount(ctx context.Context, jobID string) (int64, error) {
	if s.closed.Load() {
		return 0, job.ErrStorageClosed
	}

	var count int64
	err := s.db.WithContext(ctx).Table(s.runTable).Where("job_id = ?", jobID).Count(&count).Error
	return count, err
}

// Close 关闭存储（DB 由外部管理，不在此关闭）
func (s *GormStorage) Close() error {
	s.closed.Store(true)
	return nil
}

// Ping 检查存储状态
func (s *GormStorage) Ping(ctx context.Context) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}

// GetDueJobs 获取到期的任务
func (s *GormStorage) GetDueJobs(ctx context.Context) ([]*job.Job, error) {
	if s.closed.Load() {
		return nil, job.ErrStorageClosed
	}

	var models []JobModel
	now := time.Now()
	err := s.db.WithContext(ctx).Table(s.jobTable).
		Where("status IN ?", []string{string(job.JobStatusPending), string(job.JobStatusRunning)}).
		Where("next_run_at IS NULL OR next_run_at <= ?", now).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	jobs := make([]*job.Job, len(models))
	for i, m := range models {
		jobs[i] = toJob(&m)
	}

	return jobs, nil
}

// BatchUpdateJobs 批量更新任务（单事务提交）
func (s *GormStorage) BatchUpdateJobs(ctx context.Context, jobs []*job.Job) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, j := range jobs {
			model := toJobModel(j)
			if err := tx.Table(s.jobTable).Save(model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchUpdateRuns 批量更新执行记录（单事务提交）
func (s *GormStorage) BatchUpdateRuns(ctx context.Context, runs []*job.Run) error {
	if s.closed.Load() {
		return job.ErrStorageClosed
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, r := range runs {
			model := toRunModel(r)
			if err := tx.Table(s.runTable).Save(model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
