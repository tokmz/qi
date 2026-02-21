package qi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tokmz/qi/pkg/openapi"
)

// ============ 集成测试类型定义 ============

type itCreateUserReq struct {
	Name  string `json:"name" binding:"required" desc:"用户名"`
	Email string `json:"email" binding:"required,email" desc:"邮箱"`
}

type itUserResp struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type itListReq struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
}

type itListResp struct {
	Items []itUserResp `json:"items"`
	Total int          `json:"total"`
}

type itDeleteReq struct {
	ID int64 `uri:"id" binding:"required"`
}

type itLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type itTokenResp struct {
	Token string `json:"token"`
}

type itProfileResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ============ 集成测试 ============

func TestOpenAPIIntegration(t *testing.T) {
	engine := New(
		WithMode("test"),
		WithOpenAPI(&openapi.Config{
			Title:       "Integration Test API",
			Version:     "2.0.0",
			Description: "集成测试 API",
			Path:        "/openapi.json",
			SecuritySchemes: map[string]openapi.SecurityScheme{
				"BearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		}),
	)

	r := engine.Router()

	// 非泛型路由 + DocRoute
	r.GET("/ping", func(c *Context) { c.Success("pong") })
	r.DocRoute("GET", "/ping", openapi.Doc(
		openapi.Summary("健康检查"),
		openapi.Tags("System"),
	))

	// Full: POST 创建用户
	POST[itCreateUserReq, itUserResp](r, "/users",
		func(c *Context, req *itCreateUserReq) (*itUserResp, error) {
			return &itUserResp{ID: 1, Name: req.Name, Email: req.Email}, nil
		},
		openapi.Doc(openapi.Summary("创建用户"), openapi.Tags("Users")),
	)

	// Full: GET 列表（query params）
	GET[itListReq, itListResp](r, "/users",
		func(c *Context, req *itListReq) (*itListResp, error) {
			return &itListResp{Total: 0}, nil
		},
		openapi.Doc(openapi.Summary("用户列表"), openapi.Tags("Users")),
	)

	// Request-only: DELETE
	DELETE0[itDeleteReq](r, "/users/:id",
		func(c *Context, req *itDeleteReq) error { return nil },
		openapi.Doc(openapi.Summary("删除用户"), openapi.Tags("Users")),
	)

	// 路由组 + Security
	v1 := r.Group("/api/v1")
	v1.SetTag("V1", "V1 版本接口")
	v1.SetSecurity("BearerAuth")

	// NoSecurity 登录
	POST[itLoginReq, itTokenResp](v1, "/login",
		func(c *Context, req *itLoginReq) (*itTokenResp, error) {
			return &itTokenResp{Token: "xxx"}, nil
		},
		openapi.Doc(openapi.Summary("用户登录"), openapi.NoSecurity()),
	)

	// Response-only: 继承 Security
	GETOnly[itProfileResp](v1, "/profile",
		func(c *Context) (*itProfileResp, error) {
			return &itProfileResp{ID: 1, Name: "test"}, nil
		},
		openapi.Doc(openapi.Summary("获取个人信息")),
	)

	// 构建 spec
	engine.buildOpenAPISpec()

	// 请求 /openapi.json
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/openapi.json", nil)
	engine.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// 解析 spec
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("failed to parse spec JSON: %v", err)
	}

	// 验证基本信息
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v", doc["openapi"])
	}
	info := doc["info"].(map[string]any)
	if info["title"] != "Integration Test API" {
		t.Errorf("title = %v", info["title"])
	}
	if info["version"] != "2.0.0" {
		t.Errorf("version = %v", info["version"])
	}

	// 验证 paths 存在
	paths := doc["paths"].(map[string]any)

	// /ping
	assertPath(t, paths, "/ping", "get", "健康检查")

	// /users POST
	assertPath(t, paths, "/users", "post", "创建用户")

	// /users GET
	assertPath(t, paths, "/users", "get", "用户列表")

	// /users/{id} DELETE
	assertPath(t, paths, "/users/{id}", "delete", "删除用户")

	// /api/v1/login POST
	assertPath(t, paths, "/api/v1/login", "post", "用户登录")

	// /api/v1/profile GET
	assertPath(t, paths, "/api/v1/profile", "get", "获取个人信息")

	// 验证 /api/v1/login 有 NoSecurity（空 security 数组）
	loginOp := getOperation(paths, "/api/v1/login", "post")
	if loginOp != nil {
		sec, ok := loginOp["security"].([]any)
		if !ok || len(sec) == 0 {
			t.Error("/api/v1/login should have security (empty object for NoSecurity)")
		}
	}

	// 验证 /api/v1/profile 继承 BearerAuth
	profileOp := getOperation(paths, "/api/v1/profile", "get")
	if profileOp != nil {
		sec, ok := profileOp["security"].([]any)
		if !ok || len(sec) == 0 {
			t.Error("/api/v1/profile should inherit BearerAuth security")
		} else {
			first := sec[0].(map[string]any)
			if _, ok := first["BearerAuth"]; !ok {
				t.Error("/api/v1/profile security should reference BearerAuth")
			}
		}
	}

	// 验证 components
	components := doc["components"].(map[string]any)

	// securitySchemes
	schemes, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatal("components should have securitySchemes")
	}
	if _, ok := schemes["BearerAuth"]; !ok {
		t.Error("securitySchemes should have BearerAuth")
	}

	// responses (ErrorResponse)
	responses, ok := components["responses"].(map[string]any)
	if !ok {
		t.Fatal("components should have responses")
	}
	if _, ok := responses["ErrorResponse"]; !ok {
		t.Error("responses should have ErrorResponse")
	}

	// 验证 tags
	tags, ok := doc["tags"].([]any)
	if !ok || len(tags) == 0 {
		t.Fatal("should have tags")
	}
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagMap := tag.(map[string]any)
		tagNames[tagMap["name"].(string)] = true
	}
	for _, expected := range []string{"System", "Users", "V1"} {
		if !tagNames[expected] {
			t.Errorf("missing tag %q", expected)
		}
	}

	// 验证 GET /users 有 query parameters
	getUsersOp := getOperation(paths, "/users", "get")
	if getUsersOp != nil {
		params, ok := getUsersOp["parameters"].([]any)
		if !ok || len(params) == 0 {
			t.Error("GET /users should have query parameters")
		}
	}

	// 验证 POST /users 有 requestBody
	postUsersOp := getOperation(paths, "/users", "post")
	if postUsersOp != nil {
		if _, ok := postUsersOp["requestBody"]; !ok {
			t.Error("POST /users should have requestBody")
		}
	}

	// 验证 DELETE /users/{id} 有 path parameter
	deleteUsersOp := getOperation(paths, "/users/{id}", "delete")
	if deleteUsersOp != nil {
		params, ok := deleteUsersOp["parameters"].([]any)
		if !ok || len(params) == 0 {
			t.Error("DELETE /users/{id} should have path parameter")
		} else {
			p := params[0].(map[string]any)
			if p["in"] != "path" {
				t.Errorf("parameter in = %v, want path", p["in"])
			}
		}
	}

	// 验证所有 operation 都有 200 + 错误响应
	for path, item := range paths {
		pathItem := item.(map[string]any)
		for method, op := range pathItem {
			operation := op.(map[string]any)
			responses, ok := operation["responses"].(map[string]any)
			if !ok {
				t.Errorf("%s %s: missing responses", method, path)
				continue
			}
			if _, ok := responses["200"]; !ok {
				t.Errorf("%s %s: missing 200 response", method, path)
			}
			if _, ok := responses["400"]; !ok {
				t.Errorf("%s %s: missing 400 error response", method, path)
			}
		}
	}
}

// ============ 测试辅助函数 ============

func assertPath(t *testing.T, paths map[string]any, path, method, expectedSummary string) {
	t.Helper()
	pathItem, ok := paths[path]
	if !ok {
		t.Errorf("path %q should exist", path)
		return
	}
	op, ok := pathItem.(map[string]any)[method]
	if !ok {
		t.Errorf("%s %s should exist", method, path)
		return
	}
	opMap := op.(map[string]any)
	if summary, _ := opMap["summary"].(string); summary != expectedSummary {
		t.Errorf("%s %s summary = %q, want %q", method, path, summary, expectedSummary)
	}
}

func getOperation(paths map[string]any, path, method string) map[string]any {
	pathItem, ok := paths[path]
	if !ok {
		return nil
	}
	op, ok := pathItem.(map[string]any)[method]
	if !ok {
		return nil
	}
	return op.(map[string]any)
}
