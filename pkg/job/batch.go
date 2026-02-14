package job

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// jobUpdate 带 trace context 的任务更新
type jobUpdate struct {
	job     *Job
	spanCtx trace.SpanContext
}

// runUpdate 带 trace context 的执行记录更新
type runUpdate struct {
	run     *Run
	spanCtx trace.SpanContext
}

// BatchUpdater 批量更新器，减少数据库访问次数
type BatchUpdater struct {
	storage      Storage
	batchStorage BatchStorage  // 可选，支持批量操作的存储后端
	logger       Logger
	batchSize     int           // 批量大小
	flushInterval time.Duration // 刷新间隔
	jobQueue      chan jobUpdate // 任务更新队列
	runQueue      chan runUpdate // 执行记录更新队列
	stopChan      chan struct{}
	wg            sync.WaitGroup
	stopped       int32 // 原子操作标志位，0=运行中，1=已停止
}

// NewBatchUpdater 创建批量更新器
func NewBatchUpdater(storage Storage, logger Logger, batchSize int, flushInterval time.Duration) *BatchUpdater {
	if batchSize <= 0 {
		batchSize = 10
	}
	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	b := &BatchUpdater{
		storage:       storage,
		logger:        logger,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		jobQueue:      make(chan jobUpdate, batchSize*2),
		runQueue:      make(chan runUpdate, batchSize*2),
		stopChan:      make(chan struct{}),
	}

	// 检测存储后端是否支持批量操作
	if bs, ok := storage.(BatchStorage); ok {
		b.batchStorage = bs
	}

	return b
}

// Start 启动批量更新器
func (b *BatchUpdater) Start() {
	b.wg.Add(2)
	go b.processJobUpdates()
	go b.processRunUpdates()
}

// Stop 停止批量更新器
func (b *BatchUpdater) Stop() {
	// 设置停止标志（原子操作）
	if !atomic.CompareAndSwapInt32(&b.stopped, 0, 1) {
		return // 已经停止，避免重复关闭
	}

	close(b.stopChan)
	b.wg.Wait()
}

// UpdateJob 异步更新任务
func (b *BatchUpdater) UpdateJob(ctx context.Context, job *Job) {
	spanCtx := trace.SpanContextFromContext(ctx)

	// 检查是否已停止
	if atomic.LoadInt32(&b.stopped) == 1 {
		// 已停止，同步更新
		syncCtx, cancel := context.WithTimeout(trace.ContextWithSpanContext(context.Background(), spanCtx), DefaultSyncTimeout)
		defer cancel()
		if err := b.storage.UpdateJob(syncCtx, job); err != nil {
			b.logger.Error("[batch] 同步更新任务失败: %v", err)
		}
		return
	}

	update := jobUpdate{job: job, spanCtx: spanCtx}
	select {
	case b.jobQueue <- update:
	default:
		// 队列已满，同步更新
		syncCtx, cancel := context.WithTimeout(trace.ContextWithSpanContext(context.Background(), spanCtx), DefaultSyncTimeout)
		defer cancel()
		if err := b.storage.UpdateJob(syncCtx, job); err != nil {
			b.logger.Error("[batch] 同步更新任务失败: %v", err)
		}
	}
}

// UpdateRun 异步更新执行记录
func (b *BatchUpdater) UpdateRun(ctx context.Context, run *Run) {
	spanCtx := trace.SpanContextFromContext(ctx)

	// 检查是否已停止
	if atomic.LoadInt32(&b.stopped) == 1 {
		// 已停止，同步更新
		syncCtx, cancel := context.WithTimeout(trace.ContextWithSpanContext(context.Background(), spanCtx), DefaultSyncTimeout)
		defer cancel()
		if err := b.storage.UpdateRun(syncCtx, run); err != nil {
			b.logger.Error("[batch] 同步更新执行记录失败: %v", err)
		}
		return
	}

	update := runUpdate{run: run, spanCtx: spanCtx}
	select {
	case b.runQueue <- update:
	default:
		// 队列已满，同步更新
		syncCtx, cancel := context.WithTimeout(trace.ContextWithSpanContext(context.Background(), spanCtx), DefaultSyncTimeout)
		defer cancel()
		if err := b.storage.UpdateRun(syncCtx, run); err != nil {
			b.logger.Error("[batch] 同步更新执行记录失败: %v", err)
		}
	}
}

