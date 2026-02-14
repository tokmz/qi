package job

import (
	"sync/atomic"
)

// Metrics 性能监控指标
type Metrics struct {
	// 任务统计
	TotalJobs       atomic.Int64 // 总任务数
	PendingJobs     atomic.Int64 // 待执行任务数
	RunningJobs     atomic.Int64 // 执行中任务数
	CompletedJobs   atomic.Int64 // 已完成任务数
	FailedJobs      atomic.Int64 // 失败任务数

	// 执行统计
	TotalRuns       atomic.Int64 // 总执行次数
	SuccessRuns     atomic.Int64 // 成功执行次数
	FailedRuns      atomic.Int64 // 失败执行次数

	// 性能统计
	TotalDuration   atomic.Int64 // 总执行时间（毫秒）
	AvgDuration     atomic.Int64 // 平均执行时间（毫秒）
	MaxDuration     atomic.Int64 // 最大执行时间（毫秒）
	MinDuration     atomic.Int64 // 最小执行时间（毫秒）

	// 队列统计
	HeapSize        atomic.Int64 // 优先队列大小
	CacheSize       atomic.Int64 // 缓存大小
	CacheHits       atomic.Int64 // 缓存命中次数
	CacheMisses     atomic.Int64 // 缓存未命中次数

	// 批量更新统计
	BatchJobUpdates atomic.Int64 // 批量任务更新次数
	BatchRunUpdates atomic.Int64 // 批量执行记录更新次数

	// 错误统计
	StorageErrors   atomic.Int64 // 存储错误次数
	HandlerErrors   atomic.Int64 // 处理器错误次数
	TimeoutErrors   atomic.Int64 // 超时错误次数
}

// NewMetrics 创建指标实例
func NewMetrics() *Metrics {
	m := &Metrics{}
	m.MinDuration.Store(int64(^uint64(0) >> 1)) // 初始化为最大值
	return m
}

// RecordJobAdded 记录任务添加
func (m *Metrics) RecordJobAdded() {
	m.TotalJobs.Add(1)
	m.PendingJobs.Add(1)
}

// RecordJobRemoved 记录任务删除
func (m *Metrics) RecordJobRemoved() {
	m.TotalJobs.Add(-1)
}

// RecordJobStarted 记录任务开始执行
func (m *Metrics) RecordJobStarted() {
	m.PendingJobs.Add(-1)
	m.RunningJobs.Add(1)
}

// RecordJobCompleted 记录任务完成
func (m *Metrics) RecordJobCompleted() {
	m.RunningJobs.Add(-1)
	m.CompletedJobs.Add(1)
}

// RecordJobFailed 记录任务失败
func (m *Metrics) RecordJobFailed() {
	m.RunningJobs.Add(-1)
	m.FailedJobs.Add(1)
}

// RecordRunSuccess 记录执行成功
func (m *Metrics) RecordRunSuccess(duration int64) {
	m.TotalRuns.Add(1)
	m.SuccessRuns.Add(1)
	m.updateDuration(duration)
}

// RecordRunFailed 记录执行失败
func (m *Metrics) RecordRunFailed(duration int64) {
	m.TotalRuns.Add(1)
	m.FailedRuns.Add(1)
	m.updateDuration(duration)
}

// updateDuration 更新执行时间统计
// 注意：平均值计算是近似值（TotalDuration 和 TotalRuns 的读取不是原子的），对监控指标可接受
func (m *Metrics) updateDuration(duration int64) {
	m.TotalDuration.Add(duration)

	// 更新平均值
	totalRuns := m.TotalRuns.Load()
	if totalRuns > 0 {
		avg := m.TotalDuration.Load() / totalRuns
		m.AvgDuration.Store(avg)
	}

	// 更新最大值
	for {
		old := m.MaxDuration.Load()
		if duration <= old {
			break
		}
		if m.MaxDuration.CompareAndSwap(old, duration) {
			break
		}
	}

	// 更新最小值
	for {
		old := m.MinDuration.Load()
		if duration >= old {
			break
		}
		if m.MinDuration.CompareAndSwap(old, duration) {
			break
		}
	}
}

