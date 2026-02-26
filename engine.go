package qi

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokmz/qi/pkg/i18n"
	"github.com/tokmz/qi/pkg/openapi"
	"github.com/wdcbot/qingfeng"

	"github.com/gin-gonic/gin"
)

type Engine struct {
	config     *Config
	engine     *gin.Engine
	server     *http.Server
	translator i18n.Translator
	registry   *openapi.Registry // nil = 未启用 OpenAPI
}

// New 创建一个新的 Engine 实例，使用 Options 模式配置
func New(opts ...Option) *Engine {
	// 应用默认配置
	config := defaultConfig()

	// 应用用户提供的选项
	for _, opt := range opts {
		opt(config)
	}

	// 设置 Gin 模式（全局状态）
	// 注意：gin.SetMode 是全局操作，多次调用会相互覆盖
	// 建议在程序启动时只创建一个 Engine 实例
	gin.SetMode(config.Mode)

	// 静默 Gin 默认输出，由 Qi 自行打印
	silenceGin()

	// 创建 Gin Engine
	ginEngine := gin.New()

	// 添加默认 Recovery 中间件（使用 qi 统一响应格式）
	ginEngine.Use(wrap(Recovery()))

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

	// 初始化 i18n
	if config.I18n != nil {
		t, err := i18n.New(config.I18n)
		if err != nil {
			log.Fatalf("qi: failed to init i18n: %v", err)
		}
		e.translator = t
		e.engine.Use(wrapWithTranslator(i18nMiddleware(t), t))
	}

	// 初始化 OpenAPI registry
	if config.OpenAPI != nil {
		e.registry = openapi.NewRegistry(config.OpenAPI)
	}

	return e
}

// Default 创建一个带有默认中间件的 Engine
// 注意：Recovery 中间件已在 New() 中添加，这里只添加 Logger
func Default(opts ...Option) *Engine {
	// 创建基础 Engine（已包含 Recovery 中间件）
	e := New(opts...)

	// 添加 Logger 中间件（使用默认配置）
	e.Use(defaultLogger())

	return e
}

// wrapHandlers 将 qi.HandlerFunc 转换为 gin.HandlerFunc，自动注入 translator
func (e *Engine) wrapHandlers(handlers ...HandlerFunc) []gin.HandlerFunc {
	wrapped := make([]gin.HandlerFunc, 0, len(handlers))
	for _, h := range handlers {
		if e.translator != nil {
			wrapped = append(wrapped, wrapWithTranslator(h, e.translator))
		} else {
			wrapped = append(wrapped, wrap(h))
		}
	}
	return wrapped
}

// Use 注册全局中间件
func (e *Engine) Use(middlewares ...HandlerFunc) {
	e.engine.Use(e.wrapHandlers(middlewares...)...)
}

// Group 返回路由组
func (e *Engine) Group(path string, middlewares ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group:      e.engine.Group(path, e.wrapHandlers(middlewares...)...),
		registry:   e.registry,
		translator: e.translator,
	}
}

// Router 返回根路由组
func (e *Engine) Router() *RouterGroup {
	return &RouterGroup{
		group:      &e.engine.RouterGroup,
		registry:   e.registry,
		translator: e.translator,
	}
}

// Translator 返回 i18n 翻译器实例
func (e *Engine) Translator() i18n.Translator {
	return e.translator
}

// Run 启动 HTTP 服务器，支持优雅关机
func (e *Engine) Run(addr ...string) error {
	// 确定监听地址
	address := e.config.Server.Addr
	if len(addr) > 0 && addr[0] != "" {
		address = addr[0]
	}

	// 构建 OpenAPI spec 并注册端点
	e.buildOpenAPISpec()

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
	// 构建 OpenAPI spec 并注册端点
	e.buildOpenAPISpec()

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

// buildOpenAPISpec 构建 OpenAPI spec 并注册端点
func (e *Engine) buildOpenAPISpec() {
	if e.registry == nil {
		return
	}

	spec := e.registry.Build()

	// 序列化 spec 为 JSON
	specJSON, err := json.Marshal(spec)
	if err != nil {
		log.Printf("qi: failed to marshal OpenAPI spec: %v", err)
		return
	}

	// 注册 spec 端点（直接用 gin engine，不经过 qi wrapper，避免自身被收集）
	e.engine.GET(e.config.OpenAPI.Path, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", specJSON)
	})

	// 注册 Swagger UI（使用 qingfeng）
	if e.config.OpenAPI.SwaggerUI != "" {
		// 临时静默 qingfeng 的 banner 输出
		origWriter := log.Writer()
		log.SetOutput(io.Discard)

		uiCfg := qingfeng.Config{
			Title:       e.config.OpenAPI.Title,
			Description: e.config.OpenAPI.Description,
			Version:     e.config.OpenAPI.Version,
			BasePath:    e.config.OpenAPI.SwaggerUI,
			DocJSON:     specJSON,
			EnableDebug: true,
		}
		e.engine.GET(e.config.OpenAPI.SwaggerUI+"/*filepath", qingfeng.Handler(uiCfg))

		// 恢复 log 输出
		log.SetOutput(origWriter)
	}
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
