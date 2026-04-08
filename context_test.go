package qi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tokmz/qi/pkg/errors"
)

// newTestContext 创建用于测试的 qi.Context
func newTestContext() (*Context, *httptest.ResponseRecorder) {
	gin.SetMode("test")
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/test?foo=bar", nil)
	return &Context{ctx: ginCtx}, w
}

func newTestContextWithMethod(method, path, body string) (*Context, *httptest.ResponseRecorder) {
	gin.SetMode("test")
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		ginCtx.Request.Header.Set("Content-Type", "application/json")
	}
	return &Context{ctx: ginCtx}, w
}

func parseResponse(body []byte) (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal(body, &m)
	return m, err
}

func TestContext_OK(t *testing.T) {
	c, w := newTestContext()
	c.OK("hello")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(0) {
		t.Errorf("code = %v, want 0", m["code"])
	}
	if m["message"] != "success" {
		t.Errorf("message = %v, want success", m["message"])
	}
	if m["data"] != "hello" {
		t.Errorf("data = %v, want hello", m["data"])
	}
}

func TestContext_OK_WithMessage(t *testing.T) {
	c, w := newTestContext()
	c.OK("data", "创建成功")

	m, _ := parseResponse(w.Body.Bytes())
	if m["message"] != "创建成功" {
		t.Errorf("message = %v, want 创建成功", m["message"])
	}
}

func TestContext_Fail(t *testing.T) {
	c, w := newTestContext()
	c.Fail(ErrNotFound)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(1004) {
		t.Errorf("code = %v, want 1004", m["code"])
	}
}

func TestContext_Fail_WithCustomError(t *testing.T) {
	c, w := newTestContext()
	customErr := errors.NewWithStatus(2001, http.StatusBadRequest, "custom error")
	c.Fail(customErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(2001) {
		t.Errorf("code = %v, want 2001", m["code"])
	}
}

func TestContext_FailWithCode(t *testing.T) {
	c, w := newTestContext()
	c.FailWithCode(10001, http.StatusBadRequest, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(10001) {
		t.Errorf("code = %v, want 10001", m["code"])
	}
	if m["message"] != "参数错误" {
		t.Errorf("message = %v, want 参数错误", m["message"])
	}
}

func TestContext_Page(t *testing.T) {
	c, w := newTestContext()
	c.Page(100, []string{"a", "b"})

	m, _ := parseResponse(w.Body.Bytes())
	if m["code"] != float64(0) {
		t.Errorf("code = %v, want 0", m["code"])
	}
	data, ok := m["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be a map")
	}
	if data["total"] != float64(100) {
		t.Errorf("total = %v, want 100", data["total"])
	}
}

func TestContext_Query(t *testing.T) {
	c, _ := newTestContext()
	if v := c.Query("foo"); v != "bar" {
		t.Errorf("Query(foo) = %q, want bar", v)
	}
	if v := c.Query("missing"); v != "" {
		t.Errorf("Query(missing) = %q, want empty", v)
	}
}

func TestContext_DefaultQuery(t *testing.T) {
	c, _ := newTestContext()
	if v := c.DefaultQuery("missing", "default"); v != "default" {
		t.Errorf("DefaultQuery = %q, want default", v)
	}
	if v := c.DefaultQuery("foo", "default"); v != "bar" {
		t.Errorf("DefaultQuery = %q, want bar", v)
	}
}

func TestContext_TraceID(t *testing.T) {
	c, w := newTestContext()
	c.ctx.Set("trace_id", "trace-123")
	c.OK("data")

	m, _ := parseResponse(w.Body.Bytes())
	if m["trace_id"] != "trace-123" {
		t.Errorf("trace_id = %v, want trace-123", m["trace_id"])
	}
}

func TestContext_JSON(t *testing.T) {
	c, w := newTestContext()
	c.JSON(http.StatusOK, gin.H{"custom": true})

	m, _ := parseResponse(w.Body.Bytes())
	// JSON 方法不走统一封装，原样输出
	if m["custom"] != true {
		t.Errorf("custom = %v, want true", m["custom"])
	}
}
