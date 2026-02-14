package job

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage 测试用的简单存储实现
type mockStorage struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		jobs: make(map[string]*Job),
	}
}

func (m *mockStorage) CreateJob(ctx context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j.Clone()
	return nil
}

func (m *mockStorage) GetJob(ctx context.Context, id string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if j, ok := m.jobs[id]; ok {
		return j.Clone(), nil
	}
	return nil, ErrJobNotFound
}

func (m *mockStorage) UpdateJob(ctx context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j.Clone()
	return nil
}

func (m *mockStorage) DeleteJob(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.jobs, id)
	return nil
}

func (m *mockStorage) ListJobs(ctx context.Context, status JobStatus) ([]*Job, error) {
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

func (m *mockStorage) CreateRun(ctx context.Context, r *Run) error   { return nil }
func (m *mockStorage) UpdateRun(ctx context.Context, r *Run) error   { return nil }
func (m *mockStorage) GetRuns(ctx context.Context, jobID string, limit int) ([]*Run, error) {
	return nil, nil
}
func (m *mockStorage) GetJobRunCount(ctx context.Context, jobID string) (int64, error) {
	return 0, nil
}
func (m *mockStorage) Close() error                                  { return nil }
func (m *mockStorage) Ping(ctx context.Context) error                { return nil }
func (m *mockStorage) GetNextRunTime(ctx context.Context, status JobStatus) ([]*Job, error) {
	return nil, nil
}

// TestLRUCacheBasic 测试缓存基本功能
func TestLRUCacheBasic(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(10, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加任务到存储
	job := &Job{
		ID:          "test-1",
		Name:        "test",
		Type:        JobTypeOnce,
		HandlerName: "test",
		Status:      JobStatusPending,
	}
	err := store.CreateJob(ctx, job)
	require.NoError(t, err)

	// 第一次获取（缓存未命中）
	result, err := cache.Get(ctx, "test-1")
	require.NoError(t, err)
	assert.Equal(t, "test-1", result.ID)

	// 第二次获取（缓存命中）
	result, err = cache.Get(ctx, "test-1")
	require.NoError(t, err)
	assert.Equal(t, "test-1", result.ID)

	// 检查缓存大小
	assert.Equal(t, 1, cache.Size())
}

// TestLRUCacheEviction 测试缓存淘汰
func TestLRUCacheEviction(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(3, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加 4 个任务
	for i := 1; i <= 4; i++ {
		job := &Job{
			ID:          string(rune('0' + i)),
			Name:        "test",
			Type:        JobTypeOnce,
			HandlerName: "test",
			Status:      JobStatusPending,
		}
		err := store.CreateJob(ctx, job)
		require.NoError(t, err)

		_, err = cache.Get(ctx, job.ID)
		require.NoError(t, err)
	}

	// 缓存应该只保留 3 个（最新的）
	assert.Equal(t, 3, cache.Size())
}

// TestLRUCacheExpiration 测试缓存过期
func TestLRUCacheExpiration(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(10, 100*time.Millisecond, store, &NopLogger{})

	ctx := context.Background()

	// 添加任务
	job := &Job{
		ID:          "expire-1",
		Name:        "test",
		Type:        JobTypeOnce,
		HandlerName: "test",
		Status:      JobStatusPending,
	}
	err := store.CreateJob(ctx, job)
	require.NoError(t, err)

	// 获取任务（添加到缓存）
	_, err = cache.Get(ctx, "expire-1")
	require.NoError(t, err)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 清理过期缓存
	count := cache.CleanExpired()
	assert.Equal(t, 1, count)
	assert.Equal(t, 0, cache.Size())
}

// TestLRUCacheConcurrent 测试并发访问（防止缓存击穿）
func TestLRUCacheConcurrent(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(10, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加任务
	job := &Job{
		ID:          "concurrent-1",
		Name:        "test",
		Type:        JobTypeOnce,
		HandlerName: "test",
		Status:      JobStatusPending,
	}
	err := store.CreateJob(ctx, job)
	require.NoError(t, err)

	// 并发获取同一个任务
	var wg sync.WaitGroup
	errors := make([]error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := cache.Get(ctx, "concurrent-1")
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// 所有请求都应该成功
	for i, err := range errors {
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}

	// 缓存应该只有一个条目
	assert.Equal(t, 1, cache.Size())
}

// TestLRUCacheDelete 测试删除缓存
func TestLRUCacheDelete(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(10, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加任务
	job := &Job{
		ID:          "delete-1",
		Name:        "test",
		Type:        JobTypeOnce,
		HandlerName: "test",
		Status:      JobStatusPending,
	}
	err := store.CreateJob(ctx, job)
	require.NoError(t, err)

	// 获取任务（添加到缓存）
	_, err = cache.Get(ctx, "delete-1")
	require.NoError(t, err)
	assert.Equal(t, 1, cache.Size())

	// 删除缓存
	cache.Delete("delete-1")
	assert.Equal(t, 0, cache.Size())
}

// TestLRUCacheClear 测试清空缓存
func TestLRUCacheClear(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(10, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加多个任务
	for i := 1; i <= 5; i++ {
		job := &Job{
			ID:          string(rune('0' + i)),
			Name:        "test",
			Type:        JobTypeOnce,
			HandlerName: "test",
			Status:      JobStatusPending,
		}
		err := store.CreateJob(ctx, job)
		require.NoError(t, err)

		_, err = cache.Get(ctx, job.ID)
		require.NoError(t, err)
	}

	assert.Equal(t, 5, cache.Size())

	// 清空缓存
	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

// TestLRUCache_NoDeadlock 测试并发读写不会死锁
func TestLRUCache_NoDeadlock(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(100, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加一些初始任务
	for i := 1; i <= 10; i++ {
		job := &Job{
			ID:          string(rune('0' + i)),
			Name:        "test",
			Type:        JobTypeOnce,
			HandlerName: "test",
			Status:      JobStatusPending,
		}
		err := store.CreateJob(ctx, job)
		require.NoError(t, err)
	}

	// 并发读写测试（1000+ goroutines）
	var wg sync.WaitGroup
	numGoroutines := 1000
	numOperations := 100

	// 启动多个读 goroutine
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				jobID := string(rune('0' + (idx%10 + 1)))
				_, _ = cache.Get(ctx, jobID)
			}
		}(i)
	}

	// 启动多个写 goroutine
	for i := 0; i < numGoroutines/10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				jobID := string(rune('0' + (idx%10 + 1)))
				job := &Job{
					ID:          jobID,
					Name:        "test",
					Type:        JobTypeOnce,
					HandlerName: "test",
					Status:      JobStatusPending,
				}
				cache.Set(jobID, job)
			}
		}(i)
	}

	// 启动多个删除 goroutine
	for i := 0; i < numGoroutines/20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				jobID := string(rune('0' + (idx%10 + 1)))
				cache.Delete(jobID)
			}
		}(i)
	}

	// 等待所有 goroutine 完成（带超时）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 成功完成，没有死锁
		t.Log("并发测试完成，无死锁")
	case <-time.After(10 * time.Second):
		t.Fatal("测试超时，可能发生死锁")
	}
}

