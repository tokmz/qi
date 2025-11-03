// Package config 提供基于 Viper 的配置管理功能
package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Manager 配置管理器
type Manager struct {
	viper     *viper.Viper
	options   *Options
	logger    Logger
	callbacks []OnChangeCallback
	mu        sync.RWMutex
	watching  bool
	debounce  *time.Timer
}

var (
	// globalManager 全局配置管理器实例
	globalManager *Manager
	once          sync.Once
)

// New 创建新的配置管理器
func New(opts *Options, logger Logger) (*Manager, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	m := &Manager{
		viper:     viper.New(),
		options:   opts,
		logger:    logger,
		callbacks: make([]OnChangeCallback, 0),
		watching:  false,
	}

	// 初始化配置
	if err := m.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	return m, nil
}

// InitGlobal 初始化全局配置管理器（单例模式）
func InitGlobal(opts *Options, logger Logger) error {
	var err error
	once.Do(func() {
		globalManager, err = New(opts, logger)
	})
	return err
}

// GetGlobal 获取全局配置管理器
func GetGlobal() *Manager {
	return globalManager
}

// initialize 初始化配置管理器
func (m *Manager) initialize() error {
	// 设置配置文件信息
	if m.options.ConfigFile != "" {
		// 使用指定的配置文件路径
		m.viper.SetConfigFile(m.options.ConfigFile)
	} else {
		// 使用配置名称和搜索路径
		if m.options.ConfigName != "" {
			m.viper.SetConfigName(m.options.ConfigName)
		}

		if m.options.ConfigType != "" {
			m.viper.SetConfigType(m.options.ConfigType)
		}

		// 添加搜索路径
		for _, path := range m.options.ConfigPaths {
			m.viper.AddConfigPath(path)
		}
	}

	// 环境变量配置
	if m.options.AutomaticEnv {
		m.viper.AutomaticEnv()

		if m.options.EnvPrefix != "" {
			m.viper.SetEnvPrefix(m.options.EnvPrefix)
		}

		m.viper.AllowEmptyEnv(m.options.AllowEmptyEnv)
	}

	// 读取配置文件
	if err := m.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if m.logger != nil {
		m.logger.Info(fmt.Sprintf("Config loaded from: %s", m.viper.ConfigFileUsed()))
	}

	// 启动自动重载
	if m.options.AutoReload {
		if err := m.StartWatching(); err != nil {
			return fmt.Errorf("failed to start watching: %w", err)
		}
	}

	return nil
}

// Reload 手动重新加载配置
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	if m.logger != nil {
		m.logger.Info("Config reloaded")
	}

	// 触发回调
	m.triggerCallbacks(&ChangeEvent{
		Name: m.viper.ConfigFileUsed(),
		Op:   "reload",
		Time: time.Now(),
	})

	return nil
}

// Get 获取配置值
func (m *Manager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Get(key)
}

// GetString 获取字符串配置
func (m *Manager) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetString(key)
}

// GetInt 获取整数配置
func (m *Manager) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt(key)
}

// GetInt32 获取 int32 配置
func (m *Manager) GetInt32(key string) int32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt32(key)
}

// GetInt64 获取 int64 配置
func (m *Manager) GetInt64(key string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt64(key)
}

// GetUint 获取无符号整数配置
func (m *Manager) GetUint(key string) uint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetUint(key)
}

// GetUint32 获取 uint32 配置
func (m *Manager) GetUint32(key string) uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetUint32(key)
}

// GetUint64 获取 uint64 配置
func (m *Manager) GetUint64(key string) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetUint64(key)
}

// GetFloat64 获取浮点数配置
func (m *Manager) GetFloat64(key string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetFloat64(key)
}

// GetBool 获取布尔值配置
func (m *Manager) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetBool(key)
}

// GetStringSlice 获取字符串切片配置
func (m *Manager) GetStringSlice(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringSlice(key)
}

