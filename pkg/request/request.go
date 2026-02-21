package request

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Request 链式请求构建器
type Request struct {
	client   *Client
	method   string
	url      string
	headers  map[string]string
	query    url.Values
	bodyBytes []byte            // 缓存 body 内容，支持重试重放
	rawBody   io.Reader         // 原始 body（不可重放）
	formData map[string]string
	files    []fileField
	timeout  time.Duration
	ctx      context.Context
	retry    *RetryConfig
	err      error             // 延迟错误（SetBody 序列化失败等）
}

func newRequest(c *Client, method, rawURL string) *Request {
	return &Request{
		client:  c,
		method:  method,
		url:     rawURL,
		headers: make(map[string]string),
		query:   make(url.Values),
		ctx:     context.Background(),
	}
}

// SetMethod 设置请求方法
func (r *Request) SetMethod(method string) *Request {
	r.method = method
	return r
}

// SetURL 设置请求 URL
func (r *Request) SetURL(url string) *Request {
	r.url = url
	return r
}

// SetHeader 设置请求头
func (r *Request) SetHeader(k, v string) *Request {
	r.headers[k] = v
	return r
}

// SetHeaders 批量设置请求头
func (r *Request) SetHeaders(h map[string]string) *Request {
	for k, v := range h {
		r.headers[k] = v
	}
	return r
}

// SetQuery 设置查询参数
func (r *Request) SetQuery(k, v string) *Request {
	r.query.Set(k, v)
	return r
}

// SetQueryParams 批量设置查询参数
func (r *Request) SetQueryParams(params map[string]string) *Request {
	for k, v := range params {
		r.query.Set(k, v)
	}
	return r
}

// SetBody 设置请求体（自动 JSON 序列化）
func (r *Request) SetBody(body any) *Request {
	data, err := json.Marshal(body)
	if err != nil {
		r.err = ErrMarshal.WithError(err)
		return r
	}
	r.bodyBytes = data
	r.rawBody = nil
	if _, ok := r.headers["Content-Type"]; !ok {
		r.headers["Content-Type"] = "application/json"
	}
	return r
}

// SetRawBody 设置原始请求体（不可重试重放）
func (r *Request) SetRawBody(body io.Reader) *Request {
	r.rawBody = body
	r.bodyBytes = nil
	return r
}

// SetFormData 设置表单数据
func (r *Request) SetFormData(data map[string]string) *Request {
	r.formData = data
	return r
}

// SetFile 设置文件上传
func (r *Request) SetFile(field, filepath string) *Request {
	r.files = append(r.files, fileField{fieldName: field, filePath: filepath})
	return r
}

// SetFiles 批量设置文件上传
func (r *Request) SetFiles(files map[string]string) *Request {
	for field, path := range files {
		r.files = append(r.files, fileField{fieldName: field, filePath: path})
	}
	return r
}

// SetTimeout 覆盖客户端超时
func (r *Request) SetTimeout(d time.Duration) *Request {
	r.timeout = d
	return r
}

// SetContext 设置请求上下文
func (r *Request) SetContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// SetBearerToken 设置 Bearer Token
func (r *Request) SetBearerToken(token string) *Request {
	r.headers["Authorization"] = "Bearer " + token
	return r
}

// SetBasicAuth 设置 Basic Auth
func (r *Request) SetBasicAuth(user, pass string) *Request {
	r.headers["Authorization"] = "Basic " + basicAuth(user, pass)
	return r
}

// SetRetry 覆盖客户端重试配置
func (r *Request) SetRetry(cfg *RetryConfig) *Request {
	r.retry = cfg
	return r
}

// Do 执行请求
func (r *Request) Do() (*Response, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.client.execute(r)
}

// buildURL 构建完整 URL
func (r *Request) buildURL(baseURL string) (string, error) {
	rawURL := r.url
	if baseURL != "" && !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(rawURL, "/")
	}

	if len(r.query) == 0 {
		return rawURL, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ErrInvalidURL.WithError(err)
	}

	q := u.Query()
	for k, vs := range r.query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// buildHTTPRequest 构建 http.Request（每次调用生成新的，支持重试）
func (r *Request) buildHTTPRequest(baseURL string, mergedHeaders map[string]string) (*http.Request, error) {
	fullURL, err := r.buildURL(baseURL)
	if err != nil {
		return nil, err
	}

	var body io.Reader

	// 文件上传优先（文件上传不支持重试重放）
	if len(r.files) > 0 {
		reader, contentType, err := buildMultipart(r.formData, r.files)
		if err != nil {
			return nil, err
		}
		body = reader
		mergedHeaders["Content-Type"] = contentType
	} else if r.formData != nil && r.bodyBytes == nil && r.rawBody == nil {
		// 表单数据
		form := url.Values{}
		for k, v := range r.formData {
			form.Set(k, v)
		}
		body = strings.NewReader(form.Encode())
		if _, ok := mergedHeaders["Content-Type"]; !ok {
			mergedHeaders["Content-Type"] = "application/x-www-form-urlencoded"
		}
	} else if r.bodyBytes != nil {
		// 缓存的 body（可重放）
		body = bytes.NewReader(r.bodyBytes)
	} else {
		// 原始 body（不可重放）
		body = r.rawBody
	}

	req, err := http.NewRequestWithContext(r.ctx, r.method, fullURL, body)
	if err != nil {
		return nil, ErrRequestFailed.WithError(err)
	}

	// 设置请求头
	for k, v := range mergedHeaders {
		req.Header.Set(k, v)
	}

	return req, nil
}

// basicAuth 编码 Basic Auth
func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}
