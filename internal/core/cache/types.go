package cache

import (
	"time"
)

// SerializerType 序列化器类型
type SerializerType string

const (
	// SerializerJSON JSON 序列化器
	SerializerJSON SerializerType = "json"

	// SerializerMsgPack MessagePack 序列化器
	SerializerMsgPack SerializerType = "msgpack"

	// SerializerGob Gob 序列化器
	SerializerGob SerializerType = "gob"
)

// String 返回序列化器类型的字符串表示
func (s SerializerType) String() string {
	return string(s)
}

// LoaderFunc 数据加载函数类型
// 当缓存未命中时，调用此函数从数据源加载数据
type LoaderFunc func() (interface{}, error)

// WarmupItem 预热项
type WarmupItem struct {
	// Key 缓存键
	Key string

	// Value 缓存值
	Value interface{}

	// TTL 过期时间
	TTL time.Duration
}

// Stats 缓存统计信息
type Stats struct {
	// Requests 总请求数
	Requests int64 `json:"requests"`

	// Hits 命中数
	Hits int64 `json:"hits"`

	// Misses 未命中数
	Misses int64 `json:"misses"`

	// HitRate 命中率
	HitRate float64 `json:"hit_rate"`

	// Sets 设置次数
	Sets int64 `json:"sets"`

	// Deletes 删除次数
	Deletes int64 `json:"deletes"`

	// Errors 错误次数
	Errors int64 `json:"errors"`

	// LoaderCalls 加载函数调用次数
	LoaderCalls int64 `json:"loader_calls"`

	// SingleflightHits Singleflight 命中次数
	SingleflightHits int64 `json:"singleflight_hits"`
}

// CacheItem 缓存项（内部使用）
type CacheItem struct {
	// Key 键
	Key string

	// Value 值
	Value []byte

	// ExpiresAt 过期时间
	ExpiresAt time.Time
}

// IsExpired 检查是否过期
func (i *CacheItem) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// TTL 获取剩余时间
func (i *CacheItem) TTL() time.Duration {
	if i.IsExpired() {
		return 0
	}
	return time.Until(i.ExpiresAt)
}

