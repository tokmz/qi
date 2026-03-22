package qi

import (
	"context"
	"io/fs"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokmz/qi/pkg/errors"
)

// Context 封装 gin.Context，保留高频方法，不常用的通过 Gin() 降级使用
type Context struct {
	ctx *gin.Context
}

// Gin 获取底层 gin.Context（escape hatch）
func (c *Context) Gin() *gin.Context {
	return c.ctx
}

// Request 获取原始 http.Request
// 示例：c.Request()
// 返回：*http.Request
func (c *Context) Request() *http.Request {
	return c.ctx.Request
}

// Context 获取标准库 context.Context
func (c *Context) Context() context.Context {
	return c.ctx.Request.Context()
}

// WithValue 向 request context 中写入值
// 示例：c.WithValue("trace_id", "abc123")
func (c *Context) WithValue(key, value any) {
	c.ctx.Request = c.ctx.Request.WithContext(context.WithValue(c.ctx.Request.Context(), key, value))
}

// Deadline 实现 context.Context
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Context().Deadline()
}

// Done 实现 context.Context
func (c *Context) Done() <-chan struct{} {
	return c.Context().Done()
}

// Err 实现 context.Context
func (c *Context) Err() error {
	return c.Context().Err()
}

// Value 实现 context.Context
func (c *Context) Value(key any) any {
	return c.Context().Value(key)
}

// ===== Handler 链控制与上下文管理 =====

// Copy 复制上下文
// 示例：c.Copy()
// 返回：*Context
func (c *Context) Copy() *Context {
	return &Context{c.ctx.Copy()}
}

// Next 调用下一个 Handler
// 示例：c.Next()
func (c *Context) Next() {
	c.ctx.Next()
}

// FullPath 获取路由模板路径
// 示例：注册 /user/:id，返回 "/user/:id"
func (c *Context) FullPath() string {
	return c.ctx.FullPath()
}

// ===== 请求中止与错误处理 =====

// Abort 中止请求
// 示例：c.Abort()
func (c *Context) Abort() {
	c.ctx.Abort()
}

// IsAborted 检查是否已中止请求
// 示例：c.IsAborted()
// 返回：bool
func (c *Context) IsAborted() bool {
	return c.ctx.IsAborted()
}

// AbortWithStatus 中止请求并设置状态码
// 示例：c.AbortWithStatus(http.StatusBadRequest)
func (c *Context) AbortWithStatus(status int) {
	c.ctx.AbortWithStatus(status)
}

// AbortWithStatusJSON 中止请求并设置状态码和 JSON 响应
// 示例：c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
func (c *Context) AbortWithStatusJSON(status int, data any) {
	c.ctx.AbortWithStatusJSON(status, data)
}

// ===== 键值存储与数据共享 =====

// Set 设置键值对
// 示例：c.Set("key", "value")
func (c *Context) Set(key string, value any) {
	c.ctx.Set(key, value)
}

// Get 获取键值对
// 示例：c.Get("key")
// 返回：(val, ok)
//
//	val: 键值对值
//	ok: 是否存在键值对
func (c *Context) Get(key string) (val any, ok bool) {
	return c.ctx.Get(key)
}

// MustGet 获取键值对，不存在时 panic
// 示例：c.MustGet("key")
func (c *Context) MustGet(key string) any {
	return c.ctx.MustGet(key)
}

// Delete 删除键值对
// 示例：c.Delete("key")
func (c *Context) Delete(key string) {
	c.ctx.Delete(key)
}

// ===== URL 参数获取 =====

// Param 获取路由参数
// 示例：/user/:id -> id
func (c *Context) Param(key string) string {
	return c.ctx.Param(key)
}

// Query 获取查询参数
// 示例：?id=1 -> id
func (c *Context) Query(key string) string {
	return c.ctx.Query(key)
}

// DefaultQuery 获取查询参数，不存在时返回默认值
// 示例：?id=1 -> id
func (c *Context) DefaultQuery(key, defaultValue string) string {
	return c.ctx.DefaultQuery(key, defaultValue)
}

// GetQuery 获取查询参数
// 示例：?id=1 -> "1", true
func (c *Context) GetQuery(key string) (string, bool) {
	return c.ctx.GetQuery(key)
}

// QueryArray 获取查询参数数组
//
//	示例：?id=1&id=2 -> ["id=1", "id=2"]
func (c *Context) QueryArray(key string) []string {
	return c.ctx.QueryArray(key)
}

// QueryMap 获取查询参数映射
//
//	示例：?id=1&id=2 -> map[string]string{"id": "1", "id": "2"}
func (c *Context) QueryMap(key string) map[string]string {
	return c.ctx.QueryMap(key)
}

// ===== 表单数据获取 =====

// PostForm 获取表单数据
// 示例：c.PostForm("key")
func (c *Context) PostForm(key string) string {
	return c.ctx.PostForm(key)
}

// DefaultPostForm 获取表单数据，不存在时返回默认值
// 示例：c.DefaultPostForm("key", "default")
func (c *Context) DefaultPostForm(key, defaultValue string) string {
	return c.ctx.DefaultPostForm(key, defaultValue)
}

// GetPostForm 获取表单数据
// 示例：c.GetPostForm("key")
func (c *Context) GetPostForm(key string) (string, bool) {
	return c.ctx.GetPostForm(key)
}

// PostFormArray 获取表单数据数组
//
//	示例：c.PostFormArray("key") -> ["value1", "value2"]
func (c *Context) PostFormArray(key string) []string {
	return c.ctx.PostFormArray(key)
}

