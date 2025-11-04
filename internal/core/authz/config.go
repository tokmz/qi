package authz

import (
	"fmt"
	"time"
)

// Mode 权限模式
type Mode string

const (
	// ModeSingle 单租户模式
	ModeSingle Mode = "single"
	// ModeMulti 多租户模式
	ModeMulti Mode = "multi"
)

// AdapterType 适配器类型
type AdapterType string

const (
	// AdapterTypeFile 文件适配器
	AdapterTypeFile AdapterType = "file"
	// AdapterTypeGorm GORM 适配器
	AdapterTypeGorm AdapterType = "gorm"
)

// Config 权限管理配置
type Config struct {
	// 模式：single（单租户）或 multi（多租户）
	Mode Mode `mapstructure:"mode" json:"mode" yaml:"mode"`

	// 单租户模式配置
	Single SingleConfig `mapstructure:"single" json:"single" yaml:"single"`

	// 多租户模式配置
	Multi MultiConfig `mapstructure:"multi" json:"multi" yaml:"multi"`

	// 适配器配置
	Adapter AdapterConfig `mapstructure:"adapter" json:"adapter" yaml:"adapter"`

	// 是否自动加载策略
	AutoLoad bool `mapstructure:"auto_load" json:"auto_load" yaml:"auto_load"`

	// 策略更新间隔（秒）
	AutoLoadInterval int `mapstructure:"auto_load_interval" json:"auto_load_interval" yaml:"auto_load_interval"`

	// 是否启用日志
	EnableLog bool `mapstructure:"enable_log" json:"enable_log" yaml:"enable_log"`
}

// SingleConfig 单租户模式配置
type SingleConfig struct {
	// 模型文件路径
	ModelPath string `mapstructure:"model_path" json:"model_path" yaml:"model_path"`

	// 策略文件路径（使用文件适配器时）
	PolicyPath string `mapstructure:"policy_path" json:"policy_path" yaml:"policy_path"`
}

// MultiConfig 多租户模式配置
type MultiConfig struct {
	// 模型文件路径
	ModelPath string `mapstructure:"model_path" json:"model_path" yaml:"model_path"`

	// 策略文件路径（使用文件适配器时）
	PolicyPath string `mapstructure:"policy_path" json:"policy_path" yaml:"policy_path"`
}

// AdapterConfig 适配器配置
type AdapterConfig struct {
	// 适配器类型：file/gorm
	Type AdapterType `mapstructure:"type" json:"type" yaml:"type"`

	// 数据库 DSN（使用 GORM 适配器时）
	DSN string `mapstructure:"dsn" json:"dsn" yaml:"dsn"`

	// 数据库类型（mysql/postgres/sqlite）
	DBType string `mapstructure:"db_type" json:"db_type" yaml:"db_type"`

	// 表名（默认：casbin_rule）
	TableName string `mapstructure:"table_name" json:"table_name" yaml:"table_name"`

	// 表前缀
	TablePrefix string `mapstructure:"table_prefix" json:"table_prefix" yaml:"table_prefix"`
}

