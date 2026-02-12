package job

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "qi.job"

// DefaultScheduler 默认调度器实现
type DefaultScheduler struct {
	storage     Storage
	handlers    map[string]Handler
	cron        *cron.Cron
	mu          sync.RWMutex
	started     bool
	stopChan    chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
	jobRegistry map[string]*Job         // 本地任务缓存
	cronEntries map[string]cron.EntryID // jobID -> cron entry ID
	runningJobs map[string]bool         // 正在执行的任务ID集合
	semaphore   chan struct{}           // 并发信号量
	config      *Config
	logger      Logger
}

// NewScheduler 创建调度器
func NewScheduler(storage Storage, config *Config) *DefaultScheduler {
	if config == nil {
		config = DefaultConfig()
	}

	if config.Logger == nil {
		config.Logger = &StdLogger{}
	}

	if config.ConcurrentRuns <= 0 {
		config.ConcurrentRuns = 5 // 默认并发数
	}

	return &DefaultScheduler{
		storage:     storage,
		handlers:    make(map[string]Handler),
		cron:        cron.New(cron.WithSeconds()),
		stopChan:    make(chan struct{}),
		jobRegistry: make(map[string]*Job),
		cronEntries: make(map[string]cron.EntryID),
		runningJobs: make(map[string]bool),
		semaphore:   make(chan struct{}, config.ConcurrentRuns),
		config:      config,
		logger:      config.Logger,
	}
}

// RegisterHandler 注册任务处理器
func (s *DefaultScheduler) RegisterHandler(name string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[name] = handler
}

// RegisterHandlerFunc 注册函数式处理器
func (s *DefaultScheduler) RegisterHandlerFunc(name string, fn func(ctx context.Context, payload string) (string, error)) {
	s.RegisterHandler(name, HandlerFunc(fn))
}

// AddJob 添加任务
func (s *DefaultScheduler) AddJob(ctx context.Context, j *Job) error {
	// 验证任务参数
	if err := j.Validate(); err != nil {
		return err
	}

	// 设置ID
	if j.ID == "" {
		j.ID = uuid.New().String()
	}

	// 验证处理器
	s.mu.RLock()
	_, handlerExists := s.handlers[j.HandlerName]
	started := s.started
	s.mu.RUnlock()

	if !handlerExists {
		return NewError(ErrCodeHandlerNotFound, "handler not found: "+j.HandlerName, nil)
	}

	// 设置初始状态
	if j.Status == "" {
		j.Status = JobStatusPending
	}

	// 计算下次执行时间
	if err := s.scheduleNextRun(j); err != nil {
		return err
	}

	// 保存到存储
	if err := s.storage.CreateJob(ctx, j); err != nil {
		return err
	}

	// 添加到本地注册表
	s.mu.Lock()
	s.jobRegistry[j.ID] = j
	s.mu.Unlock()

	// 如果调度器已启动，需要将任务添加到调度器
	if started {
		if err := s.scheduleJob(j); err != nil {
			s.logger.Error("[job] 启动任务 %s 失败: %v", j.ID, err)
		}
	}

	return nil
}

// RemoveJob 删除任务
func (s *DefaultScheduler) RemoveJob(ctx context.Context, id string) error {
	// 从调度器中移除
	s.mu.Lock()
	job, exists := s.jobRegistry[id]
	if exists {
		// 从 cron 调度器中移除
		if s.started && job.Cron != "" {
			if entryID, ok := s.cronEntries[id]; ok {
				s.cron.Remove(entryID)
				delete(s.cronEntries, id)
			}
		}
	}
	delete(s.jobRegistry, id)
	s.mu.Unlock()

	// 从存储中删除
	return s.storage.DeleteJob(ctx, id)
}

