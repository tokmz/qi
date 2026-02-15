package qi

import (
	"context"
	"net/http"
	"github.com/tokmz/qi/pkg/errors"
	"github.com/tokmz/qi/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Context 包装 gin.Context，提供增强的 API
// 修复：将 gin.Context 从嵌入改为私有字段，避免暴露底层实现
type Context struct {
	ctx *gin.Context
}

// newContext 创建新的上下文
// 不暴露给外部使用
func newContext(c *gin.Context) *Context {
	return &Context{ctx: c}
}

// NewContext 创建新的上下文（公开方法，用于测试）
func NewContext(c *gin.Context) *Context {
	return &Context{ctx: c}
}

// ============ Gin Context 访问方法 ============

// Request 返回底层的 *http.Request
func (c *Context) Request() *http.Request {
	return c.ctx.Request
}

// Writer 返回底层的 http.ResponseWriter
func (c *Context) Writer() gin.ResponseWriter {
	return c.ctx.Writer
}

// SetWriter 替换底层的 ResponseWriter（用于 Gzip 等中间件）
func (c *Context) SetWriter(w gin.ResponseWriter) {
	c.ctx.Writer = w
}

// Param 获取路径参数
func (c *Context) Param(key string) string {
	return c.ctx.Param(key)
}

// FullPath 获取路由模板路径（如 /users/:id）
func (c *Context) FullPath() string {
	return c.ctx.FullPath()
}

// Query 获取 URL 查询参数
func (c *Context) Query(key string) string {
	return c.ctx.Query(key)
}

// DefaultQuery 获取 URL 查询参数（带默认值）
func (c *Context) DefaultQuery(key, defaultValue string) string {
	return c.ctx.DefaultQuery(key, defaultValue)
}

// GetQuery 获取 URL 查询参数（返回是否存在）
func (c *Context) GetQuery(key string) (string, bool) {
	return c.ctx.GetQuery(key)
}

// PostForm 获取 POST 表单参数
func (c *Context) PostForm(key string) string {
	return c.ctx.PostForm(key)
}

// DefaultPostForm 获取 POST 表单参数（带默认值）
func (c *Context) DefaultPostForm(key, defaultValue string) string {
	return c.ctx.DefaultPostForm(key, defaultValue)
}

// GetPostForm 获取 POST 表单参数（返回是否存在）
func (c *Context) GetPostForm(key string) (string, bool) {
	return c.ctx.GetPostForm(key)
}

// ShouldBind 绑定请求参数（不自动响应错误）
func (c *Context) ShouldBind(obj any) error {
	return c.ctx.ShouldBind(obj)
}

// ShouldBindJSON 绑定 JSON 请求体（不自动响应错误）
func (c *Context) ShouldBindJSON(obj any) error {
	return c.ctx.ShouldBindJSON(obj)
}

// ShouldBindQuery 绑定 URL 查询参数（不自动响应错误）
func (c *Context) ShouldBindQuery(obj any) error {
	return c.ctx.ShouldBindQuery(obj)
}

// ShouldBindUri 绑定路径参数（不自动响应错误）
func (c *Context) ShouldBindUri(obj any) error {
	return c.ctx.ShouldBindUri(obj)
}

// ShouldBindHeader 绑定请求头（不自动响应错误）
func (c *Context) ShouldBindHeader(obj any) error {
	return c.ctx.ShouldBindHeader(obj)
}

// JSON 发送 JSON 响应
func (c *Context) JSON(code int, obj any) {
	c.ctx.JSON(code, obj)
}

// Set 设置上下文键值对
func (c *Context) Set(key string, value any) {
	c.ctx.Set(key, value)
}

// Get 获取上下文键值对
func (c *Context) Get(key string) (any, bool) {
	return c.ctx.Get(key)
}

// GetString 获取字符串类型的上下文值
func (c *Context) GetString(key string) string {
	return c.ctx.GetString(key)
}

// GetInt 获取整数类型的上下文值
func (c *Context) GetInt(key string) int {
	return c.ctx.GetInt(key)
}

// GetInt64 获取 int64 类型的上下文值
func (c *Context) GetInt64(key string) int64 {
	return c.ctx.GetInt64(key)
}

// GetUint 获取 uint 类型的上下文值
func (c *Context) GetUint(key string) uint {
	return c.ctx.GetUint(key)
}

// GetUint64 获取 uint64 类型的上下文值
func (c *Context) GetUint64(key string) uint64 {
	return c.ctx.GetUint64(key)
}

// GetFloat64 获取 float64 类型的上下文值
func (c *Context) GetFloat64(key string) float64 {
	return c.ctx.GetFloat64(key)
}

// GetBool 获取布尔类型的上下文值
func (c *Context) GetBool(key string) bool {
	return c.ctx.GetBool(key)
}

// Next 执行下一个中间件或处理函数
func (c *Context) Next() {
	c.ctx.Next()
}

// Abort 中止请求处理
func (c *Context) Abort() {
	c.ctx.Abort()
}

// AbortWithStatus 中止请求并设置状态码
func (c *Context) AbortWithStatus(code int) {
	c.ctx.AbortWithStatus(code)
}

// AbortWithStatusJSON 中止请求并返回 JSON
func (c *Context) AbortWithStatusJSON(code int, jsonObj any) {
	c.ctx.AbortWithStatusJSON(code, jsonObj)
}

// IsAborted 检查请求是否已中止
func (c *Context) IsAborted() bool {
	return c.ctx.IsAborted()
}

// ClientIP 获取客户端 IP
func (c *Context) ClientIP() string {
	return c.ctx.ClientIP()
}

// ContentType 获取 Content-Type
func (c *Context) ContentType() string {
	return c.ctx.ContentType()
}

// GetHeader 获取请求头
func (c *Context) GetHeader(key string) string {
	return c.ctx.GetHeader(key)
}

// Header 设置响应头
func (c *Context) Header(key, value string) {
	c.ctx.Header(key, value)
}

// ============ 请求绑定方法 ============

// Bind 自动绑定并验证请求参数（根据 Content-Type 自动选择）
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) Bind(obj any) error {
	if err := c.ctx.ShouldBind(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindJSON 绑定 JSON 请求体
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindJSON(obj any) error {
	if err := c.ctx.ShouldBindJSON(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindQuery 绑定 URL 查询参数
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindQuery(obj any) error {
	if err := c.ctx.ShouldBindQuery(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindURI 绑定路径参数
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindURI(obj any) error {
	if err := c.ctx.ShouldBindUri(obj); err != nil {
		wrappedErr := c.wrapBindError(err)
		c.RespondError(wrappedErr)
		return wrappedErr
	}
	return nil
}

// BindHeader 绑定请求头
// 绑定失败时自动响应错误，用户只需判断 err != nil 并 return
func (c *Context) BindHeader(obj any) error {
	if err := c.ctx.ShouldBindHeader(obj); err != nil {
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

// respond 统一响应处理（自动添加 TraceID）
func (c *Context) respond(statusCode int, resp *Response) {
	if traceID := GetContextTraceID(c); traceID != "" {
		resp.WithTraceID(traceID)
	}
	c.JSON(statusCode, resp)
}

// RequestContext 返回标准库 context.Context，用于传递给 Service 层
// 自动将 TraceID、UID、Language 注入到 context.Context
// TraceID 和 UID 使用 logger 包的 context key，确保 logger.WithContext 能正确提取
func (c *Context) RequestContext() context.Context {
	ctx := c.ctx.Request.Context()

	// 注入 TraceID（使用 logger 包的 key，确保 logger 能提取）
	if traceID := GetContextTraceID(c); traceID != "" {
		ctx = context.WithValue(ctx, logger.ContextKeyTraceID(), traceID)
	}

	// 注入 UID（使用 logger 包的 key，确保 logger 能提取）
	if uid := GetContextUid(c); uid != 0 {
		ctx = context.WithValue(ctx, logger.ContextKeyUID(), uid)
	}

	// 注入 Language
	if lang := GetContextLanguage(c); lang != "" {
		ctx = context.WithValue(ctx, contextKeyLanguage, lang)
	}

	return ctx
}

// SetRequestContext 更新 Request 的 Context（用于中间件注入 SpanContext）
func (c *Context) SetRequestContext(ctx context.Context) {
	c.ctx.Request = c.ctx.Request.WithContext(ctx)
}
