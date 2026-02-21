package openapi

import (
	"encoding/json"
	"mime/multipart"
	"reflect"
	"testing"
	"time"
)

func TestGinPathToOpenAPI(t *testing.T) {
	tests := []struct {
		gin    string
		expect string
	}{
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:post_id", "/users/{id}/posts/{post_id}"},
		{"/static/*filepath", "/static/{filepath}"},
		{"/api/v1/:org/repos/:repo/*action", "/api/v1/{org}/repos/{repo}/{action}"},
		{"/", "/"},
	}
	for _, tt := range tests {
		got := GinPathToOpenAPI(tt.gin)
		if got != tt.expect {
			t.Errorf("GinPathToOpenAPI(%q) = %q, want %q", tt.gin, got, tt.expect)
		}
	}
}

func TestConfigNormalize(t *testing.T) {
	c := &Config{Title: "Test"}
	c.Normalize()

	if c.Path != DefaultPath {
		t.Errorf("Path = %q, want %q", c.Path, DefaultPath)
	}
	if c.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", c.Version, "1.0.0")
	}

	// 不覆盖已设置的值
	c2 := &Config{Path: "/api-docs.json", Version: "2.0.0"}
	c2.Normalize()
	if c2.Path != "/api-docs.json" {
		t.Errorf("Path = %q, want %q", c2.Path, "/api-docs.json")
	}
	if c2.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", c2.Version, "2.0.0")
	}
}

func TestDocOption(t *testing.T) {
	doc := Doc(
		Summary("创建用户"),
		Desc("创建一个新用户"),
		Tags("Users", "Admin"),
		Deprecated(),
		Security("BearerAuth"),
	)

	if doc.Summary != "创建用户" {
		t.Errorf("Summary = %q", doc.Summary)
	}
	if doc.Description != "创建一个新用户" {
		t.Errorf("Description = %q", doc.Description)
	}
	if len(doc.Tags) != 2 || doc.Tags[0] != "Users" || doc.Tags[1] != "Admin" {
		t.Errorf("Tags = %v", doc.Tags)
	}
	if !doc.Deprecated {
		t.Error("Deprecated should be true")
	}
	if len(doc.Security) != 1 || doc.Security[0] != "BearerAuth" {
		t.Errorf("Security = %v", doc.Security)
	}
}

func TestDocOptionNoSecurity(t *testing.T) {
	doc := Doc(NoSecurity())
	if !doc.NoSecurity {
		t.Error("NoSecurity should be true")
	}
}

type testReq struct {
	Name string `json:"name"`
}

type testResp struct {
	ID int `json:"id"`
}

func TestDocOptionRequestResponseType(t *testing.T) {
	doc := Doc(
		RequestType(testReq{}),
		ResponseType(testResp{}),
	)

	if doc.ReqType == nil {
		t.Fatal("ReqType should not be nil")
	}
	if doc.ReqType.Name() != "testReq" {
		t.Errorf("ReqType.Name() = %q", doc.ReqType.Name())
	}

	if doc.RespType == nil {
		t.Fatal("RespType should not be nil")
	}
	if doc.RespType.Name() != "testResp" {
		t.Errorf("RespType.Name() = %q", doc.RespType.Name())
	}
}

func TestDocOptionRequestTypePointer(t *testing.T) {
	doc := Doc(RequestType(&testReq{}))
	if doc.ReqType == nil {
		t.Fatal("ReqType should not be nil")
	}
	if doc.ReqType.Name() != "testReq" {
		t.Errorf("pointer should be dereferenced, got %q", doc.ReqType.Name())
	}
}

func TestDocOptionNilType(t *testing.T) {
	doc := Doc(RequestType(nil), ResponseType(nil))
	if doc.ReqType != nil {
		t.Error("ReqType should be nil")
	}
	if doc.RespType != nil {
		t.Error("RespType should be nil")
	}
}

