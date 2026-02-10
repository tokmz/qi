// Package ws provides a production-ready WebSocket framework for the Qi web framework.
//
// # Features
//
//   - Connection pooling with configurable limits
//   - Room-based broadcasting with worker pool optimization
//   - Type-safe message routing with Go generics
//   - Event-driven architecture with async event bus
//   - Built-in metrics and monitoring interfaces
//   - Graceful shutdown with timeout control
//   - Origin whitelist for security
//   - Invalid message rate limiting
//
// # Basic Usage
//
// Create a WebSocket manager and register message handlers:
//
//	// Create WebSocket manager
//	wsManager, err := ws.NewManager(
//	    ws.WithMaxConnections(10000),
//	    ws.WithHeartbeatInterval(30 * time.Second),
//	    ws.WithCheckOriginWhitelist([]string{
//	        "https://example.com",
//	    }),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start manager
//	go wsManager.Run()
//
//	// Register message handler with generics
//	ws.Handle[ChatMessage, ChatResponse](wsManager, "chat.send",
//	    func(c *ws.Client, req *ChatMessage) (*ChatResponse, error) {
//	        // Handle chat message
//	        return &ChatResponse{Success: true}, nil
//	    })
//
//	// Handle WebSocket upgrade in Qi route
//	r.GET("/ws", func(c *qi.Context) {
//	    err := wsManager.HandleUpgrade(c.Writer, c.Request,
//	        ws.WithUserID(getUserID(c)),
//	    )
//	    if err != nil {
//	        c.Fail(500, "upgrade failed")
//	    }
//	})
//
//	// Graceful shutdown
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	wsManager.Shutdown(ctx)
//
// # Room Management
//
// Create rooms and broadcast messages:
//
//	// Create a room
//	room, err := wsManager.CreateRoom("room-123", map[string]any{
//	    "name": "General Chat",
//	})
//
//	// Client joins room
//	client.JoinRoom("room-123")
//
//	// Broadcast to room (uses worker pool for efficiency)
//	wsManager.BroadcastToRoom("room-123", message, nil)
//
//	// Broadcast to specific user (all devices)
//	wsManager.BroadcastToUser(userID, message)
//
// # Message Types
//
// The framework supports different message types:
//
//	// Request message (expects response)
//	msg, _ := ws.NewMessageSimple("chat.send", data)
//
//	// Notify message (no response expected)
//	msg, _ := ws.NewNotifyMessageSimple("user.online", data)
//
//	// Response message
//	client.SendResponse(requestID, 200, "success", data)
//
//	// Error response
//	client.SendError(requestID, 400, "invalid request")
//
// # Security Configuration
//
// Production environment with Origin whitelist:
//
//	wsManager, _ := ws.NewManager(
//	    ws.WithCheckOriginWhitelist([]string{
//	        "https://example.com",
//	        "https://app.example.com",
//	    }),
//	)
//
// Development environment (allow all origins):
//
//	wsManager, _ := ws.NewManager(
//	    ws.WithAllowAllOrigins(), // Only for development!
//	)
//
// # Middleware
//
// Add middleware for authentication, logging, etc:
//
//	// Authentication middleware
//	wsManager.Use(func(c *ws.Client, msg *ws.Message, next ws.NextFunc) error {
//	    token := c.GetMetadata("token")
//	    if token == "" {
//	        return errors.New("unauthorized")
//	    }
//	    return next()
//	})
//
// # Monitoring
//
// Implement the Metrics interface for monitoring:
//
//	type MyMetrics struct {
//	    // Your metrics implementation
//	}
//
//	func (m *MyMetrics) IncrementConnections() {
//	    // Record connection metric
//	}
//
//	wsManager, _ := ws.NewManager(
//	    ws.WithMetrics(&MyMetrics{}),
//	)
//
// # Event Handling
//
// Subscribe to system events:
//
//	wsManager.Subscribe(ws.EventClientConnected, func(e ws.Event) {
//	    log.Printf("Client connected: %s", e.ClientID)
//	})
//
//	wsManager.Subscribe(ws.EventClientDisconnected, func(e ws.Event) {
//	    log.Printf("Client disconnected: %s", e.ClientID)
//	})
//
// # Performance Optimization
//
// The framework includes several performance optimizations:
//
//   - Message object pooling (use NewMessage/Release or NewMessageSimple)
//   - Worker pool for room broadcasting (max 100 concurrent workers)
//   - Pre-compiled middleware chains (call router.Freeze() after registration)
//   - Atomic operations for connection and room management
//   - Non-blocking message sending with queue overflow protection
//
// # Concurrency Safety
//
// All public APIs are concurrency-safe:
//
//   - ConnectionPool uses sync.Map and atomic counters
//   - RoomManager uses sync.Map for rooms and clients
//   - EventBus uses worker pool with buffered channels
//   - Client uses separate goroutines for read/write with proper synchronization
//
// # Error Handling
//
// The framework provides predefined errors:
//
//	var (
//	    ErrTooManyConnections = errors.New("ws: too many connections")
//	    ErrClientIDExists     = errors.New("ws: client id already exists")
//	    ErrRoomFull           = errors.New("ws: room is full")
//	    ErrHandlerNotFound    = errors.New("ws: handler not found")
//	    // ... more errors
//	)
//
// # Best Practices
//
//  1. Always set connection limits to prevent resource exhaustion
//  2. Use Origin whitelist in production environments
//  3. Implement proper authentication middleware
//  4. Monitor dropped events and messages using metrics
//  5. Use NewMessageSimple for simplicity or NewMessage+Release for performance
//  6. Call router.Freeze() after registering all handlers for better performance
//  7. Set appropriate timeouts for graceful shutdown
//  8. Handle client disconnections gracefully in your business logic
//
// # Architecture
//
// The framework consists of several core components:
//
//   - Manager: Central coordinator for all components
//   - ConnectionPool: Thread-safe connection management with limits
//   - RoomManager: Room-based broadcasting with worker pool
//   - MessageRouter: Type-safe message routing with middleware support
//   - EventBus: Async event system for observability
//   - Client: WebSocket connection wrapper with dual-queue messaging
//
// For more details, see DESIGN.md in the package directory.
package ws
