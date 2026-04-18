# pkg/mq

统一的消息队列封装，支持 **Redis Streams**、**RabbitMQ** 和 **Kafka**，提供 Producer/Consumer 接口、链路追踪、优雅关闭、批量确认等生产级特性。

## 快速开始

### Redis Streams

```go
package main

import (
    "context"
    "log"
    
    "github.com/tokmz/qi/pkg/mq"
)

func main() {
    producer, consumer, err := mq.New(&mq.Config{
        Driver:         mq.DriverRedis,
        TracingEnabled: true,
        Redis: &mq.RedisConfig{
            Addr:          "127.0.0.1:6379",
            Password:      "your-password",
            ConsumerGroup: "my-group",
            MaxRetries:    3,
            OnError: func(err error) {
                log.Printf("MQ Error: %v", err)
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer producer.Close()
    defer consumer.Close()

    // 发布消息
    producer.Publish(context.Background(), "orders", []byte("order-123"))

    // 消费消息
    consumer.Subscribe(context.Background(), "orders", func(msg []byte) error {
        log.Printf("收到: %s", msg)
        return nil // 返回 nil 确认消息
    })
}
```

### RabbitMQ

```go
producer, consumer, err := mq.New(&mq.Config{
    Driver:         mq.DriverRabbitMQ,
    TracingEnabled: true,
    RabbitMQ: &mq.RabbitMQConfig{
        URL:      "amqp://user:pass@localhost:5672/",
        Exchange: "my-exchange",
        Durable:  true,
        OnError: func(err error) {
            log.Printf("MQ Error: %v", err)
        },
    },
})
```

### Kafka

```go
producer, consumer, err := mq.New(&mq.Config{
    Driver:         mq.DriverKafka,
    TracingEnabled: true,
    Kafka: &mq.KafkaConfig{
        Brokers:       []string{"localhost:9092"},
        ConsumerGroup: "my-group",
        Compression:   "snappy",
        OnError: func(err error) {
            log.Printf("MQ Error: %v", err)
        },
    },
})
```

## 配置

### RedisConfig

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `Addr` | `string` | Redis 地址 | - |
| `Password` | `string` | 密码 | - |
| `DB` | `int` | 数据库编号 | 0 |
| `ConsumerGroup` | `string` | 消费组名称 | `"default"` |
| `ConsumerName` | `string` | 消费者名称 | 自动生成 |
| `MaxLen` | `int64` | Stream 最大长度（0=不限制） | 0 |
| `BlockTimeout` | `time.Duration` | 阻塞读取超时 | 5s |
| `BatchSize` | `int64` | 每次读取消息数 | 10 |
| `MaxRetries` | `int` | 最大重试次数（0=无限） | 3 |
| `MinIdleTime` | `time.Duration` | 认领 Pending 消息的最小空闲时间 | 5m |
| `OnError` | `func(error)` | 错误回调（可选） | nil |

### RabbitMQConfig

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `URL` | `string` | 连接 URL | - |
| `Exchange` | `string` | 交换机名称 | `""` (默认交换机) |
| `ExchangeType` | `string` | 交换机类型：direct/fanout/topic/headers | `"direct"` |
| `Durable` | `bool` | 队列是否持久化 | `true` |
| `AutoDelete` | `bool` | 队列是否自动删除 | `false` |
| `PrefetchCount` | `int` | 预取消息数 | 1 |
| `AutoAck` | `bool` | 是否自动确认 | `false` (手动确认) |
| `OnError` | `func(error)` | 错误回调（可选） | nil |

### KafkaConfig

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `Brokers` | `[]string` | Broker 地址列表 | - |
| `ConsumerGroup` | `string` | 消费组名称 | `"default"` |
| `Version` | `string` | Kafka 版本 | `"3.0.0"` |
| `Assignor` | `string` | 分区分配策略：range/roundrobin/sticky | `"range"` |
| `AutoCommit` | `bool` | 是否自动提交 offset | `false` (手动提交) |
| `RequiredAcks` | `int` | 需要的 ACK 数：0/1/-1 | 1 |
| `Compression` | `string` | 压缩算法：none/gzip/snappy/lz4/zstd | `"snappy"` |
| `MaxMessageBytes` | `int` | 最大消息大小 | 1MB |
| `RetryMax` | `int` | 发送失败重试次数 | 3 |
| `RetryBackoff` | `time.Duration` | 重试间隔 | 100ms |
| `SessionTimeout` | `time.Duration` | 会话超时 | 10s |
| `HeartbeatInterval` | `time.Duration` | 心跳间隔 | 3s |
| `RebalanceTimeout` | `time.Duration` | Rebalance 超时 | 60s |
| `MaxProcessingTime` | `time.Duration` | 最大处理时间 | 1s |
| `OnError` | `func(error)` | 错误回调（可选） | nil |

