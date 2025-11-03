package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	config   *Config
	cron     *cron.Cron
	handlers map[string]*JobHandler
	entries  map[string]cron.EntryID
	stats    map[string]*JobStats
	logger   Logger
	mu       sync.RWMutex
	running  bool
}

var (
	// globalScheduler 全局调度器实例
	globalScheduler *Scheduler
	once            sync.Once
)

// New 创建新的调度器实例
func New(cfg *Config, logger Logger) (*Scheduler, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 如果未启用，返回禁用的调度器
	if !cfg.Enabled {
		return &Scheduler{
			config:   cfg,
			handlers: make(map[string]*JobHandler),
			entries:  make(map[string]cron.EntryID),
			stats:    make(map[string]*JobStats),
			logger:   logger,
			running:  false,
		}, nil
	}

	// 加载时区
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	// 创建 cron 选项
	opts := []cron.Option{
		cron.WithLocation(location),
	}

	// 是否启用秒字段
	if cfg.WithSeconds {
		opts = append(opts, cron.WithSeconds())
	}

	// 添加日志
	if logger != nil {
		opts = append(opts, cron.WithLogger(newCronLogger(logger)))
	}

	// 添加中间件链
	chain := []cron.JobWrapper{}

	// Panic 恢复
	if cfg.RecoverPanic {
		chain = append(chain, newRecoverWrapper(logger))
	}

	// 跳过仍在运行的任务
	if cfg.SkipIfStillRunning {
		chain = append(chain, newSkipIfStillRunningWrapper(logger))
	}

	if len(chain) > 0 {
		opts = append(opts, cron.WithChain(chain...))
	}

	s := &Scheduler{
		config:   cfg,
		cron:     cron.New(opts...),
		handlers: make(map[string]*JobHandler),
		entries:  make(map[string]cron.EntryID),
		stats:    make(map[string]*JobStats),
		logger:   logger,
		running:  false,
	}

	return s, nil
}

// InitGlobal 初始化全局调度器（单例模式）
func InitGlobal(cfg *Config, logger Logger) error {
	var err error
	once.Do(func() {
		globalScheduler, err = New(cfg, logger)
	})
	return err
}

// GetGlobal 获取全局调度器实例
func GetGlobal() *Scheduler {
	return globalScheduler
}

// RegisterJob 注册任务
func (s *Scheduler) RegisterJob(name string, handler Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查任务是否已注册
	if _, exists := s.handlers[name]; exists {
		return ErrJobAlreadyExists
	}

	// 获取任务配置
	jobConfig := s.config.GetJobByName(name)
	if jobConfig == nil {
		// 如果配置中没有该任务，创建一个默认配置
		jobConfig = &JobConfig{
			Name:    name,
			Enabled: false,
		}
	}

	// 创建任务处理器
	jobHandler := NewJobHandler(name, handler, jobConfig)
	s.handlers[name] = jobHandler

	// 初始化统计信息
	s.stats[name] = NewJobStats(name)

	if s.logger != nil {
		s.logger.Info(fmt.Sprintf("Job registered: %s", name))
	}

	return nil
}

// RegisterJobFunc 注册任务函数
func (s *Scheduler) RegisterJobFunc(name string, fn JobFunc) error {
	return s.RegisterJob(name, fn)
}

// UnregisterJob 注销任务
func (s *Scheduler) UnregisterJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查任务是否存在
	if _, exists := s.handlers[name]; !exists {
		return ErrJobNotFound
	}

	// 如果任务正在运行，先移除
	if entryID, exists := s.entries[name]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, name)
	}

	// 删除任务处理器
	delete(s.handlers, name)

	if s.logger != nil {
		s.logger.Info(fmt.Sprintf("Job unregistered: %s", name))
	}

	return nil
}

// AddJob 添加任务到调度器
func (s *Scheduler) AddJob(name, spec string, handler Job) error {
	// 先注册任务
	if err := s.RegisterJob(name, handler); err != nil && err != ErrJobAlreadyExists {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	jobHandler := s.handlers[name]

	// 创建包装的任务
	wrappedJob := s.wrapJob(jobHandler)

	// 添加到 cron
	entryID, err := s.cron.AddJob(spec, wrappedJob)
	if err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}

	s.entries[name] = entryID

	if s.logger != nil {
		s.logger.Info(fmt.Sprintf("Job added: %s (spec: %s)", name, spec))
	}

	return nil
}

