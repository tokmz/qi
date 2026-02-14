package job

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobHeapBasic 测试堆基本功能
func TestJobHeapBasic(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	job1 := &Job{ID: "1", NextRunAt: &now}
	job2 := &Job{ID: "2", NextRunAt: ptrTime(now.Add(time.Second))}
	job3 := &Job{ID: "3", NextRunAt: ptrTime(now.Add(2 * time.Second))}

	// 添加任务
	h.Add(job1)
	h.Add(job2)
	h.Add(job3)

	assert.Equal(t, 3, h.Size())
}

// TestJobHeapPopDue 测试弹出到期任务
func TestJobHeapPopDue(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	job1 := &Job{ID: "1", NextRunAt: ptrTime(now.Add(-time.Second))}  // 已过期
	job2 := &Job{ID: "2", NextRunAt: ptrTime(now.Add(-2 * time.Second))} // 已过期
	job3 := &Job{ID: "3", NextRunAt: ptrTime(now.Add(time.Hour))}    // 未到期

	h.Add(job1)
	h.Add(job2)
	h.Add(job3)

	// 弹出到期任务
	dueJobs := h.PopDue(now)

	assert.Equal(t, 2, len(dueJobs), "应该弹出 2 个到期任务")
	assert.Equal(t, 1, h.Size(), "堆中应该剩余 1 个任务")

	// 验证弹出的是最早到期的任务
	assert.Equal(t, "2", dueJobs[0].ID, "最早到期的任务应该先弹出")
	assert.Equal(t, "1", dueJobs[1].ID)
}

// TestJobHeapPeek 测试查看堆顶
func TestJobHeapPeek(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	job1 := &Job{ID: "1", NextRunAt: ptrTime(now.Add(time.Second))}
	job2 := &Job{ID: "2", NextRunAt: &now} // 最早

	h.Add(job1)
	h.Add(job2)

	top := h.Peek()
	require.NotNil(t, top)
	assert.Equal(t, "2", top.ID, "堆顶应该是最早到期的任务")
	assert.Equal(t, 2, h.Size(), "Peek 不应该移除元素")
}

// TestJobHeapRemove 测试删除任务
func TestJobHeapRemove(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	job1 := &Job{ID: "1", NextRunAt: &now}
	job2 := &Job{ID: "2", NextRunAt: ptrTime(now.Add(time.Second))}
	job3 := &Job{ID: "3", NextRunAt: ptrTime(now.Add(2 * time.Second))}

	h.Add(job1)
	h.Add(job2)
	h.Add(job3)

	// 删除中间的任务
	removed := h.Remove("2")
	assert.NotNil(t, removed, "应该成功删除")
	assert.Equal(t, 2, h.Size())

	// 删除不存在的任务
	removed = h.Remove("999")
	assert.Nil(t, removed, "删除不存在的任务应该返回 nil")
}

// TestJobHeapUpdate 测试更新任务
func TestJobHeapUpdate(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	job1 := &Job{ID: "1", NextRunAt: ptrTime(now.Add(time.Hour))}
	job2 := &Job{ID: "2", NextRunAt: ptrTime(now.Add(2 * time.Hour))}

	h.Add(job1)
	h.Add(job2)

	// 更新 job1 的时间，使其成为最晚的
	job1.NextRunAt = ptrTime(now.Add(3 * time.Hour))
	h.Update(job1)

	// 堆顶应该是 job2
	top := h.Peek()
	require.NotNil(t, top)
	assert.Equal(t, "2", top.ID)
}

// TestJobHeapClear 测试清空堆
func TestJobHeapClear(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	for i := range 10 {
		job := &Job{
			ID:        string(rune('0' + i)),
			NextRunAt: ptrTime(now.Add(time.Duration(i) * time.Second)),
		}
		h.Add(job)
	}

	assert.Equal(t, 10, h.Size())

	h.Clear()
	assert.Equal(t, 0, h.Size())
	assert.Nil(t, h.Peek())
}

// TestJobHeapOrdering 测试堆排序正确性
func TestJobHeapOrdering(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	// 乱序添加
	times := []time.Duration{5, 2, 8, 1, 9, 3, 7, 4, 6}
	for i, d := range times {
		job := &Job{
			ID:        string(rune('0' + i)),
			NextRunAt: ptrTime(now.Add(d * time.Second)),
		}
		h.Add(job)
	}

	// 依次弹出，应该按时间顺序
	var lastTime time.Time
	for h.Size() > 0 {
		dueJobs := h.PopDue(now.Add(10 * time.Second))
		if len(dueJobs) == 0 {
			break
		}
		for _, job := range dueJobs {
			if !lastTime.IsZero() {
				assert.True(t, job.NextRunAt.After(lastTime) || job.NextRunAt.Equal(lastTime),
					"任务应该按时间顺序弹出")
			}
			lastTime = *job.NextRunAt
		}
	}
}

