package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
	"qi/internal/core/tracing"
)

// 全局缓存管理器
var (
	globalManager *Manager
	globalMu      sync.RWMutex
)

// Manager 缓存管理器
type Manager struct {
	config     *Config
	rdb        *redis.Client
	serializer Serializer
	logger     Logger
	sf         singleflight.Group
	closed     bool
	mu         sync.RWMutex

	// 统计信息
	stats struct {
		requests         int64
		hits             int64
		misses           int64
		sets             int64
		deletes          int64
		errors           int64
		loaderCalls      int64
		singleflightHits int64
	}

	// 清理定时器
	cleanupTimer *time.Ticker
	cleanupStop  chan struct{}
}

// New 创建新的缓存管理器
func New(cfg *Config, logger Logger) (*Manager, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 使用默认日志器
	if logger == nil {
		logger = &DefaultLogger{}
	}

	// 创建序列化器
	serializer, err := newSerializer(cfg.Serializer)
	if err != nil {
		return nil, err
	}

	// 创建管理器
	m := &Manager{
		config:       cfg,
		rdb:          cfg.Redis,
		serializer:   serializer,
		logger:       logger,
		cleanupStop:  make(chan struct{}),
	}

	// 启动统计上报（如果启用）
	if cfg.Stats.Enabled {
		m.startStatsReporter()
	}

	logger.Info("Cache manager created successfully")
	return m, nil
}

// InitGlobal 初始化全局缓存管理器
func InitGlobal(cfg *Config, logger Logger) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	manager, err := New(cfg, logger)
	if err != nil {
		return err
	}

	globalManager = manager
	return nil
}

// GetGlobal 获取全局缓存管理器
func GetGlobal() *Manager {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalManager == nil {
		panic(ErrManagerNotInitialized)
	}

	return globalManager
}

// Close 关闭缓存管理器
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrManagerAlreadyClosed
	}

	// 停止清理定时器
	if m.cleanupTimer != nil {
		m.cleanupTimer.Stop()
		close(m.cleanupStop)
	}

	m.closed = true
	m.logger.Info("Cache manager closed")
	return nil
}

// Set 设置缓存
func (m *Manager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "cache.Set",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "set"),
		),
	)
	defer tracing.EndSpan(span)

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		err := ErrManagerAlreadyClosed
		tracing.RecordError(ctx, err)
		return err
	}
	m.mu.RUnlock()

	if value == nil {
		err := ErrNilValue
		tracing.RecordError(ctx, err)
		return err
	}

	// 序列化
	data, err := m.serializer.Marshal(value)
	if err != nil {
		m.incrementError()
		m.logger.Error("Failed to marshal value", "key", key, "error", err)
		wrappedErr := fmt.Errorf("%w: %v", ErrSerializationFailed, err)
		tracing.RecordError(ctx, wrappedErr)
		return wrappedErr
	}

	// 构建完整的键
	fullKey := buildKey(m.config.KeyPrefix, key)

	// 设置到 Redis
	if ttl <= 0 {
		ttl = m.config.DefaultExpiration
	}

	// 添加 TTL 属性
	tracing.SetAttributes(ctx, attribute.String("cache.ttl", ttl.String()))

	err = m.rdb.Set(ctx, fullKey, data, ttl).Err()
	if err != nil {
		m.incrementError()
		m.logger.Error("Failed to set cache", "key", key, "error", err)
		tracing.RecordError(ctx, err)
		return err
	}

	m.incrementSets()
	m.logger.Debug("Cache set", "key", key, "ttl", ttl)
	tracing.SetSpanStatus(ctx, codes.Ok, "cache set successfully")
	return nil
}