// RemoveJob 从调度器移除任务
func (s *Scheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.entries[name]
	if !exists {
		return ErrJobNotFound
	}

	s.cron.Remove(entryID)
	delete(s.entries, name)

	if s.logger != nil {
		s.logger.Info(fmt.Sprintf("Job removed: %s", name))
	}

	return nil
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	if !s.config.Enabled {
		return ErrCronDisabled
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrCronAlreadyStarted
	}

	// 加载配置中的任务
	if err := s.loadJobs(); err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	// 启动 cron
	s.cron.Start()
	s.running = true

	if s.logger != nil {
		s.logger.Info("Cron scheduler started")
	}

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrCronNotStarted
	}

	// 停止 cron
	ctx := s.cron.Stop()
	<-ctx.Done()

	s.running = false

	if s.logger != nil {
		s.logger.Info("Cron scheduler stopped")
	}

	return nil
}

// IsRunning 检查调度器是否正在运行
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// loadJobs 加载配置中的任务
func (s *Scheduler) loadJobs() error {
	for _, jobConfig := range s.config.GetEnabledJobs() {
		// 检查任务是否已注册
		handler, exists := s.handlers[jobConfig.Name]
		if !exists {
			if s.logger != nil {
				s.logger.Warn(fmt.Sprintf("Job handler not registered: %s, skipping", jobConfig.Name))
			}
			continue
		}

		// 更新任务配置
		handler.config = &jobConfig

		// 创建包装的任务
		wrappedJob := s.wrapJob(handler)

		// 添加到 cron
		entryID, err := s.cron.AddJob(jobConfig.Spec, wrappedJob)
		if err != nil {
			return fmt.Errorf("failed to add job %s: %w", jobConfig.Name, err)
		}

		s.entries[jobConfig.Name] = entryID

		if s.logger != nil {
			s.logger.Info(fmt.Sprintf("Job loaded: %s (spec: %s)", jobConfig.Name, jobConfig.Spec))
		}
	}

	return nil
}

// wrapJob 包装任务，添加统计、超时、重试等功能
func (s *Scheduler) wrapJob(handler *JobHandler) cron.Job {
	return cron.FuncJob(func() {
		jobName := handler.Name()
		result := NewJobResult(jobName)

		// 创建 context
		ctx := context.Background()

		// 设置超时
		timeout := s.config.DefaultTimeout
		if handler.Config().Timeout > 0 {
			timeout = handler.Config().Timeout
		}

		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		// 执行任务（带重试）
		retryCount := s.config.RetryCount
		if handler.Config().RetryCount > 0 {
			retryCount = handler.Config().RetryCount
		}

		var err error
		for i := 0; i <= retryCount; i++ {
			if i > 0 {
				result.RetryCount = i
				if s.logger != nil {
					s.logger.Info(fmt.Sprintf("Job retry: %s (attempt %d/%d)", jobName, i, retryCount))
				}
				time.Sleep(s.config.RetryInterval)
			}

			err = handler.Execute(ctx)
			if err == nil {
				break
			}

			// 检查是否超时
			if ctx.Err() == context.DeadlineExceeded {
				err = ErrJobTimeout
				break
			}
		}

		// 完成任务
		result.Finish(err)

		// 更新统计
		s.mu.Lock()
		if stats, exists := s.stats[jobName]; exists {
			stats.Update(result)
		}
		s.mu.Unlock()

		// 记录日志
		if s.logger != nil {
			if result.Success {
				s.logger.Info(fmt.Sprintf("Job completed: %s (duration: %v)", jobName, result.Duration))
			} else {
				s.logger.Error(fmt.Sprintf("Job failed: %s (duration: %v, error: %v)", jobName, result.Duration, err))
			}
		}
	})
}

// GetJobStats 获取任务统计信息
func (s *Scheduler) GetJobStats(name string) (*JobStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats, exists := s.stats[name]
	if !exists {
		return nil, ErrJobNotFound
	}

	return stats, nil
}

// GetAllJobStats 获取所有任务统计信息
func (s *Scheduler) GetAllJobStats() map[string]*JobStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*JobStats, len(s.stats))
	for name, stats := range s.stats {
		result[name] = stats
	}

	return result
}

// GetEntries 获取所有任务条目
func (s *Scheduler) GetEntries() []cron.Entry {
	return s.cron.Entries()
}

// GetEntry 获取指定任务的条目
func (s *Scheduler) GetEntry(name string) *cron.Entry {
	s.mu.RLock()
	entryID, exists := s.entries[name]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	entry := s.cron.Entry(entryID)
	return &entry
}

