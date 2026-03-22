package qi

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/tokmz/qi/internal/openapi"
)

// ===== 辅助函数 =====

// ginPathToOpenAPI 将 gin 风格路径参数转换为 OpenAPI 风格。
// 例如：/users/:id/posts/:postId → /users/{id}/posts/{postId}
func ginPathToOpenAPI(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = "{" + seg[1:] + "}"
		} else if strings.HasPrefix(seg, "*") {
			segments[i] = "{" + seg[1:] + "}"
		}
	}
	return strings.Join(segments, "/")
}

// extractPathParams 从 gin 风格路径中提取所有路径参数名。
// 例如：/users/:id/posts/:postId → ["id", "postId"]
func extractPathParams(path string) []string {
	var params []string
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, ":") {
			params = append(params, seg[1:])
		} else if strings.HasPrefix(seg, "*") {
			params = append(params, seg[1:])
		}
	}
	return params
}

// buildPathParamType 动态构建一个包含路径参数字段的结构体类型。
// 每个参数生成一个 string 字段，带有 uri tag 和 binding:"required" tag。
func buildPathParamType(params []string) any {
	if len(params) == 0 {
		return nil
	}
	fields := make([]reflect.StructField, 0, len(params))
	for _, name := range params {
		if name == "" {
			continue
		}
		fields = append(fields, reflect.StructField{
			Name: strings.ToUpper(name[:1]) + name[1:],
			Type: reflect.TypeOf(""),
			Tag:  reflect.StructTag(`uri:"` + name + `" binding:"required"`),
		})
	}
	if len(fields) == 0 {
		return nil
	}
	t := reflect.StructOf(fields)
	return reflect.New(t).Elem().Interface()
}

// isBodyMethod 判断 HTTP 方法是否应将 Request 数据作为 Body 处理。
func isBodyMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

// ===== OpenAPIConfig =====

// OpenAPIOption 是 OpenAPI 配置选项类型，包装 internal/openapi.Option。
type OpenAPIOption = openapi.Option

// OpenAPIConfig 定义 OpenAPI 文档的配置。
type OpenAPIConfig struct {
	Title       string           // API 标题，默认 "OpenAPI"
	Version     string           // API 版本号，默认 "1.0.0"
	Description string           // API 描述
	Path        string           // spec 端点路径，默认 "/openapi.json"
	SwaggerUI   string           // Swagger UI 路径，空字符串=不注册
	Servers     []openapi.Server // 服务器列表
}

// normalize 填充默认值。
func (c *OpenAPIConfig) normalize() {
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if c.Path == "" {
		c.Path = "/openapi.json"
	}
	if c.SwaggerUI == "" {
		c.SwaggerUI = "/docs"
	}
}

// ===== WithOpenAPI Engine Option =====

// WithOpenAPI 启用 OpenAPI 文档收集，传入 OpenAPIConfig 结构体配置。
func WithOpenAPI(cfg *OpenAPIConfig) Option {
	return func(c *Config) {
		c.openAPIConfig = cfg
	}
}

// ===== Engine 方法 =====

// OpenAPI 返回底层 OpenAPI Manager（可能为 nil）。
func (e *Engine) OpenAPI() *openapi.Manager {
	return e.api
}

// OpenAPIJSON 构建并返回 OpenAPI 文档的 JSON 表示。
// 如果 OpenAPI 未启用，返回 nil, nil。
func (e *Engine) OpenAPIJSON() ([]byte, error) {
	if e.api == nil {
		return nil, nil
	}
	return e.api.MarshalJSON()
}

// OpenAPIYAML 构建并返回 OpenAPI 文档的 YAML 表示。
// 如果 OpenAPI 未启用，返回 nil, nil。
func (e *Engine) OpenAPIYAML() ([]byte, error) {
	if e.api == nil {
		return nil, nil
	}
	return e.api.MarshalYAML()
}

// API 创建一个 RouteBuilder，用于链式注册路由并收集 OpenAPI 信息。
func (e *Engine) API() *RouteBuilder {
	return &RouteBuilder{
		engine: e,
		prefix: "",
	}
}

// ===== RouterGroup 方法 =====

