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
	storage      Storage
	cron         *cron.Cron
	config       *Config
	logger       Logger
	semaphore    chan struct{}  // 并发安全的信号量
	stopChan     chan struct{}  // 并发安全的停止信号
	stopOnce     sync.Once      // 确保只关闭一次
	wg           sync.WaitGroup // 等待所有任务完成
	batchUpdater *BatchUpdater  // 批量更新器（可选）
	cache        *LRUCache      // LRU 缓存（可选）
	metrics      *Metrics       // 性能监控指标

	// 以下字段受 mu 保护
	mu          sync.RWMutex
	handlers    map[string]Handler      // 已注册的处理器
	started     bool                    // 调度器是否已启动
	jobRegistry map[string]*Job         // 本地任务缓存
	cronEntries map[string]cron.EntryID // jobID -> cron entry ID
	runningJobs map[string]bool         // 正在执行的任务ID集合
	jobHeap     *jobHeap                // 任务优先队列（按 NextRunAt 排序）
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

	s := &DefaultScheduler{
		storage:     storage,
		handlers:    make(map[string]Handler),
		cron:        cron.New(cron.WithSeconds()),
		stopChan:    make(chan struct{}),
		jobRegistry: make(map[string]*Job),
		cronEntries: make(map[string]cron.EntryID),
		runningJobs: make(map[string]bool),
		jobHeap:     newJobHeap(),
		semaphore:   make(chan struct{}, config.ConcurrentRuns),
		config:      config,
		logger:      config.Logger,
		metrics:     NewMetrics(),
	}

	// 启用批量更新器（可选）
	if config.EnableBatchUpdate {
		s.batchUpdater = NewBatchUpdater(storage, config.Logger, config.BatchSize, config.BatchFlushInterval)
	}

	// 启用 LRU 缓存（可选）
	if config.EnableCache {
		s.cache = NewLRUCache(config.CacheCapacity, config.CacheTTL, storage, config.Logger)
	}

	return s
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

	// 添加到优先队列（仅非 Cron 任务）
	if j.Type != JobTypeCron && j.NextRunAt != nil {
		s.jobHeap.Add(j)
	}
	heapSize := s.jobHeap.Size()
	s.mu.Unlock()

	// 记录指标
	s.metrics.RecordJobAdded()
	s.metrics.UpdateHeapSize(int64(heapSize))

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

		// 从优先队列中移除
		if job.Type != JobTypeCron {
			s.jobHeap.Remove(id)
		}
	}
	delete(s.jobRegistry, id)
	s.mu.Unlock()

	// 从存储中删除
	return s.storage.DeleteJob(ctx, id)
}

// PauseJob 暂停任务（两阶段提交）
func (s *DefaultScheduler) PauseJob(ctx context.Context, id string) error {
	// 阶段 1：检查任务存在并克隆
	s.mu.RLock()
	job, exists := s.jobRegistry[id]
	if !exists {
		s.mu.RUnlock()
		return ErrJobNotFound
	}

	// 检查是否可以暂停
	if job.Status == JobStatusPaused {
		s.mu.RUnlock()
		return nil // 已暂停，直接返回
	}

	// 在持有读锁时克隆任务，避免竞态
	jobCopy := job.Clone()
	s.mu.RUnlock()

	// 修改克隆的状态
	jobCopy.Status = JobStatusPaused

	// 阶段 2：先更新存储（不持有锁）
	if err := s.storage.UpdateJob(ctx, jobCopy); err != nil {
		return err
	}

	// 阶段 3：存储更新成功后，再更新内存状态
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查任务是否仍存在（可能已被删除）
	job, exists = s.jobRegistry[id]
	if !exists {
		return ErrJobNotFound
	}

	// 原地修改状态，不删除对象
	job.Status = JobStatusPaused

	// 清理 cron entry
	if entryID, ok := s.cronEntries[id]; ok && s.started {
		s.cron.Remove(entryID)
		delete(s.cronEntries, id)
	}

	// 从优先队列中移除
	if job.Type != JobTypeCron {
		s.jobHeap.Remove(id)
	}

	// 更新缓存
	if s.cache != nil {
		s.cache.Set(id, job)
	}

	return nil
}

