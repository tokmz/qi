package qi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	ilogging "github.com/tokmz/qi/internal/logging"
	"github.com/tokmz/qi/internal/openapi"
	itrace "github.com/tokmz/qi/internal/tracing"
	"github.com/wdcbot/qingfeng"
)

// Version 是 qi 框架的版本号。
// 构建时通过 ldflags 注入: go build -ldflags "-X github.com/tokmz/qi.Version=v0.2.0"
var Version = "v1.1.4"

// Engine 是 qi 的 HTTP 入口，负责路由注册和底层 gin.Engine 持有。
type Engine struct {
	engine          *gin.Engine      // 底层 gin.Engine
	server          *http.Server     // 底层 http.Server
	router          *routerStore     // 路由存储
	api             *openapi.Manager // OpenAPI 文档管理器（可选）
	cfg             *Config
	mode            string                      // 运行模式
	tracingShutdown func(context.Context) error // 链路追踪关闭函数
	routeMeta       map[string]RouteMeta        // 路由元信息注册表，key="METHOD /full/path"
}

// Config 定义 Engine 的常用运行配置。
type Config struct {
	Addr              string        // 监听地址
	Mode              string        // 运行模式
	ReadTimeout       time.Duration // 读取超时时间
	WriteTimeout      time.Duration // 写入超时时间
	IdleTimeout       time.Duration // 空闲超时时间
	ReadHeaderTimeout time.Duration // 读取请求头超时时间
	MaxHeaderBytes    int           // 最大请求头字节数

	ShutdownTimeout time.Duration // 关闭超时时间
	ShutdownSignals []os.Signal   // 关闭信号

	openAPIConfig *OpenAPIConfig // OpenAPI 配置（未导出）
	tracingConfig *TracingConfig // 链路追踪配置（未导出）
	loggerConfig  *LoggerConfig  // 日志中间件配置（未导出）
}

type Option func(*Config)

// WithAddr 设置监听地址。
func WithAddr(addr string) Option {
	return func(cfg *Config) { cfg.Addr = addr }
}

// WithMode 设置运行模式（debug/release/test）。
func WithMode(mode string) Option {
	return func(cfg *Config) { cfg.Mode = mode }
}

// defaultConfig 返回默认 Engine 配置。
func defaultConfig() *Config {
	return &Config{
		Addr:              ":8080",
		Mode:              gin.DebugMode,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,

		ShutdownTimeout: 5 * time.Second,
		ShutdownSignals: []os.Signal{os.Interrupt, syscall.SIGTERM},
	}
}

// New 创建一个新的 qi Engine。
func New(opts ...Option) *Engine {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	mode := normalizeMode(cfg.Mode)
	gin.SetMode(mode)

	// 静默 gin 的所有调试输出，由 qi 自行控制
	gin.DebugPrintFunc = func(format string, values ...any) {}
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {}

	engine := gin.New()

	engine.Use(gin.Recovery())

	e := &Engine{
		engine:    engine,
		router:    &routerStore{},
		routeMeta: make(map[string]RouteMeta),
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           engine,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
		cfg:  cfg,
		mode: mode,
	}

	if cfg.openAPIConfig != nil {
		cfg.openAPIConfig.normalize()
		opts := []openapi.Option{
			openapi.WithTitle(cfg.openAPIConfig.Title),
			openapi.WithDescription(cfg.openAPIConfig.Description),
			openapi.WithVersion(cfg.openAPIConfig.Version),
		}
		if len(cfg.openAPIConfig.Servers) > 0 {
			opts = append(opts, openapi.WithServers(cfg.openAPIConfig.Servers...))
		}
		e.api = openapi.New(opts...)
	}

	// 注册日志中间件
	if cfg.loggerConfig != nil {
		e.engine.Use(ilogging.Middleware(&ilogging.Config{
			Output:    cfg.loggerConfig.Output,
			SkipPaths: cfg.loggerConfig.SkipPaths,
		}))
	}

	// 初始化链路追踪并自动注册中间件
	if cfg.tracingConfig != nil {
		shutdown, err := itrace.Init(cfg.tracingConfig)
		if err != nil {
			panic("qi: tracing init failed: " + err.Error())
		}
		e.tracingShutdown = shutdown
		e.engine.Use(itrace.Middleware(cfg.tracingConfig))
	}

	return e
}

// Use 为整个应用添加中间件。
func (e *Engine) Use(handlers ...HandlerFunc) {
	e.engine.Use(toGinHandlers(handlers)...)
}

