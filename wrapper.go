package qi

import "github.com/gin-gonic/gin"

// HandlerFunc 路由处理函数和中间件函数
// 中间件需要调用 c.Next() 来继续执行后续处理
type HandlerFunc func(*Context)

// wrap 内部转换函数
func wrap(fn HandlerFunc) gin.HandlerFunc {
	if fn == nil {
		panic("qi: handler/middleware cannot be nil")
	}
	return func(c *gin.Context) {
		ctx := newContext(c)
		fn(ctx)
	}
}

// WrapHandler 将 qi.HandlerFunc 转换为 gin.HandlerFunc
func WrapHandler(handler HandlerFunc) gin.HandlerFunc {
	return wrap(handler)
}

// WrapMiddleware 将 qi.HandlerFunc 转换为 gin.HandlerFunc（中间件）
func WrapMiddleware(middleware HandlerFunc) gin.HandlerFunc {
	return wrap(middleware)
}

// WrapHandlers 批量转换多个处理函数
func WrapHandlers(handlers ...HandlerFunc) []gin.HandlerFunc {
	wrapped := make([]gin.HandlerFunc, len(handlers))
	for i, handler := range handlers {
		wrapped[i] = wrap(handler)
	}
	return wrapped
}

// WrapMiddlewares 批量转换多个中间件
func WrapMiddlewares(middlewares ...HandlerFunc) []gin.HandlerFunc {
	wrapped := make([]gin.HandlerFunc, len(middlewares))
	for i, middleware := range middlewares {
		wrapped[i] = wrap(middleware)
	}
	return wrapped
}
