package request

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Client HTTP 客户端
type Client struct {
	cfg    *Config
	client *http.Client
}

// New 创建 HTTP 客户端
func New(opts ...Option) *Client {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return NewWithConfig(cfg)
}

// NewWithConfig 使用配置创建 HTTP 客户端
func NewWithConfig(cfg *Config) *Client {
	transport := cfg.buildTransport()

	// 启用追踪时包装 Transport
	if cfg.EnableTracing {
		transport = newTracingTransport(transport)
	}

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

// Get 创建 GET 请求
func (c *Client) Get(url string) *Request {
	return newRequest(c, http.MethodGet, url)
}

// Post 创建 POST 请求
func (c *Client) Post(url string) *Request {
	return newRequest(c, http.MethodPost, url)
}

// Put 创建 PUT 请求
func (c *Client) Put(url string) *Request {
	return newRequest(c, http.MethodPut, url)
}

// Patch 创建 PATCH 请求
func (c *Client) Patch(url string) *Request {
	return newRequest(c, http.MethodPatch, url)
}

// Delete 创建 DELETE 请求
func (c *Client) Delete(url string) *Request {
	return newRequest(c, http.MethodDelete, url)
}

// Head 创建 HEAD 请求
func (c *Client) Head(url string) *Request {
	return newRequest(c, http.MethodHead, url)
}

// R 创建通用请求构建器（需通过 SetMethod/SetURL 设置）
func (c *Client) R(ctx context.Context) *Request {
	r := newRequest(c, "", "")
	r.ctx = ctx
	return r
}

// mergeHeaders 合并全局 header 和请求 header（请求级优先），返回新 map
func (c *Client) mergeHeaders(reqHeaders map[string]string) map[string]string {
	merged := make(map[string]string, len(c.cfg.Headers)+len(reqHeaders))
	for k, v := range c.cfg.Headers {
		merged[k] = v
	}
	// 请求级 header 覆盖全局
	for k, v := range reqHeaders {
		merged[k] = v
	}
	return merged
}

// execute 执行请求（含重试、拦截器、追踪）
func (c *Client) execute(r *Request) (*Response, error) {
	// 确定重试配置（clone 避免修改原始配置）
	retryCfg := r.retry
	if retryCfg == nil {
		retryCfg = c.cfg.Retry
	}

	// 无重试直接执行
	if retryCfg == nil {
		return c.doOnce(r)
	}

	// clone + normalize 避免修改用户传入的配置
	rc := *retryCfg
	rc.normalize()

	var lastResp *Response
	var lastErr error

	for attempt := 0; attempt <= rc.MaxAttempts; attempt++ {
		lastResp, lastErr = c.doOnce(r)

		if attempt == rc.MaxAttempts {
			break
		}

		var httpResp *http.Response
		if lastResp != nil {
			httpResp = &http.Response{StatusCode: lastResp.StatusCode}
		}

		if !rc.RetryIf(httpResp, lastErr) {
			return lastResp, lastErr
		}

		// 退避等待（使用 NewTimer 避免泄漏）
		delay := rc.backoff(attempt)
		timer := time.NewTimer(delay)
		select {
		case <-r.ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("%w: %w", ErrTimeout, r.ctx.Err())
		case <-timer.C:
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrMaxRetry, lastErr)
	}
	return lastResp, nil
}

// doOnce 执行单次请求
func (c *Client) doOnce(r *Request) (*Response, error) {
	// 合并 header（不修改 Request 本身）
	merged := c.mergeHeaders(r.headers)

	httpReq, err := r.buildHTTPRequest(c.cfg.BaseURL, merged)
	if err != nil {
		return nil, err
	}

	// 设置请求级超时（cancel 限定在本次调用内）
	if r.timeout > 0 {
		ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
		defer cancel()
		httpReq = httpReq.WithContext(ctx)
	}

	// 执行 BeforeRequest 拦截器
	for _, interceptor := range c.cfg.Interceptors {
		if err := interceptor.BeforeRequest(httpReq.Context(), httpReq); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRequestFailed, err)
		}
	}

	// OTel Span
	var span trace.Span
	if c.cfg.EnableTracing {
		tracer := otel.Tracer("qi.request")
		ctx, s := tracer.Start(httpReq.Context(), "HTTP "+httpReq.Method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("http.method", httpReq.Method),
				attribute.String("http.url", httpReq.URL.String()),
			),
		)
		span = s
		httpReq = httpReq.WithContext(ctx)
	}

	// 发送请求
	start := time.Now()
	httpResp, err := c.client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
		}
		if c.cfg.Logger != nil {
			c.cfg.Logger.ErrorContext(httpReq.Context(), "http request failed",
				"method", httpReq.Method,
				"url", httpReq.URL.String(),
				"error", err,
			)
		}
		return nil, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
		}
		if c.cfg.Logger != nil {
			c.cfg.Logger.ErrorContext(httpReq.Context(), "http read body failed",
				"method", httpReq.Method,
				"url", httpReq.URL.String(),
				"error", err,
			)
		}
		return nil, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       body,
		Duration:   duration,
		Request:    httpReq,
	}

	// OTel Span 记录响应
	if span != nil {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		if resp.IsError() {
			span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		}
		span.End()
	}

	// 执行 AfterResponse 拦截器
	for _, interceptor := range c.cfg.Interceptors {
		if err := interceptor.AfterResponse(httpReq.Context(), resp); err != nil {
			return resp, fmt.Errorf("%w: %w", ErrRequestFailed, err)
		}
	}

	return resp, nil
}
