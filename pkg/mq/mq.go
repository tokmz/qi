// Package mq 提供统一的消息队列接口和实现
package mq

import (
	"context"
	"fmt"
)

// Producer 消息生产者接口
type Producer interface {
	// Publish 发布消息到指定主题
	Publish(ctx context.Context, topic string, msg []byte) error
	// Close 关闭生产者
	Close() error
}

// Consumer 消息消费者接口
type Consumer interface {
	// Subscribe 订阅主题并处理消息
	// handler 返回 error 时消息不会被确认（根据驱动实现可能重试）
	Subscribe(ctx context.Context, topic string, handler func([]byte) error) error
	// Close 关闭消费者
	Close() error
}

// Driver 消息队列驱动类型
type Driver string

const (
	DriverRedis    Driver = "redis"    // Redis Streams
	DriverRabbitMQ Driver = "rabbitmq" // RabbitMQ
	DriverKafka    Driver = "kafka"    // Kafka
)

// Config 消息队列配置
type Config struct {
	Driver         Driver          // 驱动类型
	TracingEnabled bool            // 是否启用链路追踪
	Redis          *RedisConfig    // Redis 配置
	RabbitMQ       *RabbitMQConfig // RabbitMQ 配置
	Kafka          *KafkaConfig    // Kafka 配置
}

// New 创建消息队列生产者和消费者
func New(cfg *Config) (Producer, Consumer, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is nil")
	}

	var producer Producer
	var consumer Consumer
	var err error

	switch cfg.Driver {
	case DriverRedis:
		if cfg.Redis == nil {
			return nil, nil, fmt.Errorf("redis config is required for redis driver")
		}
		producer, consumer, err = newRedis(cfg.Redis)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create redis mq: %w", err)
		}
	case DriverRabbitMQ:
		if cfg.RabbitMQ == nil {
			return nil, nil, fmt.Errorf("rabbitmq config is required for rabbitmq driver")
		}
		producer, consumer, err = newRabbitMQ(cfg.RabbitMQ)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create rabbitmq mq: %w", err)
		}
	case DriverKafka:
		if cfg.Kafka == nil {
			return nil, nil, fmt.Errorf("kafka config is required for kafka driver")
		}
		producer, consumer, err = newKafka(cfg.Kafka)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create kafka mq: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}

	// 包装追踪装饰器
	if cfg.TracingEnabled {
		producer = newTracingProducer(producer)
		consumer = newTracingConsumer(consumer)
	}

	return producer, consumer, nil
}
