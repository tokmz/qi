package middleware

import (
	"context"
	"net/http"
	"time"

	"qi"
)

// TimeoutConfig 超时中间件配置
type TimeoutConfig struct {
	// Timeout 请求超时时间（默认 30 秒）
	Timeout time.Duration

	// TimeoutMessage 超时响应消息（默认 "request timeout"）
	TimeoutMessage string

	// SkipFunc 跳过超时控制的函数
	SkipFunc func(c *qi.Context) bool

	// ExcludePaths 排除的路径（不做超时控制）
	ExcludePaths []string
}

// defaultTimeoutConfig 返回默认配置
func defaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Timeout:        30 * time.Second,
		TimeoutMessage: "request timeout",
	}
}

// Timeout 创建超时中间件
// 通过 context.WithTimeout 注入超时 context，handler 应通过 ctx.Done() 感知超时
// 超时后在当前 goroutine 中检查并返回 408
func Timeout(cfgs ...*TimeoutConfig) qi.HandlerFunc {
	cfg := defaultTimeoutConfig()
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	// 构建跳过路径 map
	skipMap := make(map[string]bool)
	for _, path := range cfg.ExcludePaths {
		skipMap[path] = true
	}

	return func(c *qi.Context) {
		// 检查是否跳过
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}
		if skipMap[c.Request().URL.Path] {
			c.Next()
			return
		}

		// 注入带超时的 context，handler 可通过 ctx.Done() 感知超时
		ctx, cancel := context.WithTimeout(c.Request().Context(), cfg.Timeout)
		defer cancel()

		c.SetRequestContext(ctx)

		// 在当前 goroutine 中执行，避免并发安全问题
		c.Next()

		// handler 完成后检查是否已超时
		if ctx.Err() == context.DeadlineExceeded {
			c.Fail(http.StatusRequestTimeout, cfg.TimeoutMessage)
			c.Abort()
		}
	}
}
