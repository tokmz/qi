package cache

import "fmt"

// New 创建缓存实例
func New(cfg *Config) (Cache, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 设置默认序列化器
	if cfg.Serializer == nil {
		cfg.Serializer = &JSONSerializer{}
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 根据驱动类型创建实例
	switch cfg.Driver {
	case DriverRedis:
		return newRedisCache(cfg)
	case DriverMemory:
		return newMemoryCache(cfg)
	default:
		return nil, fmt.Errorf("%w: unsupported driver type", ErrCacheInvalidConfig)
	}
}

// NewWithOptions 使用 Options 模式创建缓存实例
func NewWithOptions(opts ...Option) (Cache, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return New(cfg)
}
