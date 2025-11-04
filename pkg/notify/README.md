# Notify - 通用消息通知包

## 📋 概述

`notify` 是一个高度可扩展的通用消息通知包，提供统一的接口来发送各种类型的通知消息。目前已内置邮件通知功能，未来可以轻松扩展短信、推送、即时消息等通知方式。

## ✨ 核心特性

### 🎯 设计特点
- **统一接口**：所有通知方式实现相同的 `Notifier` 接口
- **高度可扩展**：轻松添加新的通知方式（短信、推送、IM 等）
- **类型安全**：完整的类型定义和错误处理
- **异步支持**：支持同步和异步发送
- **自动重试**：可配置的重试策略
- **模板系统**：内置 HTML/文本模板引擎
- **批量发送**：支持批量发送优化
- **统计监控**：实时发送统计和监控
- **并发安全**：所有操作都是并发安全的

### 📧 邮件通知功能
- ✅ 支持 SMTP、SMTP over TLS、STARTTLS
- ✅ 支持 HTML 和纯文本邮件
- ✅ 支持附件（文件、图片等）
- ✅ 内嵌图片（inline images）
- ✅ HTML 模板渲染
- ✅ 批量发送
- ✅ 连接池（可选）
- ✅ 常用邮件服务商配置预设（Gmail、QQ、阿里云、Office365 等）

### 🔮 未来扩展（预留接口）
- 📱 短信通知（阿里云、腾讯云、华为云等）
- 📲 推送通知（JPush、GetUI、Firebase 等）
- 💬 即时消息（企业微信、钉钉、Slack 等）

## 📦 安装

```bash
go get -u qi/pkg/notify
```

## 🚀 快速开始

### 1. 基础邮件发送

```go
package main

import (
    "context"
    "log"
    "qi/pkg/notify"
)

func main() {
    // 1. 创建配置
    config := notify.DefaultConfig()
    config.Email = &notify.EmailConfig{
        SMTP:         notify.GmailSMTP("your-email@gmail.com", "your-app-password"),
        DefaultFrom:  "your-email@gmail.com",
        DefaultFromName: "通知系统",
    }

    // 2. 创建通知管理器
    manager, err := notify.NewManager(config)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    // 3. 创建邮件消息
    message := &notify.Message{
        Type:        notify.MessageTypeEmail,
        To:          []string{"recipient@example.com"},
        Subject:     "测试邮件",
        Content:     "这是一封测试邮件",
        ContentType: notify.ContentTypePlain,
    }

    // 4. 发送邮件
    result, err := manager.Send(context.Background(), message)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("邮件发送成功，耗时: %v", result.Cost)
}
```

### 2. HTML 邮件

```go
htmlContent := `
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif;">
    <h1 style="color: #007bff;">欢迎注册</h1>
    <p>您好，<strong>张三</strong>！</p>
    <p>感谢您注册我们的项目管理系统。</p>
    <a href="https://example.com/verify" 
       style="background-color: #28a745; color: white; 
              padding: 10px 20px; text-decoration: none; 
              border-radius: 5px;">
        验证邮箱
    </a>
</body>
</html>
`

message := &notify.Message{
    Type:        notify.MessageTypeEmail,
    To:          []string{"user@example.com"},
    Subject:     "欢迎注册",
    Content:     htmlContent,
    ContentType: notify.ContentTypeHTML,
}

result, _ := manager.Send(context.Background(), message)
```

### 3. 使用模板

```go
// 注册内置模板
notify.RegisterBuiltinTemplates(manager.GetTemplateManager())

// 使用欢迎邮件模板
message := &notify.Message{
    Type:     notify.MessageTypeEmail,
    To:       []string{"newuser@example.com"},
    Subject:  "欢迎加入",
    Template: "welcome",  // 使用模板名
    TemplateData: map[string]interface{}{
        "Username":     "张三",
        "Email":        "newuser@example.com",
        "AppName":      "项目管理系统",
        "RegisterTime": time.Now().Format("2006-01-02 15:04:05"),
        "LoginURL":     "https://example.com/login",
    },
}

result, _ := manager.Send(context.Background(), message)
```

### 4. 带附件的邮件

