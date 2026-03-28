package openapi

import "testing"

type sharedType struct {
	UserID int `json:"user_id" form:"userId" binding:"gte=1" desc:"user id"`
}

type bodyWithShared struct {
	Shared sharedType `json:"shared"`
}

type queryWithShared struct {
	Shared sharedType `form:"shared"`
}

type ptrWrapper struct {
	Shared *sharedType `json:"shared" desc:"wrapped shared"`
}

type stringConstraintReq struct {
	Name string `json:"name" binding:"gt=2,lt=10"`
	Tags []int  `json:"tags" binding:"gte=1,lte=5"`
}

func TestBuildSeparatesComponentsByMode(t *testing.T) {
	m := New()
	err := m.AddOperation(Operation{
		Method: "POST",
		Path:   "/users",
		Request: &Request{
			Body:        bodyWithShared{},
			QueryParams: queryWithShared{},
		},
		Responses: []Response{{Status: 200, Body: bodyWithShared{}}},
	})
	if err != nil {
		t.Fatalf("add operation: %v", err)
	}

	doc, err := m.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	bodySchema := doc.Components.Schemas["github.com.tokmz.qi.internal.openapi.sharedType.Body"]
	if bodySchema == nil {
		t.Fatalf("missing body schema")
	}
	if _, ok := bodySchema.Properties["user_id"]; !ok {
		t.Fatalf("body schema should use json tag")
	}

	querySchema := doc.Components.Schemas["github.com.tokmz.qi.internal.openapi.sharedType.Query"]
	if querySchema == nil {
		t.Fatalf("missing query schema")
	}
	if _, ok := querySchema.Properties["userId"]; !ok {
		t.Fatalf("query schema should use form tag")
	}

	respSchema := doc.Components.Schemas["github.com.tokmz.qi.internal.openapi.sharedType.Body"]
	if respSchema == nil {
		t.Fatalf("missing response schema")
	}
	if _, ok := respSchema.Properties["user_id"]; !ok {
		t.Fatalf("body schema should keep json tag mapping")
	}
}

func TestBuildWrapsRefMetadataWithAllOf(t *testing.T) {
	m := New()
	err := m.AddOperation(Operation{
		Method: "POST",
		Path:   "/wrappers",
		Request: &Request{
			Body: ptrWrapper{},
		},
		Responses: []Response{{Status: 200, Body: ptrWrapper{}}},
	})
	if err != nil {
		t.Fatalf("add operation: %v", err)
	}

	doc, err := m.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	wrapper := doc.Components.Schemas["github.com.tokmz.qi.internal.openapi.ptrWrapper.Body"]
	if wrapper == nil {
		t.Fatalf("missing wrapper schema")
	}
	field := wrapper.Properties["shared"]
	if field == nil {
		t.Fatalf("missing shared field")
	}
	if field.Ref != "" {
		t.Fatalf("ref with siblings should be wrapped by allOf")
	}
	if len(field.AllOf) != 2 {
		t.Fatalf("expected allOf wrapper, got %#v", field)
	}
	if field.AllOf[0].Ref == "" {
		t.Fatalf("first allOf item should be ref")
	}
	if !field.AllOf[1].Nullable {
		t.Fatalf("nullable metadata should be preserved in second allOf item")
	}
	if field.AllOf[1].Description != "wrapped shared" {
		t.Fatalf("description metadata should be preserved in second allOf item")
	}
}

func TestConstraintMappingForStringAndSlice(t *testing.T) {
	a := NewAnalyzer(AnalyzeOptions{})
	schema, err := a.AnalyzeBody(stringConstraintReq{})
	if err != nil {
		t.Fatalf("analyze body: %v", err)
	}
	if schema.Ref == "" {
		t.Fatalf("expected named type body schema to be a ref")
	}

	components := a.Components()
	schema = components["github.com.tokmz.qi.internal.openapi.stringConstraintReq.Body"]
	if schema == nil {
		t.Fatalf("missing body component schema")
	}

	name := schema.Properties["name"]
	if name == nil {
		t.Fatalf("missing name property")
	}
	if name.Constraints.MinLength == nil || *name.Constraints.MinLength != 3 {
		t.Fatalf("expected minLength 3, got %#v", name.Constraints.MinLength)
	}
	if name.Constraints.MaxLength == nil || *name.Constraints.MaxLength != 9 {
		t.Fatalf("expected maxLength 9, got %#v", name.Constraints.MaxLength)
	}
	if name.Constraints.Minimum != nil || name.Constraints.Maximum != nil {
		t.Fatalf("string constraints should not map to numeric bounds")
	}

	tags := schema.Properties["tags"]
	if tags == nil {
		t.Fatalf("missing tags property")
	}
	if tags.Constraints.MinItems == nil || *tags.Constraints.MinItems != 1 {
		t.Fatalf("expected minItems 1, got %#v", tags.Constraints.MinItems)
	}
	if tags.Constraints.MaxItems == nil || *tags.Constraints.MaxItems != 5 {
		t.Fatalf("expected maxItems 5, got %#v", tags.Constraints.MaxItems)
	}
}