// PostFormMap 获取表单数据映射
//
//	示例：c.PostFormMap("key") -> map[string]string{"key": "value"}
func (c *Context) PostFormMap(key string) map[string]string {
	return c.ctx.PostFormMap(key)
}

// ===== 文件上传 =====

// FormFile 获取文件上传
// 示例：c.FormFile("key")
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	return c.ctx.FormFile(key)
}

// MultipartForm 获取文件上传表单数据
// 示例：c.MultipartForm()
func (c *Context) MultipartForm() (*multipart.Form, error) {
	return c.ctx.MultipartForm()
}

// SaveUploadedFile 保存上传文件
// 示例：c.SaveUploadedFile("key", "path")
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string, perm ...fs.FileMode) error {
	return c.ctx.SaveUploadedFile(file, dst, perm...)
}

// ===== 数据绑定 =====
// 绑定失败时自动调用 c.Fail() 写入错误响应，调用方只需判断 err != nil 后 return

// Bind 自动绑定请求体到结构体（根据 Content-Type）
// 示例：c.Bind(&user)
func (c *Context) Bind(obj any) error {
	return c.bindOrFail(c.ctx.ShouldBind(obj))
}

// BindJSON 绑定 JSON 请求体到结构体
// 示例：c.BindJSON(&user)
func (c *Context) BindJSON(obj any) error {
	return c.bindOrFail(c.ctx.ShouldBindJSON(obj))
}

// BindQuery 绑定查询参数到结构体
// 示例：c.BindQuery(&user)
func (c *Context) BindQuery(obj any) error {
	return c.bindOrFail(c.ctx.ShouldBindQuery(obj))
}

// BindURI 绑定 URI 参数到结构体
// 示例：c.BindURI(&user)
func (c *Context) BindURI(obj any) error {
	return c.bindOrFail(c.ctx.ShouldBindUri(obj))
}

// bindOrFail 绑定失败时自动写入错误响应
func (c *Context) bindOrFail(err error) error {
	if err != nil {
		c.Fail(ErrBadRequest.WithErr(err))
		return err
	}
	return nil
}

// ===== 请求信息获取 =====

// ClientIP 获取客户端 IP
// 示例：c.ClientIP()
func (c *Context) ClientIP() string {
	return c.ctx.ClientIP()
}

// ContentType 获取请求体 Content-Type
// 示例：c.ContentType()
func (c *Context) ContentType() string {
	return c.ctx.ContentType()
}

// GetHeader 获取请求头
// 示例：c.GetHeader("key")
func (c *Context) GetHeader(key string) string {
	return c.ctx.GetHeader(key)
}

// GetRawData 获取原始请求体数据
// 示例：c.GetRawData()
func (c *Context) GetRawData() ([]byte, error) {
	return c.ctx.GetRawData()
}

// Cookie 获取请求 Cookie
// 示例：c.Cookie("key")
func (c *Context) Cookie(key string) (string, error) {
	return c.ctx.Cookie(key)
}

// ===== 响应处理 =====

// traceID 从 gin.Context 中提取 trace_id
func (c *Context) traceID() string {
	if v, ok := c.ctx.Get("trace_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// respond 统一响应，自动填充 trace_id
func (c *Context) respond(status int, code int, msg string, data any) {
	resp := NewResponse(code, msg, data)
	if tid := c.traceID(); tid != "" {
		resp.TraceID = tid
	}
	c.ctx.JSON(status, resp)
}

// Header 设置响应头
func (c *Context) Header(key, value string) {
	c.ctx.Header(key, value)
}

// JSON 响应 JSON 数据（原样输出，不经过 Response 封装）
func (c *Context) JSON(status int, data any) {
	c.ctx.JSON(status, data)
}

// OK 成功响应，code=0
// 示例：c.OK(user) 或 c.OK(user, "创建成功")
func (c *Context) OK(data any, msg ...string) {
	m := "success"
	if len(msg) > 0 {
		m = msg[0]
	}
	c.respond(http.StatusOK, 0, m, data)
}

// Fail 错误响应，自动从 *errors.Error 提取 code/status/message
// 示例：c.Fail(qi.ErrNotFound)
func (c *Context) Fail(err error) {
	if err == nil {
		c.OK(nil)
		return
	}
	code := errors.GetCode(err)
	if code == -1 {
		c.respond(http.StatusInternalServerError, ErrServer.Code, ErrServer.Message, nil)
		return
	}
	status := errors.GetStatus(err)
	c.respond(status, code, err.Error(), nil)
}

// FailWithCode 自定义 code 的错误响应
// 示例：c.FailWithCode(http.StatusBadRequest, 10001, "参数错误")
func (c *Context) FailWithCode(status, code int, msg string) {
	c.respond(status, code, msg, nil)
}

// Page 分页响应
// 示例：c.Page(total, list)
func (c *Context) Page(total int64, list any) {
	c.OK(map[string]any{
		"total": total,
		"list":  list,
	})
}

// HTML 响应 HTML 数据
// 示例：c.HTML(http.StatusOK, "index.html", nil)
func (c *Context) HTML(status int, name string, data any) {
	c.ctx.HTML(status, name, data)
}

// String 响应字符串
// 示例：c.String(http.StatusOK, "hello world")
func (c *Context) String(status int, text string) {
	c.ctx.String(status, text)
}

// File 响应文件
// 示例：c.File("path/to/file")
func (c *Context) File(path string) {
	c.ctx.File(path)
}

// Redirect 重定向
// 示例：c.Redirect(http.StatusOKFound, "/new-path")
func (c *Context) Redirect(status int, path string) {
	c.ctx.Redirect(status, path)
}
