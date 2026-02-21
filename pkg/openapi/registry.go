package openapi

import (
	"reflect"
	"sort"
	"strings"
)

// RouteType 路由类型（泛型函数变体）
type RouteType int

const (
	RouteTypeFull         RouteType = iota // Handle[Req, Resp] — 有请求 + 有响应
	RouteTypeRequestOnly                   // Handle0[Req] — 有请求，无响应数据
	RouteTypeResponseOnly                  // HandleOnly[Resp] — 无请求，有响应
)

// RouteEntry 单条路由的元数据
type RouteEntry struct {
	Method   string       // HTTP 方法：GET, POST, PUT, DELETE, PATCH
	Path     string       // Gin 格式路径，如 "/users/:id"
	BasePath string       // 路由组的 basePath
	Type     RouteType    // 路由类型
	ReqType  reflect.Type // 请求类型（RouteTypeResponseOnly 时为 nil）
	RespType reflect.Type // 响应类型（RouteTypeRequestOnly 时为 nil）
	Doc      *DocOption   // 文档元数据（可为 nil）

	// 从 RouterGroup 继承的默认值
	DefaultTag      string
	DefaultTagDesc  string
	DefaultSecurity []string
}

// Registry 收集所有路由元数据，构建最终 OpenAPI spec
type Registry struct {
	config  *Config
	entries []RouteEntry
}

// NewRegistry 创建 Registry
func NewRegistry(config *Config) *Registry {
	config.Normalize()
	return &Registry{
		config: config,
	}
}

// Add 添加一条路由记录
func (r *Registry) Add(entry RouteEntry) {
	r.entries = append(r.entries, entry)
}

// Build 构建完整的 OpenAPI Document
func (r *Registry) Build() *Document {
	builder := NewSchemaBuilder()

	doc := &Document{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       r.config.Title,
			Version:     r.config.Version,
			Description: r.config.Description,
		},
		Paths: make(map[string]*PathItem),
	}

	if len(r.config.Servers) > 0 {
		doc.Servers = r.config.Servers
	}

	tagSet := make(map[string]string) // name → description

	for i := range r.entries {
		entry := &r.entries[i]
		op := r.buildOperation(entry, builder, tagSet)

		openAPIPath := GinPathToOpenAPI(entry.Path)
		pathItem, ok := doc.Paths[openAPIPath]
		if !ok {
			pathItem = &PathItem{}
			doc.Paths[openAPIPath] = pathItem
		}
		pathItem.SetOperation(entry.Method, op)
	}

	// 构建 components
	components := &Components{}

	// schemas
	if len(builder.Schemas) > 0 {
		components.Schemas = builder.Schemas
	}

	// security schemes
	if len(r.config.SecuritySchemes) > 0 {
		components.SecuritySchemes = make(map[string]*SecurityScheme, len(r.config.SecuritySchemes))
		for name, scheme := range r.config.SecuritySchemes {
			s := scheme // copy
			components.SecuritySchemes[name] = &s
		}
	}

	// error response
	components.Responses = map[string]*Response{
		"ErrorResponse": BuildErrorResponse(),
	}

	doc.Components = components

	// tags（排序保证输出稳定）
	if len(tagSet) > 0 {
		tags := make([]Tag, 0, len(tagSet))
		for name, desc := range tagSet {
			tags = append(tags, Tag{Name: name, Description: desc})
		}
		sort.Slice(tags, func(i, j int) bool {
			return tags[i].Name < tags[j].Name
		})
		doc.Tags = tags
	}

	return doc
}

