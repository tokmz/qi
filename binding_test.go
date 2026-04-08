package qi

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type createUserReq struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email"`
}

type searchReq struct {
	Page int `form:"page"`
}

type getUserReq struct {
	ID string `uri:"id" binding:"required"`
}

type mixedReq struct {
	ID   string `uri:"id"`
	Page int    `form:"page"`
}

type userResp struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func setupRouter() (*gin.Engine, *httptest.ResponseRecorder) {
	gin.SetMode("test")
	r := gin.New()
	return r, httptest.NewRecorder()
}

func TestBind_PostBody(t *testing.T) {
	r, w := setupRouter()

	handler := Bind[createUserReq, userResp](func(c *Context, req *createUserReq) (*userResp, error) {
		return &userResp{ID: 1, Name: req.Name}, nil
	})

	r.POST("/users", toGinHandler(handler.Handler))

	body := `{"name":"alice","email":"alice@test.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	m, _ := parseResponse(w.Body.Bytes())
	data := m["data"].(map[string]any)
	if data["name"] != "alice" {
		t.Errorf("name = %v, want alice", data["name"])
	}
}

func TestBind_PostBody_ValidationError(t *testing.T) {
	r, w := setupRouter()

	handler := Bind[createUserReq, userResp](func(c *Context, req *createUserReq) (*userResp, error) {
		return &userResp{}, nil
	})

	r.POST("/users", toGinHandler(handler.Handler))

	// 缺少 required 字段 name
	req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"email":"a@b.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBind_QueryParams(t *testing.T) {
	r, w := setupRouter()

	handler := Bind[searchReq, []string](func(c *Context, req *searchReq) (*[]string, error) {
		if req.Page != 2 {
			t.Errorf("page = %d, want 2", req.Page)
		}
		result := []string{"a", "b"}
		return &result, nil
	})

	r.GET("/search", toGinHandler(handler.Handler))

	req := httptest.NewRequest("GET", "/search?page=2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBind_URIParams(t *testing.T) {
	r, w := setupRouter()

	handler := Bind[getUserReq, userResp](func(c *Context, req *getUserReq) (*userResp, error) {
		if req.ID != "42" {
			t.Errorf("id = %q, want 42", req.ID)
		}
		return &userResp{ID: 42, Name: "bob"}, nil
	})

	r.GET("/users/:id", toGinHandler(handler.Handler))

	req := httptest.NewRequest("GET", "/users/42", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBind_MixedURIAndQuery(t *testing.T) {
	r, _ := setupRouter()

	handler := Bind[mixedReq, string](func(c *Context, req *mixedReq) (*string, error) {
		result := req.ID
		return &result, nil
	})

	r.GET("/users/:id/items", toGinHandler(handler.Handler))

	// 混合场景：ID 从 URI 绑定，Page 从 query 绑定
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/5/items?page=3", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["data"] != "5" {
		t.Errorf("data = %v, want 5", m["data"])
	}
}

func TestBindR(t *testing.T) {
	r, w := setupRouter()

	handler := BindR[[]string](func(c *Context) (*[]string, error) {
		result := []string{"x", "y"}
		return &result, nil
	})

	r.GET("/list", toGinHandler(handler.Handler))

	req := httptest.NewRequest("GET", "/list", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBindE(t *testing.T) {
	r, w := setupRouter()

	handler := BindE[getUserReq](func(c *Context, req *getUserReq) error {
		if req.ID != "1" {
			t.Errorf("id = %q, want 1", req.ID)
		}
		return nil
	})

	r.DELETE("/users/:id", toGinHandler(handler.Handler))

	req := httptest.NewRequest("DELETE", "/users/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(0) {
		t.Errorf("code = %v, want 0", m["code"])
	}
	if m["data"] != nil {
		t.Errorf("data should be nil, got %v", m["data"])
	}
}

func TestBindRE(t *testing.T) {
	r, w := setupRouter()

	called := false
	handler := BindRE(func(c *Context) error {
		called = true
		return nil
	})

	r.POST("/action", toGinHandler(handler.Handler))

	req := httptest.NewRequest("POST", "/action", nil)
	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBoundHandler_Metadata(t *testing.T) {
	handler := Bind[createUserReq, userResp](func(c *Context, req *createUserReq) (*userResp, error) {
		return nil, nil
	})

	if handler.RequestType == nil {
		t.Error("RequestType should not be nil")
	}
	if handler.ResponseType == nil {
		t.Error("ResponseType should not be nil")
	}
	if handler.FuncName == "" {
		t.Error("FuncName should not be empty")
	}
	if handler.Handler == nil {
		t.Error("Handler should not be nil")
	}
}

func TestTypeHasTag(t *testing.T) {
	tests := []struct {
		name    string
		typ     reflect.Type
		tag     string
		hasTag  bool
	}{
		{"createUserReq has json", reflect.TypeFor[createUserReq](), "json", true},
		{"createUserReq has no uri", reflect.TypeFor[createUserReq](), "uri", false},
		{"getUserReq has uri", reflect.TypeFor[getUserReq](), "uri", true},
		{"searchReq has form", reflect.TypeFor[searchReq](), "form", true},
		{"mixedReq has both uri and form", reflect.TypeFor[mixedReq](), "uri", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := typeHasTag(tc.typ, tc.tag)
			if got != tc.hasTag {
				t.Errorf("typeHasTag(%s, %q) = %v, want %v", tc.typ.Name(), tc.tag, got, tc.hasTag)
			}
		})
	}
}

func TestBind_HandlerError(t *testing.T) {
	r, w := setupRouter()

	handler := Bind[createUserReq, userResp](func(c *Context, req *createUserReq) (*userResp, error) {
		return nil, ErrNotFound
	})

	r.POST("/users", toGinHandler(handler.Handler))

	req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(1004) {
		t.Errorf("code = %v, want 1004", m["code"])
	}
}


func TestBind_MixedWithRequiredBinding(t *testing.T) {
	// uri 字段有 binding:"required"，同时结构体有 form 字段
	type mixedWithBinding struct {
		ID   string `uri:"id" binding:"required"`
		Page int    `form:"page"`
	}

	r, _ := setupRouter()
	handler := Bind[mixedWithBinding, string](func(c *Context, req *mixedWithBinding) (*string, error) {
		result := req.ID
		return &result, nil
	})
	r.GET("/users/:id/items", toGinHandler(handler.Handler))

	// 发送请求：ID=5 从路径，page=3 从 query
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/5/items?page=3", nil)
	r.ServeHTTP(w, req)

	t.Logf("Status: %d, Body: %s", w.Code, w.Body.String())
	// 因为结构体有 form tag → 触发 BindQuery → 校验所有 binding tag → ID 为空 → 失败
	if w.Code == http.StatusOK {
		m, _ := parseResponse(w.Body.Bytes())
		t.Errorf("expected BindQuery validation failure, got 200, data=%v", m["data"])
	}
}

func TestBind_MixedWithoutRequiredBinding(t *testing.T) {
	// uri 字段无 binding:"required"
	type mixedNoBinding struct {
		ID   string `uri:"id"`
		Page int    `form:"page"`
	}

	r, _ := setupRouter()
	handler := Bind[mixedNoBinding, string](func(c *Context, req *mixedNoBinding) (*string, error) {
		if req.ID != "5" {
			t.Errorf("ID = %q, want 5", req.ID)
		}
		if req.Page != 3 {
			t.Errorf("Page = %d, want 3", req.Page)
		}
		result := req.ID
		return &result, nil
	})
	r.GET("/users/:id/items", toGinHandler(handler.Handler))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/5/items?page=3", nil)
	r.ServeHTTP(w, req)

	t.Logf("Status: %d, Body: %s", w.Code, w.Body.String())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["data"] != "5" {
		t.Errorf("data = %v, want 5", m["data"])
	}
}

func TestBind_PureURIWithRequired(t *testing.T) {
	// 仅 uri 字段（无 form）+ binding:"required" → 不触发 BindQuery → 安全
	type pureURI struct {
		ID string `uri:"id" binding:"required"`
	}

	r, _ := setupRouter()
	handler := Bind[pureURI, string](func(c *Context, req *pureURI) (*string, error) {
		result := req.ID
		return &result, nil
	})
	r.GET("/users/:id", toGinHandler(handler.Handler))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/42", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["data"] != "42" {
		t.Errorf("data = %v, want 42", m["data"])
	}
}
