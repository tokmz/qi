package cache

import (
	"container/list"
	"context"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// memEntry 内存缓存条目
type memEntry struct {
	key      string
	value    []byte
	expireAt time.Time // 零值 = 永不过期
}

func (e *memEntry) expired() bool {
	return !e.expireAt.IsZero() && time.Now().After(e.expireAt)
}

// memoryCache LRU 内存缓存实现
type memoryCache struct {
	mu         sync.Mutex
	items      map[string]*list.Element
	lru        *list.List
	maxSize    int
	serializer Serializer
	prefix     string
	defaultTTL time.Duration
	sf         singleflight.Group
	cancel     context.CancelFunc
}

func newMemoryCache(cfg *Config) (*memoryCache, error) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &memoryCache{
		items:      make(map[string]*list.Element),
		lru:        list.New(),
		maxSize:    cfg.Memory.MaxSize,
		serializer: cfg.Serializer,
		prefix:     cfg.KeyPrefix,
		defaultTTL: cfg.DefaultTTL,
		cancel:     cancel,
	}
	go c.cleanupLoop(ctx, cfg.Memory.CleanupInterval)
	return c, nil
}

func (c *memoryCache) k(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + key
}

func (c *memoryCache) ttl(ttl time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	return c.defaultTTL
}

func (c *memoryCache) expireAt(ttl time.Duration) time.Time {
	d := c.ttl(ttl)
	if d <= 0 {
		return time.Time{}
	}
	return time.Now().Add(d)
}

// removeElement 移除元素（调用方持锁）
func (c *memoryCache) removeElement(el *list.Element) {
	c.lru.Remove(el)
	delete(c.items, el.Value.(*memEntry).key)
}

// evictLRU 淘汰最久未使用的条目（调用方持锁）
func (c *memoryCache) evictLRU() {
	back := c.lru.Back()
	if back != nil {
		c.removeElement(back)
	}
}

// setLocked 写入条目（调用方持锁）
func (c *memoryCache) setLocked(key string, value []byte, expireAt time.Time) {
	if el, ok := c.items[key]; ok {
		c.lru.MoveToFront(el)
		el.Value.(*memEntry).value = value
		el.Value.(*memEntry).expireAt = expireAt
		return
	}
	if c.maxSize > 0 && c.lru.Len() >= c.maxSize {
		c.evictLRU()
	}
	el := c.lru.PushFront(&memEntry{key: key, value: value, expireAt: expireAt})
	c.items[key] = el
}

func (c *memoryCache) Get(_ context.Context, key string, dest any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[c.k(key)]
	if !ok {
		return ErrNotFound
	}
	entry := el.Value.(*memEntry)
	if entry.expired() {
		c.removeElement(el)
		return ErrNotFound
	}
	c.lru.MoveToFront(el)
	return c.serializer.Unmarshal(entry.value, dest)
}

func (c *memoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	if value == nil {
		return ErrNilValue
	}
	b, err := c.serializer.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.setLocked(c.k(key), b, c.expireAt(ttl))
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Del(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		if el, ok := c.items[c.k(key)]; ok {
			c.removeElement(el)
		}
	}
	return nil
}

func (c *memoryCache) Exists(_ context.Context, key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[c.k(key)]
	if !ok {
		return false, nil
	}
	if el.Value.(*memEntry).expired() {
		c.removeElement(el)
		return false, nil
	}
	return true, nil
}

func (c *memoryCache) Expire(_ context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[c.k(key)]
	if !ok {
		return ErrNotFound
	}
	entry := el.Value.(*memEntry)
	if entry.expired() {
		c.removeElement(el)
		return ErrNotFound
	}
	entry.expireAt = c.expireAt(ttl)
	return nil
}

func (c *memoryCache) TTL(_ context.Context, key string) (time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[c.k(key)]
	if !ok {
		return -1, ErrNotFound
	}
	entry := el.Value.(*memEntry)
	if entry.expired() {
		c.removeElement(el)
		return -1, ErrNotFound
	}
	if entry.expireAt.IsZero() {
		return -1, nil // 永不过期
	}
	return time.Until(entry.expireAt), nil
}

func (c *memoryCache) MGet(_ context.Context, keys []string) (map[string][]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		el, ok := c.items[c.k(key)]
		if !ok {
			continue
		}
		entry := el.Value.(*memEntry)
		if entry.expired() {
			c.removeElement(el)
			continue
		}
		c.lru.MoveToFront(el)
		result[key] = entry.value
	}
	return result, nil
}

func (c *memoryCache) MSet(_ context.Context, kvs map[string]any, ttl time.Duration) error {
	bufs := make(map[string][]byte, len(kvs))
	for k, v := range kvs {
		if v == nil {
			return ErrNilValue
		}
		b, err := c.serializer.Marshal(v)
		if err != nil {
			return err
		}
		bufs[k] = b
	}
	expAt := c.expireAt(ttl)
	c.mu.Lock()
	for k, b := range bufs {
		c.setLocked(c.k(k), b, expAt)
	}
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	if err := c.Get(ctx, key, dest); err == nil {
		return nil
	}
	v, err, _ := c.sf.Do(c.k(key), func() (any, error) {
		// double-check after singleflight
		if err2 := c.Get(ctx, key, dest); err2 == nil {
			return nil, nil // 已有值，用哨兵 nil 表示
		}
		val, err2 := fn()
		if err2 != nil {
			return nil, err2
		}
		_ = c.Set(ctx, key, val, ttl)
		return val, nil
	})
	if err != nil {
		return err
	}
	if v == nil {
		// 已被 double-check 填充，dest 已赋值，直接返回
		return c.Get(ctx, key, dest)
	}
	b, err2 := c.serializer.Marshal(v)
	if err2 != nil {
		return err2
	}
	return c.serializer.Unmarshal(b, dest)
}

func (c *memoryCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.IncrBy(ctx, key, 1)
}

func (c *memoryCache) IncrBy(_ context.Context, key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var (
		cur      int64
		expireAt time.Time
	)
	if el, ok := c.items[c.k(key)]; ok {
		entry := el.Value.(*memEntry)
		if !entry.expired() {
			if n, err := strconv.ParseInt(string(entry.value), 10, 64); err == nil {
				cur = n
			}
			expireAt = entry.expireAt // 保留原有 TTL
			c.lru.MoveToFront(el)
		} else {
			c.removeElement(el)
		}
	}
	cur += delta
	c.setLocked(c.k(key), []byte(strconv.FormatInt(cur, 10)), expireAt)
	return cur, nil
}

func (c *memoryCache) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return c.IncrBy(ctx, key, -delta)
}

func (c *memoryCache) Flush(_ context.Context) error {
	c.mu.Lock()
	c.items = make(map[string]*list.Element)
	c.lru.Init()
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Close() error {
	c.cancel()
	return nil
}

// cleanupLoop 后台定期清理过期 key
func (c *memoryCache) cleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.deleteExpired()
		}
	}
}

func (c *memoryCache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, el := range c.items {
		if el.Value.(*memEntry).expired() {
			c.removeElement(el)
		}
	}
}
