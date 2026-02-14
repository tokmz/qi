package job

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockBatchStorage 用于测试批量更新的存储
type mockBatchStorage struct {
	mu           sync.RWMutex
	jobs         map[string]*Job
	runs         map[string]*Run
	jobUpdates   int // 记录 UpdateJob 调用次数
	runUpdates   int // 记录 UpdateRun 调用次数
	updateErrors bool
}

func newMockBatchStorage() *mockBatchStorage {
	return &mockBatchStorage{
		jobs: make(map[string]*Job),
		runs: make(map[string]*Run),
	}
}

func (m *mockBatchStorage) CreateJob(ctx context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j.Clone()
	return nil
}

func (m *mockBatchStorage) GetJob(ctx context.Context, id string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if j, ok := m.jobs[id]; ok {
		return j.Clone(), nil
	}
	return nil, ErrJobNotFound
}

func (m *mockBatchStorage) UpdateJob(ctx context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErrors {
		return ErrJobNotFound
	}
	m.jobs[j.ID] = j.Clone()
	m.jobUpdates++
	return nil
}

func (m *mockBatchStorage) DeleteJob(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.jobs, id)
	return nil
}

func (m *mockBatchStorage) ListJobs(ctx context.Context, status JobStatus) ([]*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jobs := make([]*Job, 0)
	for _, j := range m.jobs {
		if status == "" || j.Status == status {
			jobs = append(jobs, j.Clone())
		}
	}
	return jobs, nil
}

func (m *mockBatchStorage) CreateRun(ctx context.Context, r *Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[r.ID] = r
	return nil
}

func (m *mockBatchStorage) UpdateRun(ctx context.Context, r *Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErrors {
		return ErrJobNotFound
	}
	m.runs[r.ID] = r
	m.runUpdates++
	return nil
}

func (m *mockBatchStorage) GetRuns(ctx context.Context, jobID string, limit int) ([]*Run, error) {
	return nil, nil
}

func (m *mockBatchStorage) GetJobRunCount(ctx context.Context, jobID string) (int64, error) {
	return 0, nil
}

func (m *mockBatchStorage) Close() error                                  { return nil }
func (m *mockBatchStorage) Ping(ctx context.Context) error                { return nil }
func (m *mockBatchStorage) GetNextRunTime(ctx context.Context, status JobStatus) ([]*Job, error) {
	return nil, nil
}

// getUpdateCounts 获取更新次数（线程安全）
func (m *mockBatchStorage) getUpdateCounts() (int, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jobUpdates, m.runUpdates
}

// setUpdateErrors 设置是否模拟更新错误（线程安全）
func (m *mockBatchStorage) setUpdateErrors(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateErrors = v
}

