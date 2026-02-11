package errors

import "errors"

type Error struct {
	Code     int    `json:"code"`    // 错误码
	HttpCode int    `json:"-"`       // http状态码
	Message  string `json:"message"` // 错误信息
	Err      error  `json:"-"`       // 原始错误
}

// Error 实现 error 接口
func (e *Error) Error() string {
	return e.Message
}

// Unwrap 实现 errors.Unwrap 接口
func (e *Error) Unwrap() error {
	return e.Err
}

// New 创建新的错误
// code 错误码
// httpCode http状态码
// message 错误信息
// err 原始错误
func New(code int, httpCode int, message string, err error) *Error {
	return &Error{
		Code:     code,
		HttpCode: httpCode,
		Message:  message,
		Err:      err,
	}
}

// Clone 克隆错误（避免修改共享的预定义错误）
func (e *Error) Clone() *Error {
	return &Error{
		Code:     e.Code,
		HttpCode: e.HttpCode,
		Message:  e.Message,
		Err:      e.Err,
	}
}

// WithError 添加原始错误（返回新实例，不修改原错误）
func (e *Error) WithError(err error) *Error {
	return &Error{
		Code:     e.Code,
		HttpCode: e.HttpCode,
		Message:  e.Message,
		Err:      err,
	}
}

// WithMessage 添加错误信息（返回新实例，不修改原错误）
func (e *Error) WithMessage(message string) *Error {
	return &Error{
		Code:     e.Code,
		HttpCode: e.HttpCode,
		Message:  message,
		Err:      e.Err,
	}
}

// As 转换为指定类型的错误
// target 目标错误类型指针
func (e *Error) As(target any) bool {
	return errors.As(e.Err, target)
}

// Is 检查错误是否为指定类型
// target 目标错误类型
func (e *Error) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// As 转换为指定类型的错误
// err 待转换错误
// target 目标错误类型指针（必须是指针类型）
// 修复：直接传递 target，不再传递指针的指针
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Is 检查错误是否为指定类型
// err 待检查错误
// target 目标错误类型
func Is(err error, target error) bool {
	return errors.Is(err, target)
}
