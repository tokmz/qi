package notify

import "errors"

var (
	// ErrNotifierNotFound 通知器不存在
	ErrNotifierNotFound = errors.New("notifier not found")

	// ErrInvalidMessage 无效的消息
	ErrInvalidMessage = errors.New("invalid message")

	// ErrEmptyRecipient 收件人为空
	ErrEmptyRecipient = errors.New("empty recipient")

	// ErrEmptyContent 内容为空
	ErrEmptyContent = errors.New("empty content")

	// ErrTemplateNotFound 模板不存在
	ErrTemplateNotFound = errors.New("template not found")

	// ErrTemplateRenderFailed 模板渲染失败
	ErrTemplateRenderFailed = errors.New("template render failed")

	// ErrSendTimeout 发送超时
	ErrSendTimeout = errors.New("send timeout")

	// ErrRetryExhausted 重试次数用尽
	ErrRetryExhausted = errors.New("retry exhausted")

	// ErrNotifierClosed 通知器已关闭
	ErrNotifierClosed = errors.New("notifier closed")

	// === 邮件相关错误 ===

	// ErrEmailInvalidAddress 无效的邮箱地址
	ErrEmailInvalidAddress = errors.New("invalid email address")

	// ErrEmailSMTPNotConfigured SMTP 未配置
	ErrEmailSMTPNotConfigured = errors.New("smtp not configured")

	// ErrEmailAuthFailed 邮件认证失败
	ErrEmailAuthFailed = errors.New("email authentication failed")

	// ErrEmailConnectionFailed 邮件服务器连接失败
	ErrEmailConnectionFailed = errors.New("email connection failed")

	// ErrEmailSendFailed 邮件发送失败
	ErrEmailSendFailed = errors.New("email send failed")

	// ErrEmailAttachmentTooLarge 附件过大
	ErrEmailAttachmentTooLarge = errors.New("attachment too large")

	// === 短信相关错误（预留）===

	// ErrSMSInvalidPhoneNumber 无效的手机号
	ErrSMSInvalidPhoneNumber = errors.New("invalid phone number")

	// ErrSMSQuotaExceeded 短信配额超限
	ErrSMSQuotaExceeded = errors.New("sms quota exceeded")

	// ErrSMSContentTooLong 短信内容过长
	ErrSMSContentTooLong = errors.New("sms content too long")
)

