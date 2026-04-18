package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// rabbitmqProducer RabbitMQ 生产者
type rabbitmqProducer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

// rabbitmqConsumer RabbitMQ 消费者
type rabbitmqConsumer struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	exchange      string
	exchangeType  string
	durable       bool
	autoDelete    bool
	prefetchCount int
	autoAck       bool
	onError       func(error)

	// 重连配置
	url           string
	reconnectWait time.Duration

	// 优雅关闭
	shutdown  chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// newRabbitMQ 创建 RabbitMQ 生产者和消费者
func newRabbitMQ(cfg *RabbitMQConfig) (Producer, Consumer, error) {
	cfg.setDefaults()

	// 创建生产者连接
	producerConn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to rabbitmq (producer): %w", err)
	}

	producerChannel, err := producerConn.Channel()
	if err != nil {
		producerConn.Close()
		return nil, nil, fmt.Errorf("failed to open producer channel: %w", err)
	}

	// 创建消费者连接
	consumerConn, err := amqp.Dial(cfg.URL)
	if err != nil {
		producerChannel.Close()
		producerConn.Close()
		return nil, nil, fmt.Errorf("failed to connect to rabbitmq (consumer): %w", err)
	}

	consumerChannel, err := consumerConn.Channel()
	if err != nil {
		producerChannel.Close()
		producerConn.Close()
		consumerConn.Close()
		return nil, nil, fmt.Errorf("failed to open consumer channel: %w", err)
	}

	// 设置 QoS
	if err := consumerChannel.Qos(cfg.PrefetchCount, 0, false); err != nil {
		producerChannel.Close()
		producerConn.Close()
		consumerChannel.Close()
		consumerConn.Close()
		return nil, nil, fmt.Errorf("failed to set qos: %w", err)
	}

	// 声明交换机（如果指定）
	if cfg.Exchange != "" {
		if err := producerChannel.ExchangeDeclare(
			cfg.Exchange,
			cfg.ExchangeType,
			cfg.Durable,
			cfg.AutoDelete,
			false, // internal
			false, // no-wait
			nil,   // arguments
		); err != nil {
			producerChannel.Close()
			producerConn.Close()
			consumerChannel.Close()
			consumerConn.Close()
			return nil, nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	producer := &rabbitmqProducer{
		conn:     producerConn,
		channel:  producerChannel,
		exchange: cfg.Exchange,
	}

	consumer := &rabbitmqConsumer{
		conn:          consumerConn,
		channel:       consumerChannel,
		exchange:      cfg.Exchange,
		exchangeType:  cfg.ExchangeType,
		durable:       cfg.Durable,
		autoDelete:    cfg.AutoDelete,
		prefetchCount: cfg.PrefetchCount,
		autoAck:       cfg.AutoAck,
		onError:       cfg.OnError,
		url:           cfg.URL,
		reconnectWait: 5 * time.Second,
		shutdown:      make(chan struct{}),
		done:          make(chan struct{}),
	}

	return producer, consumer, nil
}

// Publish 发布消息
func (p *rabbitmqProducer) Publish(ctx context.Context, topic string, msg []byte) error {
	return p.channel.PublishWithContext(
		ctx,
		p.exchange, // exchange
		topic,      // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/octet-stream",
			Body:         msg,
			DeliveryMode: amqp.Persistent, // 持久化消息
		},
	)
}

// Close 关闭生产者
func (p *rabbitmqProducer) Close() error {
	if err := p.channel.Close(); err != nil {
		p.conn.Close()
		return err
	}
	return p.conn.Close()
}

// Subscribe 订阅消息（带自动重连）
func (c *rabbitmqConsumer) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	defer close(c.done)

	for {
		// 检查关闭信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.shutdown:
			return nil
		default:
		}

		// 尝试订阅
		err := c.subscribe(ctx, topic, handler)

		// 正常退出
		if err == nil || err == context.Canceled || err == context.DeadlineExceeded {
			return err
		}

		// 连接错误，尝试重连
		c.handleError(fmt.Errorf("subscription error: %w, reconnecting in %v", err, c.reconnectWait))

		// 等待后重连
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.shutdown:
			return nil
		case <-time.After(c.reconnectWait):
			// 重新建立连接
			if err := c.reconnect(); err != nil {
				c.handleError(fmt.Errorf("reconnect failed: %w", err))
				continue
			}
		}
	}
}

// subscribe 实际订阅逻辑
func (c *rabbitmqConsumer) subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	// 声明交换机（如果指定）
	if c.exchange != "" {
		if err := c.channel.ExchangeDeclare(
			c.exchange,
			c.exchangeType,
			c.durable,
			c.autoDelete,
			false, // internal
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	// 声明队列
	queue, err := c.channel.QueueDeclare(
		topic,          // name
		c.durable,      // durable
		c.autoDelete,   // auto-delete
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机（如果指定）
	if c.exchange != "" {
		if err := c.channel.QueueBind(
			queue.Name,
			topic,      // routing key
			c.exchange, // exchange
			false,      // no-wait
			nil,        // arguments
		); err != nil {
			return fmt.Errorf("failed to bind queue: %w", err)
		}
	}

	// 开始消费
	msgs, err := c.channel.Consume(
		queue.Name,
		"",         // consumer tag
		c.autoAck,  // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// 持续消费
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.shutdown:
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("consumer channel closed")
			}

			// 处理消息
			if err := handler(msg.Body); err != nil {
				c.handleError(fmt.Errorf("handler error: %w", err))
				// 手动确认模式下，处理失败则 Nack
				if !c.autoAck {
					msg.Nack(false, true) // requeue
				}
				continue
			}

			// 手动确认
			if !c.autoAck {
				if err := msg.Ack(false); err != nil {
					c.handleError(fmt.Errorf("failed to ack message: %w", err))
				}
			}
		}
	}
}

// reconnect 重新建立连接
func (c *rabbitmqConsumer) reconnect() error {
	// 关闭旧连接
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}

	// 创建新连接
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 设置 QoS
	if err := channel.Qos(c.prefetchCount, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to set qos: %w", err)
	}

	c.conn = conn
	c.channel = channel
	return nil
}

// handleError 处理错误
func (c *rabbitmqConsumer) handleError(err error) {
	if c.onError != nil {
		c.onError(err)
	}
}

// Close 关闭消费者（优雅关闭）
func (c *rabbitmqConsumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.shutdown) // 发送关闭信号

		// 等待 Subscribe 退出（如果已启动）
		// 使用 select 避免死锁：如果 Subscribe 未启动，done 不会被关闭
		select {
		case <-c.done:
			// Subscribe 已退出
		case <-time.After(100 * time.Millisecond):
			// Subscribe 未启动或已经在关闭中
		}

		if e := c.channel.Close(); e != nil {
			err = e
		}
		if e := c.conn.Close(); e != nil && err == nil {
			err = e
		}
	})
	return err
}
