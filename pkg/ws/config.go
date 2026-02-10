package ws

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Config WebSocket 配置
type Config struct {
	// 连接配置
	MaxConnections      int           // 最大连接数
	ReadBufferSize      int           // 读缓冲区大小
	WriteBufferSize     int           // 写缓冲区大小
	HandshakeTimeout    time.Duration // 握手超时时间
	MaxMessageSize      int64         // 最大消息大小

	// 心跳配置
	HeartbeatInterval   time.Duration // 心跳间隔
	HeartbeatTimeout    time.Duration // 心跳超时

	// 消息配置
	MessageQueueSize    int           // 消息队列大小
	HighPriorityQueueSize int         // 高优先级队列大小

	// 房间配置
	RoomConfig          RoomConfig

	// 广播配置
	BroadcastConfig     BroadcastConfig

	// Upgrader 配置
	UpgraderConfig      UpgraderConfig

	// 监控
	Metrics             Metrics
}

// RoomConfig 房间配置
type RoomConfig struct {
	MaxRoomSize      int           // 单个房间最大人数
	CleanupInterval  time.Duration // 清理间隔
	EmptyRoomTTL     time.Duration // 空房间存活时间
}

// BroadcastConfig 广播配置
type BroadcastConfig struct {
	WorkerPoolSize   int           // Worker 池大小
	QueueSize        int           // 广播队列大小
	BatchSize        int           // 批量发送大小
	BatchTimeout     time.Duration // 批量超时时间
}

// UpgraderConfig Upgrader 配置
type UpgraderConfig struct {
	ReadBufferSize    int                          // 读缓冲区大小
	WriteBufferSize   int                          // 写缓冲区大小
	CheckOrigin       func(*http.Request) bool     // Origin 检查函数
	EnableCompression bool                         // 是否启用压缩
	AllowedOrigins    []string                     // 允许的 Origin 白名单
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxConnections:        10000,
		ReadBufferSize:        1024,
		WriteBufferSize:       1024,
		HandshakeTimeout:      10 * time.Second,
		MaxMessageSize:        512 * 1024, // 512KB
		HeartbeatInterval:     30 * time.Second,
		HeartbeatTimeout:      90 * time.Second,
		MessageQueueSize:      256,
		HighPriorityQueueSize: 64,
		RoomConfig: RoomConfig{
			MaxRoomSize:     1000,
			CleanupInterval: 5 * time.Minute,
			EmptyRoomTTL:    10 * time.Minute,
		},
		BroadcastConfig: BroadcastConfig{
			WorkerPoolSize: 100,
			QueueSize:      1000,
			BatchSize:      10,
			BatchTimeout:   10 * time.Millisecond,
		},
		UpgraderConfig: UpgraderConfig{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			CheckOrigin:       nil, // 将在 NewUpgrader 中设置
			EnableCompression: false,
			AllowedOrigins:    nil, // 默认为 nil，使用同源检查
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.MaxConnections <= 0 {
		return fmt.Errorf("MaxConnections must be positive, got %d", c.MaxConnections)
	}
	if c.ReadBufferSize <= 0 {
		return fmt.Errorf("ReadBufferSize must be positive, got %d", c.ReadBufferSize)
	}
	if c.WriteBufferSize <= 0 {
		return fmt.Errorf("WriteBufferSize must be positive, got %d", c.WriteBufferSize)
	}
	if c.HandshakeTimeout <= 0 {
		return fmt.Errorf("HandshakeTimeout must be positive, got %v", c.HandshakeTimeout)
	}
	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("MaxMessageSize must be positive, got %d", c.MaxMessageSize)
	}
	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("HeartbeatInterval must be positive, got %v", c.HeartbeatInterval)
	}
	if c.HeartbeatTimeout <= c.HeartbeatInterval {
		return fmt.Errorf("HeartbeatTimeout (%v) must be greater than HeartbeatInterval (%v)",
			c.HeartbeatTimeout, c.HeartbeatInterval)
	}
	if c.MessageQueueSize <= 0 {
		return fmt.Errorf("MessageQueueSize must be positive, got %d", c.MessageQueueSize)
	}
	if c.HighPriorityQueueSize <= 0 {
		return fmt.Errorf("HighPriorityQueueSize must be positive, got %d", c.HighPriorityQueueSize)
	}

	// 验证房间配置
	if c.RoomConfig.MaxRoomSize <= 0 {
		return fmt.Errorf("RoomConfig.MaxRoomSize must be positive, got %d", c.RoomConfig.MaxRoomSize)
	}
	if c.RoomConfig.CleanupInterval <= 0 {
		return fmt.Errorf("RoomConfig.CleanupInterval must be positive, got %v", c.RoomConfig.CleanupInterval)
	}
	if c.RoomConfig.EmptyRoomTTL <= 0 {
		return fmt.Errorf("RoomConfig.EmptyRoomTTL must be positive, got %v", c.RoomConfig.EmptyRoomTTL)
	}

	// 验证广播配置
	if c.BroadcastConfig.WorkerPoolSize <= 0 {
		return fmt.Errorf("BroadcastConfig.WorkerPoolSize must be positive, got %d", c.BroadcastConfig.WorkerPoolSize)
	}
	if c.BroadcastConfig.QueueSize <= 0 {
		return fmt.Errorf("BroadcastConfig.QueueSize must be positive, got %d", c.BroadcastConfig.QueueSize)
	}
	if c.BroadcastConfig.BatchSize <= 0 {
		return fmt.Errorf("BroadcastConfig.BatchSize must be positive, got %d", c.BroadcastConfig.BatchSize)
	}
	if c.BroadcastConfig.BatchTimeout <= 0 {
		return fmt.Errorf("BroadcastConfig.BatchTimeout must be positive, got %v", c.BroadcastConfig.BatchTimeout)
	}

	// 验证 Upgrader 配置
	if c.UpgraderConfig.ReadBufferSize <= 0 {
		return fmt.Errorf("UpgraderConfig.ReadBufferSize must be positive, got %d", c.UpgraderConfig.ReadBufferSize)
	}
	if c.UpgraderConfig.WriteBufferSize <= 0 {
		return fmt.Errorf("UpgraderConfig.WriteBufferSize must be positive, got %d", c.UpgraderConfig.WriteBufferSize)
	}

	return nil
}

