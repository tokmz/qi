package request

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response HTTP 响应包装
type Response struct {
	StatusCode int           // HTTP 状态码
	Headers    http.Header   // 响应头
	Body       []byte        // 响应体
	Duration   time.Duration // 请求耗时
	Request    *http.Request // 原始请求
}

// IsSuccess 判断是否为成功响应（2xx）
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError 判断是否为错误响应（4xx/5xx）
func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

// Unmarshal JSON 反序列化到任意类型
func (r *Response) Unmarshal(v any) error {
	if err := json.Unmarshal(r.Body, v); err != nil {
		return ErrUnmarshal.WithError(err)
	}
	return nil
}

// String 返回 Body 字符串
func (r *Response) String() string {
	return string(r.Body)
}

// Do 泛型解析：发送请求并将响应 JSON 反序列化为 *T
// 当 HTTP 状态码为 4xx/5xx 时返回 ErrRequestFailed
func Do[T any](req *Request) (*T, error) {
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, truncatedError(resp)
	}
	var result T
	if err := resp.Unmarshal(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DoList 泛型解析：发送请求并将响应 JSON 反序列化为 []T
// 当 HTTP 状态码为 4xx/5xx 时返回 ErrRequestFailed
func DoList[T any](req *Request) ([]T, error) {
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, truncatedError(resp)
	}
	var result []T
	if err := resp.Unmarshal(&result); err != nil {
		return nil, err
	}
	return result, nil
}

const maxErrorBodyLen = 512

// truncatedError 构建截断 body 的错误（避免巨大响应体污染错误消息）
func truncatedError(resp *Response) error {
	body := resp.Body
	if len(body) > maxErrorBodyLen {
		body = body[:maxErrorBodyLen]
	}
	return ErrRequestFailed.WithMessage(
		"HTTP " + http.StatusText(resp.StatusCode) + ": " + string(body),
	)
}