// API 创建一个 RouteBuilder，用于链式注册路由并收集 OpenAPI 信息。
func (r *RouterGroup) API() *RouteBuilder {
	return &RouteBuilder{
		engine:      r.engine,
		prefix:      r.prefix,
		middlewares: cloneHandlers(r.middlewares),
	}
}

// ===== RouteBuilder =====

// RouteBuilder 提供链式 API，一次调用完成路由注册和 OpenAPI 文档收集。
type RouteBuilder struct {
	engine      *Engine
	prefix      string
	middlewares HandlersChain

	// 路由信息
	method   string
	path     string
	handlers HandlersChain

	// OpenAPI 元信息
	summary     string
	description string
	tags        []string
	operationID string
	deprecated  bool

	// 请求响应
	request    any // Request() 自动分发
	query      any // 显式 Query
	body       any // 显式 Body
	pathParams any // 显式 PathParams
	headers    any // 显式 Headers
	cookies    any // 显式 Cookies
	response   any // Response 类型

	// Bind 推导的类型
	boundRequest  reflect.Type // Bind 推导的请求类型
	boundResponse reflect.Type // Bind 推导的响应类型
	boundFuncName string       // Bind/BindR 提取的原始函数名
}

// ----- HTTP 方法 -----

// GET 设置 GET 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) GET(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodGet
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// POST 设置 POST 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) POST(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodPost
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// PUT 设置 PUT 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) PUT(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodPut
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// PATCH 设置 PATCH 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) PATCH(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodPatch
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// DELETE 设置 DELETE 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) DELETE(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodDelete
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// HEAD 设置 HEAD 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) HEAD(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodHead
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// OPTIONS 设置 OPTIONS 方法和路径。handler 可以是 HandlerFunc、func(*Context) 或 BoundHandler。
func (b *RouteBuilder) OPTIONS(path string, handler any, middlewares ...HandlerFunc) *RouteBuilder {
	b.method = http.MethodOptions
	b.path = path
	b.setHandler(handler, middlewares)
	return b
}

// setHandler 统一处理 handler 类型断言，提取 BoundHandler 的类型信息。
func (b *RouteBuilder) setHandler(handler any, middlewares HandlersChain) {
	switch h := handler.(type) {
	case BoundHandler:
		b.handlers = append(HandlersChain{h.Handler}, middlewares...)
		b.boundRequest = h.RequestType
		b.boundResponse = h.ResponseType
		b.boundFuncName = h.FuncName
	case HandlerFunc:
		b.handlers = append(HandlersChain{h}, middlewares...)
	case func(*Context):
		b.handlers = append(HandlersChain{HandlerFunc(h)}, middlewares...)
	default:
		panic("qi: RouteBuilder handler must be qi.HandlerFunc, func(*qi.Context), or qi.BoundHandler")
	}
}

// ----- 元信息 -----

// Summary 设置操作摘要。
func (b *RouteBuilder) Summary(summary string) *RouteBuilder {
	b.summary = summary
	return b
}

// Description 设置操作描述。
func (b *RouteBuilder) Description(desc string) *RouteBuilder {
	b.description = desc
	return b
}

// Tags 设置操作标签。
func (b *RouteBuilder) Tags(tags ...string) *RouteBuilder {
	b.tags = tags
	return b
}

// OperationID 设置操作 ID。
func (b *RouteBuilder) OperationID(id string) *RouteBuilder {
	b.operationID = id
	return b
}

// Deprecated 标记操作为已弃用。
func (b *RouteBuilder) Deprecated() *RouteBuilder {
	b.deprecated = true
	return b
}

// ----- 请求响应 -----

// Request 设置请求类型。根据 HTTP 方法自动分发：
//   - GET/DELETE/HEAD → QueryParams
//   - POST/PUT/PATCH → Body
func (b *RouteBuilder) Request(v any) *RouteBuilder {
	b.request = v
	return b
}

// Response 设置响应类型（固定 200）。
func (b *RouteBuilder) Response(v any) *RouteBuilder {
	b.response = v
	return b
}

// Query 显式设置查询参数类型，覆盖 Request 自动分发。
func (b *RouteBuilder) Query(v any) *RouteBuilder {
	b.query = v
	return b
}