// processJobUpdates 处理任务更新
func (b *BatchUpdater) processJobUpdates() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	type pendingJob struct {
		job     *Job
		spanCtx trace.SpanContext
	}
	batch := make([]pendingJob, 0, b.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		// 收集 span links
		links := make([]trace.Link, 0, len(batch))
		for _, item := range batch {
			if item.spanCtx.IsValid() {
				links = append(links, trace.Link{SpanContext: item.spanCtx})
			}
		}

		// 创建 batch.flush span，关联所有原始任务的 trace
		tracer := otel.Tracer(tracerName)
		ctx, span := tracer.Start(context.Background(), "batch.flush.jobs",
			trace.WithLinks(links...),
		)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		defer span.End()

		jobs := make([]*Job, len(batch))
		for i, item := range batch {
			jobs[i] = item.job
		}

		if b.batchStorage != nil {
			if err := b.batchStorage.BatchUpdateJobs(ctx, jobs); err != nil {
				b.logger.Error("[batch] 批量更新任务失败: %v", err)
			}
		} else {
			for _, job := range jobs {
				if err := b.storage.UpdateJob(ctx, job); err != nil {
					b.logger.Error("[batch] 批量更新任务 %s 失败: %v", job.ID, err)
				}
			}
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-b.stopChan:
			// 排空队列中剩余的数据
			for {
				select {
				case update := <-b.jobQueue:
					batch = append(batch, pendingJob{job: update.job, spanCtx: update.spanCtx})
				default:
					flush()
					return
				}
			}
		case <-ticker.C:
			flush()
		case update := <-b.jobQueue:
			batch = append(batch, pendingJob{job: update.job, spanCtx: update.spanCtx})
			if len(batch) >= b.batchSize {
				flush()
			}
		}
	}
}

// processRunUpdates 处理执行记录更新
func (b *BatchUpdater) processRunUpdates() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	type pendingRun struct {
		run     *Run
		spanCtx trace.SpanContext
	}
	batch := make([]pendingRun, 0, b.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		// 收集 span links
		links := make([]trace.Link, 0, len(batch))
		for _, item := range batch {
			if item.spanCtx.IsValid() {
				links = append(links, trace.Link{SpanContext: item.spanCtx})
			}
		}

		// 创建 batch.flush span，关联所有原始任务的 trace
		tracer := otel.Tracer(tracerName)
		ctx, span := tracer.Start(context.Background(), "batch.flush.runs",
			trace.WithLinks(links...),
		)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		defer span.End()

		runs := make([]*Run, len(batch))
		for i, item := range batch {
			runs[i] = item.run
		}

		if b.batchStorage != nil {
			if err := b.batchStorage.BatchUpdateRuns(ctx, runs); err != nil {
				b.logger.Error("[batch] 批量更新执行记录失败: %v", err)
			}
		} else {
			for _, run := range runs {
				if err := b.storage.UpdateRun(ctx, run); err != nil {
					b.logger.Error("[batch] 批量更新执行记录 %s 失败: %v", run.ID, err)
				}
			}
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-b.stopChan:
			// 排空队列中剩余的数据
			for {
				select {
				case update := <-b.runQueue:
					batch = append(batch, pendingRun{run: update.run, spanCtx: update.spanCtx})
				default:
					flush()
					return
				}
			}
		case <-ticker.C:
			flush()
		case update := <-b.runQueue:
			batch = append(batch, pendingRun{run: update.run, spanCtx: update.spanCtx})
			if len(batch) >= b.batchSize {
				flush()
			}
		}
	}
}
