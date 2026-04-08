package qi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_DefaultConfig(t *testing.T) {
	e := New()
	if e == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNew_WithAddr(t *testing.T) {
	e := New(WithAddr(":9090"))
	if e.cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", e.cfg.Addr)
	}
}

func TestNew_WithMode(t *testing.T) {
	e := New(WithMode("release"))
	if e.cfg.Mode != "release" {
		t.Errorf("Mode = %q, want release", e.cfg.Mode)
	}
}

func TestEngine_GET(t *testing.T) {
	e := New()
	e.GET("/ping", func(c *Context) {
		c.OK("pong")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["data"] != "pong" {
		t.Errorf("data = %v, want pong", m["data"])
	}
}

func TestEngine_POST(t *testing.T) {
	e := New()
	e.POST("/users", func(c *Context) {
		c.OK("created")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/users", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestEngine_Group(t *testing.T) {
	e := New()
	v1 := e.Group("/api/v1")
	v1.GET("/ping", func(c *Context) {
		c.OK("v1")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["data"] != "v1" {
		t.Errorf("data = %v, want v1", m["data"])
	}
}

func TestEngine_Group_Middleware(t *testing.T) {
	e := New()
	v1 := e.Group("/api")
	v1.Use(func(c *Context) {
		c.Set("middleware", "called")
		c.Next()
	})
	v1.GET("/test", func(c *Context) {
		val, _ := c.Get("middleware")
		c.OK(val)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	e.ServeHTTP(w, req)

	m, _ := parseResponse(w.Body.Bytes())
	if m["data"] != "called" {
		t.Errorf("data = %v, want called", m["data"])
	}
}

func TestEngine_Routes(t *testing.T) {
	e := New()
	e.GET("/a", func(c *Context) {})
	e.GET("/b", func(c *Context) {})

	routes := e.Routes()
	if len(routes) < 2 {
		t.Errorf("routes count = %d, want >= 2", len(routes))
	}
}

func TestEngine_PanicRecovery(t *testing.T) {
	e := New()
	e.GET("/panic", func(c *Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(1000) {
		t.Errorf("code = %v, want 1000", m["code"])
	}
	if m["message"] != "server error" {
		t.Errorf("message = %v, want server error", m["message"])
	}
}

func TestEngine_NotFound(t *testing.T) {
	e := New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	e.ServeHTTP(w, req)

	// gin 默认 404
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestEngine_Use(t *testing.T) {
	var called bool
	e := New()
	e.Use(func(c *Context) {
		called = true
		c.Next()
	})
	e.GET("/test", func(c *Context) {
		c.OK("ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	e.ServeHTTP(w, req)

	if !called {
		t.Error("middleware was not called")
	}
}

func TestEngine_Handle_AllMethods(t *testing.T) {
	cases := []struct {
		method string
	}{
		{"GET"}, {"POST"}, {"PUT"}, {"PATCH"}, {"DELETE"}, {"HEAD"}, {"OPTIONS"},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			e := New()
			handler := func(c *Context) { c.OK(tc.method) }

			switch tc.method {
			case "GET":
				e.GET("/test", handler)
			case "POST":
				e.POST("/test", handler)
			case "PUT":
				e.PUT("/test", handler)
			case "PATCH":
				e.PATCH("/test", handler)
			case "DELETE":
				e.DELETE("/test", handler)
			case "HEAD":
				e.HEAD("/test", handler)
			case "OPTIONS":
				e.OPTIONS("/test", handler)
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/test", nil)
			e.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s status = %d, want %d", tc.method, w.Code, http.StatusOK)
			}
		})
	}
}

func TestEngine_SetRouteMeta(t *testing.T) {
	e := New()
	e.GET("/users/:id", func(c *Context) {})
	e.SetRouteMeta("GET", "/users/:id", RouteMeta{
		Summary:     "获取用户",
		Tags:        []string{"用户"},
		OperationID: "getUser",
	})

	meta := e.RouteMeta("GET", "/users/:id")
	if meta == nil {
		t.Fatal("RouteMeta should not be nil")
	}
	if meta.Summary != "获取用户" {
		t.Errorf("Summary = %q, want 获取用户", meta.Summary)
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "用户" {
		t.Errorf("Tags = %v, want [用户]", meta.Tags)
	}
}

func TestEngine_SetRouteMeta_NotFound(t *testing.T) {
	e := New()
	meta := e.RouteMeta("GET", "/nonexistent")
	if meta != nil {
		t.Error("RouteMeta should be nil for unregistered route")
	}
}

