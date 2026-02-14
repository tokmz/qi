package job

import (
	"container/list"
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	job       *Job
	expiresAt time.Time
}

// LRUCache LRU 缓存，缓存热点任务
type LRUCache struct {
	mu        sync.RWMutex
	capacity  int
	ttl       time.Duration
	items     map[string]*list.Element
	lruList   *list.List
	storage   Storage
	logger    Logger
	sf        singleflight.Group // 防止缓存击穿
	promoteCh chan string        // 有界的 LRU 提升通道
}

// cacheItem 缓存项
type cacheItem struct {
	key   string
	entry *CacheEntry
}

// NewLRUCache 创建 LRU 缓存
func NewLRUCache(capacity int, ttl time.Duration, storage Storage, logger Logger) *LRUCache {
	if capacity <= 0 {
		capacity = 100
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	return &LRUCache{
		capacity:  capacity,
		ttl:       ttl,
		items:     make(map[string]*list.Element),
		lruList:   list.New(),
		storage:   storage,
		logger:    logger,
		promoteCh: make(chan string, 256),
	}
}

// Get 获取任务（优先从缓存，使用 singleflight 防止缓存击穿）
func (c *LRUCache) Get(ctx context.Context, id string) (*Job, error) {
	// 先检查缓存（快速路径）
	c.mu.RLock()
	if elem, ok := c.items[id]; ok {
		item := elem.Value.(*cacheItem)
		// 检查是否过期
		if time.Now().Before(item.entry.expiresAt) {
			job := item.entry.job
			c.mu.RUnlock()

			// 非阻塞发送到提升通道，满了就丢弃（不影响正确性）
			select {
			case c.promoteCh <- id:
			default:
			}

			return job.Clone(), nil
		}
	}
	c.mu.RUnlock()

	// 缓存未命中或已过期，使用 singleflight 加载
	// 防止多个 goroutine 同时加载同一个任务
	v, err, _ := c.sf.Do(id, func() (any, error) {
		// 再次检查缓存（可能已被其他 goroutine 加载）
		c.mu.RLock()
		if elem, ok := c.items[id]; ok {
			item := elem.Value.(*cacheItem)
			if time.Now().Before(item.entry.expiresAt) {
				job := item.entry.job
				c.mu.RUnlock()
				return job, nil
			}
		}
		c.mu.RUnlock()

		// 从存储加载
		// 使用独立 context，避免首个调用方取消导致所有共享请求失败
		// 保留 span context 以维持链路追踪连续性
		sfCtx := trace.ContextWithSpanContext(context.Background(), trace.SpanContextFromContext(ctx))
		sfCtx, sfCancel := context.WithTimeout(sfCtx, 5*time.Second)
		defer sfCancel()
		job, err := c.storage.GetJob(sfCtx, id)
		if err != nil {
			return nil, err
		}

		// 添加到缓存
		c.Set(id, job)
		return job, nil
	})

	if err != nil {
		return nil, err
	}

	job := v.(*Job)
	return job.Clone(), nil
}

// Set 设置缓存
func (c *LRUCache) Set(id string, job *Job) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已存在
	if elem, ok := c.items[id]; ok {
		// 更新现有条目
		c.lruList.MoveToFront(elem)
		item := elem.Value.(*cacheItem)
		item.entry.job = job.Clone()
		item.entry.expiresAt = time.Now().Add(c.ttl)
		return
	}

	// 检查容量
	if c.lruList.Len() >= c.capacity {
		// 删除最久未使用的条目
		oldest := c.lruList.Back()
		if oldest != nil {
			c.lruList.Remove(oldest)
			oldItem := oldest.Value.(*cacheItem)
			delete(c.items, oldItem.key)
		}
	}

	// 添加新条目
	entry := &CacheEntry{
		job:       job.Clone(),
		expiresAt: time.Now().Add(c.ttl),
	}
	item := &cacheItem{
		key:   id,
		entry: entry,
	}
	elem := c.lruList.PushFront(item)
	c.items[id] = elem
}

// Delete 删除缓存
func (c *LRUCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.lruList.Remove(elem)
		delete(c.items, id)
	}
}

// Clear 清空缓存
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lruList = list.New()
}

// Size 返回缓存大小
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lruList.Len()
}

// CleanExpired 清理过期缓存
func (c *LRUCache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	// 遍历所有条目清理过期的（异步 MoveToFront 可能打乱 LRU 顺序）
	for elem := c.lruList.Back(); elem != nil; {
		item := elem.Value.(*cacheItem)
		prev := elem.Prev()
		if now.After(item.entry.expiresAt) {
			c.lruList.Remove(elem)
			delete(c.items, item.key)
			count++
		}
		elem = prev
	}

	return count
}

// StartCleanup 启动定期清理和 LRU 提升处理
func (c *LRUCache) StartCleanup(interval time.Duration, stopChan <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			c.drainPromotions()
			count := c.CleanExpired()
			if count > 0 {
				c.logger.Debug("[cache] 清理过期缓存: %d 条", count)
			}
		case id := <-c.promoteCh:
			c.mu.Lock()
			if elem, ok := c.items[id]; ok {
				c.lruList.MoveToFront(elem)
			}
			c.mu.Unlock()
		}
	}
}

// drainPromotions 批量处理积压的 LRU 提升请求
func (c *LRUCache) drainPromotions() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		select {
		case id := <-c.promoteCh:
			if elem, ok := c.items[id]; ok {
				c.lruList.MoveToFront(elem)
			}
		default:
			return
		}
	}
}

// GetStats 获取缓存统计信息
func (c *LRUCache) GetStats() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]any{
		"size":     c.lruList.Len(),
		"capacity": c.capacity,
		"ttl":      c.ttl.String(),
	}
}
