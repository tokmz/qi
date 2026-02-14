package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qi/pkg/job"

	"github.com/google/uuid"
)

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	mu      sync.RWMutex
	jobs    map[string]*job.Job
	runs    map[string]*job.Run // runID -> Run
	jobRuns map[string][]string // jobID -> runIDs
	closed  bool
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		jobs:    make(map[string]*job.Job),
		runs:    make(map[string]*job.Run),
		jobRuns: make(map[string][]string),
	}
}

// CreateJob 创建任务
func (s *MemoryStorage) CreateJob(ctx context.Context, j *job.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	if _, exists := s.jobs[j.ID]; exists {
		return job.ErrJobAlreadyExists
	}

	s.jobs[j.ID] = j.Clone()
	return nil
}

// GetJob 获取任务
func (s *MemoryStorage) GetJob(ctx context.Context, id string) (*job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, job.ErrStorageClosed
	}

	j, exists := s.jobs[id]
	if !exists {
		return nil, job.ErrJobNotFound
	}

	return j.Clone(), nil
}

// UpdateJob 更新任务
func (s *MemoryStorage) UpdateJob(ctx context.Context, j *job.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	if _, exists := s.jobs[j.ID]; !exists {
		return job.ErrJobNotFound
	}

	s.jobs[j.ID] = j.Clone()
	return nil
}

// DeleteJob 删除任务
func (s *MemoryStorage) DeleteJob(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	if _, exists := s.jobs[id]; !exists {
		return job.ErrJobNotFound
	}

	delete(s.jobs, id)

	if runIDs, exists := s.jobRuns[id]; exists {
		for _, runID := range runIDs {
			delete(s.runs, runID)
		}
		delete(s.jobRuns, id)
	}

	return nil
}

// ListJobs 列出任务
func (s *MemoryStorage) ListJobs(ctx context.Context, status job.JobStatus) ([]*job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, job.ErrStorageClosed
	}

	jobs := make([]*job.Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		if status == "" || j.Status == status {
			jobs = append(jobs, j.Clone())
		}
	}

	return jobs, nil
}

// CreateRun 创建执行记录
func (s *MemoryStorage) CreateRun(ctx context.Context, r *job.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	if r.ID == "" {
		r.ID = uuid.New().String()
	}

	runCopy := *r
	s.runs[r.ID] = &runCopy
	s.jobRuns[r.JobID] = append(s.jobRuns[r.JobID], r.ID)

	return nil
}

// UpdateRun 更新执行记录
func (s *MemoryStorage) UpdateRun(ctx context.Context, r *job.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	if _, exists := s.runs[r.ID]; !exists {
		return fmt.Errorf("run not found: %s", r.ID)
	}

	runCopy := *r
	s.runs[r.ID] = &runCopy

	return nil
}

// GetRuns 获取执行记录
func (s *MemoryStorage) GetRuns(ctx context.Context, jobID string, limit int) ([]*job.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, job.ErrStorageClosed
	}

	runIDs, exists := s.jobRuns[jobID]
	if !exists || len(runIDs) == 0 {
		return nil, nil
	}

	if limit <= 0 {
		limit = job.DefaultRunLimit
	}

	results := make([]*job.Run, 0, limit)
	count := 0
	for i := len(runIDs) - 1; i >= 0 && count < limit; i-- {
		if run, ok := s.runs[runIDs[i]]; ok {
			runCopy := *run
			results = append(results, &runCopy)
			count++
		}
	}

	return results, nil
}

// GetJobRunCount 获取任务执行次数
func (s *MemoryStorage) GetJobRunCount(ctx context.Context, jobID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return 0, job.ErrStorageClosed
	}

	return int64(len(s.jobRuns[jobID])), nil
}

// Close 关闭存储
func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	return nil
}

// Ping 检查存储状态
func (s *MemoryStorage) Ping(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	return nil
}

// GetNextRunTime 获取下次执行时间
func (s *MemoryStorage) GetNextRunTime(ctx context.Context, status job.JobStatus) ([]*job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, job.ErrStorageClosed
	}

	jobs := make([]*job.Job, 0)
	now := time.Now()
	for _, j := range s.jobs {
		if status == "" || j.Status == status {
			if j.NextRunAt != nil && j.NextRunAt.Before(now) {
				jobs = append(jobs, j.Clone())
			}
		}
	}

	return jobs, nil
}

// BatchUpdateJobs 批量更新任务（单次加锁）
func (s *MemoryStorage) BatchUpdateJobs(ctx context.Context, jobs []*job.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	for _, j := range jobs {
		if _, exists := s.jobs[j.ID]; !exists {
			return job.ErrJobNotFound
		}
		s.jobs[j.ID] = j.Clone()
	}
	return nil
}

// BatchUpdateRuns 批量更新执行记录（单次加锁）
func (s *MemoryStorage) BatchUpdateRuns(ctx context.Context, runs []*job.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return job.ErrStorageClosed
	}

	for _, r := range runs {
		if _, exists := s.runs[r.ID]; !exists {
			return fmt.Errorf("run not found: %s", r.ID)
		}
		runCopy := *r
		s.runs[r.ID] = &runCopy
	}
	return nil
}
