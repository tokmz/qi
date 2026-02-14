package job

import (
	"container/heap"
	"time"
)

// jobHeap 任务优先队列（按 NextRunAt 排序）
type jobHeap struct {
	items []*Job
	index map[string]int // jobID -> heap index
}

// newJobHeap 创建任务堆
func newJobHeap() *jobHeap {
	return &jobHeap{
		items: make([]*Job, 0),
		index: make(map[string]int),
	}
}

// Len 实现 heap.Interface
func (h *jobHeap) Len() int {
	return len(h.items)
}

// Less 实现 heap.Interface（按 NextRunAt 升序排序）
func (h *jobHeap) Less(i, j int) bool {
	// nil 的 NextRunAt 排在最后
	if h.items[i].NextRunAt == nil {
		return false
	}
	if h.items[j].NextRunAt == nil {
		return true
	}
	return h.items[i].NextRunAt.Before(*h.items[j].NextRunAt)
}

// Swap 实现 heap.Interface
func (h *jobHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.index[h.items[i].ID] = i
	h.index[h.items[j].ID] = j
}

// Push 实现 heap.Interface
func (h *jobHeap) Push(x any) {
	job := x.(*Job)
	h.index[job.ID] = len(h.items)
	h.items = append(h.items, job)
}

// Pop 实现 heap.Interface
func (h *jobHeap) Pop() any {
	old := h.items
	n := len(old)
	job := old[n-1]
	old[n-1] = nil // 避免内存泄漏
	h.items = old[0 : n-1]
	delete(h.index, job.ID)
	return job
}

// Add 添加任务到堆
func (h *jobHeap) Add(job *Job) {
	// 如果任务已存在，先移除
	if idx, exists := h.index[job.ID]; exists {
		heap.Remove(h, idx)
	}

	heap.Push(h, job)
}

// Remove 从堆中移除任务
func (h *jobHeap) Remove(jobID string) *Job {
	idx, exists := h.index[jobID]
	if !exists {
		return nil
	}

	job := heap.Remove(h, idx).(*Job)
	return job
}

// Update 更新任务
func (h *jobHeap) Update(job *Job) {
	idx, exists := h.index[job.ID]
	if !exists {
		heap.Push(h, job)
		return
	}

	// 更新任务并重新调整堆
	h.items[idx] = job
	heap.Fix(h, idx)
}

// Peek 查看堆顶任务（不移除）
func (h *jobHeap) Peek() *Job {
	if len(h.items) == 0 {
		return nil
	}
	return h.items[0]
}

// PopDue 弹出所有到期的任务
func (h *jobHeap) PopDue(now time.Time) []*Job {
	var dueJobs []*Job

	for len(h.items) > 0 {
		job := h.items[0]

		// 检查是否到期
		if job.NextRunAt == nil || job.NextRunAt.After(now) {
			break
		}

		// 弹出到期任务
		dueJobs = append(dueJobs, heap.Pop(h).(*Job))
	}

	return dueJobs
}

// GetNextRunTime 获取下次执行时间
func (h *jobHeap) GetNextRunTime() *time.Time {
	if len(h.items) == 0 {
		return nil
	}

	return h.items[0].NextRunAt
}

// Clear 清空堆
func (h *jobHeap) Clear() {
	h.items = make([]*Job, 0)
	h.index = make(map[string]int)
}

// Size 返回堆大小
func (h *jobHeap) Size() int {
	return len(h.items)
}

// Contains 检查任务是否在堆中
func (h *jobHeap) Contains(jobID string) bool {
	_, exists := h.index[jobID]
	return exists
}

// CleanCompleted 清理已完成或已失败的任务
func (h *jobHeap) CleanCompleted() int {
	count := 0
	var validItems []*Job

	for _, job := range h.items {
		if job.Status == JobStatusCompleted || job.Status == JobStatusFailed {
			delete(h.index, job.ID)
			count++
		} else {
			validItems = append(validItems, job)
		}
	}

	if count > 0 {
		h.items = validItems
		// 重建索引
		for i, job := range h.items {
			h.index[job.ID] = i
		}
		// 重新堆化
		heap.Init(h)
	}

	return count
}
