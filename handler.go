package qi

import "github.com/gin-gonic/gin"

// HandlerFunc qi 框架的处理函数
type HandlerFunc func(*Context)

// HandlersChain 处理函数链
type HandlersChain []HandlerFunc

// toGinHandler 将 qi.HandlerFunc 转换为 gin.HandlerFunc
func toGinHandler(h HandlerFunc) gin.HandlerFunc {
	return func(gc *gin.Context) {
		h(&Context{ctx: gc})
	}
}

// toGinHandlers 将 qi.HandlersChain 转换为 []gin.HandlerFunc
func toGinHandlers(handlers HandlersChain) []gin.HandlerFunc {
	ginHandlers := make([]gin.HandlerFunc, len(handlers))
	for i, h := range handlers {
		ginHandlers[i] = toGinHandler(h)
	}
	return ginHandlers
}
