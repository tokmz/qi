package cache

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// memoryCache 内存缓存实现
type memoryCache struct {
	cache      *gocache.Cache
	serializer Serializer
	keyPrefix  string
	defaultTTL time.Duration
	mu         sync.Mutex // 保护 IncrBy 操作的互斥锁
}

// newMemoryCache 创建内存缓存实例
func newMemoryCache(cfg *Config) (Cache, error) {
	if cfg.Memory == nil {
		cfg.Memory = DefaultMemoryConfig()
	}

	return &memoryCache{
		cache:      gocache.New(cfg.Memory.DefaultExpiration, cfg.Memory.CleanupInterval),
		serializer: cfg.Serializer,
		keyPrefix:  cfg.KeyPrefix,
		defaultTTL: cfg.DefaultTTL,
	}, nil
}

// buildKey 构建完整的键名
func (m *memoryCache) buildKey(key string) string {
	if m.keyPrefix == "" {
		return key
	}
	return m.keyPrefix + key
}

// Get 获取缓存
func (m *memoryCache) Get(ctx context.Context, key string, value any) error {
	fullKey := m.buildKey(key)
	data, found := m.cache.Get(fullKey)
	if !found {
		return ErrCacheNotFound
	}

	bytes, ok := data.([]byte)
	if !ok {
		return fmt.Errorf("%w: invalid cache data type", ErrCacheSerialization)
	}

	if err := m.serializer.Unmarshal(bytes, value); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheSerialization, err)
	}

	return nil
}

// Set 设置缓存
func (m *memoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	fullKey := m.buildKey(key)

	bytes, err := m.serializer.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCacheSerialization, err)
	}

	if ttl == 0 {
		ttl = m.defaultTTL
	}

	m.cache.Set(fullKey, bytes, ttl)
	return nil
}

// Delete 删除缓存
func (m *memoryCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		fullKey := m.buildKey(key)
		m.cache.Delete(fullKey)
	}
	return nil
}

// Exists 检查键是否存在
func (m *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := m.buildKey(key)
	_, found := m.cache.Get(fullKey)
	return found, nil
}

// MGet 批量获取
func (m *memoryCache) MGet(ctx context.Context, keys []string, values any) error {
	// 检查 values 是否为切片指针
	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("%w: values must be a pointer to slice", ErrCacheOperation)
	}

	slice := rv.Elem()
	elemType := slice.Type().Elem()

	// 清空切片
	slice.Set(reflect.MakeSlice(slice.Type(), 0, len(keys)))

	for _, key := range keys {
		fullKey := m.buildKey(key)
		data, found := m.cache.Get(fullKey)
		if !found {
			continue
		}

		bytes, ok := data.([]byte)
		if !ok {
			// 记录警告：数据类型不匹配
			continue
		}

		// 创建新元素
		elem := reflect.New(elemType)
		if err := m.serializer.Unmarshal(bytes, elem.Interface()); err != nil {
			// 记录警告：反序列化失败
			// TODO: 考虑返回部分错误信息或记录日志
			continue
		}

		// 追加到切片
		slice.Set(reflect.Append(slice, elem.Elem()))
	}

	return nil
}

// MSet 批量设置
func (m *memoryCache) MSet(ctx context.Context, items map[string]any, ttl time.Duration) error {
	if ttl == 0 {
		ttl = m.defaultTTL
	}

	for key, value := range items {
		if err := m.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}

	return nil
}

// MSetTx 批量设置（内存缓存本身就是线程安全的）
func (m *memoryCache) MSetTx(ctx context.Context, items map[string]any, ttl time.Duration) error {
	return m.MSet(ctx, items, ttl)
}

// TTL 获取键的剩余生存时间
func (m *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := m.buildKey(key)
	_, expiration, found := m.cache.GetWithExpiration(fullKey)
	if !found {
		return 0, ErrCacheNotFound
	}

	if expiration.IsZero() {
		return -1, nil // 永不过期
	}

	ttl := time.Until(expiration)
	if ttl < 0 {
		return 0, ErrCacheExpired
	}

	return ttl, nil
}

// Expire 设置键的过期时间
func (m *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := m.buildKey(key)
	data, found := m.cache.Get(fullKey)
	if !found {
		return ErrCacheNotFound
	}

	m.cache.Set(fullKey, data, ttl)
	return nil
}

// Incr 自增
func (m *memoryCache) Incr(ctx context.Context, key string) (int64, error) {
	return m.IncrBy(ctx, key, 1)
}

// Decr 自减
func (m *memoryCache) Decr(ctx context.Context, key string) (int64, error) {
	return m.IncrBy(ctx, key, -1)
}

// IncrBy 增加指定值
func (m *memoryCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	fullKey := m.buildKey(key)

	// 使用互斥锁保护整个 Get-Modify-Set 操作，确保原子性
	m.mu.Lock()
	defer m.mu.Unlock()

	// 尝试获取当前值
	data, found := m.cache.Get(fullKey)
	var current int64

	if found {
		bytes, ok := data.([]byte)
		if !ok {
			return 0, fmt.Errorf("%w: invalid cache data type", ErrCacheOperation)
		}

		if err := m.serializer.Unmarshal(bytes, &current); err != nil {
			return 0, fmt.Errorf("%w: %w", ErrCacheSerialization, err)
		}
	}

	// 计算新值
	newValue := current + value

	// 序列化并存储
	bytes, err := m.serializer.Marshal(newValue)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCacheSerialization, err)
	}

	m.cache.Set(fullKey, bytes, m.defaultTTL)
	return newValue, nil
}

// Ping 检查连接
func (m *memoryCache) Ping(ctx context.Context) error {
	return nil // Memory cache always available
}

// Close 关闭连接
func (m *memoryCache) Close() error {
	m.cache.Flush()
	return nil
}

// String 返回缓存类型
func (m *memoryCache) String() string {
	return fmt.Sprintf("MemoryCache(prefix=%s, items=%d)", m.keyPrefix, m.cache.ItemCount())
}
