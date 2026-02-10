package ws

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Room 房间
type Room struct {
	ID        string
	clients   sync.Map     // clientID -> *Client
	count     atomic.Int32
	createdAt time.Time
	mu        sync.RWMutex
	metadata  map[string]any
}

// RoomManager 房间管理器
type RoomManager struct {
	rooms  sync.Map // roomID -> *Room
	config RoomConfig
}

// NewRoomManager 创建房间管理器
func NewRoomManager(config RoomConfig) *RoomManager {
	return &RoomManager{
		config: config,
	}
}

// CreateRoom 创建房间
func (rm *RoomManager) CreateRoom(roomID string, metadata map[string]any) (*Room, error) {
	room := &Room{
		ID:        roomID,
		createdAt: time.Now(),
		metadata:  metadata,
	}

	if _, loaded := rm.rooms.LoadOrStore(roomID, room); loaded {
		return nil, ErrRoomExists
	}

	return room, nil
}

// GetRoom 获取房间
func (rm *RoomManager) GetRoom(roomID string) (*Room, bool) {
	value, ok := rm.rooms.Load(roomID)
	if !ok {
		return nil, false
	}
	room, ok := value.(*Room)
	if !ok {
		return nil, false
	}
	return room, true
}

// DeleteRoom 删除房间
func (rm *RoomManager) DeleteRoom(roomID string) {
	if value, ok := rm.rooms.LoadAndDelete(roomID); ok {
		room, ok := value.(*Room)
		if !ok {
			return
		}
		// 踢出所有客户端
		room.clients.Range(func(key, value any) bool {
			client, ok := value.(*Client)
			if !ok {
				return true
			}
			client.LeaveRoom(roomID)
			return true
		})
	}
}

// JoinRoom 加入房间
func (rm *RoomManager) JoinRoom(client *Client, roomID string) error {
	// 获取或创建房间
	value, _ := rm.rooms.LoadOrStore(roomID, &Room{
		ID:        roomID,
		createdAt: time.Now(),
		metadata:  make(map[string]any),
	})
	room, ok := value.(*Room)
	if !ok {
		return ErrInvalidConfig // 类型断言失败
	}

	// 先递增计数并检查限制
	newCount := room.count.Add(1)
	if int(newCount) > rm.config.MaxRoomSize {
		// 超过限制，回滚计数
		room.count.Add(-1)
		return ErrRoomFull
	}

	// 再添加客户端，避免重复加入
	if _, loaded := room.clients.LoadOrStore(client.ID, client); loaded {
		// 客户端已在房间中，回滚计数
		room.count.Add(-1)
		return ErrAlreadyInRoom
	}

	// 客户端记录房间
	client.rooms.Store(roomID, true)

	return nil
}

// LeaveRoom 离开房间
func (rm *RoomManager) LeaveRoom(client *Client, roomID string) {
	value, ok := rm.rooms.Load(roomID)
	if !ok {
		return
	}

	room, ok := value.(*Room)
	if !ok {
		return
	}
	if _, loaded := room.clients.LoadAndDelete(client.ID); loaded {
		room.count.Add(-1)
		client.rooms.Delete(roomID)
	}
}

// RemoveClientFromAllRooms 从所有房间移除客户端
func (rm *RoomManager) RemoveClientFromAllRooms(clientID string) {
	rm.rooms.Range(func(key, value any) bool {
		room, ok := value.(*Room)
		if !ok {
			return true
		}
		if _, loaded := room.clients.LoadAndDelete(clientID); loaded {
			room.count.Add(-1)
		}
		return true
	})
}

// BroadcastToRoom 向房间广播
func (rm *RoomManager) BroadcastToRoom(roomID string, msg []byte, exclude *Client) error {
	room, ok := rm.GetRoom(roomID)
	if !ok {
		return ErrRoomNotFound
	}

	// 收集客户端（使用当前计数作为容量提示）
	capacity := int(room.count.Load())
	if capacity < 0 {
		capacity = 0
	}
	clients := make([]*Client, 0, capacity)
	room.clients.Range(func(key, value any) bool {
		client, ok := value.(*Client)
		if !ok {
			return true
		}
		if exclude == nil || client.ID != exclude.ID {
			clients = append(clients, client)
		}
		return true
	})

	// 如果没有客户端，直接返回
	if len(clients) == 0 {
		return nil
	}

	// 使用 worker pool 模式，避免为每个客户端创建 goroutine
	const maxWorkers = 100
	workerCount := maxWorkers
	if len(clients) < maxWorkers {
		workerCount = len(clients)
	}

	// 创建任务队列
	jobs := make(chan *Client, len(clients))
	for _, client := range clients {
		jobs <- client
	}
	close(jobs)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 启动固定数量的 worker
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case client, ok := <-jobs:
					if !ok {
						// 任务队列已关闭
						return
					}
					// 发送消息，忽略错误
					_ = client.SendBytes(msg)
				case <-ctx.Done():
					// 超时，退出
					return
				}
			}
		}()
	}

	// 等待所有 worker 完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ErrBroadcastTimeout
	}
}

// RunCleanup 运行清理任务
func (rm *RoomManager) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(rm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.cleanupEmptyRooms()
		}
	}
}

// cleanupEmptyRooms 清理空房间
func (rm *RoomManager) cleanupEmptyRooms() {
	now := time.Now()
	rm.rooms.Range(func(key, value any) bool {
		room, ok := value.(*Room)
		if !ok {
			return true
		}
		if room.count.Load() == 0 && now.Sub(room.createdAt) > rm.config.EmptyRoomTTL {
			rm.rooms.Delete(key)
		}
		return true
	})
}

// GetRoomCount 获取房间数量
func (rm *RoomManager) GetRoomCount() int {
	count := 0
	rm.rooms.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

// GetRoomMembers 获取房间成员
func (rm *RoomManager) GetRoomMembers(roomID string) []*Client {
	room, ok := rm.GetRoom(roomID)
	if !ok {
		return nil
	}

	members := make([]*Client, 0, room.count.Load())
	room.clients.Range(func(key, value any) bool {
		client, ok := value.(*Client)
		if !ok {
			return true
		}
		members = append(members, client)
		return true
	})
	return members
}
