package cache

import "errors"

// 预定义错误
var (
	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("invalid cache config")

	// ErrRedisClientRequired Redis 客户端必需
	ErrRedisClientRequired = errors.New("redis client is required")

	// ErrCacheMiss 缓存未命中
	ErrCacheMiss = errors.New("cache miss")

	// ErrKeyNotFound 键不存在
	ErrKeyNotFound = errors.New("key not found")

	// ErrSerializationFailed 序列化失败
	ErrSerializationFailed = errors.New("serialization failed")

	// ErrDeserializationFailed 反序列化失败
	ErrDeserializationFailed = errors.New("deserialization failed")

	// ErrInvalidSerializer 无效的序列化器
	ErrInvalidSerializer = errors.New("invalid serializer type")

	// ErrNilValue 空值
	ErrNilValue = errors.New("nil value")

	// ErrInvalidTTL 无效的 TTL
	ErrInvalidTTL = errors.New("invalid TTL")

	// ErrManagerNotInitialized 管理器未初始化
	ErrManagerNotInitialized = errors.New("cache manager not initialized")

	// ErrManagerAlreadyClosed 管理器已关闭
	ErrManagerAlreadyClosed = errors.New("cache manager already closed")

	// ErrLoaderFuncRequired 加载函数必需
	ErrLoaderFuncRequired = errors.New("loader function is required")

	// ErrNotFound 数据不存在
	ErrNotFound = errors.New("data not found")
)

