package qi

// Response 响应结构体
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	TraceID string `json:"trace_id,omitempty"`
}

// NewResponse 创建新的响应
func NewResponse(code int, message string, data any) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// SetTraceID 设置 trace_id
func (r *Response) SetTraceID(traceID string) {
	r.TraceID = traceID
}