// Body 显式设置请求体类型，覆盖 Request 自动分发。
func (b *RouteBuilder) Body(v any) *RouteBuilder {
	b.body = v
	return b
}

// PathParams 显式设置路径参数类型。
func (b *RouteBuilder) PathParams(v any) *RouteBuilder {
	b.pathParams = v
	return b
}

// Headers 显式设置请求头参数类型。
func (b *RouteBuilder) Headers(v any) *RouteBuilder {
	b.headers = v
	return b
}

// Cookies 显式设置 Cookie 参数类型。
func (b *RouteBuilder) Cookies(v any) *RouteBuilder {
	b.cookies = v
	return b
}

// ----- 终结方法 -----

// Done 注册路由并收集 OpenAPI 信息。始终注册 gin 路由，
// 仅在 OpenAPI 启用时收集文档信息。
func (b *RouteBuilder) Done() {
	// 1. 注册 gin 路由（始终执行）
	relativePath := normalizeAbsolutePath(b.path)
	fullPath := joinPaths(b.prefix, b.path)
	b.engine.handle(b.method, relativePath, fullPath, b.middlewares, b.handlers...)

	// 如果是 Bind/BindR 注册的，用原始函数名覆盖 handler 名称
	if b.boundFuncName != "" {
		routes := b.engine.router.routes
		if len(routes) > 0 {
			routes[len(routes)-1].HandlerName = cleanHandlerName(b.boundFuncName)
		}
	}

	// 2. 如果 OpenAPI 未启用，直接返回
	if b.engine.api == nil {
		return
	}

	// 3. 构建 openapi.Operation
	op := openapi.Operation{
		Method:      strings.ToUpper(b.method),
		Path:        ginPathToOpenAPI(fullPath),
		OperationID: b.operationID,
		Summary:     b.summary,
		Description: b.description,
		Tags:        b.tags,
		Deprecated:  b.deprecated,
	}

	// 3a. 构建 Request
	req := &openapi.Request{}
	hasRequest := false

	// 路径参数: 显式 PathParams > 从 URL 自动提取
	if b.pathParams != nil {
		req.PathParams = b.pathParams
		hasRequest = true
	} else if params := extractPathParams(fullPath); len(params) > 0 {
		req.PathParams = buildPathParamType(params)
		hasRequest = true
	}

	// 查询参数/Body: 显式 > Request 自动分发
	if b.query != nil {
		req.QueryParams = b.query
		hasRequest = true
	}
	if b.body != nil {
		req.Body = b.body
		req.BodyRequired = true
		hasRequest = true
	}

	// Request 自动分发（仅当未显式设置对应字段时）
	if b.request != nil {
		if isBodyMethod(b.method) {
			if b.body == nil {
				req.Body = b.request
				req.BodyRequired = true
				hasRequest = true
			}
		} else {
			if b.query == nil {
				req.QueryParams = b.request
				hasRequest = true
			}
		}
	}

	// Bind 推导的请求类型回退（优先级最低）
	if b.request == nil && b.body == nil && b.query == nil && b.boundRequest != nil {
		v := reflect.New(b.boundRequest).Elem().Interface()
		if isBodyMethod(b.method) {
			req.Body = v
			req.BodyRequired = true
		} else {
			req.QueryParams = v
		}
		hasRequest = true
	}

	// Headers / Cookies
	if b.headers != nil {
		req.Headers = b.headers
		hasRequest = true
	}
	if b.cookies != nil {
		req.Cookies = b.cookies
		hasRequest = true
	}

	if hasRequest {
		op.Request = req
	}

	// 3b. Response: 显式 .Response() > boundResponse（Bind 推导）
	if b.response != nil {
		op.Responses = []openapi.Response{
			{
				Status:      200,
				Description: "成功",
				Body:        b.response,
			},
		}
	} else if b.boundResponse != nil {
		op.Responses = []openapi.Response{
			{
				Status:      200,
				Description: "成功",
				Body:        reflect.New(b.boundResponse).Elem().Interface(),
			},
		}
	}

	// 4. 注册 OpenAPI Operation（失败则 panic，启动时快速失败）
	if err := b.engine.api.AddOperation(op); err != nil {
		panic("qi: OpenAPI AddOperation failed: " + err.Error())
	}
}
