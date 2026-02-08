package errors

/*
	内置常用错误码
*/

var (
	// ErrServer 服务器错误
	ErrServer = New(1000, 500, "服务器异常", nil)
	// ErrBadRequest 客户端请求错误
	ErrBadRequest = New(1001, 400, "请求异常", nil)
	// ErrUnauthorized 未授权
	ErrUnauthorized = New(1002, 401, "授权异常", nil)
	// ErrForbidden 禁止访问
	ErrForbidden = New(1003, 403, "禁止访问", nil)
	// ErrNotFound 资源不存在
	ErrNotFound = New(1004, 404, "资源不存在", nil)
)
