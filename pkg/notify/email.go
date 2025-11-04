package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	netsmtp "net/smtp"
	"strings"
	"time"
)

// EmailNotifier 邮件通知器
type EmailNotifier struct {
	config *EmailConfig
	logger Logger
}

// NewEmailNotifier 创建邮件通知器
func NewEmailNotifier(config *EmailConfig, logger Logger) (*EmailNotifier, error) {
	if config == nil {
		return nil, ErrEmailSMTPNotConfigured
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 设置默认值
	if config.Charset == "" {
		config.Charset = "UTF-8"
	}

	if config.MaxAttachmentSize == 0 {
		config.MaxAttachmentSize = 10 * 1024 * 1024 // 默认 10MB
	}

	return &EmailNotifier{
		config: config,
		logger: logger,
	}, nil
}

// Name 返回通知器名称
func (e *EmailNotifier) Name() string {
	return "email"
}

// Send 发送邮件
func (e *EmailNotifier) Send(ctx context.Context, message *Message) error {
	// 验证消息
	if err := e.validateEmailMessage(message); err != nil {
		return err
	}

	// 设置默认发件人
	if message.From == "" {
		message.From = e.config.DefaultFrom
	}

	// 构建邮件内容
	emailContent, err := e.buildEmailContent(message)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	// 发送邮件
	if err := e.sendSMTP(message.From, message.To, emailContent); err != nil {
		return err
	}

	if e.logger != nil {
		e.logger.Info("email sent successfully to %v", message.To)
	}

	return nil
}

// SendBatch 批量发送邮件
func (e *EmailNotifier) SendBatch(ctx context.Context, messages []*Message) error {
	for _, message := range messages {
		if err := e.Send(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

// validateEmailMessage 验证邮件消息
func (e *EmailNotifier) validateEmailMessage(message *Message) error {
	// 验证收件人
	for _, addr := range message.To {
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("%w: %s", ErrEmailInvalidAddress, addr)
		}
	}

	// 验证发件人
	if message.From != "" {
		if _, err := mail.ParseAddress(message.From); err != nil {
			return fmt.Errorf("%w: %s", ErrEmailInvalidAddress, message.From)
		}
	}

	// 验证附件大小
	var totalSize int64
	for _, att := range message.Attachments {
		totalSize += int64(len(att.Content))
	}
	if totalSize > e.config.MaxAttachmentSize {
		return ErrEmailAttachmentTooLarge
	}

	return nil
}

// buildEmailContent 构建邮件内容
func (e *EmailNotifier) buildEmailContent(message *Message) (string, error) {
	var builder strings.Builder

	// 邮件头
	builder.WriteString(fmt.Sprintf("From: %s\r\n", e.formatAddress(message.From, e.config.DefaultFromName)))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(message.To, ", ")))

	if message.Subject != "" {
		builder.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Subject))
	}

	if e.config.ReplyTo != "" {
		builder.WriteString(fmt.Sprintf("Reply-To: %s\r\n", e.config.ReplyTo))
	}

	builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	builder.WriteString("MIME-Version: 1.0\r\n")

	// 如果有附件，使用 multipart/mixed
	if len(message.Attachments) > 0 {
		boundary := fmt.Sprintf("boundary_%d", time.Now().Unix())
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// 邮件正文
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString(fmt.Sprintf("Content-Type: %s; charset=%s\r\n", message.ContentType, e.config.Charset))
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(message.Content)
		builder.WriteString("\r\n\r\n")

		// 附件
		for _, att := range message.Attachments {
			builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			builder.WriteString(fmt.Sprintf("Content-Type: %s\r\n", att.ContentType))
			builder.WriteString("Content-Transfer-Encoding: base64\r\n")

			if att.Inline && att.ContentID != "" {
				builder.WriteString(fmt.Sprintf("Content-Disposition: inline; filename=\"%s\"\r\n", att.Filename))
				builder.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", att.ContentID))
			} else {
				builder.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
			}

			builder.WriteString("\r\n")
			builder.WriteString(e.encodeBase64(att.Content))
			builder.WriteString("\r\n\r\n")
		}

		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// 纯文本或 HTML 邮件
		contentType := message.ContentType
		if contentType == "" {
			contentType = ContentTypePlain
		}
		builder.WriteString(fmt.Sprintf("Content-Type: %s; charset=%s\r\n", contentType, e.config.Charset))
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(message.Content)
	}

	return builder.String(), nil
}

// sendSMTP 通过 SMTP 发送邮件
func (e *EmailNotifier) sendSMTP(from string, to []string, content string) error {
	smtp := e.config.SMTP

	// 连接服务器
	addr := fmt.Sprintf("%s:%d", smtp.Host, smtp.Port)

	var conn net.Conn
	var err error

	// 设置超时
	if smtp.Timeout > 0 {
		conn, err = net.DialTimeout("tcp", addr, smtp.Timeout)
	} else {
		conn, err = net.Dial("tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailConnectionFailed, err)
	}
	defer conn.Close()

	// TLS 连接
	if smtp.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: smtp.Host,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 创建 SMTP 客户端
	client, err := netsmtp.NewClient(conn, smtp.Host)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailConnectionFailed, err)
	}
	defer client.Close()

	// STARTTLS
	if smtp.UseStartTLS {
		tlsConfig := &tls.Config{
			ServerName: smtp.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("%w: %v", ErrEmailConnectionFailed, err)
		}
	}

	// 认证
	if smtp.Username != "" && smtp.Password != "" {
		auth := netsmtp.PlainAuth("", smtp.Username, smtp.Password, smtp.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%w: %v", ErrEmailAuthFailed, err)
		}
	}

	// 设置发件人
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	// 设置收件人
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	if _, err := writer.Write([]byte(content)); err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	// 退出
	return client.Quit()
}

// formatAddress 格式化邮件地址
func (e *EmailNotifier) formatAddress(email, name string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

// encodeBase64 Base64 编码
func (e *EmailNotifier) encodeBase64(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder

	for i := 0; i < len(data); i += 3 {
		b := make([]byte, 3)
		n := 0
		for j := 0; j < 3 && i+j < len(data); j++ {
			b[j] = data[i+j]
			n++
		}

		result.WriteByte(base64Table[(b[0]&0xFC)>>2])
		result.WriteByte(base64Table[((b[0]&0x03)<<4)|((b[1]&0xF0)>>4)])

		if n > 1 {
			result.WriteByte(base64Table[((b[1]&0x0F)<<2)|((b[2]&0xC0)>>6)])
		} else {
			result.WriteByte('=')
		}

		if n > 2 {
			result.WriteByte(base64Table[b[2]&0x3F])
		} else {
			result.WriteByte('=')
		}

		// 每 76 个字符换行
		if (i+3)%57 == 0 {
			result.WriteString("\r\n")
		}
	}

	return result.String()
}

// Close 关闭邮件通知器
func (e *EmailNotifier) Close() error {
	// 邮件通知器无需特殊清理
	return nil
}

