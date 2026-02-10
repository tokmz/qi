package ws

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Manager WebSocket 核心管理器
type Manager struct {
	// 核心组件
	pool   *ConnectionPool
	rooms  *RoomManager
	router *MessageRouter
	events *EventBus

	// 配置
	config   *Config
	upgrader *Upgrader

	// 生命周期
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 监控
	metrics Metrics
}

// NewManager 创建管理器
func NewManager(opts ...Option) (*Manager, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 默认监控
	if config.Metrics == nil {
		config.Metrics = &NoopMetrics{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建事件总线
	events := NewEventBus()

	// 创建各组件
	pool := NewConnectionPool(config.MaxConnections)
	rooms := NewRoomManager(config.RoomConfig)
	router := NewMessageRouter()

	m := &Manager{
		pool:     pool,
		rooms:    rooms,
		router:   router,
		events:   events,
		config:   config,
		upgrader: NewUpgrader(config.UpgraderConfig),
		ctx:      ctx,
		cancel:   cancel,
		metrics:  config.Metrics,
	}

	// 订阅事件
	m.setupEventHandlers()

	return m, nil
}

// Run 启动管理器
func (m *Manager) Run() error {
	m.wg.Add(1)

	// 启动房间清理
	go func() {
		defer m.wg.Done()
		m.rooms.RunCleanup(m.ctx)
	}()

	return nil
}

// Shutdown 优雅关闭
func (m *Manager) Shutdown(ctx context.Context) error {
	m.cancel()

	// 关闭事件总线
	if m.events != nil {
		m.events.Close()
	}

	// 并发关闭所有客户端
	var closeWg sync.WaitGroup
	m.pool.Range(func(c *Client) bool {
		closeWg.Add(1)
		go func(client *Client) {
			defer closeWg.Done()
			client.Close()
		}(c)
		return true
	})

	// 等待所有客户端关闭完成
	clientsDone := make(chan struct{})
	go func() {
		closeWg.Wait()
		close(clientsDone)
	}()

	// 等待客户端关闭或超时
	select {
	case <-clientsDone:
		// 客户端已全部关闭
	case <-ctx.Done():
		// 超时，但继续等待 goroutine 清理
	}

	// 等待所有 goroutine 结束
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HandleUpgrade 处理 WebSocket 升级
func (m *Manager) HandleUpgrade(w http.ResponseWriter, r *http.Request, opts ...ClientOption) error {
	// 升级连接
	conn, err := m.upgrader.Upgrade(w, r)
	if err != nil {
		return err
	}

	// 创建客户端
	client := NewClient(conn, m, opts...)

	// 添加到连接池（内部会原子检查连接数限制）
	if err := m.pool.Add(client); err != nil {
		// 使用 Close() 确保完整清理资源（包括 channel）
		client.Close()
		// 如果是连接数超限，返回 503 错误给客户端
		if err == ErrTooManyConnections {
			http.Error(w, "too many connections", http.StatusServiceUnavailable)
		}
		return err
	}

	// 发布连接事件
	if m.events != nil {
		m.events.Publish(Event{
			Type:     EventClientConnected,
			ClientID: client.ID,
			Time:     time.Now(),
		})
	}

	// 启动客户端
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		client.Run()
	}()

	return nil
}

// Register 注册消息处理器
func (m *Manager) Register(event string, handler Handler) error {
	return m.router.Register(event, handler)
}

// Subscribe 订阅系统事件
func (m *Manager) Subscribe(eventType EventType, handler EventHandler) {
	m.events.Subscribe(eventType, handler)
}

// Use 添加中间件
func (m *Manager) Use(middleware ...MiddlewareFunc) {
	m.router.Use(middleware...)
}

// GetClient 获取客户端
func (m *Manager) GetClient(clientID string) (*Client, bool) {
	return m.pool.Get(clientID)
}

// GetClientCount 获取连接数
func (m *Manager) GetClientCount() int {
	return m.pool.Count()
}

// BroadcastAll 全局广播
func (m *Manager) BroadcastAll(msg []byte) {
	m.pool.Range(func(c *Client) bool {
		if err := c.SendBytes(msg); err != nil {
			// 发送失败，更新监控指标
			if m.metrics != nil {
				m.metrics.IncrementDroppedMessages()
			}
		}
		return true
	})
}

// BroadcastToRoom 房间广播
func (m *Manager) BroadcastToRoom(roomID string, msg []byte, exclude *Client) error {
	return m.rooms.BroadcastToRoom(roomID, msg, exclude)
}

// BroadcastToUser 用户广播（多设备）
func (m *Manager) BroadcastToUser(userID int64, msg []byte) {
	m.pool.Range(func(c *Client) bool {
		if c.UserID == userID {
			if err := c.SendBytes(msg); err != nil {
				// 发送失败，更新监控指标
				if m.metrics != nil {
					m.metrics.IncrementDroppedMessages()
				}
			}
		}
		return true
	})
}

// CreateRoom 创建房间
func (m *Manager) CreateRoom(roomID string, metadata map[string]any) (*Room, error) {
	return m.rooms.CreateRoom(roomID, metadata)
}

// GetRoom 获取房间
func (m *Manager) GetRoom(roomID string) (*Room, bool) {
	return m.rooms.GetRoom(roomID)
}

// DeleteRoom 删除房间
func (m *Manager) DeleteRoom(roomID string) {
	m.rooms.DeleteRoom(roomID)
}

// GetRoomCount 获取房间数量
func (m *Manager) GetRoomCount() int {
	return m.rooms.GetRoomCount()
}

// setupEventHandlers 设置事件处理器
func (m *Manager) setupEventHandlers() {
	// 连接事件
	m.events.Subscribe(EventClientConnected, func(e Event) {
		if m.metrics != nil {
			m.metrics.IncrementConnections()
			m.metrics.SetConnectionCount(m.pool.Count())
		}
	})

	m.events.Subscribe(EventClientDisconnected, func(e Event) {
		if m.metrics != nil {
			m.metrics.DecrementConnections()
			m.metrics.SetConnectionCount(m.pool.Count())
		}
		// 从所有房间移除
		m.rooms.RemoveClientFromAllRooms(e.ClientID)
	})

	// 消息事件
	m.events.Subscribe(EventMessageReceived, func(e Event) {
		if m.metrics != nil {
			msg := e.Data.(*Message)
			m.metrics.IncrementMessageCount(msg.Event)
		}
	})
}