// ResumeJob 恢复任务（两阶段提交）
func (s *DefaultScheduler) ResumeJob(ctx context.Context, id string) error {
	// 阶段 1：从存储加载最新状态
	job, err := s.storage.GetJob(ctx, id)
	if err != nil {
		return err
	}

	// 验证任务状态（必须是 Paused）
	if job.Status != JobStatusPaused {
		return NewError(ErrCodeJobPaused, "job is not paused", nil)
	}

	// 验证处理器存在
	s.mu.RLock()
	_, handlerExists := s.handlers[job.HandlerName]
	s.mu.RUnlock()

	if !handlerExists {
		return NewError(ErrCodeHandlerNotFound, "handler not found: "+job.HandlerName, nil)
	}

	// 阶段 2：修改状态并计算下次执行时间
	job.Status = JobStatusPending
	if err := s.scheduleNextRun(job); err != nil {
		return err
	}

	// 阶段 3：先更新存储（不持有锁）
	if err := s.storage.UpdateJob(ctx, job); err != nil {
		return err
	}

	// 阶段 4：存储成功后，更新内存状态
	s.mu.Lock()
	defer s.mu.Unlock()

	// 原地修改或添加到注册表
	if existingJob, exists := s.jobRegistry[id]; exists {
		// 原地修改状态
		existingJob.Status = JobStatusPending
		existingJob.NextRunAt = job.NextRunAt
	} else {
		// 不存在则添加
		s.jobRegistry[id] = job
	}

	// 重新添加到调度器
	if job.Type == JobTypeCron && job.Cron != "" && s.started {
		entryID, err := s.cron.AddFunc(job.Cron, s.makeCronCallback(job.ID))
		if err != nil {
			s.logger.Error("[job] resume 添加 cron 任务 %s 失败: %v", job.ID, err)
			return err
		}
		s.cronEntries[job.ID] = entryID
	}

	// 添加到优先队列（仅非 Cron 任务）
	if job.Type != JobTypeCron && job.NextRunAt != nil {
		s.jobHeap.Add(job)
	}

	// 更新缓存
	if s.cache != nil {
		s.cache.Set(id, job)
	}

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

	// 通过参数传递避免闭包捕获变量
	s.wg.Add(1)
	go func(jobID string) {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.runningJobs, jobID)
			s.mu.Unlock()
		}()

		// 使用独立 context，避免调用方 context 取消导致任务中断
		jobCtx, cancel := context.WithTimeout(context.Background(), s.config.JobTimeout)
		defer cancel()

		j, err := s.storage.GetJob(jobCtx, jobID)
		if err != nil {
			s.logger.Error("[job] TriggerJob 获取任务 %s 失败: %v", jobID, err)
			return
		}
		s.executeJob(jobCtx, j)
	}(id)

	return nil
}

// GetJob 获取任务
func (s *DefaultScheduler) GetJob(ctx context.Context, id string) (*Job, error) {
	// 优先从缓存获取
	if s.cache != nil {
		job, err := s.cache.Get(ctx, id)
		if err == nil {
			s.metrics.RecordCacheHit()
			return job, nil
		}
		s.metrics.RecordCacheMiss()
	}

	// 缓存未启用或未命中，直接从存储获取
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
	s.stopChan = make(chan struct{})
	s.stopOnce = sync.Once{}
	s.started = true
	s.mu.Unlock()

	// 启动批量更新器（如果启用）
	if s.batchUpdater != nil {
		s.batchUpdater.Start()
	}

	// 启动缓存清理（如果启用）
	if s.cache != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.cache.StartCleanup(s.config.CacheCleanupInterval, s.stopChan)
		}()
	}

	// 恢复所有待执行的任务
	jobs, err := s.storage.ListJobs(ctx, JobStatusPending)
	if err != nil {
		return err
	}

	for _, j := range jobs {
		if err := s.scheduleJob(j); err != nil {
			s.logger.Error("[job] 启动任务 %s 失败: %v", j.ID, err)
		}

		// 添加到优先队列（仅非 Cron 任务）
		if j.Type != JobTypeCron && j.NextRunAt != nil {
			s.mu.Lock()
			s.jobHeap.Add(j)
			s.mu.Unlock()
		}

		// 添加到缓存
		if s.cache != nil {
			s.cache.Set(j.ID, j)
		}
	}

	// 恢复崩溃残留的 running 状态任务
	runningJobs, err := s.storage.ListJobs(ctx, JobStatusRunning)
	if err != nil {
		s.logger.Error("[job] 恢复 running 任务失败: %v", err)
	} else {
		for _, j := range runningJobs {
			// 重置为 pending 状态
			j.Status = JobStatusPending
			if err := s.storage.UpdateJob(ctx, j); err != nil {
				s.logger.Error("[job] 重置任务 %s 状态失败: %v", j.ID, err)
				continue
			}

			if err := s.scheduleJob(j); err != nil {
				s.logger.Error("[job] 启动任务 %s 失败: %v", j.ID, err)
			}

			if j.Type != JobTypeCron && j.NextRunAt != nil {
				s.mu.Lock()
				s.jobHeap.Add(j)
				s.mu.Unlock()
			}

			if s.cache != nil {
				s.cache.Set(j.ID, j)
			}
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

	// 停止批量更新器（如果启用）
	if s.batchUpdater != nil {
		s.batchUpdater.Stop()
	}

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
		// 间隔任务：基于上次执行时间或当前时间推进
		if job.Interval <= 0 {
			return NewError(ErrCodeInvalidCron, "interval is required for interval job", nil)
		}
		if job.NextRunAt == nil || job.NextRunAt.Before(now) {
			next := now.Add(job.Interval.Unwrap())
			job.NextRunAt = &next
		}
	}

	return nil
}

