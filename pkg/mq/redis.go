package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// redisProducer Redis Streams 生产者
type redisProducer struct {
	client *redis.Client
	maxLen int64
}

// redisConsumer Redis Streams 消费者
type redisConsumer struct {
	client        *redis.Client
	consumerGroup string
	consumerName  string
	blockTimeout  time.Duration
	batchSize     int64
	maxRetries    int
	minIdleTime   time.Duration
	onError       func(error)

	// 优雅关闭
	shutdown chan struct{}
	done     chan struct{}
	wg       sync.WaitGroup
	closeOnce sync.Once
}

// newRedis 创建 Redis 生产者和消费者
func newRedis(cfg *RedisConfig) (Producer, Consumer, error) {
	cfg.setDefaults()

	// 创建两个独立的客户端，避免 Close() 互相影响
	producerClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	consumerClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := producerClient.Ping(ctx).Err(); err != nil {
		producerClient.Close()
		consumerClient.Close()
		return nil, nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 生成消费者名称
	consumerName := cfg.ConsumerName
	if consumerName == "" {
		consumerName = fmt.Sprintf("consumer-%s", uuid.New().String()[:8])
	}

	producer := &redisProducer{
		client: producerClient,
		maxLen: cfg.MaxLen,
	}

	consumer := &redisConsumer{
		client:        consumerClient,
		consumerGroup: cfg.ConsumerGroup,
		consumerName:  consumerName,
		blockTimeout:  cfg.BlockTimeout,
		batchSize:     cfg.BatchSize,
		maxRetries:    cfg.MaxRetries,
		minIdleTime:   cfg.MinIdleTime,
		onError:       cfg.OnError,
		shutdown:      make(chan struct{}),
		done:          make(chan struct{}),
	}

	return producer, consumer, nil
}

// Publish 发布消息
func (p *redisProducer) Publish(ctx context.Context, topic string, msg []byte) error {
	args := &redis.XAddArgs{
		Stream: topic,
		Values: map[string]any{
			"data": msg,
		},
	}

	if p.maxLen > 0 {
		args.MaxLen = p.maxLen
		args.Approx = true // 使用近似裁剪，性能更好
	}

	return p.client.XAdd(ctx, args).Err()
}

// Close 关闭生产者
func (p *redisProducer) Close() error {
	return p.client.Close()
}

// Subscribe 订阅消息
func (c *redisConsumer) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	// 创建消费组（如果不存在）
	err := c.client.XGroupCreateMkStream(ctx, topic, c.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// 异步处理 Pending 列表，避免阻塞启动
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.processPending(ctx, topic, handler); err != nil {
			c.handleError(fmt.Errorf("failed to process pending messages: %w", err))
		}
	}()

	// 持续消费新消息
	defer close(c.done)
	for {
		select {
		case <-ctx.Done():
			c.wg.Wait() // 等待 Pending 处理完成
			return ctx.Err()
		case <-c.shutdown:
			c.wg.Wait() // 等待当前消息处理完成
			return nil
		default:
		}

		// 使用子 context 控制 XReadGroup 超时，避免阻塞导致无法退出
		readCtx, cancel := context.WithTimeout(ctx, c.blockTimeout)
		streams, err := c.client.XReadGroup(readCtx, &redis.XReadGroupArgs{
			Group:    c.consumerGroup,
			Consumer: c.consumerName,
			Streams:  []string{topic, ">"},
			Count:    c.batchSize,
			Block:    c.blockTimeout,
		}).Result()
		cancel()

		if err != nil {
			if err == redis.Nil || readCtx.Err() == context.DeadlineExceeded {
				continue // 超时，继续等待
			}
			// 检查是否是 ctx 取消导致的错误
			if ctx.Err() != nil {
				c.wg.Wait()
				return ctx.Err()
			}
			c.handleError(fmt.Errorf("failed to read from stream: %w", err))
			continue
		}

		// 批量处理消息
		c.wg.Add(1)
		c.processMessages(ctx, topic, streams, handler)
		c.wg.Done()
	}
}

// processMessages 批量处理消息并批量确认
func (c *redisConsumer) processMessages(ctx context.Context, topic string, streams []redis.XStream, handler func([]byte) error) {
	var ackIDs []string

	for _, stream := range streams {
		for _, message := range stream.Messages {
			// 检查关闭信号
			select {
			case <-ctx.Done():
				return
			case <-c.shutdown:
				return
			default:
			}

			if err := c.processMessage(ctx, topic, message, handler); err != nil {
				c.handleError(err)
				continue
			}

			// 收集需要确认的消息 ID
			ackIDs = append(ackIDs, message.ID)
		}
	}

	// 批量确认
	if len(ackIDs) > 0 {
		if err := c.client.XAck(ctx, topic, c.consumerGroup, ackIDs...).Err(); err != nil {
			c.handleError(fmt.Errorf("failed to ack messages: %w", err))
		}
	}
}

// processPending 处理 Pending 列表中的未确认消息
func (c *redisConsumer) processPending(ctx context.Context, topic string, handler func([]byte) error) error {
	// 循环处理直到 Pending 列表为空
	for {
		// 检查取消信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.shutdown:
			return nil
		default:
		}

		// 读取 Pending 消息
		pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: topic,
			Group:  c.consumerGroup,
			Start:  "-",
			End:    "+",
			Count:  c.batchSize,
		}).Result()

		if err != nil && err != redis.Nil {
			return err
		}

		// 没有 Pending 消息，退出
		if len(pending) == 0 {
			return nil
		}

		// 处理每条 Pending 消息
		for _, msg := range pending {
			// 检查取消信号
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.shutdown:
				return nil
			default:
			}

			// 检查重试次数
			if c.maxRetries > 0 && int(msg.RetryCount) >= c.maxRetries {
				// 超过最大重试次数，确认消息避免无限重试
				c.client.XAck(ctx, topic, c.consumerGroup, msg.ID)
				c.handleError(fmt.Errorf("message %s exceeded max retries (%d)", msg.ID, c.maxRetries))
				continue
			}

			// 认领空闲时间超过阈值的消息
			claimed, err := c.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   topic,
				Group:    c.consumerGroup,
				Consumer: c.consumerName,
				MinIdle:  c.minIdleTime,
				Messages: []string{msg.ID},
			}).Result()

			if err != nil {
				continue
			}

			for _, message := range claimed {
				if err := c.processMessage(ctx, topic, message, handler); err != nil {
					c.handleError(err)
					continue
				}
				// 单独确认 Pending 消息
				c.client.XAck(ctx, topic, c.consumerGroup, message.ID)
			}
		}
	}
}

// processMessage 处理单条消息
func (c *redisConsumer) processMessage(ctx context.Context, topic string, message redis.XMessage, handler func([]byte) error) error {
	data, ok := message.Values["data"].(string)
	if !ok {
		// 数据格式错误，确认消息避免重复处理
		c.client.XAck(ctx, topic, c.consumerGroup, message.ID)
		return fmt.Errorf("invalid message format for message %s", message.ID)
	}

	// 调用处理函数
	if err := handler([]byte(data)); err != nil {
		return fmt.Errorf("handler error for message %s: %w", message.ID, err)
	}

	return nil
}

// handleError 处理错误
func (c *redisConsumer) handleError(err error) {
	if c.onError != nil {
		c.onError(err)
	}
}

// Close 关闭消费者（优雅关闭）
func (c *redisConsumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.shutdown) // 发送关闭信号
		<-c.done          // 等待 Subscribe 退出
		err = c.client.Close()
	})
	return err
}