func TestPathItemSetOperation(t *testing.T) {
	p := &PathItem{}
	op := &Operation{Summary: "test"}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, m := range methods {
		p.SetOperation(m, op)
	}

	if p.Get == nil || p.Post == nil || p.Put == nil || p.Delete == nil || p.Patch == nil {
		t.Error("all operations should be set")
	}
}

func TestSpecJSONSerialization(t *testing.T) {
	doc := &Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Test API", Version: "1.0.0"},
		Paths: map[string]*PathItem{
			"/users/{id}": {
				Get: &Operation{
					Summary: "Get user",
					Tags:    []string{"Users"},
					Parameters: []Parameter{
						{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "integer"}},
					},
					Responses: map[string]*Response{
						"200": {Description: "成功"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v", parsed["openapi"])
	}

	paths, ok := parsed["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths should be a map")
	}
	if _, ok := paths["/users/{id}"]; !ok {
		t.Error("path /users/{id} should exist")
	}
}

func TestSchemaOmitempty(t *testing.T) {
	s := &Schema{Type: "string"}
	data, _ := json.Marshal(s)
	var m map[string]any
	json.Unmarshal(data, &m)

	// nullable 为 false 时不应出现在 JSON 中
	if _, ok := m["nullable"]; ok {
		t.Error("nullable should be omitted when false")
	}
	if _, ok := m["minimum"]; ok {
		t.Error("minimum should be omitted when nil")
	}
}

func TestSecuritySchemeJSON(t *testing.T) {
	scheme := &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}
	data, _ := json.Marshal(scheme)
	var m map[string]any
	json.Unmarshal(data, &m)

	if m["type"] != "http" {
		t.Errorf("type = %v", m["type"])
	}
	if m["scheme"] != "bearer" {
		t.Errorf("scheme = %v", m["scheme"])
	}
	// apiKey 相关字段不应出现
	if _, ok := m["in"]; ok {
		t.Error("in should be omitted for http scheme")
	}
	if _, ok := m["name"]; ok {
		t.Error("name should be omitted for http scheme")
	}
}

// ============ Phase 2: Schema Builder Tests ============

func TestSchemaPrimitiveTypes(t *testing.T) {
	b := NewSchemaBuilder()

	tests := []struct {
		typ        reflect.Type
		wantType   string
		wantFormat string
	}{
		{reflect.TypeOf(""), "string", ""},
		{reflect.TypeOf(true), "boolean", ""},
		{reflect.TypeOf(0), "integer", "int32"},
		{reflect.TypeOf(int32(0)), "integer", "int32"},
		{reflect.TypeOf(int64(0)), "integer", "int64"},
		{reflect.TypeOf(uint(0)), "integer", "int32"},
		{reflect.TypeOf(uint64(0)), "integer", "int64"},
		{reflect.TypeOf(float32(0)), "number", "float"},
		{reflect.TypeOf(float64(0)), "number", "double"},
		{reflect.TypeOf(time.Time{}), "string", "date-time"},
	}

	for _, tt := range tests {
		s := b.Build(tt.typ)
		if s.Type != tt.wantType {
			t.Errorf("Build(%v).Type = %q, want %q", tt.typ, s.Type, tt.wantType)
		}
		if s.Format != tt.wantFormat {
			t.Errorf("Build(%v).Format = %q, want %q", tt.typ, s.Format, tt.wantFormat)
		}
	}
}

func TestSchemaPointerNullable(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf((*string)(nil)))
	if s.Type != "string" {
		t.Errorf("Type = %q, want string", s.Type)
	}
	if !s.Nullable {
		t.Error("pointer type should be nullable")
	}
}

func TestSchemaSlice(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf([]int{}))
	if s.Type != "array" {
		t.Errorf("Type = %q, want array", s.Type)
	}
	if s.Items == nil || s.Items.Type != "integer" {
		t.Error("Items should be integer schema")
	}
}

func TestSchemaByteSlice(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf([]byte{}))
	if s.Type != "string" || s.Format != "byte" {
		t.Errorf("[]byte should be string/byte, got %s/%s", s.Type, s.Format)
	}
}

