package ws

import (
	"sync"
	"sync/atomic"
	"time"
)

// EventType 事件类型
type EventType string

const (
	// EventClientConnected 客户端连接
	EventClientConnected EventType = "client.connected"
	// EventClientDisconnected 客户端断开
	EventClientDisconnected EventType = "client.disconnected"
	// EventMessageReceived 收到消息
	EventMessageReceived EventType = "message.received"
	// EventMessageSent 发送消息
	EventMessageSent EventType = "message.sent"
	// EventRoomJoined 加入房间
	EventRoomJoined EventType = "room.joined"
	// EventRoomLeft 离开房间
	EventRoomLeft EventType = "room.left"
	// EventError 错误
	EventError EventType = "error"
)

// Event 事件
type Event struct {
	Type     EventType
	ClientID string
	Data     any
	Time     time.Time
}

// EventHandler 事件处理器
type EventHandler func(Event)

// EventBus 事件总线
type EventBus struct {
	handlers      map[EventType][]EventHandler
	mu            sync.RWMutex
	workerCh      chan func()
	stopCh        chan struct{}
	wg            sync.WaitGroup
	closed        atomic.Bool
	droppedEvents atomic.Int64 // 丢弃的事件计数
}

// NewEventBus 创建事件总线
func NewEventBus() *EventBus {
	eb := &EventBus{
		handlers: make(map[EventType][]EventHandler),
		workerCh: make(chan func(), 1000), // 缓冲队列
		stopCh:   make(chan struct{}),
	}

	// 启动固定数量的 worker
	for i := 0; i < 10; i++ {
		eb.wg.Add(1)
		go eb.worker()
	}

	return eb
}

// worker 工作协程
func (eb *EventBus) worker() {
	defer eb.wg.Done()
	for {
		select {
		case task := <-eb.workerCh:
			task()
		case <-eb.stopCh:
			return
		}
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish 发布事件（异步）
func (eb *EventBus) Publish(event Event) {
	// 检查是否已关闭
	if eb.closed.Load() {
		return
	}

	eb.mu.RLock()
	handlers, ok := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return
	}

	// 提交到 worker 池
	for _, handler := range handlers {
		h := handler // 捕获变量

		// 对于关键事件（连接/断开），使用阻塞发送
		if event.Type == EventClientConnected || event.Type == EventClientDisconnected {
			select {
			case eb.workerCh <- func() { h(event) }:
			case <-time.After(100 * time.Millisecond):
				// 超时后丢弃，避免阻塞
				eb.droppedEvents.Add(1)
			}
		} else {
			// 非关键事件，非阻塞发送
			select {
			case eb.workerCh <- func() { h(event) }:
			default:
				// 队列满时丢弃事件
				eb.droppedEvents.Add(1)
			}
		}
	}
}

// Close 关闭事件总线
func (eb *EventBus) Close() {
	// 标记为已关闭
	eb.closed.Store(true)

	close(eb.stopCh)
	eb.wg.Wait()

	// 不关闭 workerCh，避免并发 Publish 导致 panic
	// 剩余的事件会被丢弃，channel 会被 GC
}

// GetDroppedEventCount 获取丢弃的事件数量
func (eb *EventBus) GetDroppedEventCount() int64 {
	return eb.droppedEvents.Load()
}

// ResetDroppedEventCount 重置丢弃的事件计数（用于监控周期重置）
func (eb *EventBus) ResetDroppedEventCount() int64 {
	return eb.droppedEvents.Swap(0)
}
