package qi

import (
	"net/http"

	"github.com/tokmz/qi/pkg/errors"
)

var (
	// ErrServer 服务器错误
	// Code: 1000, Status: 500
	ErrServer = errors.NewWithStatus(1000, http.StatusInternalServerError, "server error")

	// ErrBadRequest 请求参数错误
	// Code: 1001, Status: 400
	ErrBadRequest = errors.NewWithStatus(1001, http.StatusBadRequest, "bad request")

	// ErrUnauthorized 未授权
	// Code: 1002, Status: 401
	ErrUnauthorized = errors.NewWithStatus(1002, http.StatusUnauthorized, "unauthorized")

	// ErrForbidden 禁止访问
	// Code: 1003, Status: 403
	ErrForbidden = errors.NewWithStatus(1003, http.StatusForbidden, "forbidden")

	// ErrNotFound 资源不存在
	// Code: 1004, Status: 404
	ErrNotFound = errors.NewWithStatus(1004, http.StatusNotFound, "not found")

	// ErrConflict 资源冲突
	// Code: 1005, Status: 409
	ErrConflict = errors.NewWithStatus(1005, http.StatusConflict, "conflict")

	// ErrTooManyRequests 请求过于频繁
	// Code: 1006, Status: 429
	ErrTooManyRequests = errors.NewWithStatus(1006, http.StatusTooManyRequests, "too many requests")

	// ErrInvalidParams 参数无效
	// Code: 1100, Status: 400
	ErrInvalidParams = errors.NewWithStatus(1100, http.StatusBadRequest, "invalid parameters")

	// ErrMissingParams 缺少参数
	// Code: 1101, Status: 400
	ErrMissingParams = errors.NewWithStatus(1101, http.StatusBadRequest, "missing parameters")

	// ErrInvalidFormat 格式错误
	// Code: 1102, Status: 400
	ErrInvalidFormat = errors.NewWithStatus(1102, http.StatusBadRequest, "invalid format")

	// ErrOutOfRange 超出范围
	// Code: 1103, Status: 400
	ErrOutOfRange = errors.NewWithStatus(1103, http.StatusBadRequest, "out of range")
)
