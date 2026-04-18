package mq

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestKafkaProducerConsumer(t *testing.T) {
	// 使用唯一的 topic 和 group 避免冲突
	timestamp := time.Now().UnixNano()
	topic := fmt.Sprintf("test-topic-%d", timestamp)
	group := fmt.Sprintf("test-group-%d", timestamp)

	cfg := &Config{
		Driver: DriverKafka,
		Kafka: &KafkaConfig{
			Brokers:       []string{"localhost:9092"},
			ConsumerGroup: group,
		},
	}

	producer, consumer, err := New(cfg)
	if err != nil {
		t.Skipf("跳过测试：无法连接 Kafka: %v", err)
	}
	defer producer.Close()
	defer consumer.Close()

	testMsg := []byte("Hello Kafka")

	// 启动消费者
	received := make(chan []byte, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		consumer.Subscribe(ctx, topic, func(msg []byte) error {
			received <- msg
			return nil
		})
	}()

	// 等待消费者启动和 rebalance 完成
	time.Sleep(5 * time.Second)

	// 发布消息
	if err := producer.Publish(ctx, topic, testMsg); err != nil {
		t.Fatalf("发布消息失败: %v", err)
	}

	// 等待接收消息
	select {
	case msg := <-received:
		if string(msg) != string(testMsg) {
			t.Errorf("消息内容不匹配: got %s, want %s", msg, testMsg)
		}
		cancel() // 收到消息后取消 context
	case <-time.After(20 * time.Second):
		t.Error("超时：未收到消息")
	}
}

func TestKafkaConfig_setDefaults(t *testing.T) {
	cfg := &KafkaConfig{}
	cfg.setDefaults()

	if cfg.ConsumerGroup != "default" {
		t.Errorf("ConsumerGroup = %s, want default", cfg.ConsumerGroup)
	}
	if cfg.Version != "3.0.0" {
		t.Errorf("Version = %s, want 3.0.0", cfg.Version)
	}
	if cfg.Assignor != "range" {
		t.Errorf("Assignor = %s, want range", cfg.Assignor)
	}
	if cfg.RequiredAcks != 1 {
		t.Errorf("RequiredAcks = %d, want 1", cfg.RequiredAcks)
	}
	if cfg.Compression != "snappy" {
		t.Errorf("Compression = %s, want snappy", cfg.Compression)
	}
	if cfg.MaxMessageBytes != 1024*1024 {
		t.Errorf("MaxMessageBytes = %d, want 1048576", cfg.MaxMessageBytes)
	}
	if cfg.RetryMax != 3 {
		t.Errorf("RetryMax = %d, want 3", cfg.RetryMax)
	}
	if cfg.RetryBackoff != 100*time.Millisecond {
		t.Errorf("RetryBackoff = %v, want 100ms", cfg.RetryBackoff)
	}
	if cfg.SessionTimeout != 10*time.Second {
		t.Errorf("SessionTimeout = %v, want 10s", cfg.SessionTimeout)
	}
	if cfg.HeartbeatInterval != 3*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 3s", cfg.HeartbeatInterval)
	}
	if cfg.RebalanceTimeout != 60*time.Second {
		t.Errorf("RebalanceTimeout = %v, want 60s", cfg.RebalanceTimeout)
	}
	if cfg.MaxProcessingTime != 1*time.Second {
		t.Errorf("MaxProcessingTime = %v, want 1s", cfg.MaxProcessingTime)
	}
}

