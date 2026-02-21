package request

import (
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts  int                                       // 最大重试次数（默认 3）
	InitialDelay time.Duration                             // 初始退避（默认 100ms）
	MaxDelay     time.Duration                             // 最大退避（默认 5s）
	Multiplier   float64                                   // 退避倍数（默认 2.0）
	RetryIf      func(resp *http.Response, err error) bool // 自定义重试条件
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		RetryIf:      defaultRetryIf,
	}
}

// defaultRetryIf 默认重试条件：网络错误或 5xx 状态码
func defaultRetryIf(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp != nil && resp.StatusCode >= http.StatusInternalServerError
}

// backoff 计算第 attempt 次重试的退避时间（带抖动）
func (rc *RetryConfig) backoff(attempt int) time.Duration {
	delay := float64(rc.InitialDelay) * math.Pow(rc.Multiplier, float64(attempt))
	if delay > float64(rc.MaxDelay) {
		delay = float64(rc.MaxDelay)
	}
	// 添加 ±25% 抖动
	jitter := delay * 0.25 * (rand.Float64()*2 - 1)
	d := time.Duration(delay + jitter)
	if d < 0 {
		d = 0
	}
	return d
}

// normalize 填充零值字段为默认值
func (rc *RetryConfig) normalize() {
	if rc.MaxAttempts <= 0 {
		rc.MaxAttempts = 3
	}
	if rc.InitialDelay <= 0 {
		rc.InitialDelay = 100 * time.Millisecond
	}
	if rc.MaxDelay <= 0 {
		rc.MaxDelay = 5 * time.Second
	}
	if rc.Multiplier <= 0 {
		rc.Multiplier = 2.0
	}
	if rc.RetryIf == nil {
		rc.RetryIf = defaultRetryIf
	}
}