// RecordCacheHit 记录缓存命中
func (m *Metrics) RecordCacheHit() {
	m.CacheHits.Add(1)
}

// RecordCacheMiss 记录缓存未命中
func (m *Metrics) RecordCacheMiss() {
	m.CacheMisses.Add(1)
}

// UpdateHeapSize 更新堆大小
func (m *Metrics) UpdateHeapSize(size int64) {
	m.HeapSize.Store(size)
}

// UpdateCacheSize 更新缓存大小
func (m *Metrics) UpdateCacheSize(size int64) {
	m.CacheSize.Store(size)
}

// RecordBatchJobUpdate 记录批量任务更新
func (m *Metrics) RecordBatchJobUpdate(count int64) {
	m.BatchJobUpdates.Add(count)
}

// RecordBatchRunUpdate 记录批量执行记录更新
func (m *Metrics) RecordBatchRunUpdate(count int64) {
	m.BatchRunUpdates.Add(count)
}

// RecordStorageError 记录存储错误
func (m *Metrics) RecordStorageError() {
	m.StorageErrors.Add(1)
}

// RecordHandlerError 记录处理器错误
func (m *Metrics) RecordHandlerError() {
	m.HandlerErrors.Add(1)
}

// RecordTimeoutError 记录超时错误
func (m *Metrics) RecordTimeoutError() {
	m.TimeoutErrors.Add(1)
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() map[string]int64 {
	return map[string]int64{
		"total_jobs":         m.TotalJobs.Load(),
		"pending_jobs":       m.PendingJobs.Load(),
		"running_jobs":       m.RunningJobs.Load(),
		"completed_jobs":     m.CompletedJobs.Load(),
		"failed_jobs":        m.FailedJobs.Load(),
		"total_runs":         m.TotalRuns.Load(),
		"success_runs":       m.SuccessRuns.Load(),
		"failed_runs":        m.FailedRuns.Load(),
		"total_duration_ms":  m.TotalDuration.Load(),
		"avg_duration_ms":    m.AvgDuration.Load(),
		"max_duration_ms":    m.MaxDuration.Load(),
		"min_duration_ms":    m.MinDuration.Load(),
		"heap_size":          m.HeapSize.Load(),
		"cache_size":         m.CacheSize.Load(),
		"cache_hits":         m.CacheHits.Load(),
		"cache_misses":       m.CacheMisses.Load(),
		"batch_job_updates":  m.BatchJobUpdates.Load(),
		"batch_run_updates":  m.BatchRunUpdates.Load(),
		"storage_errors":     m.StorageErrors.Load(),
		"handler_errors":     m.HandlerErrors.Load(),
		"timeout_errors":     m.TimeoutErrors.Load(),
	}
}

// GetCacheHitRate 获取缓存命中率
func (m *Metrics) GetCacheHitRate() float64 {
	hits := m.CacheHits.Load()
	misses := m.CacheMisses.Load()
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total) * 100
}

// GetSuccessRate 获取成功率
func (m *Metrics) GetSuccessRate() float64 {
	success := m.SuccessRuns.Load()
	total := m.TotalRuns.Load()

	if total == 0 {
		return 0
	}

	return float64(success) / float64(total) * 100
}

// Reset 重置所有指标
func (m *Metrics) Reset() {
	m.TotalJobs.Store(0)
	m.PendingJobs.Store(0)
	m.RunningJobs.Store(0)
	m.CompletedJobs.Store(0)
	m.FailedJobs.Store(0)
	m.TotalRuns.Store(0)
	m.SuccessRuns.Store(0)
	m.FailedRuns.Store(0)
	m.TotalDuration.Store(0)
	m.AvgDuration.Store(0)
	m.MaxDuration.Store(0)
	m.MinDuration.Store(int64(^uint64(0) >> 1))
	m.HeapSize.Store(0)
	m.CacheSize.Store(0)
	m.CacheHits.Store(0)
	m.CacheMisses.Store(0)
	m.BatchJobUpdates.Store(0)
	m.BatchRunUpdates.Store(0)
	m.StorageErrors.Store(0)
	m.HandlerErrors.Store(0)
	m.TimeoutErrors.Store(0)
}
