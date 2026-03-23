package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Config 日志中间件配置
type Config struct {
	Output    io.Writer // 输出目标，nil 时默认 os.Stdout
	SkipPaths []string  // 跳过日志记录的路径
}

// ANSI 颜色
const (
	reset  = "\033[0m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	white  = "\033[97m"
)

// Middleware 返回请求日志 gin.HandlerFunc。
// 输出格式：[QI] 2026/03/23 - 17:50:29 |  200 |       917ns |       127.0.0.1 | GET     "/path" trace_id
func Middleware(cfg *Config) gin.HandlerFunc {
	if cfg == nil {
		cfg = &Config{}
	}

	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	skipPaths := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = struct{}{}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		if _, skip := skipPaths[path]; skip {
			c.Next()
			return
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// trace_id 后缀（与 tracing 中间件联动）
		traceStr := ""
		if tid, ok := c.Get("trace_id"); ok {
			if s, ok := tid.(string); ok && s != "" {
				traceStr = " " + s
			}
		}

		statusColor := statusToColor(status)

		fmt.Fprintf(out, "%s[QI]%s %s | %s%3d%s | %12v | %15s | %-7s %q%s\n",
			cyan, reset,
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusColor, status, reset,
			latency,
			clientIP,
			method,
			path,
			traceStr,
		)
	}
}

func statusToColor(status int) string {
	switch {
	case status >= 500:
		return red
	case status >= 400:
		return yellow
	case status >= 300:
		return cyan
	case status >= 200:
		return green
	default:
		return white
	}
}
