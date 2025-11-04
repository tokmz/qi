package config

import "time"

// Options 配置管理器选项
type Options struct {
	// ConfigFile 配置文件路径
	ConfigFile string

	// ConfigName 配置文件名（不包含扩展名）
	ConfigName string

	// ConfigType 配置文件类型 (yaml, json, toml, ini, properties)
	ConfigType string

	// ConfigPaths 配置文件搜索路径列表
	ConfigPaths []string

	// AutoReload 是否自动监听配置文件变化
	AutoReload bool

	// EnvPrefix 环境变量前缀
	EnvPrefix string

	// AutomaticEnv 是否自动读取环境变量
	AutomaticEnv bool

	// AllowEmptyEnv 是否允许空环境变量
	AllowEmptyEnv bool

	// ReloadDebounce 配置重载防抖时间
	ReloadDebounce time.Duration
}

// DefaultOptions 返回默认配置选项
func DefaultOptions() *Options {
	return &Options{
		ConfigName:     "config",
		ConfigType:     "yaml",
		ConfigPaths:    []string{".", "./configs", "/etc/app"},
		AutoReload:     false,
		EnvPrefix:      "",
		AutomaticEnv:   false,
		AllowEmptyEnv:  false,
		ReloadDebounce: 500 * time.Millisecond,
	}
}

// OnChangeCallback 配置变化回调函数类型
type OnChangeCallback func(event *ChangeEvent)

// ChangeEvent 配置变化事件
type ChangeEvent struct {
	// Name 配置文件名称
	Name string

	// Op 操作类型 (write, create, remove, rename, chmod)
	Op string

	// Time 发生时间
	Time time.Time

	// Error 错误信息（如果有）
	Error error
}

// ConfigFormat 配置文件格式
type ConfigFormat string

const (
	// FormatYAML YAML 格式
	FormatYAML ConfigFormat = "yaml"

	// FormatJSON JSON 格式
	FormatJSON ConfigFormat = "json"

	// FormatTOML TOML 格式
	FormatTOML ConfigFormat = "toml"

	// FormatINI INI 格式
	FormatINI ConfigFormat = "ini"

	// FormatProperties Properties 格式
	FormatProperties ConfigFormat = "properties"

	// FormatYML YML 格式（YAML 别名）
	FormatYML ConfigFormat = "yml"
)

// String 返回配置格式的字符串表示
func (f ConfigFormat) String() string {
	return string(f)
}

// IsValid 检查配置格式是否有效
func (f ConfigFormat) IsValid() bool {
	switch f {
	case FormatYAML, FormatJSON, FormatTOML, FormatINI, FormatProperties, FormatYML:
		return true
	default:
		return false
	}
}