// TestLRUCache_ConcurrentGetSameKey 测试并发获取同一个 key 不会死锁
func TestLRUCache_ConcurrentGetSameKey(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(10, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加任务
	job := &Job{
		ID:          "same-key",
		Name:        "test",
		Type:        JobTypeOnce,
		HandlerName: "test",
		Status:      JobStatusPending,
	}
	err := store.CreateJob(ctx, job)
	require.NoError(t, err)

	// 先加载到缓存
	_, err = cache.Get(ctx, "same-key")
	require.NoError(t, err)

	// 并发获取同一个 key（触发 LRU 更新）
	var wg sync.WaitGroup
	numGoroutines := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cache.Get(ctx, "same-key")
		}()
	}

	// 等待所有 goroutine 完成（带超时）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 成功完成，没有死锁
		t.Log("并发获取同一 key 完成，无死锁")
	case <-time.After(5 * time.Second):
		t.Fatal("测试超时，可能发生死锁")
	}
}

// TestLRUCache_AsyncLRUUpdate 测试异步 LRU 更新的正确性
func TestLRUCache_AsyncLRUUpdate(t *testing.T) {
	store := newMockStorage()
	cache := NewLRUCache(3, time.Minute, store, &NopLogger{})

	ctx := context.Background()

	// 添加 3 个任务
	for i := 1; i <= 3; i++ {
		job := &Job{
			ID:          string(rune('0' + i)),
			Name:        "test",
			Type:        JobTypeOnce,
			HandlerName: "test",
			Status:      JobStatusPending,
		}
		err := store.CreateJob(ctx, job)
		require.NoError(t, err)

		_, err = cache.Get(ctx, job.ID)
		require.NoError(t, err)
	}

	// 缓存应该有 3 个条目
	assert.Equal(t, 3, cache.Size())

	// 多次访问第一个任务（触发异步 LRU 更新）
	for i := 0; i < 10; i++ {
		_, err := cache.Get(ctx, "1")
		require.NoError(t, err)
	}

	// 等待异步更新完成
	time.Sleep(100 * time.Millisecond)

	// 添加第 4 个任务，应该淘汰最久未使用的（不是 "1"）
	job4 := &Job{
		ID:          "4",
		Name:        "test",
		Type:        JobTypeOnce,
		HandlerName: "test",
		Status:      JobStatusPending,
	}
	err := store.CreateJob(ctx, job4)
	require.NoError(t, err)

	_, err = cache.Get(ctx, "4")
	require.NoError(t, err)

	// 缓存应该仍然有 3 个条目
	assert.Equal(t, 3, cache.Size())

	// "1" 应该仍在缓存中（因为最近访问过）
	_, err = cache.Get(ctx, "1")
	require.NoError(t, err)
}