func TestSchemaMap(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(map[string]int{}))
	if s.Type != "object" {
		t.Errorf("Type = %q, want object", s.Type)
	}
	if s.AdditionalProperties == nil || s.AdditionalProperties.Type != "integer" {
		t.Error("AdditionalProperties should be integer schema")
	}
}

func TestSchemaMapStringAny(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(map[string]any{}))
	if s.Type != "object" {
		t.Errorf("Type = %q, want object", s.Type)
	}
	if s.AdditionalProperties != nil {
		t.Error("map[string]any should not have typed additionalProperties")
	}
}

type schemaTestUser struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age" binding:"min=0,max=150"`
}

func TestSchemaStruct(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(schemaTestUser{}))

	if s.Type != "object" {
		t.Fatalf("Type = %q, want object", s.Type)
	}
	if len(s.Properties) != 3 {
		t.Fatalf("Properties count = %d, want 3", len(s.Properties))
	}
	if s.Properties["name"] == nil || s.Properties["name"].Type != "string" {
		t.Error("name should be string")
	}
	if s.Properties["email"] == nil || s.Properties["email"].Format != "email" {
		t.Error("email should have format=email")
	}
	if s.Properties["age"] == nil || s.Properties["age"].Minimum == nil || *s.Properties["age"].Minimum != 0 {
		t.Error("age should have minimum=0")
	}
	if s.Properties["age"].Maximum == nil || *s.Properties["age"].Maximum != 150 {
		t.Error("age should have maximum=150")
	}

	// required 字段
	requiredMap := make(map[string]bool)
	for _, r := range s.Required {
		requiredMap[r] = true
	}
	if !requiredMap["name"] || !requiredMap["email"] {
		t.Errorf("Required = %v, want name and email", s.Required)
	}
	if requiredMap["age"] {
		t.Error("age should not be required")
	}
}

type schemaTestBase struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
}

type schemaTestPost struct {
	schemaTestBase
	Title string `json:"title"`
}

func TestSchemaEmbeddedStruct(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(schemaTestPost{}))

	// 嵌入 struct 的字段应展开到父级
	if s.Properties["id"] == nil {
		t.Error("embedded field 'id' should be present")
	}
	if s.Properties["created_at"] == nil {
		t.Error("embedded field 'created_at' should be present")
	}
	if s.Properties["title"] == nil {
		t.Error("field 'title' should be present")
	}
	if len(s.Properties) != 3 {
		t.Errorf("Properties count = %d, want 3", len(s.Properties))
	}
}

type schemaTestNode struct {
	Value    string          `json:"value"`
	Children []*schemaTestNode `json:"children"`
}

func TestSchemaCircularReference(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(schemaTestNode{}))

	if s.Type != "object" {
		t.Fatalf("Type = %q, want object", s.Type)
	}
	// children 应该是 array，items 应该是 $ref（循环引用）
	children := s.Properties["children"]
	if children == nil || children.Type != "array" {
		t.Fatal("children should be array")
	}
	// items 是 *schemaTestNode，nullable + 内部是 $ref
	items := children.Items
	if items == nil {
		t.Fatal("children.Items should not be nil")
	}
	if items.Ref != "#/components/schemas/schemaTestNode" {
		t.Errorf("circular ref = %q, want $ref to schemaTestNode", items.Ref)
	}
}

func TestSchemaFileUpload(t *testing.T) {
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(multipart.FileHeader{}))
	if s.Type != "string" || s.Format != "binary" {
		t.Errorf("FileHeader should be string/binary, got %s/%s", s.Type, s.Format)
	}
}

type schemaTestUpload struct {
	File  *multipart.FileHeader   `form:"file"`
	Files []*multipart.FileHeader `form:"files"`
}

func TestHasFileUpload(t *testing.T) {
	if !HasFileUpload(reflect.TypeOf(schemaTestUpload{})) {
		t.Error("schemaTestUpload should have file upload")
	}
	if HasFileUpload(reflect.TypeOf(schemaTestUser{})) {
		t.Error("schemaTestUser should not have file upload")
	}
}

