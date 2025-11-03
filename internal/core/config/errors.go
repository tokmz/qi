package config

import "errors"

var (
	// ErrConfigNotInitialized 配置管理器未初始化
	ErrConfigNotInitialized = errors.New("config manager not initialized")

	// ErrConfigFileNotFound 配置文件不存在
	ErrConfigFileNotFound = errors.New("config file not found")

	// ErrInvalidConfigFormat 无效的配置格式
	ErrInvalidConfigFormat = errors.New("invalid config format")

	// ErrInvalidConfigPath 无效的配置路径
	ErrInvalidConfigPath = errors.New("invalid config path")

	// ErrConfigKeyNotFound 配置键不存在
	ErrConfigKeyNotFound = errors.New("config key not found")

	// ErrInvalidConfigType 无效的配置类型
	ErrInvalidConfigType = errors.New("invalid config type")

	// ErrWatcherAlreadyRunning 监听器已在运行
	ErrWatcherAlreadyRunning = errors.New("watcher already running")

	// ErrWatcherNotRunning 监听器未运行
	ErrWatcherNotRunning = errors.New("watcher not running")
)
