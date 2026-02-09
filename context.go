package qi

import (
	"net/http"
	"qi/pkg/errors"
	"qi/pkg/i18n"

	"github.com/gin-gonic/gin"
)

type Context struct {
	*gin.Context
}

// newContext 创建新的上下文
// 不暴露给外部使用
func newContext(c *gin.Context) *Context {
	return &Context{c}
}

// ============ 请求绑定方法 ============

// Bind 自动绑定并验证请求参数（根据 Content-Type 自动选择）
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) Bind(obj any) error {
	if err := c.ShouldBind(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindJSON 绑定 JSON 请求体
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindJSON(obj any) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindQuery 绑定 URL 查询参数
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindQuery(obj any) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindURI 绑定路径参数
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindURI(obj any) error {
	if err := c.ShouldBindUri(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindHeader 绑定请求头
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindHeader(obj any) error {
	if err := c.ShouldBindHeader(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// wrapBindError 包装绑定错误
func (c *Context) wrapBindError(err error) error {
	return errors.ErrBadRequest.WithError(err)
}

// ============ 响应方法 ============

// Success 成功响应
func (c *Context) Success(data any) {
	resp := Success(data)
	c.respond(http.StatusOK, resp)
}

// SuccessWithMessage 成功响应（自定义消息）
func (c *Context) SuccessWithMessage(data any, message string) {
	resp := SuccessWithMessage(data, message)
	c.respond(http.StatusOK, resp)
}

// Nil 成功响应（无数据）
func (c *Context) Nil() {
	c.Success(nil)
}

// Fail 失败响应
func (c *Context) Fail(code int, message string) {
	resp := Fail(code, message)
	c.respond(http.StatusOK, resp)
}

// RespondError 错误响应
func (c *Context) RespondError(err error) {
	var bizErr *errors.Error
	if errors.As(err, &bizErr) {
		resp := NewResponse(bizErr.Code, nil, bizErr.Message)
		c.respond(bizErr.HttpCode, resp)
		return
	}

	// 未知错误 - 使用 ErrServer 的错误码和 HTTP 状态码，但保留原始错误信息
	message := errors.ErrServer.Message
	if err != nil {
		message = err.Error()
	}
	resp := NewResponse(errors.ErrServer.Code, nil, message)
	c.respond(errors.ErrServer.HttpCode, resp)
}

// Page 分页响应
func (c *Context) Page(list any, total uint64) {
	// 确保 list 不为 nil，避免 JSON 序列化为 null
	if list == nil {
		list = []any{}
	}
	resp := Success(NewPageResp(list, total))
	c.respond(http.StatusOK, resp)
}

// T 翻译（从上下文获取语言）
func (c *Context) T(key string, args ...any) string {
	lang := GetContextLanguage(c)
	if lang == "" {
		lang = i18n.GetDefaultLanguage()
	}
	return i18n.T(lang, key, args...)
}

// respond 统一响应处理（自动添加 TraceID）
func (c *Context) respond(statusCode int, resp *Response) {
	if traceID := GetContextTraceID(c); traceID != "" {
		resp.WithTraceID(traceID)
	}
	c.JSON(statusCode, resp)
}
