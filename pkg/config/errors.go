package config

import "errors"

// 配置包专用错误定义
var (
	// ErrConfigNotFound 配置文件未找到
	ErrConfigNotFound = errors.New("配置文件未找到")
	// ErrConfigReadFailed 配置读取失败
	ErrConfigReadFailed = errors.New("配置读取失败")
)