func TestSchemaJSONSkipField(t *testing.T) {
	type hidden struct {
		Public  string `json:"public"`
		Ignored string `json:"-"`
		private string //nolint:unused
	}
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(hidden{}))
	if _, ok := s.Properties["public"]; !ok {
		t.Error("public field should be present")
	}
	if _, ok := s.Properties["-"]; ok {
		t.Error("json:\"-\" field should be skipped")
	}
	if _, ok := s.Properties["private"]; ok {
		t.Error("unexported field should be skipped")
	}
}

func TestSchemaDescAndExample(t *testing.T) {
	type annotated struct {
		Name string `json:"name" desc:"用户名" example:"张三"`
	}
	b := NewSchemaBuilder()
	s := b.Build(reflect.TypeOf(annotated{}))
	name := s.Properties["name"]
	if name.Description != "用户名" {
		t.Errorf("Description = %q, want 用户名", name.Description)
	}
	if name.Example != "张三" {
		t.Errorf("Example = %v, want 张三", name.Example)
	}
}

func TestSchemaComponentsCollected(t *testing.T) {
	b := NewSchemaBuilder()
	b.Build(reflect.TypeOf(schemaTestUser{}))
	if _, ok := b.Schemas["schemaTestUser"]; !ok {
		t.Error("named struct should be collected in Schemas")
	}
}

// ============ Phase 2: Binding Tests ============

func TestParseBindingTagRequired(t *testing.T) {
	c := ParseBindingTag("required")
	if !c.Required {
		t.Error("should be required")
	}
}

func TestParseBindingTagMinMax(t *testing.T) {
	c := ParseBindingTag("min=1,max=100")
	if c.Minimum == nil || *c.Minimum != 1 {
		t.Errorf("Minimum = %v, want 1", c.Minimum)
	}
	if c.Maximum == nil || *c.Maximum != 100 {
		t.Errorf("Maximum = %v, want 100", c.Maximum)
	}
}

func TestParseBindingTagLen(t *testing.T) {
	c := ParseBindingTag("len=10")
	if c.MinLength == nil || *c.MinLength != 10 {
		t.Errorf("MinLength = %v, want 10", c.MinLength)
	}
	if c.MaxLength == nil || *c.MaxLength != 10 {
		t.Errorf("MaxLength = %v, want 10", c.MaxLength)
	}
}

func TestParseBindingTagFormats(t *testing.T) {
	tests := []struct {
		tag    string
		format string
	}{
		{"email", "email"},
		{"url", "uri"},
		{"uuid", "uuid"},
	}
	for _, tt := range tests {
		c := ParseBindingTag(tt.tag)
		if c.Format != tt.format {
			t.Errorf("ParseBindingTag(%q).Format = %q, want %q", tt.tag, c.Format, tt.format)
		}
	}
}

func TestParseBindingTagOneof(t *testing.T) {
	c := ParseBindingTag("oneof=active inactive banned")
	if len(c.Enum) != 3 {
		t.Fatalf("Enum count = %d, want 3", len(c.Enum))
	}
	if c.Enum[0] != "active" || c.Enum[1] != "inactive" || c.Enum[2] != "banned" {
		t.Errorf("Enum = %v", c.Enum)
	}
}

func TestParseBindingTagCombined(t *testing.T) {
	c := ParseBindingTag("required,min=0,max=999,email")
	if !c.Required {
		t.Error("should be required")
	}
	if c.Minimum == nil || *c.Minimum != 0 {
		t.Error("Minimum should be 0")
	}
	if c.Maximum == nil || *c.Maximum != 999 {
		t.Error("Maximum should be 999")
	}
	if c.Format != "email" {
		t.Errorf("Format = %q, want email", c.Format)
	}
}