// Get 获取缓存
func (m *Manager) Get(ctx context.Context, key string, dest interface{}) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "cache.Get",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "get"),
		),
	)
	defer tracing.EndSpan(span)

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		err := ErrManagerAlreadyClosed
		tracing.RecordError(ctx, err)
		return err
	}
	m.mu.RUnlock()

	m.incrementRequests()

	// 构建完整的键
	fullKey := buildKey(m.config.KeyPrefix, key)

	// 从 Redis 获取
	data, err := m.rdb.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			m.incrementMisses()
			tracing.SetAttributes(ctx, attribute.Bool("cache.hit", false))
			return ErrCacheMiss
		}
		m.incrementError()
		m.logger.Error("Failed to get cache", "key", key, "error", err)
		tracing.RecordError(ctx, err)
		return err
	}

	// 检查是否为空值标记
	if isNullValue(data) {
		m.incrementHits()
		tracing.SetAttributes(ctx,
			attribute.Bool("cache.hit", true),
			attribute.Bool("cache.null_value", true),
		)
		return ErrCacheMiss
	}

	// 反序列化
	if err := m.serializer.Unmarshal(data, dest); err != nil {
		m.incrementError()
		m.logger.Error("Failed to unmarshal value", "key", key, "error", err)
		wrappedErr := fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
		tracing.RecordError(ctx, wrappedErr)
		return wrappedErr
	}

	m.incrementHits()
	m.logger.Debug("Cache hit", "key", key)
	tracing.SetAttributes(ctx, attribute.Bool("cache.hit", true))
	tracing.SetSpanStatus(ctx, codes.Ok, "cache hit")
	return nil
}

// GetOrLoad 获取或加载（防缓存击穿）
func (m *Manager) GetOrLoad(ctx context.Context, key string, dest interface{}, loader LoaderFunc, ttl time.Duration) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "cache.GetOrLoad",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "get_or_load"),
		),
	)
	defer tracing.EndSpan(span)

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		err := ErrManagerAlreadyClosed
		tracing.RecordError(ctx, err)
		return err
	}
	m.mu.RUnlock()

	if loader == nil {
		err := ErrLoaderFuncRequired
		tracing.RecordError(ctx, err)
		return err
	}

	// 先尝试从缓存获取
	err := m.Get(ctx, key, dest)
	if err == nil {
		tracing.SetAttributes(ctx, attribute.Bool("cache.loaded_from_source", false))
		return nil // 缓存命中
	}

	if err != ErrCacheMiss {
		// 其他错误，返回
		tracing.RecordError(ctx, err)
		return err
	}

	// 缓存未命中，使用 singleflight 加载
	fullKey := buildKey(m.config.KeyPrefix, key)
	
	v, err, shared := m.sf.Do(fullKey, func() (interface{}, error) {
		m.incrementLoaderCalls()
		
		// 添加事件：开始加载数据
		tracing.AddEvent(ctx, "loading from source")
		
		// 调用加载函数
		value, err := loader()
		if err != nil {
			tracing.RecordError(ctx, err)
			return nil, err
		}

		// 设置到缓存
		if err := m.Set(ctx, key, value, ttl); err != nil {
			m.logger.Warn("Failed to set cache after load", "key", key, "error", err)
		}

		return value, nil
	})

	if err != nil {
		tracing.RecordError(ctx, err)
		return err
	}

	if shared {
		m.incrementSingleflightHits()
		tracing.SetAttributes(ctx, attribute.Bool("cache.singleflight_hit", true))
	} else {
		tracing.SetAttributes(ctx, attribute.Bool("cache.singleflight_hit", false))
	}

	tracing.SetAttributes(ctx, attribute.Bool("cache.loaded_from_source", true))

	// 将结果复制到 dest
	data, err := m.serializer.Marshal(v)
	if err != nil {
		wrappedErr := fmt.Errorf("%w: %v", ErrSerializationFailed, err)
		tracing.RecordError(ctx, wrappedErr)
		return wrappedErr
	}

	if err := m.serializer.Unmarshal(data, dest); err != nil {
		wrappedErr := fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
		tracing.RecordError(ctx, wrappedErr)
		return wrappedErr
	}

	tracing.SetSpanStatus(ctx, codes.Ok, "data loaded successfully")
	return nil
}