// TestBatchUpdaterBasic 测试批量更新器基本功能
func TestBatchUpdaterBasic(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 5
	flushInterval := 100 * time.Millisecond
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	// 添加少量任务（不触发批量刷新）
	for i := range 3 {
		job := &Job{
			ID:     string(rune('0' + i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 等待自动刷新
	time.Sleep(200 * time.Millisecond)

	jobUpdates, _ := store.getUpdateCounts()
	assert.Equal(t, 3, jobUpdates, "应该更新 3 个任务")
}

// TestBatchUpdaterBatchSize 测试批量大小触发
func TestBatchUpdaterBatchSize(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 5
	flushInterval := time.Second // 长间隔，不依赖时间触发
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	// 添加正好 batchSize 个任务
	for i := range batchSize {
		job := &Job{
			ID:     string(rune('0' + i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 短暂等待批量处理
	time.Sleep(100 * time.Millisecond)

	jobUpdates, _ := store.getUpdateCounts()
	assert.Equal(t, batchSize, jobUpdates, "达到批量大小应该立即刷新")
}

// TestBatchUpdaterRuns 测试执行记录批量更新
func TestBatchUpdaterRuns(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 5
	flushInterval := 100 * time.Millisecond
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	// 添加执行记录
	for i := range 3 {
		run := &Run{
			ID:     string(rune('0' + i)),
			JobID:  "test-job",
			Status: RunStatusRunning,
		}
		updater.UpdateRun(context.Background(), run)
	}

	// 等待自动刷新
	time.Sleep(200 * time.Millisecond)

	_, runUpdates := store.getUpdateCounts()
	assert.Equal(t, 3, runUpdates, "应该更新 3 个执行记录")
}

// TestBatchUpdaterMixed 测试混合更新
func TestBatchUpdaterMixed(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 10
	flushInterval := 100 * time.Millisecond
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	// 混合添加任务和执行记录
	for i := range 5 {
		job := &Job{
			ID:     "job-" + string(rune('0'+i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)

		run := &Run{
			ID:     "run-" + string(rune('0'+i)),
			JobID:  job.ID,
			Status: RunStatusRunning,
		}
		updater.UpdateRun(context.Background(), run)
	}

	// 等待自动刷新
	time.Sleep(200 * time.Millisecond)

	jobUpdates, runUpdates := store.getUpdateCounts()
	assert.Equal(t, 5, jobUpdates, "应该更新 5 个任务")
	assert.Equal(t, 5, runUpdates, "应该更新 5 个执行记录")
}

// TestBatchUpdaterQueueFull 测试队列满时的降级处理
func TestBatchUpdaterQueueFull(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 2
	flushInterval := time.Second // 长间隔，让队列填满
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	// 快速添加大量任务，填满队列
	for i := range 100 {
		job := &Job{
			ID:     string(rune('0' + i%10)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	jobUpdates, _ := store.getUpdateCounts()
	assert.Greater(t, jobUpdates, 0, "即使队列满也应该有更新（通过同步降级）")
}

// TestBatchUpdaterStop 测试停止时刷新剩余数据
func TestBatchUpdaterStop(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 10
	flushInterval := time.Hour // 很长的间隔，不会自动刷新
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()

	// 添加少量任务
	for i := range 3 {
		job := &Job{
			ID:     string(rune('0' + i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 短暂等待确保任务进入队列
	time.Sleep(50 * time.Millisecond)

	// 立即停止（应该刷新剩余数据）
	updater.Stop()

	jobUpdates, _ := store.getUpdateCounts()
	assert.Equal(t, 3, jobUpdates, "停止时应该刷新所有剩余数据")
}

// TestBatchUpdaterConcurrent 测试并发更新
func TestBatchUpdaterConcurrent(t *testing.T) {
	store := newMockBatchStorage()
	batchSize := 10
	flushInterval := 100 * time.Millisecond
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	var wg sync.WaitGroup
	concurrency := 10
	updatesPerGoroutine := 10

	// 并发更新
	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := range updatesPerGoroutine {
				job := &Job{
					ID:     string(rune('0' + (idx*updatesPerGoroutine + j))),
					Status: JobStatusPending,
				}
				updater.UpdateJob(context.Background(), job)
			}
		}(i)
	}

	wg.Wait()

	// 等待所有批次刷新
	time.Sleep(300 * time.Millisecond)

	jobUpdates, _ := store.getUpdateCounts()
	assert.Equal(t, concurrency*updatesPerGoroutine, jobUpdates,
		"所有并发更新都应该被处理")
}

// TestBatchUpdaterDefaultValues 测试默认值
func TestBatchUpdaterDefaultValues(t *testing.T) {
	store := newMockBatchStorage()

	// 使用无效的配置值
	updater := NewBatchUpdater(store, &NopLogger{}, 0, 0)

	// 验证使用了默认值（通过能正常工作来验证）
	updater.Start()
	defer updater.Stop()

	job := &Job{ID: "test", Status: JobStatusPending}
	updater.UpdateJob(context.Background(), job)

	// 等待足够长的时间让默认刷新间隔触发
	time.Sleep(1500 * time.Millisecond)

	jobUpdates, _ := store.getUpdateCounts()
	assert.Equal(t, 1, jobUpdates, "应该使用默认配置正常工作")
}

// TestBatchUpdaterErrorHandling 测试错误处理
func TestBatchUpdaterErrorHandling(t *testing.T) {
	store := newMockBatchStorage()
	store.setUpdateErrors(true) // 模拟更新错误

	batchSize := 5
	flushInterval := 100 * time.Millisecond
	updater := NewBatchUpdater(store, &NopLogger{}, batchSize, flushInterval)

	updater.Start()
	defer updater.Stop()

	// 添加任务（会失败，但不应该崩溃）
	for i := range 3 {
		job := &Job{
			ID:     string(rune('0' + i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	// 验证更新器仍在运行（通过能继续添加来验证）
	store.setUpdateErrors(false)
	job := &Job{ID: "test", Status: JobStatusPending}
	updater.UpdateJob(context.Background(), job)

	time.Sleep(200 * time.Millisecond)
	jobUpdates, _ := store.getUpdateCounts()
	assert.Greater(t, jobUpdates, 0, "错误恢复后应该能继续工作")
}

// TestBatchUpdater_StopSafety 测试停止后的安全性
func TestBatchUpdater_StopSafety(t *testing.T) {
	store := newMockBatchStorage()
	updater := NewBatchUpdater(store, &NopLogger{}, 10, 100*time.Millisecond)

	updater.Start()

	// 添加一些任务
	for i := 0; i < 5; i++ {
		job := &Job{
			ID:     string(rune('0' + i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 停止更新器
	updater.Stop()

	// 停止后尝试发送（不应该 panic）
	for i := 0; i < 10; i++ {
		job := &Job{
			ID:     "after-stop",
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job) // 应该同步更新，不会 panic
	}

	// 验证停止后的更新被同步处理
	jobUpdates, _ := store.getUpdateCounts()
	assert.Greater(t, jobUpdates, 0, "停止后的更新应该被同步处理")
}

// TestBatchUpdater_MultipleStop 测试多次停止
func TestBatchUpdater_MultipleStop(t *testing.T) {
	store := newMockBatchStorage()
	updater := NewBatchUpdater(store, &NopLogger{}, 10, 100*time.Millisecond)

	updater.Start()

	// 多次调用 Stop（不应该 panic）
	updater.Stop()
	updater.Stop()
	updater.Stop()

	// 验证没有 panic
	t.Log("多次停止测试通过")
}

// TestBatchUpdater_ConcurrentStop 测试并发停止
func TestBatchUpdater_ConcurrentStop(t *testing.T) {
	store := newMockBatchStorage()
	updater := NewBatchUpdater(store, &NopLogger{}, 10, 100*time.Millisecond)

	updater.Start()

	// 并发调用 Stop
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			updater.Stop()
		}()
	}

	wg.Wait()

	// 验证没有 panic
	t.Log("并发停止测试通过")
}

// TestBatchUpdater_UpdateAfterStop 测试停止后更新的正确性
func TestBatchUpdater_UpdateAfterStop(t *testing.T) {
	store := newMockBatchStorage()
	updater := NewBatchUpdater(store, &NopLogger{}, 10, 100*time.Millisecond)

	updater.Start()
	updater.Stop()

	// 停止后发送更新
	job := &Job{
		ID:     "after-stop",
		Status: JobStatusPending,
	}
	updater.UpdateJob(context.Background(), job)

	run := &Run{
		ID:    "run-after-stop",
		JobID: "after-stop",
	}
	updater.UpdateRun(context.Background(), run)

	// 验证更新被同步处理
	jobUpdates, runUpdates := store.getUpdateCounts()
	assert.Equal(t, 1, jobUpdates, "停止后的 Job 更新应该被同步处理")
	assert.Equal(t, 1, runUpdates, "停止后的 Run 更新应该被同步处理")
}

// TestBatchUpdater_StopWithPendingUpdates 测试停止时有待处理的更新
func TestBatchUpdater_StopWithPendingUpdates(t *testing.T) {
	store := newMockBatchStorage()
	updater := NewBatchUpdater(store, &NopLogger{}, 100, 10*time.Second) // 大批量，长间隔

	updater.Start()

	// 添加一些任务（不会立即刷新）
	for i := 0; i < 10; i++ {
		job := &Job{
			ID:     string(rune('0' + i)),
			Status: JobStatusPending,
		}
		updater.UpdateJob(context.Background(), job)
	}

	// 等待一小段时间确保任务进入队列
	time.Sleep(50 * time.Millisecond)

	// 停止（应该刷新待处理的更新）
	updater.Stop()

	// 验证所有更新都被处理
	jobUpdates, _ := store.getUpdateCounts()
	assert.Equal(t, 10, jobUpdates, "停止时应该刷新所有待处理的更新")
}
