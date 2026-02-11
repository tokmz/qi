package qi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterGroup 路由组
type RouterGroup struct {
	group *gin.RouterGroup
}

// ============ 路由组管理 ============

// Group 创建子路由组
func (rg *RouterGroup) Group(path string, middlewares ...HandlerFunc) *RouterGroup {
	handlers := WrapMiddlewares(middlewares...)
	return &RouterGroup{
		group: rg.group.Group(path, handlers...),
	}
}

// Use 注册中间件
func (rg *RouterGroup) Use(middlewares ...HandlerFunc) {
	handlers := WrapMiddlewares(middlewares...)
	rg.group.Use(handlers...)
}

// ============ 基础路由方法 ============

// GET 注册 GET 路由
func (rg *RouterGroup) GET(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.GET(path, handlers...)
	} else {
		rg.group.GET(path, WrapHandler(handler))
	}
}

// POST 注册 POST 路由
func (rg *RouterGroup) POST(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.POST(path, handlers...)
	} else {
		rg.group.POST(path, WrapHandler(handler))
	}
}

// PUT 注册 PUT 路由
func (rg *RouterGroup) PUT(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.PUT(path, handlers...)
	} else {
		rg.group.PUT(path, WrapHandler(handler))
	}
}

// DELETE 注册 DELETE 路由
func (rg *RouterGroup) DELETE(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.DELETE(path, handlers...)
	} else {
		rg.group.DELETE(path, WrapHandler(handler))
	}
}

// PATCH 注册 PATCH 路由
func (rg *RouterGroup) PATCH(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.PATCH(path, handlers...)
	} else {
		rg.group.PATCH(path, WrapHandler(handler))
	}
}

// HEAD 注册 HEAD 路由
func (rg *RouterGroup) HEAD(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.HEAD(path, handlers...)
	} else {
		rg.group.HEAD(path, WrapHandler(handler))
	}
}

// OPTIONS 注册 OPTIONS 路由
func (rg *RouterGroup) OPTIONS(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.OPTIONS(path, handlers...)
	} else {
		rg.group.OPTIONS(path, WrapHandler(handler))
	}
}

// Any 注册所有 HTTP 方法的路由
func (rg *RouterGroup) Any(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	if len(middlewares) > 0 {
		handlers := append(WrapMiddlewares(middlewares...), WrapHandler(handler))
		rg.group.Any(path, handlers...)
	} else {
		rg.group.Any(path, WrapHandler(handler))
	}
}

// ============ 静态文件服务 ============

// Static 注册静态文件服务（单个文件）
func (rg *RouterGroup) Static(relativePath, root string) {
	rg.group.Static(relativePath, root)
}

// StaticFile 注册静态文件服务（单个文件）
func (rg *RouterGroup) StaticFile(relativePath, filepath string) {
	rg.group.StaticFile(relativePath, filepath)
}

// StaticFS 注册静态文件系统服务
func (rg *RouterGroup) StaticFS(relativePath string, fs http.FileSystem) {
	rg.group.StaticFS(relativePath, fs)
}

// ============ 高级泛型路由（自动绑定 + 自动响应）============

// RouteRegister 路由注册函数类型
type RouteRegister func(path string, handler HandlerFunc, middlewares ...HandlerFunc)

// Handle 有请求参数，有响应数据
// 自动绑定请求参数，自动处理响应
func Handle[Req any, Resp any](register RouteRegister, path string, handler func(*Context, *Req) (*Resp, error), middlewares ...HandlerFunc) {
	wrappedHandler := func(c *Context) {
		var req Req
		if err := autoBind(c, &req); err != nil {
			c.RespondError(err)
			return
		}
		resp, err := handler(c, &req)
		if err != nil {
			c.RespondError(err)
			return
		}
		c.Success(resp)
	}
	register(path, wrappedHandler, middlewares...)
}

// Handle0 有请求参数，无响应数据
// 自动绑定请求参数，自动处理响应
func Handle0[Req any](register RouteRegister, path string, handler func(*Context, *Req) error, middlewares ...HandlerFunc) {
	wrappedHandler := func(c *Context) {
		var req Req
		if err := autoBind(c, &req); err != nil {
			c.RespondError(err)
			return
		}
		if err := handler(c, &req); err != nil {
			c.RespondError(err)
			return
		}
		c.Nil()
	}
	register(path, wrappedHandler, middlewares...)
}

// HandleOnly 无请求参数，有响应数据
// 自动处理响应
func HandleOnly[Resp any](register RouteRegister, path string, handler func(*Context) (*Resp, error), middlewares ...HandlerFunc) {
	wrappedHandler := func(c *Context) {
		resp, err := handler(c)
		if err != nil {
			c.RespondError(err)
			return
		}
		c.Success(resp)
	}
	register(path, wrappedHandler, middlewares...)
}

// autoBind 根据请求方法自动选择绑定策略
func autoBind(c *Context, obj any) error {
	method := c.Request().Method

	// GET/DELETE: Query + URI
	if method == "GET" || method == "DELETE" {
		if err := c.ShouldBindQuery(obj); err != nil {
			return c.wrapBindError(err)
		}
		// URI 绑定失败不阻断（路由可能没有 URI 参数）
		_ = c.ShouldBindUri(obj)
		return nil
	}

	// POST/PUT/PATCH: 根据 Content-Type 自动选择 + URI
	if method == "POST" || method == "PUT" || method == "PATCH" {
		// ShouldBind 会根据 Content-Type 自动选择：
		// - application/json -> JSON
		// - application/xml -> XML
		// - application/x-www-form-urlencoded -> Form
		// - multipart/form-data -> Multipart Form
		if err := c.ShouldBind(obj); err != nil {
			return c.wrapBindError(err)
		}
		// URI 绑定失败不阻断
		_ = c.ShouldBindUri(obj)
		return nil
	}

	// 其他方法使用默认绑定
	if err := c.ShouldBind(obj); err != nil {
		return c.wrapBindError(err)
	}
	return nil
}
