package request

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		w.Header().Set("X-Custom", "test")
		json.NewEncoder(w).Encode(map[string]string{"msg": "ok"})
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/data").SetQuery("page", "1").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
	assert.Equal(t, "test", resp.Headers.Get("X-Custom"))

	var body map[string]string
	require.NoError(t, resp.Unmarshal(&body))
	assert.Equal(t, "ok", body["msg"])
}

func TestPost_JSON(t *testing.T) {
	type Req struct {
		Name string `json:"name"`
	}
	type Resp struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req Req
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(Resp{ID: 1, Name: req.Name})
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	result, err := Do[Resp](client.Post("/users").SetBody(&Req{Name: "test"}))
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "test", result.Name)
}

func TestDoList(t *testing.T) {
	type Item struct {
		ID int `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]Item{{ID: 1}, {ID: 2}})
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	items, err := DoList[Item](client.Get("/items"))
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, 1, items[0].ID)
}

func TestPut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Put("/resource").SetBody(map[string]string{"key": "val"}).Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Delete("/resource/1").Do()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Patch("/resource/1").SetBody(map[string]string{"name": "new"}).Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestHead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.Header().Set("X-Total", "42")
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Head("/resource").Do()
	require.NoError(t, err)
	assert.Equal(t, "42", resp.Headers.Get("X-Total"))
}

func TestBearerToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/secure").SetBearerToken("mytoken").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "admin", user)
		assert.Equal(t, "secret", pass)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/auth").SetBasicAuth("admin", "secret").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestGlobalHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "app/1.0", r.Header.Get("X-App"))
		assert.Equal(t, "override", r.Header.Get("X-Override"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithHeader("X-App", "app/1.0"),
		WithHeader("X-Override", "global"),
	)
	// 请求级 header 优先于全局
	resp, err := client.Get("/").SetHeader("X-Override", "override").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestFormData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "test", r.FormValue("name"))
		assert.Equal(t, "123", r.FormValue("age"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Post("/form").SetFormData(map[string]string{
		"name": "test",
		"age":  "123",
	}).Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestFileUpload(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("hello"), 0644))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(32 << 20)
		assert.Equal(t, "extra", r.FormValue("field"))

		file, header, err := r.FormFile("upload")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "test.txt", header.Filename)

		data, _ := io.ReadAll(file)
		assert.Equal(t, "hello", string(data))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Post("/upload").
		SetFile("upload", tmpFile).
		SetFormData(map[string]string{"field": "extra"}).
		Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestRetry(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithRetry(&RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     50 * time.Millisecond,
			Multiplier:   2.0,
		}),
	)

	resp, err := client.Get("/flaky").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
	assert.Equal(t, int32(3), attempts.Load())
}

func TestRetry_MaxExhausted(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithRetry(&RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     50 * time.Millisecond,
		}),
	)

	resp, err := client.Get("/always-fail").Do()
	// 重试用尽后返回最后一次响应（非 nil）或错误
	if err != nil {
		assert.ErrorIs(t, err, ErrMaxRetry)
	} else {
		assert.True(t, resp.IsError())
	}
	assert.Equal(t, int32(3), attempts.Load()) // 1 initial + 2 retries
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	_, err := client.Get("/slow").SetTimeout(50 * time.Millisecond).Do()
	require.Error(t, err)
}

func TestRequestTimeout_Override(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 客户端超时 5s，请求级超时 50ms
	client := New(WithBaseURL(srv.URL), WithTimeout(5*time.Second))
	_, err := client.Get("/slow").SetTimeout(50 * time.Millisecond).Do()
	require.Error(t, err)
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	client := New(WithBaseURL(srv.URL))
	_, err := client.Get("/").SetContext(ctx).Do()
	require.Error(t, err)
}

func TestInterceptor(t *testing.T) {
	var beforeCalled, afterCalled bool

	interceptor := &testInterceptor{
		before: func(_ context.Context, req *http.Request) error {
			beforeCalled = true
			req.Header.Set("X-Intercepted", "true")
			return nil
		},
		after: func(_ context.Context, resp *Response) error {
			afterCalled = true
			return nil
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.Header.Get("X-Intercepted"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL), WithInterceptor(interceptor))
	resp, err := client.Get("/").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestAuthInterceptor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer dynamic-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithInterceptor(NewAuthInterceptor(func() string { return "dynamic-token" })),
	)
	resp, err := client.Get("/").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestResponse_IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/missing").Do()
	require.NoError(t, err)
	assert.True(t, resp.IsError())
	assert.False(t, resp.IsSuccess())
	assert.Contains(t, resp.String(), "not found")
}

func TestQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "10", r.URL.Query().Get("size"))
		assert.Equal(t, "name", r.URL.Query().Get("sort"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/list").
		SetQuery("page", "1").
		SetQueryParams(map[string]string{"size": "10", "sort": "name"}).
		Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestSetHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "v1", r.Header.Get("X-A"))
		assert.Equal(t, "v2", r.Header.Get("X-B"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/").SetHeaders(map[string]string{"X-A": "v1", "X-B": "v2"}).Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestPerRequestRetry(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 客户端无重试，请求级重试
	client := New(WithBaseURL(srv.URL))
	resp, err := client.Get("/").SetRetry(&RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
	}).Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

// testInterceptor 测试用拦截器
type testInterceptor struct {
	before func(context.Context, *http.Request) error
	after  func(context.Context, *Response) error
}

func (t *testInterceptor) BeforeRequest(ctx context.Context, req *http.Request) error {
	if t.before != nil {
		return t.before(ctx, req)
	}
	return nil
}

func (t *testInterceptor) AfterResponse(ctx context.Context, resp *Response) error {
	if t.after != nil {
		return t.after(ctx, resp)
	}
	return nil
}

func TestRetry_BodyReplay(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		n := attempts.Add(1)
		if n < 3 {
			// 验证每次重试都能读到完整 body
			assert.Equal(t, `{"name":"test"}`, string(body))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		assert.Equal(t, `{"name":"test"}`, string(body))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	client := New(
		WithBaseURL(srv.URL),
		WithRetry(&RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		}),
	)

	resp, err := client.Post("/retry-body").
		SetBody(map[string]string{"name": "test"}).
		Do()
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
	assert.Equal(t, int32(3), attempts.Load())
}

func TestSetBody_MarshalError(t *testing.T) {
	client := New(WithBaseURL("http://localhost"))
	// channel 类型无法 JSON 序列化
	_, err := client.Post("/").SetBody(make(chan int)).Do()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMarshal)
}

func TestDo_ErrorOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	type Resp struct {
		Msg string `json:"msg"`
	}

	client := New(WithBaseURL(srv.URL))
	result, err := Do[Resp](client.Get("/missing"))
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRequestFailed)
}
