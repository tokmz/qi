package errors

/*
	内置常用错误码
*/

var (
	// ErrServer 服务器错误
	ErrServer = New(1000, "服务器异常", 500)
	// ErrBadRequest 客户端请求错误
	ErrBadRequest = New(1001, "请求异常", 400)
	// ErrUnauthorized 未授权
	ErrUnauthorized = New(1002, "授权异常", 401)
	// ErrForbidden 禁止访问
	ErrForbidden = New(1003, "禁止访问", 403)
	// ErrNotFound 资源不存在
	ErrNotFound = New(1004, "资源不存在", 404)
)
