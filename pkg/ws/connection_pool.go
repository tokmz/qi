package ws

import (
	"sync"
	"sync/atomic"
)

// ConnectionPool 连接池管理器
type ConnectionPool struct {
	clients  sync.Map     // clientID -> *Client
	count    atomic.Int64 // 连接数
	maxConns int          // 最大连接数
}

// NewConnectionPool 创建连接池
func NewConnectionPool(maxConns int) *ConnectionPool {
	return &ConnectionPool{
		maxConns: maxConns,
	}
}

// Add 添加客户端
func (p *ConnectionPool) Add(client *Client) error {
	// 先检查 ID 是否存在，避免计数不一致
	if _, loaded := p.clients.LoadOrStore(client.ID, client); loaded {
		return ErrClientIDExists
	}

	// 递增计数并检查限制
	newCount := p.count.Add(1)
	if int(newCount) > p.maxConns {
		// 超过限制，回滚操作
		p.count.Add(-1)
		p.clients.Delete(client.ID)
		return ErrTooManyConnections
	}

	return nil
}

// Remove 移除客户端
func (p *ConnectionPool) Remove(clientID string) {
	if _, loaded := p.clients.LoadAndDelete(clientID); loaded {
		p.count.Add(-1)
	}
}

// Get 获取客户端
func (p *ConnectionPool) Get(clientID string) (*Client, bool) {
	value, ok := p.clients.Load(clientID)
	if !ok {
		return nil, false
	}
	client, ok := value.(*Client)
	if !ok {
		return nil, false
	}
	return client, true
}

// Count 获取连接数
func (p *ConnectionPool) Count() int {
	return int(p.count.Load())
}

// Range 遍历所有客户端
func (p *ConnectionPool) Range(f func(*Client) bool) {
	p.clients.Range(func(key, value any) bool {
		client, ok := value.(*Client)
		if !ok {
			return true
		}
		return f(client)
	})
}

// GetAll 获取所有客户端（快照）
// 注意：在高并发场景下，此方法可能导致内存峰值，建议使用 Range 方法
func (p *ConnectionPool) GetAll() []*Client {
	// 使用当前计数作为容量提示
	capacity := p.Count()
	if capacity < 0 {
		capacity = 0
	}
	clients := make([]*Client, 0, capacity)
	p.Range(func(c *Client) bool {
		clients = append(clients, c)
		return true
	})
	return clients
}