// GetStringMap 获取字符串映射配置
func (m *Manager) GetStringMap(key string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringMap(key)
}

// GetStringMapString 获取字符串到字符串的映射配置
func (m *Manager) GetStringMapString(key string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringMapString(key)
}

// GetStringMapStringSlice 获取字符串到字符串切片的映射配置
func (m *Manager) GetStringMapStringSlice(key string) map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringMapStringSlice(key)
}

// GetTime 获取时间配置
func (m *Manager) GetTime(key string) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetTime(key)
}

// GetDuration 获取时间间隔配置
func (m *Manager) GetDuration(key string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetDuration(key)
}

// GetIntSlice 获取整数切片配置
func (m *Manager) GetIntSlice(key string) []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetIntSlice(key)
}

// GetSizeInBytes 获取字节大小配置
func (m *Manager) GetSizeInBytes(key string) uint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetSizeInBytes(key)
}

// Unmarshal 将配置解析到结构体
func (m *Manager) Unmarshal(rawVal interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Unmarshal(rawVal)
}

// UnmarshalKey 将指定键的配置解析到结构体
func (m *Manager) UnmarshalKey(key string, rawVal interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.UnmarshalKey(key, rawVal)
}

// IsSet 检查配置键是否存在
func (m *Manager) IsSet(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.IsSet(key)
}

// AllKeys 获取所有配置键
func (m *Manager) AllKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.AllKeys()
}

// AllSettings 获取所有配置
func (m *Manager) AllSettings() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.AllSettings()
}

// Set 设置配置值（运行时）
func (m *Manager) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viper.Set(key, value)
}

// SetDefault 设置默认配置值
func (m *Manager) SetDefault(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viper.SetDefault(key, value)
}

// OnChange 注册配置变化回调
func (m *Manager) OnChange(callback OnChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// triggerCallbacks 触发所有回调函数
func (m *Manager) triggerCallbacks(event *ChangeEvent) {
	for _, callback := range m.callbacks {
		if callback != nil {
			// 异步调用回调，避免阻塞
			go func(cb OnChangeCallback) {
				defer func() {
					if r := recover(); r != nil {
						if m.logger != nil {
							m.logger.Error(fmt.Sprintf("Config callback panic: %v", r))
						}
					}
				}()
				cb(event)
			}(callback)
		}
	}
}

// StartWatching 开始监听配置文件变化
func (m *Manager) StartWatching() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watching {
		return ErrWatcherAlreadyRunning
	}

	m.viper.OnConfigChange(func(e fsnotify.Event) {
		// 防抖处理
		if m.debounce != nil {
			m.debounce.Stop()
		}

		m.debounce = time.AfterFunc(m.options.ReloadDebounce, func() {
			if m.logger != nil {
				m.logger.Info("Config file changed, reloading...")
			}

			event := &ChangeEvent{
				Name: e.Name,
				Op:   e.Op.String(),
				Time: time.Now(),
			}

			// 触发回调
			m.triggerCallbacks(event)
		})
	})

	m.viper.WatchConfig()
	m.watching = true

	if m.logger != nil {
		m.logger.Info("Started watching config file")
	}

	return nil
}

// StopWatching 停止监听配置文件变化
func (m *Manager) StopWatching() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.watching {
		return ErrWatcherNotRunning
	}

	if m.debounce != nil {
		m.debounce.Stop()
		m.debounce = nil
	}

	m.watching = false

	if m.logger != nil {
		m.logger.Info("Stopped watching config file")
	}

	return nil
}

// IsWatching 检查是否正在监听配置文件
func (m *Manager) IsWatching() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.watching
}

// GetConfigFile 获取当前使用的配置文件路径
func (m *Manager) GetConfigFile() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.ConfigFileUsed()
}

// GetViper 获取底层的 Viper 实例（高级用法）
func (m *Manager) GetViper() *viper.Viper {
	return m.viper
}
