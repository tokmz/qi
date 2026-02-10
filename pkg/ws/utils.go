package ws

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var (
	// clientIDCounter 客户端 ID 计数器
	clientIDCounter atomic.Uint64
	// requestIDCounter 请求 ID 计数器
	requestIDCounter atomic.Uint64
)

// generateID 生成唯一 ID
func generateID(prefix string, counter *atomic.Uint64) string {
	// 使用时间戳 + 计数器 + 随机数
	timestamp := time.Now().UnixNano()
	count := counter.Add(1)

	// 生成 4 字节随机数
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// 降级到纯计数器
		return fmt.Sprintf("%s_%d_%d", prefix, timestamp, count)
	}

	return fmt.Sprintf("%s_%d_%d_%s", prefix, timestamp, count, hex.EncodeToString(b))
}

// generateClientID 生成客户端 ID
func generateClientID() string {
	return generateID("client", &clientIDCounter)
}

// generateRoomID 生成房间 ID
func generateRoomID() string {
	return generateID("room", &clientIDCounter) // 房间 ID 可以共用计数器
}

// generateTraceID 生成追踪 ID
func generateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 降级到时间戳
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