// TestJobHeapConcurrent 测试并发安全
// 注意：jobHeap 自身不再持有锁，由外部调用方（如 DefaultScheduler.mu）保护。
// 此测试使用外部 mutex 模拟调度器的锁保护模式。
func TestJobHeapConcurrent(t *testing.T) {
	h := newJobHeap()
	var mu sync.Mutex

	now := time.Now()
	done := make(chan bool)

	// 并发添加
	for i := range 10 {
		go func(idx int) {
			job := &Job{
				ID:        string(rune('0' + idx)),
				NextRunAt: ptrTime(now.Add(time.Duration(idx) * time.Second)),
			}
			mu.Lock()
			h.Add(job)
			mu.Unlock()
			done <- true
		}(i)
	}

	// 等待所有添加完成
	for range 10 {
		<-done
	}

	assert.Equal(t, 10, h.Size())

	// 并发删除
	for i := range 5 {
		go func(idx int) {
			mu.Lock()
			h.Remove(string(rune('0' + idx)))
			mu.Unlock()
			done <- true
		}(i)
	}

	// 等待所有删除完成
	for range 5 {
		<-done
	}

	assert.Equal(t, 5, h.Size())
}

// TestJobHeapNilNextRunAt 测试 NextRunAt 为 nil 的情况
func TestJobHeapNilNextRunAt(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	job1 := &Job{ID: "1", NextRunAt: nil}
	job2 := &Job{ID: "2", NextRunAt: &now}

	// 添加 NextRunAt 为 nil 的任务（应该被忽略或放在最后）
	h.Add(job1)
	h.Add(job2)

	// 堆顶应该是有时间的任务
	top := h.Peek()
	if top != nil {
		assert.Equal(t, "2", top.ID)
	}
}

// ptrTime 辅助函数：返回时间指针
func ptrTime(t time.Time) *time.Time {
	return &t
}

// TestJobHeapCleanCompleted 测试清理已完成任务
func TestJobHeapCleanCompleted(t *testing.T) {
	h := newJobHeap()

	now := time.Now()

	// 添加不同状态的任务
	h.Add(&Job{ID: "1", Status: JobStatusPending, NextRunAt: ptrTime(now.Add(time.Second))})
	h.Add(&Job{ID: "2", Status: JobStatusCompleted, NextRunAt: ptrTime(now.Add(2 * time.Second))})
	h.Add(&Job{ID: "3", Status: JobStatusFailed, NextRunAt: ptrTime(now.Add(3 * time.Second))})
	h.Add(&Job{ID: "4", Status: JobStatusPending, NextRunAt: ptrTime(now.Add(4 * time.Second))})
	h.Add(&Job{ID: "5", Status: JobStatusRunning, NextRunAt: ptrTime(now.Add(5 * time.Second))})

	assert.Equal(t, 5, h.Size())

	// 清理已完成和已失败的任务
	count := h.CleanCompleted()
	assert.Equal(t, 2, count, "应该清理 2 个任务（completed + failed）")
	assert.Equal(t, 3, h.Size(), "应该剩余 3 个任务")

	// 验证剩余任务的正确性
	assert.True(t, h.Contains("1"))
	assert.False(t, h.Contains("2"))
	assert.False(t, h.Contains("3"))
	assert.True(t, h.Contains("4"))
	assert.True(t, h.Contains("5"))

	// 验证堆排序仍然正确
	top := h.Peek()
	require.NotNil(t, top)
	assert.Equal(t, "1", top.ID, "堆顶应该是最早到期的 pending 任务")
}

// TestJobHeapCleanCompleted_Empty 测试空堆清理
func TestJobHeapCleanCompleted_Empty(t *testing.T) {
	h := newJobHeap()
	count := h.CleanCompleted()
	assert.Equal(t, 0, count)
}

// TestJobHeapCleanCompleted_NoneToClean 测试无需清理的情况
func TestJobHeapCleanCompleted_NoneToClean(t *testing.T) {
	h := newJobHeap()

	now := time.Now()
	h.Add(&Job{ID: "1", Status: JobStatusPending, NextRunAt: ptrTime(now)})
	h.Add(&Job{ID: "2", Status: JobStatusRunning, NextRunAt: ptrTime(now.Add(time.Second))})

	count := h.CleanCompleted()
	assert.Equal(t, 0, count)
	assert.Equal(t, 2, h.Size())
}
