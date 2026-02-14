package qi

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

type Engine struct {
	config *Config
	engine *gin.Engine
	server *http.Server
}

// New 创建一个新的 Engine 实例，使用 Options 模式配置
func New(opts ...Option) *Engine {
	// 应用默认配置
	config := defaultConfig()

	// 应用用户提供的选项
	for _, opt := range opts {
		opt(config)
	}

	// 设置 Gin 模式（全局状态，仅在首次调用时设置）
	// 注意：gin.SetMode 是全局操作，多次调用会相互覆盖
	// 建议在程序启动时只创建一个 Engine 实例
	if gin.Mode() == gin.DebugMode || config.Mode != gin.DebugMode {
		gin.SetMode(config.Mode)
	}

	// 静默 Gin 默认输出，由 Qi 自行打印
	silenceGin()

	// 创建 Gin Engine
	ginEngine := gin.New()

	// 添加默认 Recovery 中间件（防止 panic 导致服务崩溃）
	ginEngine.Use(gin.Recovery())

	// 设置信任的代理
	if config.TrustedProxies != nil {
		if err := ginEngine.SetTrustedProxies(config.TrustedProxies); err != nil {
			log.Printf("设置信任代理失败: %v", err)
		}
	}

	// 设置 MaxMultipartMemory
	ginEngine.MaxMultipartMemory = config.MaxMultipartMemory

	// 创建 Engine
	e := &Engine{
		engine: ginEngine,
		config: config,
	}

	return e
}

// Default 创建一个带有默认中间件的 Engine
// 注意：Recovery 中间件已在 New() 中添加，这里只添加 Logger
func Default(opts ...Option) *Engine {
	// 创建基础 Engine（已包含 Recovery 中间件）
	e := New(opts...)

	// 添加 Logger 中间件
	e.engine.Use(gin.Logger())

	return e
}

// Use 注册全局中间件
func (e *Engine) Use(middlewares ...HandlerFunc) {
	handlers := WrapMiddlewares(middlewares...)
	e.engine.Use(handlers...)
}

// Group 返回路由组
func (e *Engine) Group(path string) *RouterGroup {
	return &RouterGroup{
		group: e.engine.Group(path),
	}
}

// RouterGroup 返回根路由组
func (e *Engine) RouterGroup() *RouterGroup {
	return &RouterGroup{
		group: &e.engine.RouterGroup,
	}
}

// Run 启动 HTTP 服务器，支持优雅关机
func (e *Engine) Run(addr ...string) error {
	// 确定监听地址
	address := e.config.Server.Addr
	if len(addr) > 0 && addr[0] != "" {
		address = addr[0]
	}

	// 创建 HTTP Server
	e.server = &http.Server{
		Addr:           address,
		Handler:        e.engine,
		ReadTimeout:    e.config.Server.ReadTimeout,
		WriteTimeout:   e.config.Server.WriteTimeout,
		IdleTimeout:    e.config.Server.IdleTimeout,
		MaxHeaderBytes: e.config.Server.MaxHeaderBytes,
	}

	// 打印 banner 和路由表
	e.printBanner(address)

	// 启动服务器
	return e.serve(func() error {
		return e.server.ListenAndServe()
	})
}

// RunTLS 启动 HTTPS 服务器，支持优雅关机
func (e *Engine) RunTLS(addr, certFile, keyFile string) error {
	// 创建 HTTP Server
	e.server = &http.Server{
		Addr:           addr,
		Handler:        e.engine,
		ReadTimeout:    e.config.Server.ReadTimeout,
		WriteTimeout:   e.config.Server.WriteTimeout,
		IdleTimeout:    e.config.Server.IdleTimeout,
		MaxHeaderBytes: e.config.Server.MaxHeaderBytes,
	}

	// 打印 banner 和路由表
	e.printBanner(addr)

	// 启动服务器
	return e.serve(func() error {
		return e.server.ListenAndServeTLS(certFile, keyFile)
	})
}

// serve 统一的服务器启动和优雅关机逻辑
func (e *Engine) serve(startFunc func() error) error {
	// 用于传递启动错误的 channel
	errChan := make(chan error, 1)

	// 启动服务器（在 goroutine 中）
	go func() {
		if err := startFunc(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// 等待中断信号或启动错误
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-errChan:
		// 服务器启动失败
		return err
	case <-quit:
		// 收到关机信号
		log.Println("正在关闭服务器...")
	}

	// 执行优雅关机
	return e.gracefulShutdown()
}

// gracefulShutdown 执行优雅关机流程
func (e *Engine) gracefulShutdown() error {
	// 执行关机前回调
	if e.config.Shutdown.BeforeShutdown != nil {
		e.config.Shutdown.BeforeShutdown()
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), e.config.Shutdown.Timeout)
	defer cancel()

	// 优雅关闭服务器
	if err := e.server.Shutdown(ctx); err != nil {
		log.Printf("服务器强制关闭: %v", err)
		return err
	}

	// 执行关机后回调
	if e.config.Shutdown.AfterShutdown != nil {
		e.config.Shutdown.AfterShutdown()
	}

	log.Println("服务器已退出")
	return nil
}

// Shutdown 手动关闭服务器
func (e *Engine) Shutdown(ctx context.Context) error {
	if e.server == nil {
		return nil
	}

	// 执行关机前回调
	if e.config.Shutdown.BeforeShutdown != nil {
		e.config.Shutdown.BeforeShutdown()
	}

	// 关闭服务器
	err := e.server.Shutdown(ctx)

	// 执行关机后回调
	if e.config.Shutdown.AfterShutdown != nil {
		e.config.Shutdown.AfterShutdown()
	}

	return err
}
