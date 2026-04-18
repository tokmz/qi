package mq

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
)

// kafkaProducer Kafka 生产者
type kafkaProducer struct {
	producer sarama.SyncProducer
	onError  func(error)
}

// kafkaConsumer Kafka 消费者
type kafkaConsumer struct {
	client        sarama.ConsumerGroup
	consumerGroup string
	onError       func(error)
	shutdown      chan struct{}
	done          chan struct{}
}

// newKafka 创建 Kafka 生产者和消费者
func newKafka(cfg *KafkaConfig) (Producer, Consumer, error) {
	cfg.setDefaults()

	// 解析 Kafka 版本
	version, err := sarama.ParseKafkaVersion(cfg.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse kafka version: %w", err)
	}

	// Producer 配置
	producerConfig := sarama.NewConfig()
	producerConfig.Version = version
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.RequiredAcks = sarama.RequiredAcks(cfg.RequiredAcks)
	producerConfig.Producer.Retry.Max = cfg.RetryMax
	producerConfig.Producer.Retry.Backoff = cfg.RetryBackoff
	producerConfig.Producer.MaxMessageBytes = cfg.MaxMessageBytes

	// 设置压缩算法
	switch cfg.Compression {
	case "gzip":
		producerConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		producerConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		producerConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		producerConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		producerConfig.Producer.Compression = sarama.CompressionNone
	}

	// 创建 Producer
	producer, err := sarama.NewSyncProducer(cfg.Brokers, producerConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	// Consumer 配置
	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = version
	consumerConfig.Consumer.Return.Errors = true
	consumerConfig.Consumer.Offsets.AutoCommit.Enable = cfg.AutoCommit
	consumerConfig.Consumer.Group.Session.Timeout = cfg.SessionTimeout
	consumerConfig.Consumer.Group.Heartbeat.Interval = cfg.HeartbeatInterval
	consumerConfig.Consumer.Group.Rebalance.Timeout = cfg.RebalanceTimeout
	consumerConfig.Consumer.MaxProcessingTime = cfg.MaxProcessingTime

	// 设置分区分配策略
	switch cfg.Assignor {
	case "roundrobin":
		consumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	case "sticky":
		consumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategySticky()}
	default:
		consumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	}

	// 创建 Consumer Group
	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, consumerConfig)
	if err != nil {
		producer.Close()
		return nil, nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	return &kafkaProducer{
			producer: producer,
			onError:  cfg.OnError,
		}, &kafkaConsumer{
			client:        client,
			consumerGroup: cfg.ConsumerGroup,
			onError:       cfg.OnError,
			shutdown:      make(chan struct{}),
			done:          make(chan struct{}),
		}, nil
}

// Publish 发布消息
func (p *kafkaProducer) Publish(ctx context.Context, topic string, msg []byte) error {
	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	}

	_, _, err := p.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Close 关闭生产者
func (p *kafkaProducer) Close() error {
	return p.producer.Close()
}

// Subscribe 订阅主题
func (c *kafkaConsumer) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	defer close(c.done)

	// 创建消费者处理器
	consumerHandler := &kafkaConsumerHandler{
		handler: handler,
		onError: c.onError,
	}

	// 消费消息（Consume 会自动处理 rebalance 和重连）
	// 注意：Consume 是阻塞调用，只有在 ctx 取消或发生致命错误时才会返回
	err := c.client.Consume(ctx, []string{topic}, consumerHandler)
	if err != nil {
		c.handleError(fmt.Errorf("consumer error: %w", err))
		return err
	}

	return nil
}

// Close 关闭消费者
func (c *kafkaConsumer) Close() error {
	close(c.shutdown)
	<-c.done
	return c.client.Close()
}

// handleError 处理错误
func (c *kafkaConsumer) handleError(err error) {
	if c.onError != nil {
		c.onError(err)
	}
}

// kafkaConsumerHandler 实现 sarama.ConsumerGroupHandler 接口
type kafkaConsumerHandler struct {
	handler func([]byte) error
	onError func(error)
	ready   chan bool
}

// Setup 在消费者组 rebalance 之前调用
func (h *kafkaConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	if h.ready != nil {
		close(h.ready)
	}
	return nil
}

// Cleanup 在消费者组 rebalance 之后调用
func (h *kafkaConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 处理消息
func (h *kafkaConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	var wg sync.WaitGroup
	defer wg.Wait() // 确保所有 goroutine 完成

	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			wg.Add(1)
			go func(message *sarama.ConsumerMessage) {
				defer wg.Done()

				// 处理消息
				if err := h.handler(message.Value); err != nil {
					if h.onError != nil {
						h.onError(fmt.Errorf("handler error: %w", err))
					}
					return
				}

				// 标记消息已处理
				session.MarkMessage(message, "")
			}(msg)
		}
	}
}
