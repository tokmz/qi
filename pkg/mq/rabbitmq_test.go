package mq

import (
	"context"
	"testing"
	"time"
)

func TestRabbitMQProducerConsumer(t *testing.T) {
	cfg := &Config{
		Driver: DriverRabbitMQ,
		RabbitMQ: &RabbitMQConfig{
			URL: "amqp://aikzy:wui11413@localhost:5672/",
		},
	}

	producer, consumer, err := New(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 RabbitMQ: %v", err)
	}
	defer producer.Close()
	defer consumer.Close()

	topic := "test-rabbitmq-topic"
	testMsg := []byte("hello rabbitmq")
	received := make(chan []byte, 1)

	// 启动消费者
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		err := consumer.Subscribe(ctx, topic, func(msg []byte) error {
			received <- msg
			cancel() // 收到消息后取消
			return nil
		})
		if err != nil && err != context.Canceled {
			t.Errorf("消费失败: %v", err)
		}
	}()

	// 等待消费者准备好
	time.Sleep(100 * time.Millisecond)

	// 发布消息
	if err := producer.Publish(context.Background(), topic, testMsg); err != nil {
		t.Fatalf("发布失败: %v", err)
	}

	// 验证接收
	select {
	case msg := <-received:
		if string(msg) != string(testMsg) {
			t.Errorf("消息不匹配: got %s, want %s", msg, testMsg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("超时：未收到消息")
	}
}

func TestRabbitMQConfig_setDefaults(t *testing.T) {
	cfg := &RabbitMQConfig{}
	cfg.setDefaults()

	if cfg.ExchangeType != "direct" {
		t.Errorf("ExchangeType = %s, want direct", cfg.ExchangeType)
	}
	if cfg.PrefetchCount != 1 {
		t.Errorf("PrefetchCount = %d, want 1", cfg.PrefetchCount)
	}
	// Durable 和 AutoDelete 没有默认值，由用户显式设置
	if cfg.Durable {
		t.Error("Durable should be false by default (user must set explicitly)")
	}
	if cfg.AutoDelete {
		t.Error("AutoDelete should be false by default")
	}
}

func TestNew_RabbitMQDriver(t *testing.T) {
	cfg := &Config{
		Driver: DriverRabbitMQ,
		RabbitMQ: &RabbitMQConfig{
			URL: "amqp://aikzy:wui11413@localhost:5672/",
		},
	}

	producer, consumer, err := New(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 RabbitMQ: %v", err)
	}
	defer producer.Close()
	defer consumer.Close()

	if producer == nil || consumer == nil {
		t.Fatal("producer or consumer is nil")
	}
}

func TestNew_MissingRabbitMQConfig(t *testing.T) {
	cfg := &Config{
		Driver: DriverRabbitMQ,
	}

	_, _, err := New(cfg)
	if err == nil {
		t.Fatal("期望错误，但成功了")
	}
}