func TestParseBindingTagEmpty(t *testing.T) {
	c := ParseBindingTag("")
	if c.Required || c.Minimum != nil || c.Maximum != nil || c.Format != "" {
		t.Error("empty tag should produce zero constraints")
	}
	c2 := ParseBindingTag("-")
	if c2.Required {
		t.Error("dash tag should produce zero constraints")
	}
}

func TestApplyToSchema(t *testing.T) {
	min := 1.0
	c := BindingConstraints{
		Minimum: &min,
		Format:  "email",
		Enum:    []any{"a", "b"},
	}
	s := &Schema{Type: "string"}
	c.ApplyToSchema(s)

	if s.Minimum == nil || *s.Minimum != 1 {
		t.Error("Minimum should be applied")
	}
	if s.Format != "email" {
		t.Error("Format should be applied")
	}
	if len(s.Enum) != 2 {
		t.Error("Enum should be applied")
	}
}

func TestApplyToSchemaNoOverrideFormat(t *testing.T) {
	c := BindingConstraints{Format: "email"}
	s := &Schema{Type: "string", Format: "date-time"}
	c.ApplyToSchema(s)
	if s.Format != "date-time" {
		t.Error("existing format should not be overridden")
	}
}

func TestInferFieldLocation(t *testing.T) {
	tests := []struct {
		method    string
		hasURI    bool
		hasHeader bool
		want      FieldLocation
	}{
		{"GET", false, false, LocQuery},
		{"GET", true, false, LocPath},
		{"DELETE", false, false, LocQuery},
		{"POST", false, false, LocBody},
		{"POST", true, false, LocPath},
		{"PUT", false, false, LocBody},
		{"PATCH", false, false, LocBody},
		{"GET", false, true, LocHeader},
		{"POST", false, true, LocHeader},
	}
	for _, tt := range tests {
		got := InferFieldLocation(tt.method, tt.hasURI, tt.hasHeader)
		if got != tt.want {
			t.Errorf("InferFieldLocation(%q, uri=%v, header=%v) = %d, want %d",
				tt.method, tt.hasURI, tt.hasHeader, got, tt.want)
		}
	}
}

// ============ Phase 3: Tags Tests ============

