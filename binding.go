package qi

import (
	"reflect"
	"runtime"
)

// BoundHandler 携带类型元信息的包装 handler。
// 由 Bind / BindR 生成，RouteBuilder 在注册时提取类型信息用于 OpenAPI 文档。
type BoundHandler struct {
	Handler      HandlerFunc
	RequestType  reflect.Type // 可能为 nil（BindR 场景）
	ResponseType reflect.Type // 可能为 nil
	FuncName     string       // 原始函数名（如 "main.CreateUser"）
}

// Bind 将 func(*Context, *Req) (*Resp, error) 包装为 BoundHandler。
// 请求路径上自动完成绑定和响应包装，注册时提取 Req/Resp 类型用于 OpenAPI。
func Bind[Req any, Resp any](fn func(*Context, *Req) (*Resp, error)) BoundHandler {
	reqType := reflect.TypeFor[Req]()
	respType := reflect.TypeFor[Resp]()

	handler := func(c *Context) {
		req := new(Req)
		if !bindRequest(c, req, reqType) {
			return
		}
		resp, err := fn(c, req)
		if err != nil {
			c.Fail(err)
			return
		}
		c.OK(resp)
	}

	return BoundHandler{
		Handler:      handler,
		RequestType:  reqType,
		ResponseType: respType,
		FuncName:     runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
	}
}

// BindR 将 func(*Context) (*Resp, error) 包装为 BoundHandler。
// 无请求绑定，仅包装响应。适用于不需要请求体的场景（如 GET）。
func BindR[Resp any](fn func(*Context) (*Resp, error)) BoundHandler {
	respType := reflect.TypeFor[Resp]()

	handler := func(c *Context) {
		resp, err := fn(c)
		if err != nil {
			c.Fail(err)
			return
		}
		c.OK(resp)
	}

	return BoundHandler{
		Handler:      handler,
		RequestType:  nil,
		ResponseType: respType,
		FuncName:     runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
	}
}

// BindE 将 func(*Context, *Req) error 包装为 BoundHandler。
// 适用于无需返回响应体的场景（如 DELETE），成功时自动返回 c.OK(nil)。
func BindE[Req any](fn func(*Context, *Req) error) BoundHandler {
	reqType := reflect.TypeFor[Req]()

	handler := func(c *Context) {
		req := new(Req)
		if !bindRequest(c, req, reqType) {
			return
		}
		if err := fn(c, req); err != nil {
			c.Fail(err)
			return
		}
		c.OK(nil)
	}

	return BoundHandler{
		Handler:      handler,
		RequestType:  reqType,
		ResponseType: nil,
		FuncName:     runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
	}
}

// BindRE 将 func(*Context) error 包装为 BoundHandler。
// 无请求绑定，无响应体，成功时自动返回 c.OK(nil)。
func BindRE(fn func(*Context) error) BoundHandler {
	handler := func(c *Context) {
		if err := fn(c); err != nil {
			c.Fail(err)
			return
		}
		c.OK(nil)
	}

	return BoundHandler{
		Handler:      handler,
		RequestType:  nil,
		ResponseType: nil,
		FuncName:     runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
	}
}

// typeHasTag 扫描结构体字段（含嵌入）检查是否有指定 tag。
// 注册时调用一次，闭包捕获结果。
func typeHasTag(t reflect.Type, tagName string) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if _, ok := field.Tag.Lookup(tagName); ok {
			return true
		}
		// 递归检查嵌入字段
		if field.Anonymous {
			if typeHasTag(field.Type, tagName) {
				return true
			}
		}
	}
	return false
}

// bindRequest 统一请求绑定逻辑。
// 根据 HTTP 方法和结构体 tag 自动选择 Bind/BindQuery/BindURI。
// 返回 false 表示绑定失败（已自动写入错误响应）。
func bindRequest(c *Context, req any, reqType reflect.Type) bool {
	hasURITag := typeHasTag(reqType, "uri")
	hasFormTag := typeHasTag(reqType, "form")

	if isBodyMethod(c.Request().Method) {
		if err := c.Bind(req); err != nil {
			return false
		}
	} else if hasFormTag {
		// 仅当结构体含有 form tag 时才调用 BindQuery，
		// 否则会验证仅有 uri tag 的字段导致必填校验失败
		if err := c.BindQuery(req); err != nil {
			return false
		}
	}

	if hasURITag {
		if err := c.BindURI(req); err != nil {
			return false
		}
	}

	return true
}
