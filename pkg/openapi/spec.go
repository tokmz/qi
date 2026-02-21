package openapi

// Document OpenAPI 3.0 文档根对象
type Document struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Paths      map[string]*PathItem  `json:"paths"`
	Components *Components           `json:"components,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Info API 元信息
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Server 服务器信息
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Tag 标签定义
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem 路径项，每个 HTTP 方法对应一个 Operation
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

// Operation 单个 API 操作
type Operation struct {
	Summary     string               `json:"summary,omitempty"`
	Description string               `json:"description,omitempty"`
	OperationID string               `json:"operationId,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Parameters  []Parameter          `json:"parameters,omitempty"`
	RequestBody *RequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*Response `json:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty"`
	Deprecated  bool                 `json:"deprecated,omitempty"`
}

// Parameter 请求参数（query, path, header, cookie）
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // query, path, header, cookie
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody 请求体
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

// Response 响应定义
type Response struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Ref         string               `json:"$ref,omitempty"`
}

// MediaType 媒体类型
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Schema JSON Schema 子集（OpenAPI 3.0）
type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Example              any                `json:"example,omitempty"`
}

// Components 可复用组件
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme 安全方案定义
type SecurityScheme struct {
	Type         string `json:"type"`                   // http, apiKey, oauth2, openIdConnect
	Scheme       string `json:"scheme,omitempty"`       // bearer, basic（type=http 时）
	BearerFormat string `json:"bearerFormat,omitempty"` // JWT 等（type=http, scheme=bearer 时）
	In           string `json:"in,omitempty"`           // query, header, cookie（type=apiKey 时）
	Name         string `json:"name,omitempty"`         // 参数名（type=apiKey 时）
}

// SecurityRequirement 安全需求（key 为 scheme 名，value 为 scope 列表）
type SecurityRequirement map[string][]string

// SetOperation 根据 HTTP 方法设置 PathItem 上的 Operation
func (p *PathItem) SetOperation(method string, op *Operation) {
	switch method {
	case "GET":
		p.Get = op
	case "POST":
		p.Post = op
	case "PUT":
		p.Put = op
	case "DELETE":
		p.Delete = op
	case "PATCH":
		p.Patch = op
	}
}