// PauseJob 暂停任务
func (s *DefaultScheduler) PauseJob(ctx context.Context, id string) error {
	s.mu.Lock()
	job, exists := s.jobRegistry[id]
	if !exists {
		s.mu.Unlock()
		return ErrJobNotFound
	}

	// 深拷贝任务对象
	jobCopy := job.Clone()
	jobCopy.Status = JobStatusPaused

	// 从注册表中移除
	delete(s.jobRegistry, id)

	// 清除 cron entry 映射
	if entryID, ok := s.cronEntries[id]; ok {
		if s.started {
			s.cron.Remove(entryID)
		}
		delete(s.cronEntries, id)
	}
	s.mu.Unlock()

	// 更新存储
	if err := s.storage.UpdateJob(ctx, jobCopy); err != nil {
		// 更新失败，恢复注册表
		s.mu.Lock()
		s.jobRegistry[id] = job
		s.mu.Unlock()
		return err
	}

	return nil
}

// ResumeJob 恢复任务
func (s *DefaultScheduler) ResumeJob(ctx context.Context, id string) error {
	job, err := s.storage.GetJob(ctx, id)
	if err != nil {
		return err
	}

	s.mu.RLock()
	_, handlerExists := s.handlers[job.HandlerName]
	s.mu.RUnlock()

	if !handlerExists {
		return NewError(ErrCodeHandlerNotFound, "handler not found: "+job.HandlerName, nil)
	}

	s.mu.Lock()
	job.Status = JobStatusPending
	if err := s.scheduleNextRun(job); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()

	if err := s.storage.UpdateJob(ctx, job); err != nil {
		return err
	}

	s.mu.Lock()
	s.jobRegistry[id] = job
	if job.Type == JobTypeCron && job.Cron != "" {
		jobID := job.ID
		entryID, err := s.cron.AddFunc(job.Cron, func() {
			jobCtx, cancel := context.WithTimeout(context.Background(), s.config.JobTimeout)
			defer cancel()
			j, err := s.storage.GetJob(jobCtx, jobID)
			if err != nil {
				s.logger.Error("[job] resume cron 回调获取任务 %s 失败: %v", jobID, err)
				return
			}
			s.executeJob(jobCtx, j)
		})
		if err == nil {
			s.cronEntries[jobID] = entryID
		}
	}
	s.mu.Unlock()

	return nil
}

// TriggerJob 手动触发任务
func (s *DefaultScheduler) TriggerJob(ctx context.Context, id string) error {
	s.mu.RLock()
	started := s.started
	s.mu.RUnlock()

	if !started {
		return ErrSchedulerNotStarted
	}

	j, err := s.storage.GetJob(ctx, id)
	if err != nil {
		return err
	}

	if j.Status == JobStatusRunning {
		return ErrJobRunning
	}
	if j.Status == JobStatusPaused {
		return ErrJobPaused
	}

	jobID := id
	go func() {
		jobCtx, cancel := context.WithTimeout(context.Background(), s.config.JobTimeout)
		defer cancel()
		j, err := s.storage.GetJob(jobCtx, jobID)
		if err != nil {
			return
		}
		s.executeJob(jobCtx, j)
	}()

	return nil
}

// GetJob 获取任务
func (s *DefaultScheduler) GetJob(ctx context.Context, id string) (*Job, error) {
	return s.storage.GetJob(ctx, id)
}

// ListJobs 列出所有任务
func (s *DefaultScheduler) ListJobs(ctx context.Context) ([]*Job, error) {
	return s.storage.ListJobs(ctx, "")
}

// GetRuns 获取执行记录
func (s *DefaultScheduler) GetRuns(ctx context.Context, jobID string, limit int) ([]*Run, error) {
	return s.storage.GetRuns(ctx, jobID, limit)
}

// Start 启动调度器
func (s *DefaultScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return ErrSchedulerAlreadyStarted
	}
	s.started = true
	s.mu.Unlock()

	// 恢复所有待执行的任务
	jobs, err := s.storage.ListJobs(ctx, JobStatusPending)
	if err != nil {
		return err
	}

	for _, j := range jobs {
		if err := s.scheduleJob(j); err != nil {
			s.logger.Error("[job] 启动任务 %s 失败: %v", j.ID, err)
		}
	}

	// 启动cron调度器
	s.cron.Start()

	// 启动后台执行器
	s.wg.Add(1)
	go s.runScheduler(ctx)

	return nil
}

