package notify

import (
	"context"
	"time"
)

// Notifier 通知器接口
// 所有通知方式（邮件、短信、推送等）都需要实现此接口
type Notifier interface {
	// Send 发送通知
	Send(ctx context.Context, message *Message) error

	// SendBatch 批量发送通知
	SendBatch(ctx context.Context, messages []*Message) error

	// Name 返回通知器名称
	Name() string

	// Close 关闭通知器，释放资源
	Close() error
}

// Message 通用消息结构
type Message struct {
	// ID 消息ID（可选，用于追踪）
	ID string

	// Type 消息类型（email/sms/push等）
	Type MessageType

	// To 接收者（可以是邮箱、手机号、用户ID等）
	To []string

	// From 发送者（邮箱、短信签名等）
	From string

	// Subject 主题（邮件主题、推送标题等）
	Subject string

	// Content 消息内容
	Content string

	// ContentType 内容类型（text/plain, text/html等）
	ContentType ContentType

	// Template 模板名称（如果使用模板）
	Template string

	// TemplateData 模板数据
	TemplateData map[string]interface{}

	// Attachments 附件（邮件附件等）
	Attachments []*Attachment

	// Priority 优先级
	Priority Priority

	// Metadata 元数据（额外信息）
	Metadata map[string]string

	// ScheduledAt 定时发送时间（可选）
	ScheduledAt *time.Time

	// CreatedAt 创建时间
	CreatedAt time.Time
}

// MessageType 消息类型
type MessageType string

const (
	MessageTypeEmail MessageType = "email" // 邮件
	MessageTypeSMS   MessageType = "sms"   // 短信
	MessageTypePush  MessageType = "push"  // 推送
	MessageTypeIM    MessageType = "im"    // 即时消息（如企业微信、钉钉）
)

// ContentType 内容类型
type ContentType string

const (
	ContentTypePlain ContentType = "text/plain"       // 纯文本
	ContentTypeHTML  ContentType = "text/html"        // HTML
	ContentTypeJSON  ContentType = "application/json" // JSON
)

// Priority 消息优先级
type Priority int

const (
	PriorityLow    Priority = 1 // 低优先级
	PriorityNormal Priority = 5 // 普通优先级
	PriorityHigh   Priority = 9 // 高优先级
)

// Attachment 附件
type Attachment struct {
	// Filename 文件名
	Filename string

	// Content 文件内容
	Content []byte

	// ContentType 内容类型（如 image/png, application/pdf）
	ContentType string

	// Inline 是否内嵌（用于邮件中的图片）
	Inline bool

	// ContentID 内容ID（用于 HTML 中引用，如 <img src="cid:logo">）
	ContentID string
}

// SendResult 发送结果
type SendResult struct {
	// MessageID 消息ID
	MessageID string

	// Success 是否成功
	Success bool

	// Error 错误信息
	Error error

	// Provider 提供商（如 SMTP、阿里云、腾讯云）
	Provider string

	// ProviderMessageID 提供商返回的消息ID
	ProviderMessageID string

	// SentAt 发送时间
	SentAt time.Time

	// Cost 发送耗时
	Cost time.Duration
}

// SendOptions 发送选项
type SendOptions struct {
	// Async 是否异步发送
	Async bool

	// Retry 重试次数
	Retry int

	// RetryInterval 重试间隔
	RetryInterval time.Duration

	// Timeout 超时时间
	Timeout time.Duration

	// Callback 发送完成回调
	Callback func(*SendResult)
}

// DefaultSendOptions 默认发送选项
func DefaultSendOptions() *SendOptions {
	return &SendOptions{
		Async:         false,
		Retry:         3,
		RetryInterval: time.Second * 5,
		Timeout:       time.Second * 30,
	}
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Stats 统计信息
type Stats struct {
	// TotalSent 总发送数
	TotalSent int64

	// TotalSuccess 成功数
	TotalSuccess int64

	// TotalFailed 失败数
	TotalFailed int64

	// TotalRetry 重试数
	TotalRetry int64

	// LastSentAt 最后发送时间
	LastSentAt time.Time

	// LastError 最后错误
	LastError error
}

