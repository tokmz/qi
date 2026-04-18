package mq

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/tokmz/qi/pkg/mq"

// tracingProducer 追踪装饰器 - 生产者
type tracingProducer struct {
	producer Producer
	tracer   trace.Tracer
}

// tracingConsumer 追踪装饰器 - 消费者
type tracingConsumer struct {
	consumer Consumer
	tracer   trace.Tracer
}

// newTracingProducer 创建追踪生产者
func newTracingProducer(producer Producer) Producer {
	return &tracingProducer{
		producer: producer,
		tracer:   otel.Tracer(tracerName),
	}
}

// newTracingConsumer 创建追踪消费者
func newTracingConsumer(consumer Consumer) Consumer {
	return &tracingConsumer{
		consumer: consumer,
		tracer:   otel.Tracer(tracerName),
	}
}

// Publish 发布消息（带追踪）
func (t *tracingProducer) Publish(ctx context.Context, topic string, msg []byte) error {
	ctx, span := t.tracer.Start(ctx, "mq.Publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("mq.topic", topic),
			attribute.Int("mq.message.size", len(msg)),
		),
	)
	defer span.End()

	err := t.producer.Publish(ctx, topic, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Close 关闭生产者
func (t *tracingProducer) Close() error {
	return t.producer.Close()
}

// Subscribe 订阅消息（带追踪）
func (t *tracingConsumer) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	// 包装 handler，为每条消息创建 span
	wrappedHandler := func(msg []byte) error {
		msgCtx, span := t.tracer.Start(ctx, "mq.Consume",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("mq.topic", topic),
				attribute.Int("mq.message.size", len(msg)),
			),
		)
		defer span.End()

		// 将带追踪的 context 传递给 handler（如果 handler 需要）
		// 注意：当前 handler 签名不接受 context，这里仅用于追踪
		_ = msgCtx

		err := handler(msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		span.SetStatus(codes.Ok, "")
		return nil
	}

	// 创建订阅 span
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("mq.Subscribe:%s", topic),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("mq.topic", topic),
		),
	)
	defer span.End()

	err := t.consumer.Subscribe(ctx, topic, wrappedHandler)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// Close 关闭消费者
func (t *tracingConsumer) Close() error {
	return t.consumer.Close()
}