// Delete 删除缓存
func (m *Manager) Delete(ctx context.Context, key string) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "cache.Delete",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "delete"),
		),
	)
	defer tracing.EndSpan(span)

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		err := ErrManagerAlreadyClosed
		tracing.RecordError(ctx, err)
		return err
	}
	m.mu.RUnlock()

	// 构建完整的键
	fullKey := buildKey(m.config.KeyPrefix, key)

	// 从 Redis 删除
	err := m.rdb.Del(ctx, fullKey).Err()
	if err != nil {
		m.incrementError()
		m.logger.Error("Failed to delete cache", "key", key, "error", err)
		tracing.RecordError(ctx, err)
		return err
	}

	m.incrementDeletes()
	m.logger.Debug("Cache deleted", "key", key)
	tracing.SetSpanStatus(ctx, codes.Ok, "cache deleted successfully")
	return nil
}

// Exists 检查键是否存在
func (m *Manager) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return false, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullKey := buildKey(m.config.KeyPrefix, key)
	
	n, err := m.rdb.Exists(ctx, fullKey).Result()
	if err != nil {
		m.incrementError()
		return false, err
	}

	return n > 0, nil
}

// Expire 设置过期时间
func (m *Manager) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	if ttl <= 0 {
		return ErrInvalidTTL
	}

	fullKey := buildKey(m.config.KeyPrefix, key)
	return m.rdb.Expire(ctx, fullKey, ttl).Err()
}

// TTL 获取剩余过期时间
func (m *Manager) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return 0, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullKey := buildKey(m.config.KeyPrefix, key)
	return m.rdb.TTL(ctx, fullKey).Result()
}

// GetMulti 批量获取
func (m *Manager) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// 构建完整的键
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = buildKey(m.config.KeyPrefix, key)
	}

	// 使用 Pipeline 批量获取
	pipe := m.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(fullKeys))
	
	for i, fullKey := range fullKeys {
		cmds[i] = pipe.Get(ctx, fullKey)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		m.incrementError()
		return nil, err
	}

	// 收集结果
	results := make(map[string][]byte)
	for i, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == nil {
			results[keys[i]] = data
		}
	}

	return results, nil
}

// SetMulti 批量设置
func (m *Manager) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "cache.SetMulti",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			attribute.Int("cache.batch_size", len(items)),
			attribute.String("cache.operation", "set_multi"),
		),
	)
	defer tracing.EndSpan(span)

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		err := ErrManagerAlreadyClosed
		tracing.RecordError(ctx, err)
		return err
	}
	m.mu.RUnlock()

	if len(items) == 0 {
		return nil
	}

	if ttl <= 0 {
		ttl = m.config.DefaultExpiration
	}

	tracing.SetAttributes(ctx, attribute.String("cache.ttl", ttl.String()))

	// 使用 Pipeline 批量设置
	pipe := m.rdb.Pipeline()

	for key, value := range items {
		data, err := m.serializer.Marshal(value)
		if err != nil {
			m.logger.Warn("Failed to marshal value", "key", key, "error", err)
			continue
		}

		fullKey := buildKey(m.config.KeyPrefix, key)
		pipe.Set(ctx, fullKey, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		m.incrementError()
		tracing.RecordError(ctx, err)
		return err
	}

	atomic.AddInt64(&m.stats.sets, int64(len(items)))
	tracing.SetSpanStatus(ctx, codes.Ok, fmt.Sprintf("%d items set successfully", len(items)))
	return nil
}

// DeleteMulti 批量删除
func (m *Manager) DeleteMulti(ctx context.Context, keys []string) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	if len(keys) == 0 {
		return nil
	}

	// 构建完整的键
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = buildKey(m.config.KeyPrefix, key)
	}

	err := m.rdb.Del(ctx, fullKeys...).Err()
	if err != nil {
		m.incrementError()
		return err
	}

	atomic.AddInt64(&m.stats.deletes, int64(len(keys)))
	return nil
}

