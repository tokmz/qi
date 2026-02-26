package qi

import (
	"github.com/gin-gonic/gin"
	"github.com/tokmz/qi/pkg/i18n"
)

// HandlerFunc 路由处理函数和中间件函数
// 中间件需要调用 c.Next() 来继续执行后续处理
type HandlerFunc func(*Context)

// wrapWithTranslator 内部转换函数
// translator 参数用于注入 i18n 翻译器（可能为 nil）
func wrapWithTranslator(fn HandlerFunc, translator i18n.Translator) gin.HandlerFunc {
	if fn == nil {
		panic("qi: handler/middleware cannot be nil")
	}
	return func(c *gin.Context) {
		ctx := &Context{ctx: c}
		// 如果提供了 translator，注入到 context
		if translator != nil {
			SetContextTranslator(ctx, translator)
		}
		fn(ctx)
	}
}

// wrap 内部转换函数（兼容旧代码，translator 为 nil）
func wrap(fn HandlerFunc) gin.HandlerFunc {
	return wrapWithTranslator(fn, nil)
}

// WrapHandler 将 qi.HandlerFunc 转换为 gin.HandlerFunc
// 已废弃：建议使用 Engine/RouterGroup 的路由方法，会自动注入 translator
func WrapHandler(handler HandlerFunc) gin.HandlerFunc {
	return wrap(handler)
}

// WrapMiddleware 将 qi.HandlerFunc 转换为 gin.HandlerFunc（中间件）
// 已废弃：建议使用 Engine/RouterGroup 的 Use 方法，会自动注入 translator
func WrapMiddleware(middleware HandlerFunc) gin.HandlerFunc {
	return wrap(middleware)
}

// WrapHandlers 批量转换多个处理函数
// 已废弃：建议使用 Engine/RouterGroup 的路由方法，会自动注入 translator
func WrapHandlers(handlers ...HandlerFunc) []gin.HandlerFunc {
	wrapped := make([]gin.HandlerFunc, len(handlers))
	for i, handler := range handlers {
		wrapped[i] = wrap(handler)
	}
	return wrapped
}

// WrapMiddlewares 批量转换多个中间件
// 已废弃：建议使用 Engine/RouterGroup 的 Use 方法，会自动注入 translator
func WrapMiddlewares(middlewares ...HandlerFunc) []gin.HandlerFunc {
	wrapped := make([]gin.HandlerFunc, len(middlewares))
	for i, middleware := range middlewares {
		wrapped[i] = wrap(middleware)
	}
	return wrapped
}
