package job

import "time"

// 重试相关常量
const (
	// DefaultRetryAttempts 默认重试次数
	DefaultRetryAttempts = 3
	// DefaultRetryBaseDelay 默认重试基础延迟
	DefaultRetryBaseDelay = time.Second
)

// 超时相关常量
const (
	// DefaultSyncTimeout 默认同步操作超时
	DefaultSyncTimeout = 5 * time.Second
	// DefaultAsyncTimeout 默认异步操作超时
	DefaultAsyncTimeout = 10 * time.Second
)

// 批量更新相关常量
const (
	// DefaultBatchSize 默认批量大小
	DefaultBatchSize = 10
	// DefaultBatchFlushInterval 默认批量刷新间隔
	DefaultBatchFlushInterval = time.Second
)

// 缓存相关常量
const (
	// DefaultCacheCapacity 默认缓存容量
	DefaultCacheCapacity = 100
	// DefaultCacheTTL 默认缓存 TTL
	DefaultCacheTTL = 5 * time.Minute
	// DefaultCacheCleanupInterval 默认缓存清理间隔
	DefaultCacheCleanupInterval = time.Minute
)

// 调度器相关常量
const (
	// DefaultConcurrentRuns 默认并发执行数
	DefaultConcurrentRuns = 5
	// DefaultJobTimeout 默认任务超时时间
	DefaultJobTimeout = 5 * time.Minute
	// DefaultRetryDelay 默认重试间隔
	DefaultRetryDelay = 5 * time.Second
	// DefaultSchedulerTickInterval 默认调度器轮询间隔
	DefaultSchedulerTickInterval = time.Second
)
