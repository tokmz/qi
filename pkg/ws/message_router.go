package ws

import (
	"encoding/json"
	"sync"
)

// Handler 消息处理器
type Handler func(*Client, *Message) error

// NextFunc 中间件下一步函数
type NextFunc func() error

// MiddlewareFunc 中间件函数
type MiddlewareFunc func(*Client, *Message, NextFunc) error

// MessageRouter 消息路由器
type MessageRouter struct {
	handlers   map[string]Handler
	middleware []MiddlewareFunc
	compiled   map[string]Handler // 预编译的处理器链
	mu         sync.RWMutex
	frozen     bool
}

// NewMessageRouter 创建路由器
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make(map[string]Handler),
	}
}

// Register 注册处理器
func (r *MessageRouter) Register(event string, handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.frozen {
		return ErrRouterFrozen
	}

	if _, exists := r.handlers[event]; exists {
		return ErrHandlerExists
	}

	r.handlers[event] = handler
	return nil
}

// Use 添加中间件
func (r *MessageRouter) Use(middleware ...MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// Freeze 冻结路由器（启动后不可修改）
func (r *MessageRouter) Freeze() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frozen = true

	// 预编译所有处理器链
	r.compiled = make(map[string]Handler, len(r.handlers))
	for event, handler := range r.handlers {
		r.compiled[event] = r.buildChain(handler)
	}
}

// buildChain 构建中间件链
func (r *MessageRouter) buildChain(handler Handler) Handler {
	// 从后向前构建中间件链
	finalHandler := handler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		mw := r.middleware[i]
		next := finalHandler
		finalHandler = func(mw MiddlewareFunc, next Handler) Handler {
			return func(c *Client, m *Message) error {
				return mw(c, m, func() error {
					return next(c, m)
				})
			}
		}(mw, next)
	}
	return finalHandler
}

// Route 路由消息
func (r *MessageRouter) Route(client *Client, msg *Message) error {
	r.mu.RLock()
	// 优先使用预编译的处理器链
	if r.frozen && r.compiled != nil {
		handler, exists := r.compiled[msg.Event]
		r.mu.RUnlock()
		if !exists {
			return ErrHandlerNotFound
		}
		return handler(client, msg)
	}

	// 未冻结时，动态构建（兼容性）
	handler, exists := r.handlers[msg.Event]
	middlewareCopy := r.middleware
	r.mu.RUnlock()

	if !exists {
		return ErrHandlerNotFound
	}

	// 执行中间件链
	finalHandler := handler
	for i := len(middlewareCopy) - 1; i >= 0; i-- {
		mw := middlewareCopy[i]
		next := finalHandler
		finalHandler = func(mw MiddlewareFunc, next Handler) Handler {
			return func(c *Client, m *Message) error {
				return mw(c, m, func() error {
					return next(c, m)
				})
			}
		}(mw, next)
	}

	return finalHandler(client, msg)
}

// HandlerFunc 泛型处理器函数（有请求有响应）
type HandlerFunc[Req any, Resp any] func(*Client, *Req) (*Resp, error)

// HandlerFunc0 泛型处理器函数（有请求无响应）
type HandlerFunc0[Req any] func(*Client, *Req) error

// HandlerFuncOnly 泛型处理器函数（无请求有响应）
type HandlerFuncOnly[Resp any] func(*Client) (*Resp, error)

// Handle 注册泛型处理器（有请求有响应）
func Handle[Req any, Resp any](router *MessageRouter, event string, handler HandlerFunc[Req, Resp]) error {
	return router.Register(event, func(c *Client, msg *Message) error {
		// 解析请求
		var req Req
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return c.SendError(msg.RequestID, 400, "invalid request data")
		}

		// 执行处理器
		resp, err := handler(c, &req)
		if err != nil {
			return c.SendError(msg.RequestID, 500, err.Error())
		}

		// 发送响应
		return c.SendResponse(msg.RequestID, 200, "success", resp)
	})
}

// Handle0 注册泛型处理器（有请求无响应）
func Handle0[Req any](router *MessageRouter, event string, handler HandlerFunc0[Req]) error {
	return router.Register(event, func(c *Client, msg *Message) error {
		// 解析请求
		var req Req
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return c.SendError(msg.RequestID, 400, "invalid request data")
		}

		// 执行处理器
		if err := handler(c, &req); err != nil {
			return c.SendError(msg.RequestID, 500, err.Error())
		}

		// 发送响应
		return c.SendResponse(msg.RequestID, 200, "success", nil)
	})
}

// HandleOnly 注册泛型处理器（无请求有响应）
func HandleOnly[Resp any](router *MessageRouter, event string, handler HandlerFuncOnly[Resp]) error {
	return router.Register(event, func(c *Client, msg *Message) error {
		// 执行处理器
		resp, err := handler(c)
		if err != nil {
			return c.SendError(msg.RequestID, 500, err.Error())
		}

		// 发送响应
		return c.SendResponse(msg.RequestID, 200, "success", resp)
	})
}
