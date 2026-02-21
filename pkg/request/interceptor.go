package request

import (
	"context"
	"net/http"
)

// Interceptor 拦截器接口
type Interceptor interface {
	// BeforeRequest 请求发送前调用
	BeforeRequest(ctx context.Context, req *http.Request) error
	// AfterResponse 响应返回后调用
	AfterResponse(ctx context.Context, resp *Response) error
}

// loggingInterceptor 日志拦截器
type loggingInterceptor struct {
	log Logger
}

// NewLoggingInterceptor 创建日志拦截器
func NewLoggingInterceptor(log Logger) Interceptor {
	return &loggingInterceptor{log: log}
}

func (l *loggingInterceptor) BeforeRequest(ctx context.Context, req *http.Request) error {
	l.log.InfoContext(ctx, "http request",
		"method", req.Method,
		"url", req.URL.String(),
	)
	return nil
}

func (l *loggingInterceptor) AfterResponse(ctx context.Context, resp *Response) error {
	l.log.InfoContext(ctx, "http response",
		"method", resp.Request.Method,
		"url", resp.Request.URL.String(),
		"status", resp.StatusCode,
		"duration", resp.Duration,
	)
	return nil
}

// authInterceptor 认证拦截器
type authInterceptor struct {
	tokenFunc func() string
}

// NewAuthInterceptor 创建认证拦截器
func NewAuthInterceptor(tokenFunc func() string) Interceptor {
	return &authInterceptor{tokenFunc: tokenFunc}
}

func (a *authInterceptor) BeforeRequest(_ context.Context, req *http.Request) error {
	if token := a.tokenFunc(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func (a *authInterceptor) AfterResponse(_ context.Context, _ *Response) error {
	return nil
}
