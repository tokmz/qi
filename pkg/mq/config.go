package mq

import "time"

// RedisConfig Redis Streams 配置
type RedisConfig struct {
	Addr          string        // Redis 地址，如 "127.0.0.1:6379"
	Password      string        // 密码
	DB            int           // 数据库编号
	ConsumerGroup string        // 消费组名称（默认 "default"）
	ConsumerName  string        // 消费者名称（默认自动生成）
	MaxLen        int64         // Stream 最大长度（0 表示不限制）
	BlockTimeout  time.Duration // 阻塞读取超时（默认 5s）
	BatchSize     int64         // 每次读取消息数（默认 10）
	MaxRetries    int           // 消息最大重试次数（0 表示无限重试，默认 3）
	MinIdleTime   time.Duration // 认领 Pending 消息的最小空闲时间（默认 5 分钟）
	OnError       func(error)   // 错误回调（可选，用于日志记录）
}

// setDefaults 设置默认值
func (c *RedisConfig) setDefaults() {
	if c.ConsumerGroup == "" {
		c.ConsumerGroup = "default"
	}
	if c.BlockTimeout == 0 {
		c.BlockTimeout = 5 * time.Second
	}
	if c.BatchSize == 0 {
		c.BatchSize = 10
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.MinIdleTime == 0 {
		c.MinIdleTime = 5 * time.Minute
	}
}

// RabbitMQConfig RabbitMQ 配置
type RabbitMQConfig struct {
	URL           string      // RabbitMQ 连接 URL，如 "amqp://user:pass@localhost:5672/"
	Exchange      string      // 交换机名称（默认 ""，使用默认交换机）
	ExchangeType  string      // 交换机类型：direct/fanout/topic/headers（默认 "direct"）
	Durable       bool        // 队列是否持久化（默认 false）
	AutoDelete    bool        // 队列是否自动删除（默认 false）
	PrefetchCount int         // 预取消息数（默认 1）
	AutoAck       bool        // 是否自动确认（默认 false，手动确认）
	OnError       func(error) // 错误回调（可选）
}

// setDefaults 设置默认值
func (c *RabbitMQConfig) setDefaults() {
	if c.ExchangeType == "" {
		c.ExchangeType = "direct"
	}
	if c.PrefetchCount == 0 {
		c.PrefetchCount = 1
	}
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers       []string      // Kafka broker 地址列表，如 []string{"localhost:9092"}
	ConsumerGroup string        // 消费组名称（默认 "default"）
	Version       string        // Kafka 版本（默认 "3.0.0"）
	Assignor      string        // 分区分配策略：range/roundrobin/sticky（默认 "range"）
	AutoCommit    bool          // 是否自动提交 offset（默认 false，手动提交）
	OnError       func(error)   // 错误回调（可选）

	// Producer 配置
	RequiredAcks      int           // 需要的 ACK 数：0/1/-1（默认 1）
	Compression       string        // 压缩算法：none/gzip/snappy/lz4/zstd（默认 "snappy"）
	MaxMessageBytes   int           // 最大消息大小（默认 1MB）
	RetryMax          int           // 发送失败重试次数（默认 3）
	RetryBackoff      time.Duration // 重试间隔（默认 100ms）

	// Consumer 配置
	SessionTimeout    time.Duration // 会话超时（默认 10s）
	HeartbeatInterval time.Duration // 心跳间隔（默认 3s）
	RebalanceTimeout  time.Duration // Rebalance 超时（默认 60s）
	MaxProcessingTime time.Duration // 最大处理时间（默认 1s）
}

// setDefaults 设置默认值
func (c *KafkaConfig) setDefaults() {
	if c.ConsumerGroup == "" {
		c.ConsumerGroup = "default"
	}
	if c.Version == "" {
		c.Version = "3.0.0"
	}
	if c.Assignor == "" {
		c.Assignor = "range"
	}
	if c.RequiredAcks == 0 {
		c.RequiredAcks = 1
	}
	if c.Compression == "" {
		c.Compression = "snappy"
	}
	if c.MaxMessageBytes == 0 {
		c.MaxMessageBytes = 1024 * 1024 // 1MB
	}
	if c.RetryMax == 0 {
		c.RetryMax = 3
	}
	if c.RetryBackoff == 0 {
		c.RetryBackoff = 100 * time.Millisecond
	}
	if c.SessionTimeout == 0 {
		c.SessionTimeout = 10 * time.Second
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = 3 * time.Second
	}
	if c.RebalanceTimeout == 0 {
		c.RebalanceTimeout = 60 * time.Second
	}
	if c.MaxProcessingTime == 0 {
		c.MaxProcessingTime = 1 * time.Second
	}
}
