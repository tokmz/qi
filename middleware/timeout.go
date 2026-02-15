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
// 通过 context.WithTimeout 控制请求超时，超时后返回 408
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

		ctx, cancel := context.WithTimeout(c.Request().Context(), cfg.Timeout)
		defer cancel()

		c.SetRequestContext(ctx)

		// 用 channel 等待处理完成
		done := make(chan struct{}, 1)
		go func() {
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			// 正常完成
		case <-ctx.Done():
			// 超时
			c.Fail(http.StatusRequestTimeout, cfg.TimeoutMessage)
			c.Abort()
		}
	}
}
