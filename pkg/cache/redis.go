package cache

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCache Redis 缓存实现
type redisCache struct {
	client     redis.UniversalClient
	serializer Serializer
	keyPrefix  string
	defaultTTL time.Duration
}

// newRedisCache 创建 Redis 缓存实例
func newRedisCache(cfg *Config) (Cache, error) {
	if cfg.Redis == nil {
		return nil, fmt.Errorf("%w: redis config is required", ErrCacheInvalidConfig)
	}

	var client redis.UniversalClient

	switch cfg.Redis.Mode {
	case RedisStandalone, "":
		// 单机模式
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Redis.Addr,
			Username:     cfg.Redis.Username,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			MaxRetries:   cfg.Redis.MaxRetries,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		})

	case RedisCluster:
		// 集群模式
		if len(cfg.Redis.Addrs) == 0 {
			return nil, fmt.Errorf("%w: cluster mode requires addrs", ErrCacheInvalidConfig)
		}
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Redis.Addrs,
			Username:     cfg.Redis.Username,
			Password:     cfg.Redis.Password,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			MaxRetries:   cfg.Redis.MaxRetries,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		})

	case RedisSentinel:
		// 哨兵模式
		if len(cfg.Redis.Addrs) == 0 {
			return nil, fmt.Errorf("%w: sentinel mode requires addrs", ErrCacheInvalidConfig)
		}
		if cfg.Redis.MasterName == "" {
			return nil, fmt.Errorf("%w: sentinel mode requires master name", ErrCacheInvalidConfig)
		}
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.Redis.MasterName,
			SentinelAddrs: cfg.Redis.Addrs,
			Username:      cfg.Redis.Username,
			Password:      cfg.Redis.Password,
			DB:            cfg.Redis.DB,
			PoolSize:      cfg.Redis.PoolSize,
			MinIdleConns:  cfg.Redis.MinIdleConns,
			MaxRetries:    cfg.Redis.MaxRetries,
			DialTimeout:   cfg.Redis.DialTimeout,
			ReadTimeout:   cfg.Redis.ReadTimeout,
			WriteTimeout:  cfg.Redis.WriteTimeout,
		})

	default:
		return nil, fmt.Errorf("%w: unsupported redis mode: %s", ErrCacheInvalidConfig, cfg.Redis.Mode)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCacheConnection, err)
	}

	return &redisCache{
		client:     client,
		serializer: cfg.Serializer,
		keyPrefix:  cfg.KeyPrefix,
		defaultTTL: cfg.DefaultTTL,
	}, nil
}

// buildKey 构建完整的键名
func (r *redisCache) buildKey(key string) string {
	if r.keyPrefix == "" {
		return key
	}
	return r.keyPrefix + key
}

// Get 获取缓存
func (r *redisCache) Get(ctx context.Context, key string, value any) error {
	fullKey := r.buildKey(key)

	data, err := r.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheNotFound
		}
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	if err := r.serializer.Unmarshal(data, value); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheSerialization, err)
	}

	return nil
}

// Set 设置缓存
func (r *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	fullKey := r.buildKey(key)

	bytes, err := r.serializer.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCacheSerialization, err)
	}

	if ttl == 0 {
		ttl = r.defaultTTL
	}

	if err := r.client.Set(ctx, fullKey, bytes, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return nil
}

// Delete 删除缓存
func (r *redisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	if err := r.client.Del(ctx, fullKeys...).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return nil
}

// Exists 检查键是否存在
func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.buildKey(key)

	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return count > 0, nil
}

// MGet 批量获取
func (r *redisCache) MGet(ctx context.Context, keys []string, values any) error {
	if len(keys) == 0 {
		return nil
	}

	// 检查 values 是否为切片指针
	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("%w: values must be a pointer to slice", ErrCacheOperation)
	}

	slice := rv.Elem()
	elemType := slice.Type().Elem()

	// 构建完整键名
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = r.buildKey(key)
	}

	// 批量获取
	results, err := r.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	// 清空切片
	slice.Set(reflect.MakeSlice(slice.Type(), 0, len(keys)))

	// 反序列化结果
	for _, result := range results {
		if result == nil {
			continue
		}

		str, ok := result.(string)
		if !ok {
			continue
		}

		// 创建新元素
		elem := reflect.New(elemType)
		if err := r.serializer.Unmarshal([]byte(str), elem.Interface()); err != nil {
			continue
		}

		// 追加到切片
		slice.Set(reflect.Append(slice, elem.Elem()))
	}

	return nil
}

// MSet 批量设置（使用 Pipeline）
func (r *redisCache) MSet(ctx context.Context, items map[string]any, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	if ttl == 0 {
		ttl = r.defaultTTL
	}

	// 使用 Pipeline 批量设置
	pipe := r.client.Pipeline()

	for key, value := range items {
		fullKey := r.buildKey(key)

		bytes, err := r.serializer.Marshal(value)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrCacheSerialization, err)
		}

		pipe.Set(ctx, fullKey, bytes, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return nil
}

// MSetTx 批量设置（使用事务，确保原子性）
// 如果任何一个 item 失败，整个事务回滚
func (r *redisCache) MSetTx(ctx context.Context, items map[string]any, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	if ttl == 0 {
		ttl = r.defaultTTL
	}

	// 使用 Watch 监控所有键
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, r.buildKey(key))
	}

	err := r.client.Watch(ctx, func(tx *redis.Tx) error {
		// 监控所有键
		pipe := tx.Pipeline()
		for key, value := range items {
			fullKey := r.buildKey(key)
			bytes, err := r.serializer.Marshal(value)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrCacheSerialization, err)
			}
			pipe.Set(ctx, fullKey, bytes, ttl)
		}
		_, err := pipe.Exec(ctx)
		return err
	}, keys...)

	if err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return nil
}

// TTL 获取键的剩余生存时间
func (r *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.buildKey(key)

	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	if ttl == -2 {
		return 0, ErrCacheNotFound
	}

	if ttl == -1 {
		return -1, nil // 永不过期
	}

	return ttl, nil
}

// Expire 设置键的过期时间
func (r *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := r.buildKey(key)

	ok, err := r.client.Expire(ctx, fullKey, ttl).Result()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	if !ok {
		return ErrCacheNotFound
	}

	return nil
}

// Incr 自增
func (r *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)

	val, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return val, nil
}

// Decr 自减
func (r *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)

	val, err := r.client.Decr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return val, nil
}

// IncrBy 增加指定值
func (r *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	fullKey := r.buildKey(key)

	val, err := r.client.IncrBy(ctx, fullKey, value).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}

	return val, nil
}

// Ping 检查连接
func (r *redisCache) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheConnection, err)
	}
	return nil
}

// Close 关闭连接
func (r *redisCache) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrCacheOperation, err)
	}
	return nil
}

// String 返回缓存类型
func (r *redisCache) String() string {
	return fmt.Sprintf("RedisCache(prefix=%s)", r.keyPrefix)
}
