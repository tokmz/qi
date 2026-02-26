package qi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tokmz/qi/pkg/i18n"
	"github.com/tokmz/qi/pkg/openapi"
)

// RouterGroup 路由组
type RouterGroup struct {
	group           *gin.RouterGroup
	registry        *openapi.Registry // nil = 未启用 OpenAPI
	translator      i18n.Translator   // nil = 未启用 i18n
	defaultTag      string            // SetTag 设置的默认 tag
	defaultTagDesc  string            // SetTag 设置的默认 tag 描述
	defaultSecurity []string          // SetSecurity 设置的默认认证
}

// ============ 路由组管理 ============

// Group 创建子路由组
func (rg *RouterGroup) Group(path string, middlewares ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group:           rg.group.Group(path, rg.wrapMiddlewares(middlewares...)...),
		registry:        rg.registry,
		translator:      rg.translator,
		defaultTag:      rg.defaultTag,
		defaultTagDesc:  rg.defaultTagDesc,
		defaultSecurity: rg.defaultSecurity,
	}
}

// Use 注册中间件
func (rg *RouterGroup) Use(middlewares ...HandlerFunc) {
	rg.group.Use(rg.wrapMiddlewares(middlewares...)...)
}

// wrapMiddlewares 将中间件转换为 gin.HandlerFunc（不包含 handler）
func (rg *RouterGroup) wrapMiddlewares(middlewares ...HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(middlewares))
	for _, mw := range middlewares {
		if rg.translator != nil {
			handlers = append(handlers, wrapWithTranslator(mw, rg.translator))
		} else {
			handlers = append(handlers, wrap(mw))
		}
	}
	return handlers
}

// SetTag 设置路由组的默认 tag 名称和描述
func (rg *RouterGroup) SetTag(name, description string) {
	rg.defaultTag = name
	rg.defaultTagDesc = description
}

// SetSecurity 设置路由组的默认安全方案引用
func (rg *RouterGroup) SetSecurity(schemes ...string) {
	rg.defaultSecurity = schemes
}

// DocRoute 为非泛型路由手动注册文档
func (rg *RouterGroup) DocRoute(method, path string, doc *openapi.DocOption) {
	if rg.registry == nil || doc == nil {
		return
	}
	fullPath := strings.TrimRight(rg.group.BasePath(), "/") + path
	rg.registry.Add(openapi.RouteEntry{
		Method:          method,
		Path:            fullPath,
		BasePath:        rg.group.BasePath(),
		Type:            openapi.RouteTypeFull,
		ReqType:         doc.ReqType,
		RespType:        doc.RespType,
		Doc:             doc,
		DefaultTag:      rg.defaultTag,
		DefaultTagDesc:  rg.defaultTagDesc,
		DefaultSecurity: rg.defaultSecurity,
	})
}

// ============ 基础路由方法 ============

// wrapHandlers 将 qi.HandlerFunc 转换为 gin.HandlerFunc，自动注入 translator
func (rg *RouterGroup) wrapHandlers(handler HandlerFunc, middlewares ...HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(middlewares)+1)

	// 转换中间件
	for _, mw := range middlewares {
		if rg.translator != nil {
			handlers = append(handlers, wrapWithTranslator(mw, rg.translator))
		} else {
			handlers = append(handlers, wrap(mw))
		}
	}

	// 转换处理函数
	if rg.translator != nil {
		handlers = append(handlers, wrapWithTranslator(handler, rg.translator))
	} else {
		handlers = append(handlers, wrap(handler))
	}

	return handlers
}

// GET 注册 GET 路由
func (rg *RouterGroup) GET(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.GET(path, rg.wrapHandlers(handler, middlewares...)...)
}

// POST 注册 POST 路由
func (rg *RouterGroup) POST(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.POST(path, rg.wrapHandlers(handler, middlewares...)...)
}

// PUT 注册 PUT 路由
func (rg *RouterGroup) PUT(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.PUT(path, rg.wrapHandlers(handler, middlewares...)...)
}

// DELETE 注册 DELETE 路由
func (rg *RouterGroup) DELETE(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.DELETE(path, rg.wrapHandlers(handler, middlewares...)...)
}

// PATCH 注册 PATCH 路由
func (rg *RouterGroup) PATCH(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.PATCH(path, rg.wrapHandlers(handler, middlewares...)...)
}

// HEAD 注册 HEAD 路由
func (rg *RouterGroup) HEAD(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.HEAD(path, rg.wrapHandlers(handler, middlewares...)...)
}

// OPTIONS 注册 OPTIONS 路由
func (rg *RouterGroup) OPTIONS(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.OPTIONS(path, rg.wrapHandlers(handler, middlewares...)...)
}

// Any 注册所有 HTTP 方法的路由
func (rg *RouterGroup) Any(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	rg.group.Any(path, rg.wrapHandlers(handler, middlewares...)...)
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

	// GET/DELETE: URI + Query（先绑定 URI，再绑定 Query）
	if method == "GET" || method == "DELETE" {
		// URI 绑定失败不阻断（路由可能没有 URI 参数）
		_ = c.ShouldBindUri(obj)
		if err := c.ShouldBindQuery(obj); err != nil {
			return c.wrapBindError(err)
		}
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