// Option 配置选项
type Option func(*Config)

// WithMaxConnections 设置最大连接数
func WithMaxConnections(max int) Option {
	return func(c *Config) {
		c.MaxConnections = max
	}
}

// WithHeartbeatInterval 设置心跳间隔
func WithHeartbeatInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.HeartbeatInterval = interval
	}
}

// WithHeartbeatTimeout 设置心跳超时
func WithHeartbeatTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.HeartbeatTimeout = timeout
	}
}

// WithMessageSizeLimit 设置消息大小限制
func WithMessageSizeLimit(size int64) Option {
	return func(c *Config) {
		c.MaxMessageSize = size
	}
}

// WithMessageQueueSize 设置消息队列大小
func WithMessageQueueSize(size int) Option {
	return func(c *Config) {
		c.MessageQueueSize = size
	}
}

// WithCheckOrigin 设置 Origin 检查函数
func WithCheckOrigin(fn func(*http.Request) bool) Option {
	return func(c *Config) {
		c.UpgraderConfig.CheckOrigin = fn
	}
}

// WithCheckOriginWhitelist 设置 Origin 白名单
// 示例：WithCheckOriginWhitelist([]string{"https://example.com", "https://app.example.com"})
func WithCheckOriginWhitelist(allowedOrigins []string) Option {
	return func(c *Config) {
		c.UpgraderConfig.AllowedOrigins = allowedOrigins
		// 自动设置 CheckOrigin 函数
		c.UpgraderConfig.CheckOrigin = createWhitelistChecker(allowedOrigins)
	}
}

// WithAllowAllOrigins 允许所有来源（仅用于开发环境，生产环境禁用）
func WithAllowAllOrigins() Option {
	return func(c *Config) {
		c.UpgraderConfig.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}
}

// WithMetrics 设置监控
func WithMetrics(metrics Metrics) Option {
	return func(c *Config) {
		c.Metrics = metrics
	}
}

// WithEnableCompression 启用压缩
func WithEnableCompression(enable bool) Option {
	return func(c *Config) {
		c.UpgraderConfig.EnableCompression = enable
	}
}

// defaultCheckOrigin 默认 Origin 检查（同源策略）
// 生产环境建议使用 WithCheckOriginWhitelist 设置白名单
func defaultCheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// 严格模式：拒绝空 Origin
		// 如需允许非浏览器客户端，使用 WithAllowAllOrigins()
		return false
	}
	// 同源检查
	return origin == "http://"+r.Host || origin == "https://"+r.Host
}

// createWhitelistChecker 创建白名单检查器
func createWhitelistChecker(allowedOrigins []string) func(*http.Request) bool {
	// 构建白名单 map 用于快速查找
	whitelist := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		whitelist[origin] = true
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// 白名单模式下拒绝空 Origin
			return false
		}
		// 检查是否在白名单中
		return whitelist[origin]
	}
}

// Upgrader WebSocket 升级器
type Upgrader struct {
	upgrader websocket.Upgrader
}

// NewUpgrader 创建升级器
func NewUpgrader(config UpgraderConfig) *Upgrader {
	// 如果没有设置 CheckOrigin，使用默认的同源检查
	checkOrigin := config.CheckOrigin
	if checkOrigin == nil {
		if len(config.AllowedOrigins) > 0 {
			// 如果设置了白名单，使用白名单检查
			checkOrigin = createWhitelistChecker(config.AllowedOrigins)
		} else {
			// 否则使用默认的同源检查
			checkOrigin = defaultCheckOrigin
		}
	}

	return &Upgrader{
		upgrader: websocket.Upgrader{
			ReadBufferSize:    config.ReadBufferSize,
			WriteBufferSize:   config.WriteBufferSize,
			CheckOrigin:       checkOrigin,
			EnableCompression: config.EnableCompression,
		},
	}
}

// Upgrade 升级 HTTP 连接为 WebSocket
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return u.upgrader.Upgrade(w, r, nil)
}