## 特性对比

| 特性 | Redis Streams | RabbitMQ | Kafka |
|------|---------------|----------|-------|
| 消费组 | ✅ | ✅ | ✅ |
| 持久化 | ✅ | ✅ | ✅ |
| 优先级队列 | ❌ | ✅ | ❌ |
| 延迟队列 | ❌ | ✅ (插件) | ❌ |
| 死信队列 | ❌ | ✅ | ❌ |
| 路由模式 | 简单 | 丰富 (direct/fanout/topic/headers) | Topic 分区 |
| 消息回溯 | ✅ | ❌ | ✅ |
| 吞吐量 | 极高 (10w+ msg/s) | 高 (1w+ msg/s) | 极高 (100w+ msg/s) |
| 延迟 | 极低 (ms) | 低 (ms) | 低 (ms) |
| 运维复杂度 | 低 | 中 | 高 |
| 适用场景 | 轻量级队列 | 复杂路由 | 大数据流处理 |

## 核心特性

### 统一接口
- `Producer` / `Consumer` 接口，切换驱动无需改代码
- 支持 Redis Streams、RabbitMQ 和 Kafka

### 可靠性保障
- **Pending 处理** (Redis): 启动时异步处理未确认消息
- **最大重试** (Redis): 超过 `MaxRetries` 自动确认，防止无限重试
- **手动确认** (RabbitMQ): 处理失败自动 Nack + Requeue
- **批量确认** (Redis): 批量 ACK 提升性能
- **消费组** (Kafka): 自动负载均衡和故障转移
- **Offset 提交** (Kafka): 手动提交 offset，确保消息不丢失

### 优雅关闭
- `Close()` 等待当前消息处理完成
- 独立客户端，Producer/Consumer 互不影响
- 支持 context 取消和 shutdown 信号

### 可观测性
- 错误回调 `OnError` 记录所有错误
- 链路追踪集成 OpenTelemetry
- 自动创建 `mq.Publish` / `mq.Consume` span

## 使用示例

### 基础用法

```go
producer, consumer, _ := mq.New(&mq.Config{
    Driver: mq.DriverRedis,
    Redis:  &mq.RedisConfig{Addr: "127.0.0.1:6379"},
})

// 发布
producer.Publish(ctx, "topic", []byte("data"))

// 消费
consumer.Subscribe(ctx, "topic", func(msg []byte) error {
    // 处理消息
    return nil
})
```

### 错误处理

```go
producer, consumer, _ := mq.New(&mq.Config{
    Driver: mq.DriverRedis,
    Redis: &mq.RedisConfig{
        Addr: "127.0.0.1:6379",
        OnError: func(err error) {
            log.Printf("[MQ Error] %v", err)
        },
    },
})
```

### 优雅关闭

```go
ctx, cancel := context.WithCancel(context.Background())

go consumer.Subscribe(ctx, "topic", handler)

// 收到信号时
cancel()              // 取消 context
consumer.Close()      // 等待当前消息处理完成
```

### 链路追踪

```go
producer, consumer, _ := mq.New(&mq.Config{
    Driver:         mq.DriverRedis,
    TracingEnabled: true, // 启用追踪
    Redis:          &mq.RedisConfig{Addr: "127.0.0.1:6379"},
})

// 自动创建 span：mq.Publish / mq.Consume
```

### RabbitMQ 交换机模式

```go
// Direct 模式（默认）
producer, consumer, _ := mq.New(&mq.Config{
    Driver: mq.DriverRabbitMQ,
    RabbitMQ: &mq.RabbitMQConfig{
        URL:          "amqp://guest:guest@localhost:5672/",
        Exchange:     "logs",
        ExchangeType: "direct",
    },
})

// Fanout 模式（广播）
producer, consumer, _ := mq.New(&mq.Config{
    Driver: mq.DriverRabbitMQ,
    RabbitMQ: &mq.RabbitMQConfig{
        URL:          "amqp://guest:guest@localhost:5672/",
        Exchange:     "broadcast",
        ExchangeType: "fanout",
    },
})

// Topic 模式（通配符路由）
producer, consumer, _ := mq.New(&mq.Config{
    Driver: mq.DriverRabbitMQ,
    RabbitMQ: &mq.RabbitMQConfig{
        URL:          "amqp://guest:guest@localhost:5672/",
        Exchange:     "events",
        ExchangeType: "topic",
    },
})
```

