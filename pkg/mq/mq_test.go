package mq

import (
	"context"
	"testing"
	"time"
)

func TestRedisProducerConsumer(t *testing.T) {
	cfg := &Config{
		Driver: DriverRedis,
		Redis: &RedisConfig{
			Addr: "127.0.0.1:6379",
		},
	}

	producer, consumer, err := New(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}
	defer producer.Close()
	defer consumer.Close()

	topic := "test-topic"
	testMsg := []byte("hello world")
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

func TestRedisConfig_setDefaults(t *testing.T) {
	cfg := &RedisConfig{}
	cfg.setDefaults()

	if cfg.ConsumerGroup != "default" {
		t.Errorf("ConsumerGroup = %s, want default", cfg.ConsumerGroup)
	}
	if cfg.BlockTimeout != 5*time.Second {
		t.Errorf("BlockTimeout = %v, want 5s", cfg.BlockTimeout)
	}
	if cfg.BatchSize != 10 {
		t.Errorf("BatchSize = %d, want 10", cfg.BatchSize)
	}
}

func TestNew_InvalidDriver(t *testing.T) {
	cfg := &Config{
		Driver: "invalid",
	}

	_, _, err := New(cfg)
	if err == nil {
		t.Fatal("期望错误，但成功了")
	}
}

func TestNew_NilConfig(t *testing.T) {
	_, _, err := New(nil)
	if err == nil {
		t.Fatal("期望错误，但成功了")
	}
}
