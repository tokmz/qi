package notify

import "time"

// Config 通知配置
type Config struct {
	// DefaultFrom 默认发送者
	DefaultFrom string

	// DefaultTimeout 默认超时时间
	DefaultTimeout time.Duration

	// MaxRetry 最大重试次数
	MaxRetry int

	// RetryInterval 重试间隔
	RetryInterval time.Duration

	// Email 邮件配置
	Email *EmailConfig

	// SMS 短信配置（预留）
	SMS *SMSConfig

	// Push 推送配置（预留）
	Push *PushConfig

	// Template 模板配置
	Template *TemplateConfig

	// Logger 日志配置
	Logger Logger
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		DefaultTimeout: time.Second * 30,
		MaxRetry:       3,
		RetryInterval:  time.Second * 5,
		Template: &TemplateConfig{
			TemplateDir: "templates",
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Email != nil {
		if err := c.Email.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// EmailConfig 邮件配置
type EmailConfig struct {
	// SMTP 配置
	SMTP *SMTPConfig

	// DefaultFrom 默认发件人邮箱
	DefaultFrom string

	// DefaultFromName 默认发件人名称
	DefaultFromName string

	// MaxAttachmentSize 最大附件大小（字节）
	MaxAttachmentSize int64

	// ReplyTo 默认回复地址
	ReplyTo string

	// Charset 字符集（默认 UTF-8）
	Charset string
}

// Validate 验证邮件配置
func (c *EmailConfig) Validate() error {
	if c.SMTP == nil {
		return ErrEmailSMTPNotConfigured
	}
	return c.SMTP.Validate()
}

// SMTPConfig SMTP 配置
type SMTPConfig struct {
	// Host SMTP 服务器地址
	Host string

	// Port SMTP 端口
	Port int

	// Username 用户名
	Username string

	// Password 密码
	Password string

	// UseTLS 是否使用 TLS（建议使用）
	UseTLS bool

	// UseStartTLS 是否使用 STARTTLS
	UseStartTLS bool

	// Timeout 连接超时时间
	Timeout time.Duration

	// PoolSize 连接池大小（0 表示不使用连接池）
	PoolSize int

	// KeepAlive 是否保持连接
	KeepAlive bool
}

// Validate 验证 SMTP 配置
func (c *SMTPConfig) Validate() error {
	if c.Host == "" {
		return ErrEmailConnectionFailed
	}
	if c.Port == 0 {
		c.Port = 25 // 默认端口
	}
	if c.Timeout == 0 {
		c.Timeout = time.Second * 30
	}
	return nil
}

// SMSConfig 短信配置（预留）
type SMSConfig struct {
	// Provider 提供商（aliyun/tencent/huawei等）
	Provider string

	// AccessKeyID 访问密钥ID
	AccessKeyID string

	// AccessKeySecret 访问密钥
	AccessKeySecret string

	// SignName 签名名称
	SignName string

	// Region 区域
	Region string
}

// PushConfig 推送配置（预留）
type PushConfig struct {
	// Provider 提供商（jpush/getui/firebase等）
	Provider string

	// AppKey 应用Key
	AppKey string

	// AppSecret 应用密钥
	AppSecret string
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	// TemplateDir 模板目录
	TemplateDir string

	// DefaultLang 默认语言
	DefaultLang string

	// SupportedLangs 支持的语言列表
	SupportedLangs []string

	// AutoReload 是否自动重载模板（开发环境）
	AutoReload bool
}

// 常用邮件服务商 SMTP 配置预设

// GmailSMTP Gmail SMTP 配置
func GmailSMTP(username, password string) *SMTPConfig {
	return &SMTPConfig{
		Host:        "smtp.gmail.com",
		Port:        587,
		Username:    username,
		Password:    password,
		UseStartTLS: true,
		UseTLS:      false,
		Timeout:     time.Second * 30,
	}
}

// QQMailSMTP QQ邮箱 SMTP 配置
func QQMailSMTP(username, password string) *SMTPConfig {
	return &SMTPConfig{
		Host:        "smtp.qq.com",
		Port:        465,
		Username:    username,
		Password:    password,
		UseTLS:      true,
		UseStartTLS: false,
		Timeout:     time.Second * 30,
	}
}

// AliyunSMTP 阿里云邮件推送 SMTP 配置
func AliyunSMTP(username, password string) *SMTPConfig {
	return &SMTPConfig{
		Host:        "smtpdm.aliyun.com",
		Port:        465,
		Username:    username,
		Password:    password,
		UseTLS:      true,
		UseStartTLS: false,
		Timeout:     time.Second * 30,
	}
}

// Office365SMTP Office 365 SMTP 配置
func Office365SMTP(username, password string) *SMTPConfig {
	return &SMTPConfig{
		Host:        "smtp.office365.com",
		Port:        587,
		Username:    username,
		Password:    password,
		UseStartTLS: true,
		UseTLS:      false,
		Timeout:     time.Second * 30,
	}
}

// OutlookSMTP Outlook SMTP 配置
func OutlookSMTP(username, password string) *SMTPConfig {
	return &SMTPConfig{
		Host:        "smtp-mail.outlook.com",
		Port:        587,
		Username:    username,
		Password:    password,
		UseStartTLS: true,
		UseTLS:      false,
		Timeout:     time.Second * 30,
	}
}

// SendGridSMTP SendGrid SMTP 配置
func SendGridSMTP(apiKey string) *SMTPConfig {
	return &SMTPConfig{
		Host:        "smtp.sendgrid.net",
		Port:        587,
		Username:    "apikey", // SendGrid 固定使用 "apikey" 作为用户名
		Password:    apiKey,
		UseStartTLS: true,
		UseTLS:      false,
		Timeout:     time.Second * 30,
	}
}