### Kafka 消费组

```go
// 多个消费者自动负载均衡
producer, consumer, _ := mq.New(&mq.Config{
    Driver: mq.DriverKafka,
    Kafka: &mq.KafkaConfig{
        Brokers:       []string{"localhost:9092"},
        ConsumerGroup: "order-processors",
        Assignor:      "sticky", // 粘性分配策略
    },
})
```

## 消息可靠性

### Redis Streams

| 问题 | 方案 |
|------|------|
| 消息丢失 | Pending 列表 + 启动时重新处理 |
| 重复消费 | 消费组 + ACK 机制 |
| 无限重试 | MaxRetries 限制 + 自动确认 |
| 消费者宕机 | MinIdleTime 后其他消费者认领 |

### RabbitMQ

| 问题 | 方案 |
|------|------|
| 消息丢失 | 持久化队列 + 持久化消息 |
| 重复消费 | 手动确认 + 幂等处理 |
| 处理失败 | Nack + Requeue 重试 |
| 消费者宕机 | 自动重新分配未确认消息 |

### Kafka

| 问题 | 方案 |
|------|------|
| 消息丢失 | 持久化 + 副本机制 |
| 重复消费 | Offset 提交 + 幂等处理 |
| 处理失败 | 不提交 offset，下次重新消费 |
| 消费者宕机 | Rebalance 自动重新分配分区 |
| 消息回溯 | 重置 offset 到指定位置 |

## 性能优化

### Redis Streams
- **批量读取**: `BatchSize` 控制每次读取消息数
- **批量确认**: 自动批量 ACK，减少 Redis 往返
- **近似裁剪**: `MaxLen` 使用 `XTRIM MAXLEN ~ N`，性能更好
- **阻塞读取**: 避免轮询，降低 CPU 和网络开销

### RabbitMQ
- **预取控制**: `PrefetchCount` 控制未确认消息数
- **持久化权衡**: 非关键消息可关闭 `Durable` 提升性能
- **批量发布**: 业务层批量调用 `Publish`

### Kafka
- **批量发送**: 自动批量发送消息
- **压缩**: 使用 `snappy`/`lz4`/`zstd` 压缩
- **分区**: 合理设置 topic 分区数，提升并行度
- **消费组**: 多消费者并行消费不同分区

## 示例

- Redis Streams: [`examples/mq_test/main.go`](../../examples/mq_test/main.go)
- RabbitMQ: [`examples/rabbitmq_test/main.go`](../../examples/rabbitmq_test/main.go)
- Kafka: [`examples/kafka_test/main.go`](../../examples/kafka_test/main.go)

## 测试

```bash
# Redis Streams（需要本地 Redis）
go test ./pkg/mq/ -v -run TestRedis

# RabbitMQ（需要本地 RabbitMQ）
go test ./pkg/mq/ -v -run TestRabbitMQ

# Kafka（需要本地 Kafka）
go test ./pkg/mq/ -v -run TestKafka

# 所有测试
go test ./pkg/mq/ -v
```

## 架构设计

```
New() → 根据 Driver 创建对应实现
  ↓
TracingEnabled? → 包装追踪装饰器
  ↓
Subscribe() → 启动消费循环
  ↓
Redis: 异步处理 Pending → 批量消费 → 批量 ACK
RabbitMQ: 声明队列/交换机 → 消费 → 手动 ACK/Nack
Kafka: 加入消费组 → 分区分配 → 消费 → 提交 Offset
  ↓
Close() → 等待处理完成 → 关闭连接
```

## 选型建议

**选择 Redis Streams**
- 已有 Redis 基础设施
- 简单的发布/订阅场景
- 追求极致性能
- 运维成本敏感

**选择 RabbitMQ**
- 需要复杂路由（topic/fanout）
- 需要延迟队列、死信队列
- 需要优先级队列
- 消息可靠性要求极高

**选择 Kafka**
- 大数据量、高吞吐场景
- 需要消息回溯和重放
- 日志收集、事件溯源
- 流式数据处理

