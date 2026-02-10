package ws

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Client WebSocket 客户端
type Client struct {
	ID      string
	conn    *websocket.Conn
	manager *Manager

	// 发送队列
	send     chan []byte
	sendHigh chan []byte // 高优先级队列（系统消息）

	// 元数据
	UserID   int64
	metadata sync.Map

	// 房间
	rooms sync.Map // roomID -> bool

	// 心跳
	lastPong atomic.Int64 // Unix timestamp

	// 生命周期
	ctx       context.Context
	cancel    context.CancelFunc
	closed    atomic.Bool
	closeOnce sync.Once
	writeDone chan struct{} // 标记 writePump 已退出

	// 限流
	invalidMsgCount atomic.Int32 // 无效消息计数

	// 配置
	config *ClientConfig
}

// ClientConfig 客户端配置
type ClientConfig struct {
	SendQueueSize     int
	SendHighQueueSize int
	WriteWait         time.Duration
	PongWait          time.Duration
	MaxMessageSize    int64
}

// ClientOption 客户端选项
type ClientOption func(*Client)

// WithClientID 设置客户端 ID
func WithClientID(id string) ClientOption {
	return func(c *Client) {
		c.ID = id
	}
}

// WithUserID 设置用户 ID
func WithUserID(uid int64) ClientOption {
	return func(c *Client) {
		c.UserID = uid
	}
}

// WithMetadata 设置元数据
func WithMetadata(key string, value any) ClientOption {
	return func(c *Client) {
		c.metadata.Store(key, value)
	}
}

// NewClient 创建客户端
func NewClient(conn *websocket.Conn, manager *Manager, opts ...ClientOption) *Client {
	ctx, cancel := context.WithCancel(manager.ctx)

	config := &ClientConfig{
		SendQueueSize:     manager.config.MessageQueueSize,
		SendHighQueueSize: manager.config.HighPriorityQueueSize,
		WriteWait:         10 * time.Second,
		PongWait:          manager.config.HeartbeatTimeout,
		MaxMessageSize:    manager.config.MaxMessageSize,
	}

	client := &Client{
		ID:        generateClientID(),
		conn:      conn,
		manager:   manager,
		send:      make(chan []byte, config.SendQueueSize),
		sendHigh:  make(chan []byte, config.SendHighQueueSize),
		ctx:       ctx,
		cancel:    cancel,
		config:    config,
		writeDone: make(chan struct{}),
	}

	// 应用选项
	for _, opt := range opts {
		opt(client)
	}

	// 初始化心跳时间
	client.lastPong.Store(time.Now().Unix())

	return client
}

// Run 运行客户端
func (c *Client) Run() {
	var wg sync.WaitGroup
	wg.Add(2)

	// 读协程
	go func() {
		defer wg.Done()
		c.readPump()
	}()

	// 写协程
	go func() {
		defer wg.Done()
		c.writePump()
	}()

	wg.Wait()
	c.Close()
}

// readPump 读取消息
func (c *Client) readPump() {
	defer func() {
		c.Close()
	}()

	c.conn.SetReadLimit(c.config.MaxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait)); err != nil {
		if c.manager.metrics != nil {
			c.manager.metrics.IncrementReadErrors()
		}
		return
	}
	c.conn.SetPongHandler(func(string) error {
		c.lastPong.Store(time.Now().Unix())
		return c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				// 区分错误类型
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// 异常关闭，记录日志（如果有日志系统）
				}
				return
			}

			// 解析消息
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				// 记录无效消息指标
				if c.manager.metrics != nil {
					c.manager.metrics.IncrementInvalidMessages()
				}
				// 累计无效消息次数
				count := c.invalidMsgCount.Add(1)
				if count > 10 {
					// 超过阈值，关闭连接
					return
				}
				// 尝试发送错误响应，忽略发送失败
				_ = c.SendError("", 400, "invalid message format")
				continue
			}

			// 成功解析，重置计数器
			c.invalidMsgCount.Store(0)

			// 发布消息接收事件
			if c.manager.events != nil {
				c.manager.events.Publish(Event{
					Type:     EventMessageReceived,
					ClientID: c.ID,
					Data:     &msg,
					Time:     time.Now(),
				})
			}

			// 路由消息
			if err := c.manager.router.Route(c, &msg); err != nil {
				// 尝试发送错误响应，忽略发送失败
				_ = c.SendError(msg.RequestID, 500, err.Error())
			}
		}
	}
}