func TestRegistryRejectsDuplicateOperation(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(Operation{Method: "get", Path: "/users"}); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if err := r.Add(Operation{Method: "GET", Path: "/users"}); err == nil {
		t.Fatalf("expected duplicate operation error")
	}
}

func TestRegistryRejectsDuplicateOperationID(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(Operation{Method: "GET", Path: "/users", OperationID: "ListUsers"}); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if err := r.Add(Operation{Method: "POST", Path: "/users", OperationID: "ListUsers"}); err == nil {
		t.Fatalf("expected duplicate operationId error")
	}
}

func TestQueryParametersFallbackToFieldName(t *testing.T) {
	type req struct {
		Page int
	}

	a := NewAnalyzer(AnalyzeOptions{})
	params, err := a.AnalyzeParameters(req{}, ParamInQuery)
	if err != nil {
		t.Fatalf("analyze params: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("expected one param, got %d", len(params))
	}
	if params[0].Name != "Page" {
		t.Fatalf("expected fallback field name, got %q", params[0].Name)
	}
}

func TestQueryParametersUseConfigurableFieldNamer(t *testing.T) {
	type req struct {
		PageNo   int
		HTTPCode int
	}

	a := NewAnalyzer(AnalyzeOptions{
		FieldNamer: SnakeCaseFieldNamer{},
	})
	params, err := a.AnalyzeParameters(req{}, ParamInQuery)
	if err != nil {
		t.Fatalf("analyze params: %v", err)
	}
	if len(params) != 2 {
		t.Fatalf("expected two params, got %d", len(params))
	}
	if params[0].Name != "http_code" || params[1].Name != "page_no" {
		t.Fatalf("unexpected param names: %#v", params)
	}
}

func TestExplicitTagOverridesFieldNamer(t *testing.T) {
	type req struct {
		PageNo int `form:"page"`
	}

	a := NewAnalyzer(AnalyzeOptions{
		FieldNamer: SnakeCaseFieldNamer{},
	})
	params, err := a.AnalyzeParameters(req{}, ParamInQuery)
	if err != nil {
		t.Fatalf("analyze params: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("expected one param, got %d", len(params))
	}
	if params[0].Name != "page" {
		t.Fatalf("explicit tag should override field namer, got %q", params[0].Name)
	}
}

func TestRecommendedFieldNamerByMode(t *testing.T) {
	type req struct {
		RequestID int
		HTTPCode  int
	}

	a := NewAnalyzer(AnalyzeOptions{
		FieldNamer: RecommendedFieldNamer{},
	})

	queryParams, err := a.AnalyzeParameters(req{}, ParamInQuery)
	if err != nil {
		t.Fatalf("analyze query params: %v", err)
	}
	if len(queryParams) != 2 {
		t.Fatalf("expected two query params, got %d", len(queryParams))
	}
	if queryParams[0].Name != "http_code" || queryParams[1].Name != "request_id" {
		t.Fatalf("unexpected query param names: %#v", queryParams)
	}

	headerParams, err := a.AnalyzeParameters(req{}, ParamInHeader)
	if err != nil {
		t.Fatalf("analyze header params: %v", err)
	}
	if len(headerParams) != 2 {
		t.Fatalf("expected two header params, got %d", len(headerParams))
	}
	if headerParams[0].Name != "http-code" || headerParams[1].Name != "request-id" {
		t.Fatalf("unexpected header param names: %#v", headerParams)
	}

	bodySchema, err := a.AnalyzeBody(req{})
	if err != nil {
		t.Fatalf("analyze body: %v", err)
	}
	if bodySchema.Ref == "" {
		t.Fatalf("expected named body schema to be a ref")
	}
	component := a.Components()["github.com.tokmz.qi.internal.openapi.req.Body"]
	if component == nil {
		t.Fatalf("missing body component")
	}
	if _, ok := component.Properties["RequestID"]; !ok {
		t.Fatalf("recommended body fallback should keep Go field name")
	}
}

func TestWithRecommendedDefaultsSetsRecommendedFieldNamer(t *testing.T) {
	cfg := defaultOptions()
	WithRecommendedDefaults()(&cfg)

	if _, ok := cfg.FieldNamer.(RecommendedFieldNamer); !ok {
		t.Fatalf("expected recommended field namer, got %T", cfg.FieldNamer)
	}
}

func TestMustAddOperationPanicsOnDuplicate(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()

	m := New()
	m.MustAddOperation(Operation{Method: "GET", Path: "/users"})
	m.MustAddOperation(Operation{Method: "GET", Path: "/users"})
}

func TestCloneWithOptionsDoesNotMutateOriginal(t *testing.T) {
	m := New(
		WithTitle("Origin"),
		WithVersion("1.0.0"),
	)
	if err := m.AddOperation(Operation{Method: "GET", Path: "/users"}); err != nil {
		t.Fatalf("add operation: %v", err)
	}

	clone := m.CloneWithOptions(
		WithTitle("Clone"),
		WithVersion("2.0.0"),
		WithRecommendedDefaults(),
	)

	origDoc, err := m.Build()
	if err != nil {
		t.Fatalf("build original: %v", err)
	}
	cloneDoc, err := clone.Build()
	if err != nil {
		t.Fatalf("build clone: %v", err)
	}

	if origDoc.Info.Title != "Origin" || origDoc.Info.Version != "1.0.0" {
		t.Fatalf("original manager should remain unchanged: %#v", origDoc.Info)
	}
	if cloneDoc.Info.Title != "Clone" || cloneDoc.Info.Version != "2.0.0" {
		t.Fatalf("clone options not applied: %#v", cloneDoc.Info)
	}
	if _, ok := clone.opts.FieldNamer.(RecommendedFieldNamer); !ok {
		t.Fatalf("clone should apply recommended field namer, got %T", clone.opts.FieldNamer)
	}
	if _, ok := m.opts.FieldNamer.(KeepCaseFieldNamer); !ok {
		t.Fatalf("original should keep original field namer, got %T", m.opts.FieldNamer)
	}
}

func TestBuildWithInfoOverridesInfoOnClone(t *testing.T) {
	m := New(
		WithTitle("Origin"),
		WithVersion("1.0.0"),
	)
	if err := m.AddOperation(Operation{Method: "GET", Path: "/users"}); err != nil {
		t.Fatalf("add operation: %v", err)
	}

	doc, err := m.BuildWithInfo("Preview", "9.9.9", WithDescription("preview doc"))
	if err != nil {
		t.Fatalf("build with info: %v", err)
	}
	if doc.Info.Title != "Preview" || doc.Info.Version != "9.9.9" || doc.Info.Description != "preview doc" {
		t.Fatalf("unexpected override info: %#v", doc.Info)
	}

	origDoc, err := m.Build()
	if err != nil {
		t.Fatalf("build original: %v", err)
	}
	if origDoc.Info.Title != "Origin" || origDoc.Info.Version != "1.0.0" {
		t.Fatalf("original manager info should not be mutated: %#v", origDoc.Info)
	}
}

func TestUriFieldsExcludedFromQueryParams(t *testing.T) {
	type detailReq struct {
		ID   int64  `uri:"id" binding:"required" desc:"用户ID"`
		Name string `form:"name" desc:"用户名"`
	}

	m := New()
	err := m.AddOperation(Operation{
		Method: "GET",
		Path:   "/users/{id}",
		Request: &Request{
			QueryParams: detailReq{},
			PathParams:  detailReq{},
		},
	})
	if err != nil {
		t.Fatalf("add operation: %v", err)
	}

	doc, err := m.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	pathItem := doc.Paths["/users/{id}"]
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("missing GET /users/{id}")
	}

	op := pathItem.Get
	// 路径参数应包含 id，且带 desc
	var foundPathID bool
	for _, p := range op.Parameters {
		if p.In == "path" && p.Name == "id" {
			foundPathID = true
			if p.Description != "用户ID" {
				t.Fatalf("path param id should have desc '用户ID', got %q", p.Description)
			}
		}
	}
	if !foundPathID {
		t.Fatal("path param 'id' not found")
	}

	// query 参数不应包含 id（uri 字段），应只有 name
	for _, p := range op.Parameters {
		if p.In == "query" && p.Name == "id" {
			t.Fatal("uri field 'id' should not appear as query param")
		}
	}

	var foundQueryName bool
	for _, p := range op.Parameters {
		if p.In == "query" && p.Name == "name" {
			foundQueryName = true
			if p.Description != "用户名" {
				t.Fatalf("query param name should have desc '用户名', got %q", p.Description)
			}
		}
	}
	if !foundQueryName {
		t.Fatal("query param 'name' not found")
	}
}
