package i18n

import (
	"errors"
)

// i18n 错误定义

var (
	// ErrLanguageNotSupported 语言不支持
	ErrLanguageNotSupported = errors.New("language not supported")
)