// writePump 写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(c.manager.config.HeartbeatInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		close(c.writeDone) // 标记 writePump 已退出
	}()

	for {
		select {
		case <-c.ctx.Done():
			// 尝试发送关闭消息，忽略错误
			_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message, ok := <-c.sendHigh:
			// 高优先级消息
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.writeMessage(message); err != nil {
				return
			}

		case message, ok := <-c.send:
			// 普通消息
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.writeMessage(message); err != nil {
				return
			}

		case <-ticker.C:
			// 发送心跳
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// writeMessage 写入消息
func (c *Client) writeMessage(message []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait)); err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, message)
}

// SendBytes 发送字节消息（非阻塞）
func (c *Client) SendBytes(msg []byte) error {
	if c.closed.Load() {
		return ErrConnectionClosed
	}

	select {
	case c.send <- msg:
		return nil
	default:
		return ErrChannelFull
	}
}

// SendBytesHigh 发送高优先级字节消息
func (c *Client) SendBytesHigh(msg []byte) error {
	if c.closed.Load() {
		return ErrConnectionClosed
	}

	select {
	case c.sendHigh <- msg:
		return nil
	default:
		return ErrChannelFull
	}
}

// SendJSON 发送 JSON 消息
func (c *Client) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.SendBytes(data)
}

// SendResponse 发送响应
func (c *Client) SendResponse(requestID string, code int, message string, data any) error {
	resp := NewResponse(requestID, code, message, data)
	return c.SendJSON(resp)
}

// SendError 发送错误响应
func (c *Client) SendError(requestID string, code int, message string) error {
	errResp := NewErrorResponse(requestID, code, message)
	return c.SendJSON(errResp)
}

// Close 关闭客户端
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.cancel()

		// 从连接池移除
		c.manager.pool.Remove(c.ID)

		// 收集所有房间ID（避免在遍历时修改）
		roomIDs := make([]string, 0, 8)
		c.rooms.Range(func(roomID, _ any) bool {
			if id, ok := roomID.(string); ok {
				roomIDs = append(roomIDs, id)
			}
			return true
		})

		// 从所有房间移除
		for _, roomID := range roomIDs {
			c.manager.rooms.LeaveRoom(c, roomID)
		}

		// 关闭连接（会触发 writePump 退出）
		c.conn.Close()

		// 等待 writePump 退出后再关闭通道，使用超时避免永久阻塞
		go func() {
			select {
			case <-c.writeDone:
				// writePump 已退出，安全关闭 channel
				close(c.send)
				close(c.sendHigh)
			case <-time.After(5 * time.Second):
				// 超时保护：writePump 可能未启动，直接关闭
				close(c.send)
				close(c.sendHigh)
			}
		}()

		// 发布断开事件
		if c.manager.events != nil {
			c.manager.events.Publish(Event{
				Type:     EventClientDisconnected,
				ClientID: c.ID,
				Time:     time.Now(),
			})
		}
	})
}

// IsClosed 检查是否已关闭
func (c *Client) IsClosed() bool {
	return c.closed.Load()
}

// GetMetadata 获取元数据
func (c *Client) GetMetadata(key string) (any, bool) {
	return c.metadata.Load(key)
}

// SetMetadata 设置元数据
func (c *Client) SetMetadata(key string, value any) {
	c.metadata.Store(key, value)
}

// JoinRoom 加入房间
func (c *Client) JoinRoom(roomID string) error {
	return c.manager.rooms.JoinRoom(c, roomID)
}

// LeaveRoom 离开房间
func (c *Client) LeaveRoom(roomID string) {
	c.manager.rooms.LeaveRoom(c, roomID)
}

// GetRooms 获取所有房间
func (c *Client) GetRooms() []string {
	rooms := make([]string, 0, 8)
	c.rooms.Range(func(key, _ any) bool {
		if roomID, ok := key.(string); ok {
			rooms = append(rooms, roomID)
		}
		return true
	})
	return rooms
}

// RemoteAddr 获取远程地址
func (c *Client) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}