// ServeHTTP 实现 http.Handler，便于测试和作为上层路由的子处理器使用。
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.engine.ServeHTTP(w, r)
}

// Group 创建一个路由分组。
func (e *Engine) Group(prefix string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		engine:      e,
		prefix:      normalizeAbsolutePath(prefix),
		middlewares: cloneHandlers(handlers),
	}
}

// Routes 返回已注册路由的快照。
func (e *Engine) Routes() []Route {
	return e.router.snapshot()
}

// SetRouteMeta 写入路由元信息，key 格式为 "METHOD:/full/path"。
// 由 RouteBuilder.Done() 调用，用户无需直接调用。
func (e *Engine) SetRouteMeta(method, fullPath string, meta RouteMeta) {
	e.routeMeta[strings.ToUpper(method)+":"+fullPath] = meta
}

// RouteMeta 按 method + fullPath 查询路由元信息。
// 在中间件中使用：meta := e.RouteMeta(c.Request().Method, c.FullPath())
// 通过 RouteBuilder 注册的路由返回完整元信息；直接注册的路由 Summary 为 handlerName，其余字段为零值。
func (e *Engine) RouteMeta(method, fullPath string) *RouteMeta {
	if m, ok := e.routeMeta[strings.ToUpper(method)+":"+fullPath]; ok {
		return &m
	}
	return nil
}

// Handle 注册一条路由。
func (e *Engine) Handle(method, path string, handlers ...HandlerFunc) {
	normalized := normalizeAbsolutePath(path)
	e.handle(method, normalized, normalized, nil, handlers...)
}

// GET 注册 GET 路由。
func (e *Engine) GET(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodGet, path, handlers...)
}

// POST 注册 POST 路由。
func (e *Engine) POST(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodPost, path, handlers...)
}

// PUT 注册 PUT 路由。
func (e *Engine) PUT(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodPut, path, handlers...)
}

// PATCH 注册 PATCH 路由。
func (e *Engine) PATCH(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodPatch, path, handlers...)
}

// DELETE 注册 DELETE 路由。
func (e *Engine) DELETE(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodDelete, path, handlers...)
}

// HEAD 注册 HEAD 路由。
func (e *Engine) HEAD(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodHead, path, handlers...)
}

// OPTIONS 注册 OPTIONS 路由。
func (e *Engine) OPTIONS(path string, handlers ...HandlerFunc) {
	e.Handle(http.MethodOptions, path, handlers...)
}

// Any 为常见 HTTP 方法注册同一路由。
func (e *Engine) Any(path string, handlers ...HandlerFunc) {
	for _, method := range anyMethods() {
		e.Handle(method, path, handlers...)
	}
}

// Run 启动 HTTP 服务并阻塞，直到收到关闭信号后优雅退出。
func (e *Engine) Run() error {
	// 构建 OpenAPI spec 并注册端点（所有路由已注册完毕）
	e.buildOpenAPISpec()

	// 打印 banner + 路由表 + 运行信息
	e.printBanner()

	errCh := make(chan error, 1)
	go func() {
		if err := e.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, e.cfg.ShutdownSignals...)

	select {
	case err := <-errCh:
		return err
	case <-quit:
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.cfg.ShutdownTimeout)
	defer cancel()

	// flush span 数据后再关闭 HTTP server
	if e.tracingShutdown != nil {
		_ = e.tracingShutdown(ctx)
	}

	return e.server.Shutdown(ctx)
}

// buildOpenAPISpec 构建 OpenAPI spec 并注册相关端点。
// 在 Run() 中调用，此时所有路由已注册完毕，不需要 sync.Once。
func (e *Engine) buildOpenAPISpec() {
	if e.api == nil {
		return
	}
	cfg := e.cfg.openAPIConfig

	// 序列化 OpenAPI spec
	docJSON, err := e.api.MarshalJSON()
	if err != nil {
		return
	}

	// 注册 spec 端点（使用配置的路径）
	e.engine.GET(cfg.Path, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", docJSON)
	})
	e.router.add(Route{
		Method:      "GET",
		Path:        cfg.Path,
		FullPath:    cfg.Path,
		HandlerName: "qi.OpenAPIJSON",
	})

	// 仅当配置了 SwaggerUI 路径时才注册 UI
	if cfg.SwaggerUI != "" {
		// 静默 qingfeng 的 banner 输出
		origWriter := log.Writer()
		log.SetOutput(io.Discard)

		uiHandler := qingfeng.Handler(qingfeng.Config{
			BasePath:    cfg.SwaggerUI,
			Title:       cfg.Title,
			DocJSON:     docJSON,
			EnableDebug: e.mode == gin.DebugMode,
			DarkMode:    true,
		})

		log.SetOutput(origWriter)

		uiPath := cfg.SwaggerUI + "/*filepath"
		e.engine.GET(uiPath, uiHandler)
		e.router.add(Route{
			Method:      "GET",
			Path:        uiPath,
			FullPath:    uiPath,
			HandlerName: "qi.SwaggerUI",
		})
	}
}