func TestDeriveTag(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users", "users"},
		{"/api/v1/users/:id/posts", "users"},
		{"/admin", "admin"},
		{"/", ""},
		{"/api/v2", ""},
		{"/api/v1/:id", ""},
		{"/orders/items", "orders"},
	}
	for _, tt := range tests {
		got := DeriveTag(tt.path)
		if got != tt.want {
			t.Errorf("DeriveTag(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// ============ Phase 3: Response Tests ============

func TestWrapResponseSchema(t *testing.T) {
	data := &Schema{Type: "object"}
	s := WrapResponseSchema(data)

	if s.Type != "object" {
		t.Errorf("Type = %q, want object", s.Type)
	}
	if s.Properties["code"] == nil || s.Properties["code"].Type != "integer" {
		t.Error("code property should be integer")
	}
	if s.Properties["data"] != data {
		t.Error("data property should be the input schema")
	}
	if s.Properties["message"] == nil || s.Properties["message"].Type != "string" {
		t.Error("message property should be string")
	}
	if s.Properties["trace_id"] == nil {
		t.Error("trace_id property should exist")
	}
}

func TestWrapResponseSchemaNilInput(t *testing.T) {
	s := WrapResponseSchema(nil)
	if s.Properties["data"] == nil {
		t.Error("data should not be nil even with nil input")
	}
}

func TestNullDataResponseSchema(t *testing.T) {
	s := NullDataResponseSchema()
	data := s.Properties["data"]
	if data == nil || !data.Nullable {
		t.Error("data should be nullable")
	}
}

func TestBuildErrorResponse(t *testing.T) {
	resp := BuildErrorResponse()
	if resp.Description != "业务错误" {
		t.Errorf("Description = %q", resp.Description)
	}
	ct, ok := resp.Content["application/json"]
	if !ok {
		t.Fatal("should have application/json content")
	}
	if ct.Schema == nil || ct.Schema.Type != "object" {
		t.Error("schema should be object")
	}
}

func TestDefaultErrorResponses(t *testing.T) {
	resps := DefaultErrorResponses()
	for _, code := range []string{"400", "401", "500"} {
		r, ok := resps[code]
		if !ok {
			t.Errorf("missing %s response", code)
			continue
		}
		if r.Ref != ErrorResponseRef {
			t.Errorf("%s ref = %q, want %q", code, r.Ref, ErrorResponseRef)
		}
	}
}

// ============ Phase 3: Registry Tests ============

type registryTestCreateReq struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type registryTestCreateResp struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type registryTestListReq struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
}

type registryTestListResp struct {
	Items []registryTestCreateResp `json:"items"`
	Total int                      `json:"total"`
}

type registryTestDeleteReq struct {
	ID int `uri:"id" binding:"required"`
}

func TestRegistryBuildBasic(t *testing.T) {
	cfg := &Config{
		Title:   "Test API",
		Version: "1.0.0",
	}
	reg := NewRegistry(cfg)

	reg.Add(RouteEntry{
		Method:   "POST",
		Path:     "/users",
		BasePath: "/",
		Type:     RouteTypeFull,
		ReqType:  reflect.TypeOf(registryTestCreateReq{}),
		RespType: reflect.TypeOf(registryTestCreateResp{}),
		Doc:      Doc(Summary("创建用户"), Tags("Users")),
	})

	doc := reg.Build()

	if doc.OpenAPI != "3.0.3" {
		t.Errorf("OpenAPI = %q", doc.OpenAPI)
	}
	if doc.Info.Title != "Test API" {
		t.Errorf("Title = %q", doc.Info.Title)
	}

	pathItem, ok := doc.Paths["/users"]
	if !ok {
		t.Fatal("path /users should exist")
	}
	if pathItem.Post == nil {
		t.Fatal("POST operation should exist")
	}
	if pathItem.Post.Summary != "创建用户" {
		t.Errorf("Summary = %q", pathItem.Post.Summary)
	}
	if len(pathItem.Post.Tags) != 1 || pathItem.Post.Tags[0] != "Users" {
		t.Errorf("Tags = %v", pathItem.Post.Tags)
	}

	// 应有 requestBody
	if pathItem.Post.RequestBody == nil {
		t.Fatal("RequestBody should exist")
	}
	ct, ok := pathItem.Post.RequestBody.Content["application/json"]
	if !ok {
		t.Fatal("RequestBody should have application/json")
	}
	if ct.Schema == nil || ct.Schema.Properties["name"] == nil {
		t.Error("RequestBody schema should have name property")
	}

	// 应有 200 响应
	resp200, ok := pathItem.Post.Responses["200"]
	if !ok {
		t.Fatal("200 response should exist")
	}
	if resp200.Description != "成功" {
		t.Errorf("200 description = %q", resp200.Description)
	}

	// 应有错误响应
	for _, code := range []string{"400", "401", "500"} {
		if _, ok := pathItem.Post.Responses[code]; !ok {
			t.Errorf("missing %s error response", code)
		}
	}
}

func TestRegistryBuildGETWithQueryParams(t *testing.T) {
	reg := NewRegistry(&Config{Title: "Test"})

	reg.Add(RouteEntry{
		Method:   "GET",
		Path:     "/users",
		BasePath: "/",
		Type:     RouteTypeFull,
		ReqType:  reflect.TypeOf(registryTestListReq{}),
		RespType: reflect.TypeOf(registryTestListResp{}),
		Doc:      Doc(Summary("用户列表")),
	})

	doc := reg.Build()
	op := doc.Paths["/users"].Get
	if op == nil {
		t.Fatal("GET operation should exist")
	}

	// GET 请求的 form 字段应变为 query parameters
	if len(op.Parameters) == 0 {
		t.Fatal("should have query parameters")
	}

	paramMap := make(map[string]Parameter)
	for _, p := range op.Parameters {
		paramMap[p.Name] = p
	}

	page, ok := paramMap["page"]
	if !ok {
		t.Fatal("page parameter should exist")
	}
	if page.In != "query" {
		t.Errorf("page.In = %q, want query", page.In)
	}

	pageSize, ok := paramMap["page_size"]
	if !ok {
		t.Fatal("page_size parameter should exist")
	}
	if pageSize.In != "query" {
		t.Errorf("page_size.In = %q, want query", pageSize.In)
	}
}

func TestRegistryBuildDELETEWithURIParam(t *testing.T) {
	reg := NewRegistry(&Config{Title: "Test"})

	reg.Add(RouteEntry{
		Method:   "DELETE",
		Path:     "/users/:id",
		BasePath: "/",
		Type:     RouteTypeRequestOnly,
		ReqType:  reflect.TypeOf(registryTestDeleteReq{}),
		Doc:      Doc(Summary("删除用户")),
	})

	doc := reg.Build()
	op := doc.Paths["/users/{id}"].Delete
	if op == nil {
		t.Fatal("DELETE operation should exist")
	}

	// uri 字段应变为 path parameter
	if len(op.Parameters) != 1 {
		t.Fatalf("Parameters count = %d, want 1", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.Name != "id" || p.In != "path" || !p.Required {
		t.Errorf("Parameter = %+v, want id/path/required", p)
	}

	// Handle0 的 200 响应 data 应为 nullable
	resp200 := op.Responses["200"]
	if resp200 == nil {
		t.Fatal("200 response should exist")
	}
}

func TestRegistrySecurityInheritance(t *testing.T) {
	reg := NewRegistry(&Config{
		Title: "Test",
		SecuritySchemes: map[string]SecurityScheme{
			"BearerAuth": {Type: "http", Scheme: "bearer"},
		},
	})

	// 继承组级 security
	reg.Add(RouteEntry{
		Method:          "GET",
		Path:            "/api/v1/profile",
		BasePath:        "/api/v1",
		Type:            RouteTypeResponseOnly,
		RespType:        reflect.TypeOf(registryTestCreateResp{}),
		Doc:             Doc(Summary("个人信息")),
		DefaultSecurity: []string{"BearerAuth"},
	})

	// NoSecurity 覆盖
	reg.Add(RouteEntry{
		Method:          "POST",
		Path:            "/api/v1/login",
		BasePath:        "/api/v1",
		Type:            RouteTypeFull,
		ReqType:         reflect.TypeOf(registryTestCreateReq{}),
		RespType:        reflect.TypeOf(registryTestCreateResp{}),
		Doc:             Doc(Summary("登录"), NoSecurity()),
		DefaultSecurity: []string{"BearerAuth"},
	})

	// 路由级显式指定
	reg.Add(RouteEntry{
		Method:   "PUT",
		Path:     "/api/v1/admin",
		BasePath: "/api/v1",
		Type:     RouteTypeFull,
		ReqType:  reflect.TypeOf(registryTestCreateReq{}),
		RespType: reflect.TypeOf(registryTestCreateResp{}),
		Doc:      Doc(Summary("管理"), Security("ApiKeyAuth")),
	})

	doc := reg.Build()

	// 继承 BearerAuth
	profileOp := doc.Paths["/api/v1/profile"].Get
	if len(profileOp.Security) != 1 {
		t.Fatalf("profile security count = %d, want 1", len(profileOp.Security))
	}
	if _, ok := profileOp.Security[0]["BearerAuth"]; !ok {
		t.Error("profile should inherit BearerAuth")
	}

	// NoSecurity → 空 security 数组
	loginOp := doc.Paths["/api/v1/login"].Post
	if len(loginOp.Security) != 1 {
		t.Fatalf("login security count = %d, want 1 (empty object)", len(loginOp.Security))
	}
	if len(loginOp.Security[0]) != 0 {
		t.Error("login security should be empty object (no auth)")
	}

	// 显式指定 ApiKeyAuth
	adminOp := doc.Paths["/api/v1/admin"].Put
	if len(adminOp.Security) != 1 {
		t.Fatalf("admin security count = %d, want 1", len(adminOp.Security))
	}
	if _, ok := adminOp.Security[0]["ApiKeyAuth"]; !ok {
		t.Error("admin should use ApiKeyAuth")
	}

	// components 应包含 SecuritySchemes
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		t.Fatal("Components.SecuritySchemes should exist")
	}
	if _, ok := doc.Components.SecuritySchemes["BearerAuth"]; !ok {
		t.Error("BearerAuth scheme should be in components")
	}
}

func TestRegistryTagInheritance(t *testing.T) {
	reg := NewRegistry(&Config{Title: "Test"})

	// 显式 tag
	reg.Add(RouteEntry{
		Method:   "GET",
		Path:     "/users",
		BasePath: "/",
		Type:     RouteTypeResponseOnly,
		RespType: reflect.TypeOf(registryTestCreateResp{}),
		Doc:      Doc(Tags("CustomTag")),
	})

	// 默认 tag
	reg.Add(RouteEntry{
		Method:         "GET",
		Path:           "/api/v1/orders",
		BasePath:       "/api/v1/orders",
		Type:           RouteTypeResponseOnly,
		RespType:       reflect.TypeOf(registryTestCreateResp{}),
		DefaultTag:     "Orders",
		DefaultTagDesc: "订单管理",
	})

	// 自动推导 tag
	reg.Add(RouteEntry{
		Method:   "GET",
		Path:     "/api/v1/products",
		BasePath: "/api/v1/products",
		Type:     RouteTypeResponseOnly,
		RespType: reflect.TypeOf(registryTestCreateResp{}),
	})

	doc := reg.Build()

	op1 := doc.Paths["/users"].Get
	if len(op1.Tags) != 1 || op1.Tags[0] != "CustomTag" {
		t.Errorf("explicit tag = %v, want [CustomTag]", op1.Tags)
	}

	op2 := doc.Paths["/api/v1/orders"].Get
	if len(op2.Tags) != 1 || op2.Tags[0] != "Orders" {
		t.Errorf("default tag = %v, want [Orders]", op2.Tags)
	}

	op3 := doc.Paths["/api/v1/products"].Get
	if len(op3.Tags) != 1 || op3.Tags[0] != "products" {
		t.Errorf("derived tag = %v, want [products]", op3.Tags)
	}

	// Tags 列表应包含所有 tag 且按字母排序
	if len(doc.Tags) < 3 {
		t.Fatalf("Tags count = %d, want >= 3", len(doc.Tags))
	}
	// 验证 Orders tag 有描述
	for _, tag := range doc.Tags {
		if tag.Name == "Orders" && tag.Description != "订单管理" {
			t.Errorf("Orders tag description = %q, want 订单管理", tag.Description)
		}
	}
}

func TestRegistryBuildSpecJSON(t *testing.T) {
	reg := NewRegistry(&Config{
		Title:       "My API",
		Version:     "2.0.0",
		Description: "测试 API",
		Servers:     []Server{{URL: "https://api.example.com"}},
	})

	reg.Add(RouteEntry{
		Method:   "GET",
		Path:     "/ping",
		BasePath: "/",
		Type:     RouteTypeResponseOnly,
		RespType: reflect.TypeOf(""),
		Doc:      Doc(Summary("健康检查")),
	})

	doc := reg.Build()

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v", parsed["openapi"])
	}

	info, ok := parsed["info"].(map[string]any)
	if !ok {
		t.Fatal("info should be a map")
	}
	if info["title"] != "My API" {
		t.Errorf("title = %v", info["title"])
	}
	if info["version"] != "2.0.0" {
		t.Errorf("version = %v", info["version"])
	}

	servers, ok := parsed["servers"].([]any)
	if !ok || len(servers) != 1 {
		t.Fatal("servers should have 1 entry")
	}
}