// makeCronCallback 创建 cron 回调函数
func (s *DefaultScheduler) makeCronCallback(jobID string) func() {
	return func() {
		jobCtx, cancel := context.WithTimeout(context.Background(), s.config.JobTimeout)
		defer cancel()

		j, err := s.storage.GetJob(jobCtx, jobID)
		if err != nil {
			s.logger.Error("[job] cron 回调获取任务 %s 失败: %v", jobID, err)
			return
		}
		s.executeJob(jobCtx, j)
	}
}

// scheduleJob 将任务添加到调度器
func (s *DefaultScheduler) scheduleJob(job *Job) error {
	// 验证处理器是否存在
	s.mu.RLock()
	_, ok := s.handlers[job.HandlerName]
	s.mu.RUnlock()
	if !ok {
		return NewError(ErrCodeHandlerNotFound, "handler not found: "+job.HandlerName, nil)
	}

	switch job.Type {
	case JobTypeCron:
		entryID, err := s.cron.AddFunc(job.Cron, s.makeCronCallback(job.ID))
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.cronEntries[job.ID] = entryID
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

	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.processDueJobs(ctx, now)
		case <-cleanupTicker.C:
			s.mu.Lock()
			count := s.jobHeap.CleanCompleted()
			heapSize := s.jobHeap.Size()
			s.mu.Unlock()
			if count > 0 {
				s.logger.Debug("[job] 清理已完成任务: %d 个", count)
				s.metrics.UpdateHeapSize(int64(heapSize))
			}
		}
	}
}

// processDueJobs 处理到期的任务（优化版：使用优先队列）
func (s *DefaultScheduler) processDueJobs(ctx context.Context, now time.Time) {
	s.mu.Lock()
	// 从优先队列中弹出所有到期的任务
	dueJobs := s.jobHeap.PopDue(now)

	if len(dueJobs) == 0 {
		s.mu.Unlock()
		return
	}

	// 过滤掉已在运行的任务
	var readyJobs []*Job
	for _, job := range dueJobs {
		// 检查任务是否存在于注册表且未在运行
		if registryJob, exists := s.jobRegistry[job.ID]; exists {
			if registryJob.Status == JobStatusPending && !s.runningJobs[job.ID] {
				readyJobs = append(readyJobs, registryJob)
				s.runningJobs[job.ID] = true
			} else {
				// 任务状态不符合执行条件，重新加入堆
				if registryJob.NextRunAt != nil {
					s.jobHeap.Add(registryJob)
				}
			}
		}
	}
	s.mu.Unlock()

	// 异步执行任务，避免阻塞调度器（使用 WaitGroup 追踪）
	for _, job := range readyJobs {
		s.wg.Add(1)
		go func(j *Job) {
			defer s.wg.Done()
			jobCtx, cancel := context.WithTimeout(context.Background(), s.config.JobTimeout)
			defer cancel()
			s.executeJob(jobCtx, j)
		}(job)
	}
}