func normalizeMode(mode string) string {
	switch mode {
	case gin.DebugMode, gin.ReleaseMode, gin.TestMode:
		return mode
	default:
		return gin.DebugMode
	}
}

// printBanner 打印 banner、路由表和运行信息到 os.Stdout。
func (e *Engine) printBanner() {
	w := os.Stdout

	// 颜色定义
	const (
		cyan    = "\033[36m"
		green   = "\033[32m"
		yellow  = "\033[33m"
		red     = "\033[31m"
		blue    = "\033[34m"
		magenta = "\033[35m"
		gray    = "\033[37m"
		dim     = "\033[2m"
		bold    = "\033[1m"
		reset   = "\033[0m"
	)

	// 构建 open URL
	addr := e.cfg.Addr
	if strings.HasPrefix(addr, ":") {
		addr = "http://127.0.0.1" + addr
	}
	openURL := addr
	if e.cfg.openAPIConfig != nil && e.cfg.openAPIConfig.SwaggerUI != "" {
		openURL = addr + e.cfg.openAPIConfig.SwaggerUI + "/"
	}

	// Banner 左侧 + 右侧介绍
	bannerLines := []struct{ art, info string }{
		{" ██████╗ ██╗", "Qi 基于Gin的Go Web 框架"},
		{"██╔═══██╗██║", "QQ: 81288369"},
		{"██║   ██║██║", "github: https://github.com/tokmz/qi"},
		{"██║▄▄ ██║██║", "open: " + openURL},
		{"╚██████╔╝██║", "version: " + Version},
		{" ╚══▀▀═╝ ╚═╝", ""},
	}

	fmt.Fprintln(w)
	for _, line := range bannerLines {
		if line.info != "" {
			fmt.Fprintf(w, "%s%s%s    %s%s%s\n", bold+cyan, line.art, reset, dim, line.info, reset)
		} else {
			fmt.Fprintf(w, "%s%s%s\n", bold+cyan, line.art, reset)
		}
	}
	fmt.Fprintln(w)

	// 路由表：从 routerStore 获取（保留注册顺序和真实 handler 名称）
	routes := e.router.snapshot()

	if len(routes) > 0 {
		// 计算路径列最大宽度
		maxPathLen := 0
		for _, r := range routes {
			if len(r.FullPath) > maxPathLen {
				maxPathLen = len(r.FullPath)
			}
		}
		if maxPathLen < 20 {
			maxPathLen = 20
		}

		pathFmt := fmt.Sprintf("%%-%ds", maxPathLen+2)

		for _, r := range routes {
			methodColor := methodToColor(r.Method)
			fmt.Fprintf(w, "%s[QI-%s]%s %s%-7s%s %s %s-->%s %s%s%s\n",
				cyan, e.mode, reset,
				methodColor, r.Method, reset,
				fmt.Sprintf(pathFmt, r.FullPath),
				dim, reset,
				green, r.HandlerName, reset,
			)
		}
		fmt.Fprintln(w)
	}

	// 运行信息
	fmt.Fprintf(w, "%s[Qi]%s Running in %s\"%s\"%s mode.\n", cyan, reset, yellow, e.mode, reset)
	fmt.Fprintf(w, "%s[Qi]%s Go version: %s | OS: %s/%s\n", cyan, reset, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "%s[Qi]%s Listening on %s%s%s\n", cyan, reset, green, e.cfg.Addr, reset)
}

// methodToColor 返回 HTTP 方法对应的 ANSI 颜色代码。
func methodToColor(method string) string {
	switch method {
	case http.MethodGet:
		return "\033[34m" // 蓝
	case http.MethodPost:
		return "\033[32m" // 绿
	case http.MethodPut:
		return "\033[33m" // 黄
	case http.MethodDelete:
		return "\033[31m" // 红
	case http.MethodPatch:
		return "\033[36m" // 青
	case http.MethodHead:
		return "\033[35m" // 紫
	case http.MethodOptions:
		return "\033[37m" // 灰
	default:
		return "\033[0m"
	}
}