// DeletePattern 模式匹配删除
func (m *Manager) DeletePattern(ctx context.Context, pattern string) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullPattern := buildKey(m.config.KeyPrefix, pattern)
	
	// 使用 SCAN 命令迭代删除
	iter := m.rdb.Scan(ctx, 0, fullPattern, 0).Iterator()
	
	count := 0
	for iter.Next(ctx) {
		if err := m.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			m.logger.Error("Failed to delete key", "key", iter.Val(), "error", err)
			continue
		}
		count++
	}

	if err := iter.Err(); err != nil {
		m.incrementError()
		return err
	}

	atomic.AddInt64(&m.stats.deletes, int64(count))
	m.logger.Info("Pattern deleted", "pattern", pattern, "count", count)
	return nil
}

// Warmup 缓存预热
func (m *Manager) Warmup(ctx context.Context, items []WarmupItem) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "cache.Warmup",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			attribute.Int("cache.warmup_items", len(items)),
			attribute.String("cache.operation", "warmup"),
		),
	)
	defer tracing.EndSpan(span)

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		err := ErrManagerAlreadyClosed
		tracing.RecordError(ctx, err)
		return err
	}
	m.mu.RUnlock()

	if len(items) == 0 {
		return nil
	}

	m.logger.Info("Starting cache warmup", "items", len(items))
	tracing.AddEvent(ctx, "warmup started")

	// 使用 Pipeline 批量设置
	pipe := m.rdb.Pipeline()
	count := 0

	for _, item := range items {
		data, err := m.serializer.Marshal(item.Value)
		if err != nil {
			m.logger.Warn("Failed to marshal warmup item", "key", item.Key, "error", err)
			continue
		}

		fullKey := buildKey(m.config.KeyPrefix, item.Key)
		ttl := item.TTL
		if ttl <= 0 {
			ttl = m.config.DefaultExpiration
		}

		pipe.Set(ctx, fullKey, data, ttl)
		count++
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		m.incrementError()
		tracing.RecordError(ctx, err)
		return err
	}

	m.logger.Info("Cache warmup completed", "loaded", count)
	tracing.SetAttributes(ctx, attribute.Int("cache.warmup_loaded", count))
	tracing.AddEvent(ctx, "warmup completed")
	tracing.SetSpanStatus(ctx, codes.Ok, fmt.Sprintf("%d items warmed up", count))
	return nil
}

// Incr 自增
func (m *Manager) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return 0, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullKey := buildKey(m.config.KeyPrefix, key)
	return m.rdb.Incr(ctx, fullKey).Result()
}

// Decr 自减
func (m *Manager) Decr(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return 0, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullKey := buildKey(m.config.KeyPrefix, key)
	return m.rdb.Decr(ctx, fullKey).Result()
}

// IncrBy 增加指定值
func (m *Manager) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return 0, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullKey := buildKey(m.config.KeyPrefix, key)
	return m.rdb.IncrBy(ctx, fullKey, value).Result()
}

// GetRaw 获取原始字节数据
func (m *Manager) GetRaw(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	fullKey := buildKey(m.config.KeyPrefix, key)
	return m.rdb.Get(ctx, fullKey).Bytes()
}

// Ping 检查 Redis 连接
func (m *Manager) Ping(ctx context.Context) error {
	return m.rdb.Ping(ctx).Err()
}

// RandomExpiration 生成随机过期时间
func (m *Manager) RandomExpiration(baseTTL time.Duration, jitter float64) time.Duration {
	return randomExpiration(baseTTL, jitter)
}

