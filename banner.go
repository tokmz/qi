package qi

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/tokmz/qi/utils/strings"

	"github.com/gin-gonic/gin"
)

// Version 框架版本号
const Version = "1.0.5"

// banner ASCII Art
const banner = `
 ██████╗ ██╗    Qi 基于Gin的Go Web 框架
██╔═══██╗██║	Qi 是一个基于 Gin 的轻量级 Web 框架，提供统一的响应格式、自动参数绑定、泛型路由支持和优雅关机功能
██║   ██║██║	github: https://github.com/tokmz/qi
██║▄▄ ██║██║	QQ: 81288369
╚██████╔╝██║	open: %s
 ╚══▀▀═╝ ╚═╝  	version: %s
`

// printBanner 打印启动 banner 和路由表
func (e *Engine) printBanner(addr string) {
	out := os.Stdout

	// 拼接访问地址
	var open string
	if strings.HasPrefix(addr, ":") {
		open = "http://127.0.0.1" + addr
	} else if strings.Contains(addr, ":") {
		open = "http://" + addr
	} else {
		open = "http://127.0.0.1:" + addr
	}

	// 打印 banner
	fPrint(out, banner, open, Version)
	fPrint(out, "\n")

	// 打印路由表
	routes := e.engine.Routes()
	if len(routes) > 0 {
		printRoutes(out, routes, e.config.Mode)
		fPrint(out, "\n")
	}

	// 打印运行模式
	mode := e.config.Mode
	if mode == "debug" {
		fPrint(out, "[Qi] Running in \"%s\" mode. Switch to \"release\" mode in production.\n", mode)
	} else {
		fPrint(out, "[Qi] Running in \"%s\" mode.\n", mode)
	}

	// 打印环境信息
	fPrint(out, "[Qi] Go version: %s | OS: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// 打印启动信息
	fPrint(out, "[Qi] Listening on %s\n", addr)
}

// methodColor 根据 HTTP 方法返回 ANSI 颜色码
func methodColor(method string) string {
	switch method {
	case "GET":
		return "\033[34m" // 蓝色
	case "POST":
		return "\033[32m" // 绿色
	case "PUT":
		return "\033[33m" // 黄色
	case "DELETE":
		return "\033[31m" // 红色
	case "PATCH":
		return "\033[36m" // 青色
	case "HEAD":
		return "\033[35m" // 紫色
	case "OPTIONS":
		return "\033[37m" // 灰色
	default:
		return "\033[0m"
	}
}

const resetColor = "\033[0m"

// printRoutes 格式化打印路由表（Gin 风格 + 颜色）
func printRoutes(out io.Writer, routes gin.RoutesInfo, mode string) {
	// 计算路径列最大宽度，用于对齐
	maxPathLen := 0
	for _, r := range routes {
		if len(r.Path) > maxPathLen {
			maxPathLen = len(r.Path)
		}
	}

	for _, r := range routes {
		fPrint(out, "[Qi-%s] %s %-7s %s %-*s --> %s\n",
			mode,
			methodColor(r.Method), r.Method, resetColor,
			maxPathLen, r.Path,
			r.Handler)
	}
}

// silenceGin 静默 Gin 的默认输出
func silenceGin() {
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
}

// fPrint 打印到 writer，忽略错误（banner 输出场景）
func fPrint(out io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(out, format, a...)
}