// TenantConfig 租户配置
type TenantConfig struct {
	// 是否启用多租户
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`

	// 租户识别方式：subdomain/header/jwt
	Identifier TenantIdentifier `mapstructure:"identifier" json:"identifier" yaml:"identifier"`

	// 子域名配置
	Subdomain SubdomainConfig `mapstructure:"subdomain" json:"subdomain" yaml:"subdomain"`

	// 请求头配置
	Header HeaderConfig `mapstructure:"header" json:"header" yaml:"header"`

	// JWT 配置
	JWT JWTConfig `mapstructure:"jwt" json:"jwt" yaml:"jwt"`

	// 默认租户（用于开发测试）
	DefaultTenant string `mapstructure:"default_tenant" json:"default_tenant" yaml:"default_tenant"`
}

// TenantIdentifier 租户识别方式
type TenantIdentifier string

const (
	// TenantIdentifierSubdomain 子域名方式
	TenantIdentifierSubdomain TenantIdentifier = "subdomain"
	// TenantIdentifierHeader 请求头方式
	TenantIdentifierHeader TenantIdentifier = "header"
	// TenantIdentifierJWT JWT Token 方式
	TenantIdentifierJWT TenantIdentifier = "jwt"
)

// SubdomainConfig 子域名配置
type SubdomainConfig struct {
	// 主域名（例如：example.com）
	Domain string `mapstructure:"domain" json:"domain" yaml:"domain"`
}

// HeaderConfig 请求头配置
type HeaderConfig struct {
	// 请求头字段名
	Key string `mapstructure:"key" json:"key" yaml:"key"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	// JWT 中的租户ID字段名
	TenantClaim string `mapstructure:"tenant_claim" json:"tenant_claim" yaml:"tenant_claim"`

	// JWT 中的用户ID字段名
	UserClaim string `mapstructure:"user_claim" json:"user_claim" yaml:"user_claim"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证模式
	if c.Mode != ModeSingle && c.Mode != ModeMulti {
		return ErrInvalidMode
	}

	// 验证单租户模式配置
	if c.Mode == ModeSingle {
		if c.Single.ModelPath == "" {
			return ErrMissingModelPath
		}
		if c.Adapter.Type == AdapterTypeFile && c.Single.PolicyPath == "" {
			return ErrMissingPolicyPath
		}
	}

	// 验证多租户模式配置
	if c.Mode == ModeMulti {
		if c.Multi.ModelPath == "" {
			return ErrMissingModelPath
		}
		if c.Adapter.Type == AdapterTypeFile && c.Multi.PolicyPath == "" {
			return ErrMissingPolicyPath
		}
	}

	// 验证适配器配置
	if c.Adapter.Type != AdapterTypeFile && c.Adapter.Type != AdapterTypeGorm {
		return ErrInvalidAdapterType
	}

	if c.Adapter.Type == AdapterTypeGorm {
		if c.Adapter.DSN == "" {
			return ErrMissingDSN
		}
		if c.Adapter.DBType == "" {
			return ErrMissingDBType
		}
	}

	// 验证自动加载间隔
	if c.AutoLoad && c.AutoLoadInterval <= 0 {
		return ErrInvalidAutoLoadInterval
	}

	return nil
}

// GetModelPath 获取模型文件路径
func (c *Config) GetModelPath() string {
	if c.Mode == ModeSingle {
		return c.Single.ModelPath
	}
	return c.Multi.ModelPath
}

// GetPolicyPath 获取策略文件路径
func (c *Config) GetPolicyPath() string {
	if c.Mode == ModeSingle {
		return c.Single.PolicyPath
	}
	return c.Multi.PolicyPath
}

// GetTableName 获取表名
func (c *Config) GetTableName() string {
	if c.Adapter.TableName == "" {
		return "casbin_rule"
	}
	return c.Adapter.TableName
}

// GetAutoLoadDuration 获取自动加载间隔时长
func (c *Config) GetAutoLoadDuration() time.Duration {
	return time.Duration(c.AutoLoadInterval) * time.Second
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode: ModeSingle,
		Single: SingleConfig{
			ModelPath:  "configs/casbin/model.conf",
			PolicyPath: "configs/casbin/policy.csv",
		},
		Multi: MultiConfig{
			ModelPath:  "configs/casbin/model_tenant.conf",
			PolicyPath: "configs/casbin/policy_tenant.csv",
		},
		Adapter: AdapterConfig{
			Type:      AdapterTypeFile,
			TableName: "casbin_rule",
		},
		AutoLoad:         true,
		AutoLoadInterval: 60,
		EnableLog:        true,
	}
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	return fmt.Sprintf(
		"Mode=%s, ModelPath=%s, AdapterType=%s, AutoLoad=%v",
		c.Mode,
		c.GetModelPath(),
		c.Adapter.Type,
		c.AutoLoad,
	)
}