// GetStats 获取统计信息
func (m *Manager) GetStats() *Stats {
	requests := atomic.LoadInt64(&m.stats.requests)
	hits := atomic.LoadInt64(&m.stats.hits)
	misses := atomic.LoadInt64(&m.stats.misses)

	return &Stats{
		Requests:         requests,
		Hits:             hits,
		Misses:           misses,
		HitRate:          calculateHitRate(hits, requests),
		Sets:             atomic.LoadInt64(&m.stats.sets),
		Deletes:          atomic.LoadInt64(&m.stats.deletes),
		Errors:           atomic.LoadInt64(&m.stats.errors),
		LoaderCalls:      atomic.LoadInt64(&m.stats.loaderCalls),
		SingleflightHits: atomic.LoadInt64(&m.stats.singleflightHits),
	}
}

// ResetStats 重置统计
func (m *Manager) ResetStats() {
	atomic.StoreInt64(&m.stats.requests, 0)
	atomic.StoreInt64(&m.stats.hits, 0)
	atomic.StoreInt64(&m.stats.misses, 0)
	atomic.StoreInt64(&m.stats.sets, 0)
	atomic.StoreInt64(&m.stats.deletes, 0)
	atomic.StoreInt64(&m.stats.errors, 0)
	atomic.StoreInt64(&m.stats.loaderCalls, 0)
	atomic.StoreInt64(&m.stats.singleflightHits, 0)

	m.logger.Info("Stats reset")
}

// GetHitRate 获取命中率
func (m *Manager) GetHitRate() float64 {
	requests := atomic.LoadInt64(&m.stats.requests)
	hits := atomic.LoadInt64(&m.stats.hits)
	return calculateHitRate(hits, requests)
}

// GetConfig 获取配置（只读）
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Clone()
}

// 内部方法：增加统计计数器

func (m *Manager) incrementRequests() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.requests, 1)
	}
}

func (m *Manager) incrementHits() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.hits, 1)
	}
}

func (m *Manager) incrementMisses() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.misses, 1)
	}
}

func (m *Manager) incrementSets() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.sets, 1)
	}
}

func (m *Manager) incrementDeletes() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.deletes, 1)
	}
}

func (m *Manager) incrementError() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.errors, 1)
	}
}

func (m *Manager) incrementLoaderCalls() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.loaderCalls, 1)
	}
}

func (m *Manager) incrementSingleflightHits() {
	if m.config.Stats.Enabled {
		atomic.AddInt64(&m.stats.singleflightHits, 1)
	}
}

// startStatsReporter 启动统计上报
func (m *Manager) startStatsReporter() {
	m.cleanupTimer = time.NewTicker(m.config.Stats.ReportInterval)

	go func() {
		for {
			select {
			case <-m.cleanupTimer.C:
				stats := m.GetStats()
				m.logger.Info("Cache stats",
					"requests", stats.Requests,
					"hits", stats.Hits,
					"misses", stats.Misses,
					"hit_rate", fmt.Sprintf("%.2f%%", stats.HitRate*100),
					"sets", stats.Sets,
					"deletes", stats.Deletes,
					"errors", stats.Errors,
					"loader_calls", stats.LoaderCalls,
					"singleflight_hits", stats.SingleflightHits,
				)
			case <-m.cleanupStop:
				return
			}
		}
	}()

	m.logger.Info("Stats reporter started", "interval", m.config.Stats.ReportInterval)
}

// FlushAll 清空所有缓存（危险操作，谨慎使用）
func (m *Manager) FlushAll(ctx context.Context) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	// 只删除带前缀的键
	if m.config.KeyPrefix != "" {
		return m.DeletePattern(ctx, "*")
	}

	// 警告：这会删除整个 Redis 数据库
	m.logger.Warn("Flushing all cache (no prefix set)")
	return m.rdb.FlushDB(ctx).Err()
}

// Health 健康检查
func (m *Manager) Health(ctx context.Context) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return fmt.Errorf("manager is closed")
	}
	m.mu.RUnlock()

	// 检查 Redis 连接
	if err := m.Ping(ctx); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