```go
pdfContent := []byte("PDF file content...")
imageContent := []byte("Image content...")

message := &notify.Message{
    Type:    notify.MessageTypeEmail,
    To:      []string{"recipient@example.com"},
    Subject: "项目报告",
    Content: "请查收附件中的项目报告。",
    Attachments: []*notify.Attachment{
        {
            Filename:    "report.pdf",
            Content:     pdfContent,
            ContentType: "application/pdf",
        },
        {
            Filename:    "chart.png",
            Content:     imageContent,
            ContentType: "image/png",
        },
    },
}

result, _ := manager.Send(context.Background(), message)
```

## 📚 详细使用

### 配置管理

#### 邮件配置

```go
config := notify.DefaultConfig()
config.Email = &notify.EmailConfig{
    // SMTP 配置
    SMTP: &notify.SMTPConfig{
        Host:        "smtp.example.com",
        Port:        587,
        Username:    "user@example.com",
        Password:    "password",
        UseStartTLS: true,
        UseTLS:      false,
        Timeout:     time.Second * 30,
    },
    
    // 默认发件人
    DefaultFrom:     "noreply@example.com",
    DefaultFromName: "通知系统",
    
    // 最大附件大小（字节）
    MaxAttachmentSize: 10 * 1024 * 1024, // 10MB
    
    // 回复地址
    ReplyTo: "support@example.com",
    
    // 字符集
    Charset: "UTF-8",
}
```

#### 常用邮件服务商预设

```go
// Gmail
config.Email.SMTP = notify.GmailSMTP("user@gmail.com", "app-password")

// QQ 邮箱
config.Email.SMTP = notify.QQMailSMTP("user@qq.com", "auth-code")

// 阿里云邮件推送
config.Email.SMTP = notify.AliyunSMTP("username", "password")

// Office 365
config.Email.SMTP = notify.Office365SMTP("user@company.com", "password")

// Outlook
config.Email.SMTP = notify.OutlookSMTP("user@outlook.com", "password")

// SendGrid
config.Email.SMTP = notify.SendGridSMTP("api-key")
```

### 发送选项

#### 异步发送

```go
opts := &notify.SendOptions{
    Async: true,  // 异步发送
    Callback: func(result *notify.SendResult) {
        if result.Success {
            log.Printf("异步发送成功: %s", result.MessageID)
        } else {
            log.Printf("异步发送失败: %v", result.Error)
        }
    },
}

result, _ := manager.Send(context.Background(), message, opts)
```

#### 重试策略

```go
opts := &notify.SendOptions{
    Retry:         5,                // 重试 5 次
    RetryInterval: time.Second * 10, // 每次重试间隔 10 秒
    Timeout:       time.Minute,      // 总超时 1 分钟
}

result, err := manager.Send(context.Background(), message, opts)
```

### 批量发送

```go
messages := []*notify.Message{
    {
        Type:    notify.MessageTypeEmail,
        To:      []string{"user1@example.com"},
        Subject: "通知 1",
        Content: "这是第一条通知",
    },
    {
        Type:    notify.MessageTypeEmail,
        To:      []string{"user2@example.com"},
        Subject: "通知 2",
        Content: "这是第二条通知",
    },
}

// 批量发送（并发）
results, _ := manager.SendBatch(context.Background(), messages)

// 统计结果
successCount := 0
for _, result := range results {
    if result.Success {
        successCount++
    }
}
log.Printf("成功: %d/%d", successCount, len(messages))
```

### 模板管理

#### 注册自定义模板

```go
tm := manager.GetTemplateManager()

// HTML 模板
customTemplate := `
<html>
<body>
    <h1>任务提醒</h1>
    <p>您有一个新任务：{{.TaskName}}</p>
    <p>截止时间：{{.Deadline}}</p>
    <p>优先级：{{.Priority}}</p>
</body>
</html>
`
tm.RegisterHTMLTemplate("task_reminder", customTemplate)

// 使用模板
message := &notify.Message{
    Type:     notify.MessageTypeEmail,
    To:       []string{"user@example.com"},
    Subject:  "任务提醒",
    Template: "task_reminder",
    TemplateData: map[string]interface{}{
        "TaskName": "完成项目报告",
        "Deadline": "2024-12-31",
        "Priority": "高",
    },
}
```

#### 从文件加载模板

