package middleware

import (
	"sync"
	"time"

	"qi"
	"qi/pkg/logger"

	"go.uber.org/zap"
)

// RateLimiterConfig 限流中间件配置
type RateLimiterConfig struct {
	// RequestsPerSecond 每秒允许的请求数（默认 100）
	RequestsPerSecond float64

	// Burst 突发容量（默认等于 RequestsPerSecond）
	Burst int

	// KeyFunc 自定义限流 key 函数（默认使用客户端 IP）
	KeyFunc func(c *qi.Context) string

	// SkipFunc 跳过限流的函数
	SkipFunc func(c *qi.Context) bool

	// ExcludePaths 排除的路径（不限流）
	ExcludePaths []string

	// Logger 日志实例
	Logger logger.Logger

	// CleanupInterval 过期桶清理间隔（默认 10 分钟）
	CleanupInterval time.Duration

	// BucketExpiry 桶过期时间（默认 30 分钟无访问则清理）
	BucketExpiry time.Duration
}

// defaultRateLimiterConfig 返回默认配置
func defaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerSecond: 100,
		Burst:             100,
		CleanupInterval:   10 * time.Minute,
		BucketExpiry:      30 * time.Minute,
	}
}

// tokenBucket 令牌桶
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// newTokenBucket 创建令牌桶
func newTokenBucket(rate float64, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// allow 检查是否允许请求
func (t *tokenBucket) allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastRefill).Seconds()
	t.tokens += elapsed * t.refillRate
	if t.tokens > t.maxTokens {
		t.tokens = t.maxTokens
	}
	t.lastRefill = now

	if t.tokens >= 1 {
		t.tokens--
		return true
	}
	return false
}

// rateLimiterStore 限流存储
type rateLimiterStore struct {
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	done    chan struct{}
}

// newRateLimiterStore 创建限流存储
func newRateLimiterStore() *rateLimiterStore {
	return &rateLimiterStore{
		buckets: make(map[string]*tokenBucket),
		done:    make(chan struct{}),
	}
}

// getBucket 获取或创建令牌桶
func (s *rateLimiterStore) getBucket(key string, rate float64, burst int) *tokenBucket {
	s.mu.RLock()
	bucket, exists := s.buckets[key]
	s.mu.RUnlock()

	if exists {
		return bucket
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if bucket, exists = s.buckets[key]; exists {
		return bucket
	}

	bucket = newTokenBucket(rate, burst)
	s.buckets[key] = bucket
	return bucket
}

// cleanup 清理过期的令牌桶
func (s *rateLimiterStore) cleanup(expiry time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, bucket := range s.buckets {
		bucket.mu.Lock()
		expired := now.Sub(bucket.lastRefill) > expiry
		bucket.mu.Unlock()
		if expired {
			delete(s.buckets, key)
		}
	}
}

// RateLimiter 创建限流中间件
// 使用令牌桶算法，按 key（默认客户端 IP）进行限流
func RateLimiter(cfgs ...*RateLimiterConfig) qi.HandlerFunc {
	cfg := defaultRateLimiterConfig()
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	// 默认使用客户端 IP 作为 key
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *qi.Context) string {
			return c.ClientIP()
		}
	}

	// 默认日志
	if cfg.Logger == nil {
		var err error
		cfg.Logger, err = logger.NewDevelopment()
		if err != nil {
			panic("qi/middleware: failed to create rate limiter logger: " + err.Error())
		}
	}

	// 构建跳过路径 map
	skipMap := make(map[string]bool)
	for _, path := range cfg.ExcludePaths {
		skipMap[path] = true
	}

	// 每个中间件实例独立存储
	limiterStore := newRateLimiterStore()

	// 启动后台清理 goroutine（可通过 close(done) 停止）
	go func() {
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				limiterStore.cleanup(cfg.BucketExpiry)
			case <-limiterStore.done:
				return
			}
		}
	}()

	return func(c *qi.Context) {
		// 检查是否跳过
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}
		if skipMap[c.Request().URL.Path] {
			c.Next()
			return
		}

		key := cfg.KeyFunc(c)
		bucket := limiterStore.getBucket(key, cfg.RequestsPerSecond, cfg.Burst)

		if !bucket.allow() {
			cfg.Logger.Warn("rate limit exceeded",
				zap.String("key", key),
				zap.String("path", c.Request().URL.Path),
				zap.Float64("rate", cfg.RequestsPerSecond),
			)
			c.Fail(429, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