// buildOperation 为单条路由构建 Operation
func (r *Registry) buildOperation(entry *RouteEntry, builder *SchemaBuilder, tagSet map[string]string) *Operation {
	op := &Operation{
		Responses: make(map[string]*Response),
	}

	// 应用 DocOption
	if entry.Doc != nil {
		op.Summary = entry.Doc.Summary
		op.Description = entry.Doc.Description
		op.Deprecated = entry.Doc.Deprecated
	}

	// Tags 解析：DocOption > DefaultTag > DeriveTag
	tags := r.resolveTags(entry, tagSet)
	if len(tags) > 0 {
		op.Tags = tags
	}

	// Security 解析
	security := r.resolveSecurity(entry)
	if security != nil {
		op.Security = security
	}

	// 请求参数 / RequestBody
	if entry.ReqType != nil {
		r.buildRequest(op, entry, builder)
	}

	// 响应
	r.buildResponse(op, entry, builder)

	return op
}

// resolveTags 解析最终 tag 列表
func (r *Registry) resolveTags(entry *RouteEntry, tagSet map[string]string) []string {
	// DocOption 显式指定
	if entry.Doc != nil && len(entry.Doc.Tags) > 0 {
		for _, t := range entry.Doc.Tags {
			if _, ok := tagSet[t]; !ok {
				tagSet[t] = ""
			}
		}
		return entry.Doc.Tags
	}

	// RouterGroup 默认 tag
	if entry.DefaultTag != "" {
		tagSet[entry.DefaultTag] = entry.DefaultTagDesc
		return []string{entry.DefaultTag}
	}

	// 自动推导
	if tag := DeriveTag(entry.BasePath); tag != "" {
		if _, ok := tagSet[tag]; !ok {
			tagSet[tag] = ""
		}
		return []string{tag}
	}

	return nil
}

// resolveSecurity 解析最终 security 配置
func (r *Registry) resolveSecurity(entry *RouteEntry) []SecurityRequirement {
	// NoSecurity 显式取消
	if entry.Doc != nil && entry.Doc.NoSecurity {
		return []SecurityRequirement{{}} // 空对象 = 无认证
	}

	// DocOption 显式指定
	if entry.Doc != nil && len(entry.Doc.Security) > 0 {
		reqs := make([]SecurityRequirement, 0, len(entry.Doc.Security))
		for _, s := range entry.Doc.Security {
			reqs = append(reqs, SecurityRequirement{s: {}})
		}
		return reqs
	}

	// RouterGroup 默认 security
	if len(entry.DefaultSecurity) > 0 {
		reqs := make([]SecurityRequirement, 0, len(entry.DefaultSecurity))
		for _, s := range entry.DefaultSecurity {
			reqs = append(reqs, SecurityRequirement{s: {}})
		}
		return reqs
	}

	return nil
}

// buildRequest 构建请求参数和 RequestBody
func (r *Registry) buildRequest(op *Operation, entry *RouteEntry, builder *SchemaBuilder) {
	reqType := entry.ReqType
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}
	if reqType.Kind() != reflect.Struct {
		return
	}

	var bodyFields []reflect.StructField
	hasFile := HasFileUpload(reqType)

	for i := range reqType.NumField() {
		field := reqType.Field(i)
		if !field.IsExported() {
			continue
		}

		// 嵌入 struct 展开
		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				r.collectFieldParams(op, ft, entry.Method, builder, &bodyFields)
				continue
			}
		}

		r.classifyField(op, field, entry.Method, builder, &bodyFields)
	}

	// 构建 RequestBody
	if len(bodyFields) > 0 {
		bodySchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
		}
		for _, f := range bodyFields {
			name := fieldName(f)
			if name == "-" {
				continue
			}
			fs := builder.Build(f.Type)

			if bindTag := f.Tag.Get("binding"); bindTag != "" {
				constraints := ParseBindingTag(bindTag)
				constraints.ApplyToSchema(fs)
				if constraints.Required {
					bodySchema.Required = append(bodySchema.Required, name)
				}
			}
			if validateTag := f.Tag.Get("validate"); validateTag != "" {
				constraints := ParseBindingTag(validateTag)
				constraints.ApplyToSchema(fs)
				if constraints.Required {
					if !containsString(bodySchema.Required, name) {
						bodySchema.Required = append(bodySchema.Required, name)
					}
				}
			}

			if desc := f.Tag.Get("desc"); desc != "" {
				fs.Description = desc
			}

			bodySchema.Properties[name] = fs
		}

		contentType := "application/json"
		if hasFile {
			contentType = "multipart/form-data"
		}

		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				contentType: {Schema: bodySchema},
			},
		}
	}
}