// Stop 停止调度器
func (s *DefaultScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	s.mu.Unlock()

	// 使用 sync.Once 确保只关闭一次
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})

	// 停止cron调度器
	s.cron.Stop()

	// 等待所有任务完成
	s.wg.Wait()

	return nil
}

// IsStarted 检查调度器是否已启动
func (s *DefaultScheduler) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// scheduleNextRun 计算下次执行时间
func (s *DefaultScheduler) scheduleNextRun(job *Job) error {
	now := time.Now()

	switch job.Type {
	case JobTypeCron:
		if job.Cron == "" {
			return NewError(ErrCodeInvalidCron, "cron expression required", nil)
		}
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(job.Cron)
		if err != nil {
			return NewError(ErrCodeInvalidCron, "invalid cron expression", err)
		}
		next := schedule.Next(now)
		job.NextRunAt = &next

	case JobTypeOnce:
		// 一次性任务，如果还没执行过，设置NextRunAt为当前时间
		if job.NextRunAt == nil || job.NextRunAt.Before(now) {
			job.NextRunAt = &now
		}

	case JobTypeInterval:
		// 间隔任务
		if job.NextRunAt == nil {
			job.NextRunAt = &now
		}
	}

	return nil
}

// scheduleJob 将任务添加到调度器
func (s *DefaultScheduler) scheduleJob(job *Job) error {
	// 验证处理器是否存在
	if _, ok := s.handlers[job.HandlerName]; !ok {
		return NewError(ErrCodeHandlerNotFound, "handler not found: "+job.HandlerName, nil)
	}

	switch job.Type {
	case JobTypeCron:
		jobID := job.ID
		entryID, err := s.cron.AddFunc(job.Cron, func() {
			jobCtx, cancel := context.WithTimeout(context.Background(), s.config.JobTimeout)
			defer cancel()
			j, err := s.storage.GetJob(jobCtx, jobID)
			if err != nil {
				s.logger.Error("[job] cron 回调获取任务 %s 失败: %v", jobID, err)
				return
			}
			s.executeJob(jobCtx, j)
		})
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.cronEntries[jobID] = entryID
		s.mu.Unlock()

	case JobTypeOnce, JobTypeInterval:
	}

	return nil
}

// runScheduler 后台调度器循环
func (s *DefaultScheduler) runScheduler(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.processDueJobs(ctx, now)
		}
	}
}

// processDueJobs 处理到期的任务
func (s *DefaultScheduler) processDueJobs(ctx context.Context, now time.Time) {
	s.mu.Lock()
	var dueJobs []*Job
	for _, job := range s.jobRegistry {
		if job.Status == JobStatusPending && job.NextRunAt != nil && job.NextRunAt.Before(now) {
			if !s.runningJobs[job.ID] {
				dueJobs = append(dueJobs, job)
				s.runningJobs[job.ID] = true
			}
		}
	}
	s.mu.Unlock()

	for _, job := range dueJobs {
		s.executeJob(ctx, job)
	}
}