```go
config.Template = &notify.TemplateConfig{
    TemplateDir: "templates",  // 模板目录
    AutoReload:  true,         // 自动重载（开发环境）
}

// 目录结构：
// templates/
//   ├── welcome.html
//   ├── verification.html
//   └── reset_password.html
```

### 统计信息

```go
// 获取指定类型的统计
stats := manager.GetStats(notify.MessageTypeEmail)
log.Printf("总发送: %d", stats.TotalSent)
log.Printf("成功: %d", stats.TotalSuccess)
log.Printf("失败: %d", stats.TotalFailed)
log.Printf("重试: %d", stats.TotalRetry)
log.Printf("最后发送: %v", stats.LastSentAt)

// 获取所有统计
allStats := manager.GetAllStats()
for msgType, stats := range allStats {
    log.Printf("%s: 成功率 %.2f%%", 
        msgType, 
        float64(stats.TotalSuccess)/float64(stats.TotalSent)*100)
}
```

### 全局管理器

```go
// 初始化全局管理器（建议在 main 函数中执行一次）
config := notify.DefaultConfig()
config.Email = &notify.EmailConfig{
    SMTP:        notify.GmailSMTP("user@gmail.com", "password"),
    DefaultFrom: "user@gmail.com",
}
notify.InitGlobal(config)

// 在任何地方使用全局函数发送
message := &notify.Message{
    Type:    notify.MessageTypeEmail,
    To:      []string{"user@example.com"},
    Subject: "通知",
    Content: "使用全局管理器",
}
result, _ := notify.Send(context.Background(), message)
```

## 🎨 内置模板

### 1. 欢迎邮件（welcome）

```go
message := &notify.Message{
    Template: "welcome",
    TemplateData: map[string]interface{}{
        "Username":     "张三",
        "Email":        "user@example.com",
        "AppName":      "项目管理系统",
        "RegisterTime": "2024-01-01 10:00:00",
        "LoginURL":     "https://example.com/login",
    },
}
```

### 2. 邮箱验证（verification）

```go
message := &notify.Message{
    Template: "verification",
    TemplateData: map[string]interface{}{
        "Username":   "张三",
        "VerifyURL":  "https://example.com/verify?token=xxx",
        "Code":       "123456",
        "ExpireTime": "24 小时",
    },
}
```

### 3. 重置密码（reset_password）

```go
message := &notify.Message{
    Template: "reset_password",
    TemplateData: map[string]interface{}{
        "Username":   "张三",
        "ResetURL":   "https://example.com/reset?token=xxx",
        "ExpireTime": "1 小时",
    },
}
```

## 🔧 扩展自定义通知器

### 实现 Notifier 接口

```go
package notify

import "context"

// CustomNotifier 自定义通知器
type CustomNotifier struct {
    // 自定义配置
}

// Name 返回通知器名称
func (n *CustomNotifier) Name() string {
    return "custom"
}

// Send 发送通知
func (n *CustomNotifier) Send(ctx context.Context, message *Message) error {
    // 实现发送逻辑
    return nil
}

// SendBatch 批量发送
func (n *CustomNotifier) SendBatch(ctx context.Context, messages []*Message) error {
    for _, msg := range messages {
        if err := n.Send(ctx, msg); err != nil {
            return err
        }
    }
    return nil
}

// Close 关闭通知器
func (n *CustomNotifier) Close() error {
    return nil
}
```

### 注册自定义通知器

```go
// 创建管理器
manager, _ := notify.NewManager(config)

// 创建并注册自定义通知器
customNotifier := &CustomNotifier{}
manager.RegisterNotifier("custom", customNotifier)

// 使用自定义通知器
message := &notify.Message{
    Type:    "custom",
    To:      []string{"recipient"},
    Content: "custom notification",
}
manager.Send(context.Background(), message)
```

## 📖 完整示例

查看 `example_test.go` 文件获取更多完整示例：

- ✅ 基础邮件发送
- ✅ HTML 邮件
- ✅ 模板邮件
- ✅ 带附件邮件
- ✅ 异步发送
- ✅ 批量发送
- ✅ 重试策略
- ✅ 全局管理器
- ✅ 统计信息
- ✅ 自定义模板

## ⚙️ 配置参考

### 完整配置示例

