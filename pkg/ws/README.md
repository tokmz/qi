# Qi WebSocket 包

`pkg/ws` 为 Qi 框架提供生产级、高性能的 WebSocket 框架。它具备企业级特性，包括连接池管理、房间管理、类型安全的消息路由以及全面的监控能力。

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Code Quality](https://img.shields.io/badge/Quality-Production%20Ready-brightgreen)](AUDIT.md)

## 🚀 核心特性

- **高性能**：优化支持 10k+ 并发连接，内存占用低，使用对象池和 Worker 池减少资源开销。
- **连接池管理**：线程安全的连接管理，支持可配置的连接数限制，原子操作保证并发安全。
- **房间管理**：内置房间/频道支持，使用工作池实现高效广播，支持房间人数限制和自动清理。
- **类型安全路由**：基于 Go 泛型的消息路由，提供类型安全的请求/响应处理，减少样板代码。
- **事件驱动**：异步事件总线，支持系统级可观测性（连接、断开、消息等事件），便于监控和日志记录。
- **弹性设计**：心跳管理、自动连接清理、panic 恢复和优雅关闭。
- **安全性**：Origin 白名单、消息大小限制、无效消息频率限制和连接数限制。
- **可观测性**：内置指标接口，可轻松集成 Prometheus 等监控系统，支持自定义指标收集。

## 📦 安装

此包是 Qi 项目的一部分。在你的 Go 代码中导入：

```go
import "qi/pkg/ws"
```

## ⚡ 快速开始

### 1. 初始化管理器

使用所需配置创建 WebSocket 管理器：

```go
// 使用选项创建 WebSocket 管理器
wsManager, err := ws.NewManager(
    ws.WithMaxConnections(10000),
    ws.WithHeartbeatInterval(30 * time.Second),
    ws.WithCheckOriginWhitelist([]string{
        "https://example.com",
        "http://localhost:8080",
    }),
)
if err != nil {
    log.Fatal(err)
}

// 在 goroutine 中启动管理器
go wsManager.Run()

// 确保优雅关闭
defer wsManager.Shutdown(context.Background())
```

### 2. 注册处理器

定义消息结构并使用 Go 泛型注册处理器：

```go
// 定义请求和响应类型
type ChatMessage struct {
    RoomID  string `json:"room_id"`
    Content string `json:"content"`
}

type ChatResponse struct {
    Success bool   `json:"success"`
    Time    int64  `json:"time"`
}

// 为 "chat.send" 事件注册处理器
ws.Handle[ChatMessage, ChatResponse](wsManager, "chat.send",
    func(c *ws.Client, req *ChatMessage) (*ChatResponse, error) {
        log.Printf("收到来自 %s 的消息: %s", c.ID, req.Content)

        // 向指定房间广播
        wsManager.BroadcastToRoom(req.RoomID, []byte(req.Content), c)

        return &ChatResponse{
            Success: true,
            Time:    time.Now().Unix(),
        }, nil
    })
```

### 3. 集成 HTTP 路由

在 Qi 控制器中将 HTTP 请求升级为 WebSocket 连接：

```go
r.GET("/ws", func(c *qi.Context) {
    // 认证用户（可选）
    userID := c.Query("user_id")

    // 升级连接
    err := wsManager.HandleUpgrade(c.Writer, c.Request,
        ws.WithUserID(convert.ToInt64(userID)), // 绑定用户 ID 到客户端
        ws.WithMetadata("ip", c.ClientIP()),
    )

    if err != nil {
        c.Fail(500, "升级失败")
    }
})
```

## 🏗 架构

框架围绕以下核心组件构建：

- **Manager（管理器）**：中央协调器，管理所有组件的生命周期。
- **ConnectionPool（连接池）**：使用 `sync.Map` 和原子计数器管理活跃连接，支持高并发。
- **RoomManager（房间管理器）**：处理房间的创建、加入、离开和广播。使用工作池高效广播消息，避免阻塞。
- **MessageRouter（消息路由器）**：根据 `event` 字段将传入的 JSON 消息路由到已注册的处理器。支持中间件。
- **EventBus（事件总线）**：异步发布系统事件（如 `client.connected`、`message.received`）给订阅的监听器。
- **Client（客户端）**：封装 WebSocket 连接，处理读写泵、心跳和消息队列。

## ⚙️ 配置

`NewManager` 函数接受函数式选项来自定义行为：

| 选项 | 说明 | 默认值 |
|--------|-------------|---------|
| `WithMaxConnections(int)` | 最大并发连接数 | 10,000 |
| `WithHeartbeatInterval(duration)` | Ping 间隔 | 30s |
| `WithHeartbeatTimeout(duration)` | 等待 Pong 的超时时间（超时后断开连接） | 90s |
| `WithMessageSizeLimit(int64)` | 最大消息大小（字节） | 512 KB |
| `WithMessageQueueSize(int)` | 每个客户端的发送队列大小 | 256 |
| `WithCheckOriginWhitelist([]string)` | CORS 允许的 Origin | 同源 |
| `WithMetrics(Metrics)` | 自定义指标实现 | 空操作 |

## 🛡 安全性与最佳实践

1. **Origin 白名单**：生产环境中务必配置 `WithCheckOriginWhitelist` 以防止跨站 WebSocket 劫持（CSWSH）。
2. **身份认证**：在 HTTP 升级阶段（调用 `HandleUpgrade` 之前）使用中间件或验证令牌。
3. **频率限制**：客户端发送过多无效消息时会自动断开连接。如需应用级频率限制，可使用中间件实现。
4. **资源限制**：根据服务器容量设置合理的 `MaxConnections` 和 `MaxMessageSize`。

## 📊 监控

实现 `ws.Metrics` 接口以集成你的监控系统（如 Prometheus）：

```go
type MyMetrics struct{}

func (m *MyMetrics) IncrementConnections() {
    // metrics.Connections.Inc()
}
// ... 实现其他方法

wsManager, _ := ws.NewManager(ws.WithMetrics(&MyMetrics{}))
```

## 🤝 事件

订阅系统事件以进行日志记录或自定义逻辑：

```go
wsManager.Subscribe(ws.EventClientConnected, func(e ws.Event) {
    log.Printf("客户端 %s 在 %s 连接", e.ClientID, e.Time)
})

wsManager.Subscribe(ws.EventMessageReceived, func(e ws.Event) {
    msg := e.Data.(*ws.Message)
    log.Printf("收到消息: %s", msg.Event)
})
```

## 📚 完整示例

### 聊天室应用

```go
package main

import (
    "context"
    "log"
    "time"
    "qi"
    "qi/pkg/ws"
)

type JoinRoomReq struct {
    RoomID string `json:"room_id"`
}

type ChatMessageReq struct {
    RoomID  string `json:"room_id"`
    Content string `json:"content"`
}

type ChatMessageResp struct {
    Success   bool   `json:"success"`
    Timestamp int64  `json:"timestamp"`
}

func main() {
    // 创建 WebSocket 管理器
    wsManager, err := ws.NewManager(
        ws.WithMaxConnections(10000),
        ws.WithHeartbeatInterval(30 * time.Second),
        ws.WithCheckOriginWhitelist([]string{
            "https://example.com",
            "http://localhost:8080",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 启动管理器
    go wsManager.Run()
    defer wsManager.Shutdown(context.Background())

    // 注册加入房间处理器
    ws.Handle0[JoinRoomReq](wsManager, "room.join",
        func(c *ws.Client, req *JoinRoomReq) error {
            return c.JoinRoom(req.RoomID)
        })

    // 注册聊天消息处理器
    ws.Handle[ChatMessageReq, ChatMessageResp](wsManager, "chat.send",
        func(c *ws.Client, req *ChatMessageReq) (*ChatMessageResp, error) {
            // 向房间广播消息
            wsManager.BroadcastToRoom(req.RoomID, []byte(req.Content), c)

            return &ChatMessageResp{
                Success:   true,
                Timestamp: time.Now().Unix(),
            }, nil
        })

    // 订阅连接事件
    wsManager.Subscribe(ws.EventClientConnected, func(e ws.Event) {
        log.Printf("客户端 %s 已连接", e.ClientID)
    })

    // 创建 Qi 引擎
    engine := qi.New()

    // WebSocket 路由
    engine.GET("/ws", func(c *qi.Context) {
        userID := c.Query("user_id")

        err := wsManager.HandleUpgrade(c.Writer, c.Request,
            ws.WithUserID(convert.ToInt64(userID)),
            ws.WithMetadata("ip", c.ClientIP()),
        )

        if err != nil {
            c.Fail(500, "WebSocket 升级失败")
        }
    })

    // 启动服务器
    engine.Run(":8080")
}
```

## 🔧 高级用法

### 中间件

为消息处理器添加中间件：

```go
// 认证中间件
func authMiddleware(c *ws.Client, msg *ws.Message, next ws.NextFunc) error {
    token, ok := c.GetMetadata("token")
    if !ok {
        return c.SendError(msg.RequestID, 401, "未授权")
    }

    // 验证 token
    if !validateToken(token.(string)) {
        return c.SendError(msg.RequestID, 401, "Token 无效")
    }

    return next()
}

// 日志中间件
func logMiddleware(c *ws.Client, msg *ws.Message, next ws.NextFunc) error {
    start := time.Now()
    err := next()
    log.Printf("处理消息 %s 耗时: %v", msg.Event, time.Since(start))
    return err
}

// 使用中间件
wsManager.Use(logMiddleware, authMiddleware)
```

### 自定义指标

集成 Prometheus 监控：

```go
import "github.com/prometheus/client_golang/prometheus"

type PrometheusMetrics struct {
    connections    prometheus.Gauge
    messages       *prometheus.CounterVec
    droppedMsgs    prometheus.Counter
}

func NewPrometheusMetrics() *PrometheusMetrics {
    m := &PrometheusMetrics{
        connections: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "websocket_connections",
            Help: "当前 WebSocket 连接数",
        }),
        messages: prometheus.NewCounterVec(prometheus.CounterOpts{
            Name: "websocket_messages_total",
            Help: "WebSocket 消息总数",
        }, []string{"event"}),
        droppedMsgs: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "websocket_dropped_messages_total",
            Help: "丢弃的消息总数",
        }),
    }

    prometheus.MustRegister(m.connections, m.messages, m.droppedMsgs)
    return m
}

func (m *PrometheusMetrics) IncrementConnections() {
    m.connections.Inc()
}

func (m *PrometheusMetrics) DecrementConnections() {
    m.connections.Dec()
}

func (m *PrometheusMetrics) IncrementMessageCount(event string) {
    m.messages.WithLabelValues(event).Inc()
}

func (m *PrometheusMetrics) IncrementDroppedMessages() {
    m.droppedMsgs.Inc()
}

// ... 实现其他方法

// 使用自定义指标
wsManager, _ := ws.NewManager(
    ws.WithMetrics(NewPrometheusMetrics()),
)
```

### 房间管理

```go
// 创建房间
room, err := wsManager.CreateRoom("room-123", map[string]any{
    "name": "技术讨论室",
    "type": "public",
})

// 获取房间成员
members := wsManager.GetRoomMembers("room-123")
log.Printf("房间成员数: %d", len(members))

// 删除房间（会踢出所有成员）
wsManager.DeleteRoom("room-123")

// 客户端加入/离开房间
client.JoinRoom("room-123")
client.LeaveRoom("room-123")

// 获取客户端所在的所有房间
rooms := client.GetRooms()
```

### 广播消息

```go
// 全局广播
wsManager.BroadcastAll([]byte(`{"type":"notify","event":"system.announcement","data":"系统维护通知"}`))

// 房间广播（排除发送者）
wsManager.BroadcastToRoom("room-123", []byte("消息内容"), senderClient)

// 用户广播（多设备）
wsManager.BroadcastToUser(12345, []byte("私信内容"))

// 使用消息对象池（高性能场景）
msg, err := ws.NewMessage("user.online", map[string]any{
    "user_id": 123,
    "status":  "online",
})
if err != nil {
    return err
}
defer msg.Release() // 释放到对象池

data, _ := json.Marshal(msg)
wsManager.BroadcastAll(data)
```

### 客户端操作

```go
// 发送 JSON 消息
client.SendJSON(map[string]any{
    "type":  "notify",
    "event": "user.online",
    "data":  userData,
})

// 发送字节消息
client.SendBytes([]byte("raw message"))

// 发送高优先级消息（系统消息）
client.SendBytesHigh([]byte("urgent message"))

// 发送响应
client.SendResponse("req-123", 200, "success", responseData)

// 发送错误
client.SendError("req-123", 400, "参数错误")

// 获取/设置元数据
client.SetMetadata("last_active", time.Now())
lastActive, _ := client.GetMetadata("last_active")

// 检查连接状态
if client.IsClosed() {
    log.Println("连接已关闭")
}

// 获取远程地址
remoteAddr := client.RemoteAddr()
```

## 🔒 安全配置

### Origin 白名单（推荐）

```go
wsManager, _ := ws.NewManager(
    ws.WithCheckOriginWhitelist([]string{
        "https://example.com",
        "https://app.example.com",
        "https://*.example.com", // 不支持通配符，需手动列出
    }),
)
```

### 自定义 Origin 检查

```go
wsManager, _ := ws.NewManager(
    ws.WithCheckOrigin(func(r *http.Request) bool {
        origin := r.Header.Get("Origin")

        // 自定义逻辑
        if strings.HasSuffix(origin, ".example.com") {
            return true
        }

        // 检查 IP 白名单
        ip := r.RemoteAddr
        return isWhitelistedIP(ip)
    }),
)
```

### 开发环境配置

```go
// 仅用于开发环境，生产环境禁用
wsManager, _ := ws.NewManager(
    ws.WithAllowAllOrigins(),
)
```

## ⚡ 性能优化

### 配置建议

```go
wsManager, _ := ws.NewManager(
    // 连接配置
    ws.WithMaxConnections(10000),           // 根据服务器容量调整
    ws.WithMessageQueueSize(512),           // 增大队列减少丢消息

    // 心跳配置
    ws.WithHeartbeatInterval(30 * time.Second),
    ws.WithHeartbeatTimeout(90 * time.Second),

    // 消息配置
    ws.WithMessageSizeLimit(1024 * 1024),   // 1MB

    // 启用压缩（适用于大消息）
    ws.WithEnableCompression(true),
)
```

### 对象池使用

```go
// 使用对象池（推荐）
msg, err := ws.NewMessage("event", data)
if err != nil {
    return err
}
defer msg.Release() // 必须释放

// 不使用对象池（简单场景）
msg, err := ws.NewMessageSimple("event", data)
// 无需 Release()，GC 自动回收
```

### 路由器预编译

```go
// 注册所有处理器后，冻结路由器以提升性能
wsManager.Router.Freeze()

// 冻结后无法再注册新处理器
// wsManager.Register("new.event", handler) // 会返回错误
```

## 🐛 故障排查

### 常见问题

**1. 连接立即断开**
- 检查 Origin 配置是否正确
- 确认客户端发送了正确的 Origin 头
- 查看服务器日志中的错误信息

**2. 消息发送失败**
- 检查发送队列是否已满（`ErrChannelFull`）
- 增大 `MessageQueueSize` 配置
- 检查客户端是否已断开连接

**3. 房间广播慢**
- 检查房间成员数是否过多
- 考虑分片广播或使用消息队列
- 增大 `BroadcastConfig.WorkerPoolSize`

**4. 内存占用高**
- 检查是否有连接泄漏（未正确关闭）
- 确认使用了对象池并正确调用 `Release()`
- 监控 `GetClientCount()` 和 `GetRoomCount()`

### 调试技巧

```go
// 订阅所有事件进行调试
wsManager.Subscribe(ws.EventClientConnected, func(e ws.Event) {
    log.Printf("[DEBUG] 客户端连接: %s", e.ClientID)
})

wsManager.Subscribe(ws.EventClientDisconnected, func(e ws.Event) {
    log.Printf("[DEBUG] 客户端断开: %s", e.ClientID)
})

wsManager.Subscribe(ws.EventMessageReceived, func(e ws.Event) {
    msg := e.Data.(*ws.Message)
    log.Printf("[DEBUG] 收到消息: event=%s, client=%s", msg.Event, e.ClientID)
})

wsManager.Subscribe(ws.EventError, func(e ws.Event) {
    log.Printf("[ERROR] 错误: %v", e.Data)
})

// 监控丢弃的事件
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        dropped := wsManager.EventBus.GetDroppedEventCount()
        if dropped > 0 {
            log.Printf("[WARN] 丢弃的事件数: %d", dropped)
        }
    }
}()
```

## 📖 消息协议

### 请求消息格式

```json
{
  "type": "request",
  "event": "chat.send",
  "request_id": "req_1234567890_1_a1b2c3d4",
  "data": {
    "room_id": "room-123",
    "content": "Hello, World!"
  },
  "timestamp": 1707552000
}
```

### 响应消息格式

```json
{
  "type": "response",
  "request_id": "req_1234567890_1_a1b2c3d4",
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "time": 1707552000
  },
  "trace_id": "trace-abc123",
  "timestamp": 1707552001
}
```

### 错误消息格式

```json
{
  "type": "error",
  "request_id": "req_1234567890_1_a1b2c3d4",
  "code": 400,
  "message": "参数错误",
  "trace_id": "trace-abc123",
  "timestamp": 1707552001
}
```

### 通知消息格式

```json
{
  "type": "notify",
  "event": "user.online",
  "data": {
    "user_id": 123,
    "status": "online"
  },
  "timestamp": 1707552000
}
```

## 🧪 测试

### 单元测试

```bash
go test ./pkg/ws/... -v
```

### 压力测试

```bash
# 使用 websocket-bench 进行压力测试
go get github.com/hashrocket/ws-bench
ws-bench -c 1000 -s 10 ws://localhost:8080/ws
```

### 基准测试

```bash
go test -bench=. -benchmem ./pkg/ws/...
```

## 📋 代码审计

本包已通过完整的代码审计，详见 [AUDIT.md](AUDIT.md)。

**审计结果**：
- ✅ 并发安全性：⭐⭐⭐⭐⭐
- ✅ 错误处理：⭐⭐⭐⭐
- ✅ 资源管理：⭐⭐⭐⭐
- ✅ 性能优化：⭐⭐⭐⭐⭐
- ⚠️ 测试覆盖率：待完善

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License
