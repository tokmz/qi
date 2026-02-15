package cache

import "github.com/tokmz/qi/pkg/errors"

// 预定义错误
var (
	ErrCacheNotFound      = errors.New(3001, 404, "cache key not found", nil)
	ErrCacheExpired       = errors.New(3002, 404, "cache key expired", nil)
	ErrCacheConnection    = errors.New(3003, 500, "cache connection failed", nil)
	ErrCacheSerialization = errors.New(3004, 500, "cache serialization failed", nil)
	ErrCacheInvalidConfig = errors.New(3005, 500, "cache invalid config", nil)
	ErrCacheOperation     = errors.New(3006, 500, "cache operation failed", nil)
)