// executeJob 执行任务
func (s *DefaultScheduler) executeJob(ctx context.Context, j *Job) {
	// 尝试获取信号量，如果并发数已满则跳过
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	default:
		// 并发数已满，跳过本次执行
		return
	}

	// 启动链路追踪 span
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "job.execute",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("job.id", j.ID),
			attribute.String("job.name", j.Name),
			attribute.String("job.handler", j.HandlerName),
		),
	)
	defer span.End()

	job := j.Clone()

	handler, ok := s.handlers[job.HandlerName]
	if !ok {
		s.mu.Lock()
		delete(s.runningJobs, job.ID)
		s.mu.Unlock()
		span.SetStatus(codes.Error, "handler not found")
		return
	}

	s.mu.Lock()

	if job.Status == JobStatusRunning {
		delete(s.runningJobs, job.ID)
		s.mu.Unlock()
		span.SetStatus(codes.Ok, "already running")
		return
	}

	job.Status = JobStatusRunning
	job.RetryCount = 0
	now := time.Now()
	job.LastRunAt = &now

	if registryJob, exists := s.jobRegistry[job.ID]; exists {
		registryJob.Status = JobStatusRunning
	}
	s.mu.Unlock()

	if err := s.storage.UpdateJob(ctx, job); err != nil {
		s.mu.Lock()
		if registryJob, exists := s.jobRegistry[job.ID]; exists {
			registryJob.Status = JobStatusPending
		}
		delete(s.runningJobs, job.ID)
		s.mu.Unlock()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		job.Status = JobStatusPending
		if updateErr := s.storage.UpdateJob(ctx, job); updateErr != nil {
			s.logger.Error("[job] 恢复任务 %s 状态失败: %v", job.ID, updateErr)
		}
		return
	}

	run := &Run{
		ID:      uuid.New().String(),
		JobID:   job.ID,
		Status:  RunStatusRunning,
		StartAt: now,
		TraceID: span.SpanContext().TraceID().String(),
	}

	if err := s.storage.CreateRun(ctx, run); err != nil {
		s.mu.Lock()
		if registryJob, exists := s.jobRegistry[job.ID]; exists {
			registryJob.Status = JobStatusPending
		}
		delete(s.runningJobs, job.ID)
		s.mu.Unlock()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		job.Status = JobStatusPending
		if updateErr := s.storage.UpdateJob(ctx, job); updateErr != nil {
			s.logger.Error("[job] 恢复任务 %s 状态失败: %v", job.ID, updateErr)
		}
		return
	}

	// 执行任务（不在锁内执行，避免阻塞）
	span.AddEvent("handler_executing")
	startTime := time.Now()
	output, execErr := handler.Execute(ctx, job.Payload)
	duration := time.Since(startTime).Milliseconds()

	// 更新执行记录
	endTime := time.Now()
	run.EndAt = &endTime
	run.Duration = duration

	// 获取锁并更新状态
	s.mu.Lock()

	if execErr != nil {
		run.Status = RunStatusFailed
		run.Error = execErr.Error()

		// 处理重试
		if job.RetryCount < job.MaxRetry {
			job.RetryCount++
			job.Status = JobStatusPending
			nextRun := time.Now().Add(s.config.RetryDelay)
			job.NextRunAt = &nextRun
			span.AddEvent("retry_scheduled", trace.WithAttributes(
				attribute.Int("retry_count", job.RetryCount),
			))
		} else {
			job.Status = JobStatusFailed
			job.LastResult = execErr.Error()
		}
	} else {
		run.Status = RunStatusSuccess
		run.Output = output
		job.Status = JobStatusCompleted
		job.LastResult = output
		span.SetStatus(codes.Ok, "completed")

		// 调度下次执行
		if job.Type == JobTypeCron || job.Type == JobTypeInterval {
			if err := s.scheduleNextRun(job); err == nil {
				job.Status = JobStatusPending
				if job.NextRunAt != nil {
					span.AddEvent("next_run_scheduled", trace.WithAttributes(
						attribute.String("next_run_at", job.NextRunAt.Format(time.RFC3339)),
					))
				}
			}
		}
	}

	// 更新注册表中的任务
	if registryJob, exists := s.jobRegistry[job.ID]; exists {
		registryJob.Status = job.Status
		registryJob.LastRunAt = job.LastRunAt
		registryJob.LastResult = job.LastResult
		registryJob.RetryCount = job.RetryCount
		registryJob.NextRunAt = job.NextRunAt
	}

	delete(s.runningJobs, job.ID)

	s.mu.Unlock()

	// 设置执行结果属性
	span.SetAttributes(
		attribute.Int64("run.duration_ms", duration),
		attribute.String("run.status", string(run.Status)),
		attribute.Int("job.retry_count", job.RetryCount),
	)

	if execErr != nil {
		span.RecordError(execErr)
		span.SetStatus(codes.Error, execErr.Error())
	}

	if err := s.storage.UpdateJob(ctx, job); err != nil {
		s.logger.Error("[job] 更新任务 %s 状态失败: %v", job.ID, err)
	}
	if err := s.storage.UpdateRun(ctx, run); err != nil {
		s.logger.Error("[job] 更新执行记录 %s 失败: %v", run.ID, err)
	}
}

// GetHandler 获取已注册的处理器
func (s *DefaultScheduler) GetHandler(name string) (Handler, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.handlers[name]
	return h, ok
}
