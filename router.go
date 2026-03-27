package qi

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

// RouteMeta 路由的 OpenAPI 元信息，供中间件运行时查询。
type RouteMeta struct {
	Summary     string
	Description string
	Tags        []string
	OperationID string
	Deprecated  bool
}

// Route 描述一条已注册路由，供调试、文档生成和扩展能力使用。
type Route struct {
	Method      string        // HTTP 方法
	Path        string        // 注册时的相对路径
	FullPath    string        // 拼接分组前缀后的绝对路径
	HandlerName string        // 清理后的 handler 名称，用于路由表打印
	Handlers    HandlersChain // 处理器链
}

type routerStore struct {
	routes []*Route
}

// RouterGroup 表示一个路由分组。
type RouterGroup struct {
	engine      *Engine
	prefix      string
	middlewares HandlersChain
}

// Use 为当前分组追加中间件。
func (r *RouterGroup) Use(handlers ...HandlerFunc) {
	r.middlewares = append(r.middlewares, handlers...)
}

// Group 创建子分组，并继承当前分组中间件。
func (r *RouterGroup) Group(prefix string, handlers ...HandlerFunc) *RouterGroup {
	inherited := append(cloneHandlers(r.middlewares), handlers...)
	return &RouterGroup{
		engine:      r.engine,
		prefix:      joinPaths(r.prefix, prefix),
		middlewares: inherited,
	}
}

// Handle 在当前分组下注册一条路由。
func (r *RouterGroup) Handle(method, path string, handlers ...HandlerFunc) {
	r.engine.handle(method, normalizeAbsolutePath(path), joinPaths(r.prefix, path), r.middlewares, handlers...)
}

// GET 注册 GET 路由。
func (r *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodGet, path, handlers...)
}

// POST 注册 POST 路由。
func (r *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodPost, path, handlers...)
}

// PUT 注册 PUT 路由。
func (r *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodPut, path, handlers...)
}

// PATCH 注册 PATCH 路由。
func (r *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodPatch, path, handlers...)
}

// DELETE 注册 DELETE 路由。
func (r *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodDelete, path, handlers...)
}

// HEAD 注册 HEAD 路由。
func (r *RouterGroup) HEAD(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodHead, path, handlers...)
}

// OPTIONS 注册 OPTIONS 路由。
func (r *RouterGroup) OPTIONS(path string, handlers ...HandlerFunc) {
	r.Handle(http.MethodOptions, path, handlers...)
}

// Any 为常见 HTTP 方法注册同一路由。
func (r *RouterGroup) Any(path string, handlers ...HandlerFunc) {
	for _, method := range anyMethods() {
		r.Handle(method, path, handlers...)
	}
}

func (e *Engine) handle(method, relativePath, fullPath string, middlewares HandlersChain, handlers ...HandlerFunc) {
	fullPath = normalizeAbsolutePath(fullPath)
	chain := make(HandlersChain, 0, len(middlewares)+len(handlers))
	chain = append(chain, middlewares...)
	chain = append(chain, handlers...)

	// 通过反射获取最后一个 handler（用户 handler）的真实函数名
	lastHandler := handlers[len(handlers)-1]
	handlerName := cleanHandlerName(runtime.FuncForPC(reflect.ValueOf(lastHandler).Pointer()).Name())

	e.engine.Handle(method, fullPath, toGinHandlers(chain)...)
	e.router.add(Route{
		Method:      strings.ToUpper(strings.TrimSpace(method)),
		Path:        relativePath,
		FullPath:    fullPath,
		HandlerName: handlerName,
		Handlers:    cloneHandlers(chain),
	})

	// 写入默认元信息（handlerName 作为 Summary），RouteBuilder.Done() 会在之后覆盖
	e.routeMeta[strings.ToUpper(method)+":"+fullPath] = RouteMeta{Summary: handlerName}
}

// cleanHandlerName 清理反射获取的函数全名。
// 截取最后一段包路径，移除泛型类型参数噪音，并去除重复的包名前缀。
func cleanHandlerName(name string) string {
	// 截取最后一段包路径: "github.com/foo/bar.MyHandler" → "bar.MyHandler"
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	// 清理泛型类型参数: "qi.Bind[...].func1" → "qi.Bind.func1"
	for {
		open := strings.Index(name, "[")
		if open < 0 {
			break
		}
		close := strings.Index(name[open:], "]")
		if close < 0 {
			break
		}
		name = name[:open] + name[open+close+1:]
	}
	// 去除重复的包名前缀: "main.main.func2" → "main.func2"
	if dot := strings.Index(name, "."); dot >= 0 {
		pkg := name[:dot]
		rest := name[dot+1:]
		if strings.HasPrefix(rest, pkg+".") {
			name = pkg + "." + rest[len(pkg)+1:]
		}
	}
	return name
}

func (s *routerStore) add(route Route) {
	cp := route
	s.routes = append(s.routes, &cp)
}

func (s *routerStore) snapshot() []Route {
	out := make([]Route, 0, len(s.routes))
	for _, route := range s.routes {
		if route == nil {
			continue
		}
		cp := *route
		cp.Handlers = cloneHandlers(route.Handlers)
		out = append(out, cp)
	}
	return out
}

func cloneHandlers(handlers HandlersChain) HandlersChain {
	if len(handlers) == 0 {
		return nil
	}
	out := make(HandlersChain, len(handlers))
	copy(out, handlers)
	return out
}

func joinPaths(prefix, path string) string {
	if prefix == "" || prefix == "/" {
		return normalizeAbsolutePath(path)
	}

	if path == "" || path == "/" {
		return normalizeAbsolutePath(prefix)
	}

	return normalizeAbsolutePath(strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/"))
}

func normalizeAbsolutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.ReplaceAll(path, "//", "/")
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	return path
}

func anyMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}
}
