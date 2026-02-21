package openapi

// WrapResponseSchema 将业务数据 schema 包装为 Qi 统一响应格式
//
//	{code: 200, data: <dataSchema>, message: "success", trace_id: "..."}
func WrapResponseSchema(dataSchema *Schema) *Schema {
	if dataSchema == nil {
		dataSchema = &Schema{}
	}
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"code":     {Type: "integer", Example: 200},
			"data":     dataSchema,
			"message":  {Type: "string", Example: "success"},
			"trace_id": {Type: "string"},
		},
		Required: []string{"code", "message"},
	}
}

// NullDataResponseSchema 返回 data 为 null 的统一响应 schema（Handle0 场景）
func NullDataResponseSchema() *Schema {
	return WrapResponseSchema(&Schema{Nullable: true})
}

// ErrorResponseRef 错误响应的 $ref 路径
const ErrorResponseRef = "#/components/responses/ErrorResponse"

// BuildErrorResponse 构建统一错误响应定义
func BuildErrorResponse() *Response {
	return &Response{
		Description: "业务错误",
		Content: map[string]MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"code":     {Type: "integer", Example: 1001},
						"data":     {Nullable: true},
						"message":  {Type: "string", Example: "参数错误"},
						"trace_id": {Type: "string"},
					},
				},
			},
		},
	}
}

// DefaultErrorResponses 返回默认的错误响应引用（400/401/500）
func DefaultErrorResponses() map[string]*Response {
	return map[string]*Response{
		"400": {Ref: ErrorResponseRef},
		"401": {Ref: ErrorResponseRef},
		"500": {Ref: ErrorResponseRef},
	}
}