// executeJob 执行任务
func (s *DefaultScheduler) executeJob(ctx context.Context, j *Job) {
	// 尝试获取信号量，如果并发数已满则跳过
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	default:
		// 并发数已满，将任务重新加入堆等待下次调度
		s.mu.Lock()
		delete(s.runningJobs, j.ID)
		if registryJob, exists := s.jobRegistry[j.ID]; exists {
			if registryJob.NextRunAt != nil {
				s.jobHeap.Add(registryJob)
			}
		}
		s.mu.Unlock()
		s.logger.Warn("[job] 任务 %s 跳过执行（并发数已满）", j.ID)
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

	// 加锁检查状态和获取处理器
	s.mu.Lock()

	handler, ok := s.handlers[j.HandlerName]
	if !ok {
		delete(s.runningJobs, j.ID)
		s.mu.Unlock()
		span.SetStatus(codes.Error, "handler not found")
		return
	}

	// 检查注册表中的实际状态
	registryJob, exists := s.jobRegistry[j.ID]
	if !exists {
		delete(s.runningJobs, j.ID)
		s.mu.Unlock()
		span.SetStatus(codes.Error, "job not found in registry")
		return
	}

	if registryJob.Status == JobStatusRunning {
		delete(s.runningJobs, j.ID)
		s.mu.Unlock()
		span.SetStatus(codes.Ok, "already running")
		return
	}

	// 更新注册表状态
	registryJob.Status = JobStatusRunning
	now := time.Now()
	registryJob.LastRunAt = &now

	// 克隆任务用于后续处理（在释放锁前）
	job := registryJob.Clone()

	// 释放锁，避免在 I/O 操作时持有锁
	s.mu.Unlock()

	// 更新存储（不持有锁）
	if err := s.storage.UpdateJob(ctx, job); err != nil {
		s.logger.Error("[job] 更新任务 %s 状态失败: %v，尝试恢复", job.ID, err)
		s.recoverJobState(job.ID, JobStatusPending)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	// 创建执行记录（不持有锁）
	run := &Run{
		ID:      uuid.New().String(),
		JobID:   job.ID,
		Status:  RunStatusRunning,
		StartAt: now,
		TraceID: span.SpanContext().TraceID().String(),
	}

	if err := s.storage.CreateRun(ctx, run); err != nil {
		s.logger.Error("[job] 创建执行记录失败: %v，尝试恢复", err)
		s.recoverJobState(job.ID, JobStatusPending)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	// 执行任务（不在锁内执行，避免阻塞）
	span.AddEvent("handler_executing")
	startTime := time.Now()
	output, execErr := handler.Execute(ctx, job.Payload)
	duration := time.Since(startTime).Milliseconds()

	// 记录执行指标
	if execErr != nil {
		s.metrics.RecordRunFailed(duration)
		s.metrics.RecordHandlerError()
	} else {
		s.metrics.RecordRunSuccess(duration)
	}

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

		// 如果任务需要重新调度，添加到优先队列
		if job.Type != JobTypeCron && job.Status == JobStatusPending && job.NextRunAt != nil {
			s.jobHeap.Update(registryJob)
		}
	}

	delete(s.runningJobs, job.ID)

	// Clone for persistence (while still holding lock)
	jobForPersist := job.Clone()
	runForPersist := *run

	s.mu.Unlock()

	// 设置执行结果属性
	span.SetAttributes(
		attribute.Int64("run.duration_ms", duration),
		attribute.String("run.status", string(runForPersist.Status)),
		attribute.Int("job.retry_count", jobForPersist.RetryCount),
	)

	if execErr != nil {
		span.RecordError(execErr)
		span.SetStatus(codes.Error, execErr.Error())
	}

	// 使用批量更新器（如果启用）
	if s.batchUpdater != nil {
		s.batchUpdater.UpdateJob(ctx, jobForPersist)
		s.batchUpdater.UpdateRun(ctx, &runForPersist)
	} else {
		// 同步更新
		if err := s.storage.UpdateJob(ctx, jobForPersist); err != nil {
			s.logger.Error("[job] 更新任务 %s 状态失败: %v", jobForPersist.ID, err)
		}
		if err := s.storage.UpdateRun(ctx, &runForPersist); err != nil {
			s.logger.Error("[job] 更新执行记录 %s 失败: %v", runForPersist.ID, err)
		}
	}
}

// GetMetrics 获取性能指标
func (s *DefaultScheduler) GetMetrics() *Metrics {
	// 更新实时指标
	s.mu.RLock()
	heapSize := s.jobHeap.Size()
	s.mu.RUnlock()
	s.metrics.UpdateHeapSize(int64(heapSize))
	if s.cache != nil {
		s.metrics.UpdateCacheSize(int64(s.cache.Size()))
	}

	return s.metrics
}

// GetHandler 获取已注册的处理器
func (s *DefaultScheduler) GetHandler(name string) (Handler, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.handlers[name]
	return h, ok
}

// recoverJobState 恢复任务状态（错误处理辅助函数）
func (s *DefaultScheduler) recoverJobState(jobID string, status JobStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if regJob, exists := s.jobRegistry[jobID]; exists {
		regJob.Status = status
	}
	delete(s.runningJobs, jobID)
}
