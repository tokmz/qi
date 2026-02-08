package qi

import "net/http"

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`               // 业务状态码
	Data    any    `json:"data"`               // 响应数据
	Message string `json:"message"`            // 响应消息
	TraceID string `json:"trace_id,omitempty"` // 追踪ID（可选）
}

// NewResponse 创建响应
func NewResponse(code int, data any, message string) *Response {
	return &Response{
		Code:    code,
		Data:    data,
		Message: message,
	}
}

// WithTraceID 设置追踪ID
func (r *Response) WithTraceID(traceID string) *Response {
	r.TraceID = traceID
	return r
}

// Success 创建成功响应
func Success(data any) *Response {
	return NewResponse(http.StatusOK, data, "success")
}

// SuccessWithMessage 创建成功响应（自定义消息）
func SuccessWithMessage(data any, message string) *Response {
	return NewResponse(http.StatusOK, data, message)
}

// Fail 创建失败响应
func Fail(code int, message string) *Response {
	return NewResponse(code, nil, message)
}

// PageResp 分页响应结构
type PageResp struct {
	List  any    `json:"list"`  // 数据列表
	Total uint64 `json:"total"` // 总数
}

// NewPageResp 创建分页响应
func NewPageResp(list any, total uint64) *PageResp {
	// 确保 list 不为 nil，避免 JSON 序列化为 null
	if list == nil {
		list = []any{}
	}
	return &PageResp{
		List:  list,
		Total: total,
	}
}

// PageData 分页数据包装器
func PageData(list any, total uint64) *Response {
	return Success(NewPageResp(list, total))
}
