package cache

import "errors"

// 预定义错误
var (
	ErrCacheNotFound      = errors.New("cache key not found")
	ErrCacheExpired       = errors.New("cache key expired")
	ErrCacheConnection    = errors.New("cache connection failed")
	ErrCacheSerialization = errors.New("cache serialization failed")
	ErrCacheInvalidConfig = errors.New("cache invalid config")
	ErrCacheOperation     = errors.New("cache operation failed")
)