```go
config := &notify.Config{
    // 默认发送者
    DefaultFrom: "noreply@example.com",
    
    // 默认超时
    DefaultTimeout: time.Second * 30,
    
    // 最大重试次数
    MaxRetry: 3,
    
    // 重试间隔
    RetryInterval: time.Second * 5,
    
    // 邮件配置
    Email: &notify.EmailConfig{
        SMTP: &notify.SMTPConfig{
            Host:        "smtp.example.com",
            Port:        587,
            Username:    "user@example.com",
            Password:    "password",
            UseStartTLS: true,
            UseTLS:      false,
            Timeout:     time.Second * 30,
            PoolSize:    10,
            KeepAlive:   true,
        },
        DefaultFrom:       "noreply@example.com",
        DefaultFromName:   "通知系统",
        MaxAttachmentSize: 10 * 1024 * 1024,
        ReplyTo:           "support@example.com",
        Charset:           "UTF-8",
    },
    
    // 模板配置
    Template: &notify.TemplateConfig{
        TemplateDir:    "templates",
        DefaultLang:    "zh-CN",
        SupportedLangs: []string{"zh-CN", "en-US"},
        AutoReload:     false,
    },
    
    // 日志配置
    Logger: myLogger,
}
```

## 🔍 错误处理

```go
result, err := manager.Send(ctx, message)
if err != nil {
    switch {
    case errors.Is(err, notify.ErrEmailInvalidAddress):
        log.Println("无效的邮箱地址")
    case errors.Is(err, notify.ErrEmailAuthFailed):
        log.Println("邮件认证失败，请检查用户名和密码")
    case errors.Is(err, notify.ErrEmailConnectionFailed):
        log.Println("邮件服务器连接失败")
    case errors.Is(err, notify.ErrSendTimeout):
        log.Println("发送超时")
    case errors.Is(err, notify.ErrRetryExhausted):
        log.Println("重试次数用尽")
    default:
        log.Printf("发送失败: %v", err)
    }
}
```

## 🎯 最佳实践

### 1. 使用连接池

```go
config.Email.SMTP.PoolSize = 10      // 连接池大小
config.Email.SMTP.KeepAlive = true   // 保持连接
```

### 2. 异步发送大量邮件

```go
opts := &notify.SendOptions{
    Async: true,
    Callback: func(result *notify.SendResult) {
        // 记录发送结果到数据库
        saveToDatabase(result)
    },
}

for _, message := range messages {
    manager.Send(ctx, message, opts)
}
```

### 3. 使用全局管理器

```go
// main.go
func main() {
    config := notify.DefaultConfig()
    // ... 配置
    notify.InitGlobal(config)
}

// service.go
func SendNotification() {
    message := &notify.Message{...}
    notify.Send(context.Background(), message)
}
```

### 4. 监控发送状态

```go
// 定期检查统计信息
ticker := time.NewTicker(time.Minute * 5)
go func() {
    for range ticker.C {
        stats := manager.GetStats(notify.MessageTypeEmail)
        if stats.TotalFailed > 100 {
            // 告警
            alertOps("邮件发送失败过多")
        }
    }
}()
```

## 📝 注意事项

### Gmail 配置

Gmail 需要使用应用专用密码：
1. 启用两步验证
2. 生成应用专用密码
3. 使用应用密码而不是账户密码

### QQ 邮箱配置

QQ 邮箱需要获取授权码：
1. 进入邮箱设置 → 账户
2. 开启 POP3/SMTP 服务
3. 获取授权码（不是登录密码）

### 阿里云邮件推送

需要先在阿里云控制台：
1. 创建发信域名
2. 配置发信地址
3. 获取 SMTP 用户名和密码

## 🚀 性能优化

### 1. 批量发送

使用 `SendBatch` 而不是循环调用 `Send`

### 2. 连接池

设置合适的连接池大小减少连接开销

### 3. 异步发送

对于非关键通知，使用异步发送提高响应速度

### 4. 模板缓存

模板会被自动缓存，避免重复解析

## 🐛 故障排查

### 连接失败
```
检查防火墙设置
确认 SMTP 服务器地址和端口
检查网络连接
```

### 认证失败
```
确认用户名和密码正确
Gmail 需要应用专用密码
QQ 需要授权码而不是密码
检查是否启用了 SMTP 服务
```

### 发送超时
```
增加超时时间
检查网络延迟
减小附件大小
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📮 联系方式

如有问题或建议，请联系项目维护者。