// collectFieldParams 递归收集嵌入 struct 的字段
func (r *Registry) collectFieldParams(op *Operation, t reflect.Type, method string, builder *SchemaBuilder, bodyFields *[]reflect.StructField) {
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			// 嵌入 struct 即使未导出也要展开
			if field.Anonymous {
				ft := field.Type
				if ft.Kind() == reflect.Ptr {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					r.collectFieldParams(op, ft, method, builder, bodyFields)
				}
			}
			continue
		}
		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				r.collectFieldParams(op, ft, method, builder, bodyFields)
				continue
			}
		}
		r.classifyField(op, field, method, builder, bodyFields)
	}
}

// classifyField 根据 struct tag 将字段分类为 parameter 或 body field
func (r *Registry) classifyField(op *Operation, field reflect.StructField, method string, builder *SchemaBuilder, bodyFields *[]reflect.StructField) {
	hasURI := field.Tag.Get("uri") != ""
	hasHeader := field.Tag.Get("header") != ""

	loc := InferFieldLocation(method, hasURI, hasHeader)

	switch loc {
	case LocPath:
		name := field.Tag.Get("uri")
		if name == "" {
			name = fieldName(field)
		}
		p := Parameter{
			Name:     name,
			In:       "path",
			Required: true,
			Schema:   builder.Build(field.Type),
		}
		if desc := field.Tag.Get("desc"); desc != "" {
			p.Description = desc
		}
		op.Parameters = append(op.Parameters, p)

	case LocQuery:
		name := field.Tag.Get("form")
		if name == "" {
			name = fieldName(field)
		}
		if n, _, ok := strings.Cut(name, ","); ok {
			name = n
		}
		p := Parameter{
			Name:   name,
			In:     "query",
			Schema: builder.Build(field.Type),
		}
		if bindTag := field.Tag.Get("binding"); bindTag != "" {
			c := ParseBindingTag(bindTag)
			p.Required = c.Required
			c.ApplyToSchema(p.Schema)
		}
		if desc := field.Tag.Get("desc"); desc != "" {
			p.Description = desc
		}
		op.Parameters = append(op.Parameters, p)

	case LocHeader:
		name := field.Tag.Get("header")
		if n, _, ok := strings.Cut(name, ","); ok {
			name = n
		}
		p := Parameter{
			Name:   name,
			In:     "header",
			Schema: builder.Build(field.Type),
		}
		if bindTag := field.Tag.Get("binding"); bindTag != "" {
			c := ParseBindingTag(bindTag)
			p.Required = c.Required
		}
		if desc := field.Tag.Get("desc"); desc != "" {
			p.Description = desc
		}
		op.Parameters = append(op.Parameters, p)

	case LocBody:
		*bodyFields = append(*bodyFields, field)
	}
}

// buildResponse 构建响应 schema
func (r *Registry) buildResponse(op *Operation, entry *RouteEntry, builder *SchemaBuilder) {
	var dataSchema *Schema

	switch entry.Type {
	case RouteTypeRequestOnly:
		// Handle0: data 固定为 null
		dataSchema = &Schema{Nullable: true}
	case RouteTypeFull, RouteTypeResponseOnly:
		if entry.RespType != nil {
			dataSchema = builder.Build(entry.RespType)
		} else {
			dataSchema = &Schema{}
		}
	}

	wrapped := WrapResponseSchema(dataSchema)

	op.Responses["200"] = &Response{
		Description: "成功",
		Content: map[string]MediaType{
			"application/json": {Schema: wrapped},
		},
	}

	// 添加默认错误响应引用
	for code, resp := range DefaultErrorResponses() {
		op.Responses[code] = resp
	}
}
