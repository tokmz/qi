package config

import "github.com/tokmz/qi/pkg/errors"

// 配置包专用错误定义
var (
	// ErrConfigNotFound 配置文件未找到
	ErrConfigNotFound = errors.New(3001, 500, "配置文件未找到", nil)
	// ErrConfigReadFailed 配置读取失败
	ErrConfigReadFailed = errors.New(3003, 500, "配置读取失败", nil)
)
