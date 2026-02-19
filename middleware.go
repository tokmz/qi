package qi

import (
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/tokmz/qi/pkg/logger"
	"go.uber.org/zap"
)

// defaultLogger 创建默认日志中间件（供 Default() 使用）
func defaultLogger() HandlerFunc {
	log, err := logger.NewDevelopment()
	if err != nil {
		panic("qi: failed to create default logger: " + err.Error())
	}

	return func(c *Context) {
		start := time.Now()
		path := c.Request().URL.Path
		method := c.Request().Method
		clientIP := c.ClientIP()

		c.Next()

		latency := time.Since(start)
		status := c.Writer().Status()

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		}

		if status >= 500 {
			log.Error("request", fields...)
		} else if status >= 400 {
			log.Warn("request", fields...)
		} else {
			log.Info("request", fields...)
		}
	}
}

// Recovery 创建 panic 恢复中间件
// panic 时返回 qi 统一响应格式（500），并记录错误日志
func Recovery(logs ...logger.Logger) HandlerFunc {
	var log logger.Logger
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	} else {
		var err error
		log, err = logger.NewDevelopment()
		if err != nil {
			panic("qi: failed to create recovery logger: " + err.Error())
		}
	}

	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否为断开的连接（客户端主动断开）
				if isBrokenPipe(err) {
					log.Error("broken pipe",
						zap.Any("error", err),
						zap.String("path", c.Request().URL.Path),
					)
					c.Abort()
					return
				}

				// 获取堆栈信息
				stack := string(debug.Stack())

				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request().Method),
					zap.String("path", c.Request().URL.Path),
					zap.String("client_ip", c.ClientIP()),
					zap.String("stack", stack),
				)

				c.Fail(http.StatusInternalServerError, "Internal Server Error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// isBrokenPipe 检查是否为断开的连接错误
func isBrokenPipe(err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	ne, ok := e.(*net.OpError)
	if !ok {
		return false
	}
	se, ok := ne.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	msg := strings.ToLower(se.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}
