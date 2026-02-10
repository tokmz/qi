package ws

// Metrics 监控接口
type Metrics interface {
	// 连接指标
	IncrementConnections()
	DecrementConnections()
	SetConnectionCount(count int)

	// 消息指标
	IncrementMessageCount(msgType string)
	RecordMessageLatency(msgType string, duration int64)
	IncrementMessageErrors(msgType string)

	// 房间指标
	SetRoomCount(count int)
	SetRoomMemberCount(roomID string, count int)

	// 性能指标
	RecordBroadcastLatency(duration int64)
	IncrementDroppedMessages()

	// 错误指标
	IncrementReadErrors()
	IncrementWriteErrors()
	IncrementInvalidMessages()
}

// NoopMetrics 空实现（默认）
type NoopMetrics struct{}

func (m *NoopMetrics) IncrementConnections()                               {}
func (m *NoopMetrics) DecrementConnections()                               {}
func (m *NoopMetrics) SetConnectionCount(count int)                        {}
func (m *NoopMetrics) IncrementMessageCount(msgType string)                {}
func (m *NoopMetrics) RecordMessageLatency(msgType string, duration int64) {}
func (m *NoopMetrics) IncrementMessageErrors(msgType string)               {}
func (m *NoopMetrics) SetRoomCount(count int)                              {}
func (m *NoopMetrics) SetRoomMemberCount(roomID string, count int)         {}
func (m *NoopMetrics) RecordBroadcastLatency(duration int64)               {}
func (m *NoopMetrics) IncrementDroppedMessages()                           {}
func (m *NoopMetrics) IncrementReadErrors()                                {}
func (m *NoopMetrics) IncrementWriteErrors()                               {}
func (m *NoopMetrics) IncrementInvalidMessages()                           {}
