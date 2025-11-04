package notify_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"qi/pkg/notify"
)

// Example_basicEmail 基础邮件发送示例
func Example_basicEmail() {
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
		Type:    notify.MessageTypeEmail,
		To:      []string{"recipient@example.com"},
		Subject: "测试邮件",
		Content: "这是一封测试邮件",
		ContentType: notify.ContentTypePlain,
	}

	// 4. 发送邮件
	result, err := manager.Send(context.Background(), message)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("邮件发送成功，耗时: %v\n", result.Cost)
}

// Example_htmlEmail HTML 邮件示例
func Example_htmlEmail() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:         notify.QQMailSMTP("your-qq@qq.com", "your-auth-code"),
		DefaultFrom:  "your-qq@qq.com",
		DefaultFromName: "项目管理系统",
	}

	manager, err := notify.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// HTML 内容
	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
		</head>
		<body style="font-family: Arial, sans-serif;">
			<h1 style="color: #007bff;">欢迎注册</h1>
			<p>您好，<strong>张三</strong>！</p>
			<p>感谢您注册我们的项目管理系统。</p>
			<a href="https://example.com/verify" style="background-color: #28a745; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
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
	fmt.Printf("HTML 邮件发送成功: %v\n", result.Success)
}

// Example_templateEmail 模板邮件示例
func Example_templateEmail() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.AliyunSMTP("your-username", "your-password"),
		DefaultFrom: "noreply@example.com",
	}

	manager, err := notify.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// 注册内置模板
	if err := notify.RegisterBuiltinTemplates(manager.GetTemplateManager()); err != nil {
		log.Fatal(err)
	}

	// 使用欢迎邮件模板
	message := &notify.Message{
		Type:     notify.MessageTypeEmail,
		To:       []string{"newuser@example.com"},
		Subject:  "欢迎加入",
		Template: "welcome",
		TemplateData: map[string]interface{}{
			"Username":     "张三",
			"Email":        "newuser@example.com",
			"AppName":      "项目管理系统",
			"RegisterTime": time.Now().Format("2006-01-02 15:04:05"),
			"LoginURL":     "https://example.com/login",
		},
	}

	result, _ := manager.Send(context.Background(), message)
	fmt.Printf("模板邮件发送成功: %v\n", result.Success)
}

// Example_emailWithAttachment 带附件的邮件示例
func Example_emailWithAttachment() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.Office365SMTP("user@company.com", "password"),
		DefaultFrom: "user@company.com",
	}

	manager, _ := notify.NewManager(config)
	defer manager.Close()

	// 创建附件
	pdfContent := []byte("PDF file content here...")
	imageContent := []byte("Image file content here...")

	message := &notify.Message{
		Type:        notify.MessageTypeEmail,
		To:          []string{"recipient@example.com"},
		Subject:     "项目报告",
		Content:     "请查收附件中的项目报告。",
		ContentType: notify.ContentTypePlain,
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
	fmt.Printf("带附件邮件发送成功: %v\n", result.Success)
}

// Example_asyncSend 异步发送示例
func Example_asyncSend() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.SendGridSMTP("your-api-key"),
		DefaultFrom: "noreply@example.com",
	}

	manager, _ := notify.NewManager(config)
	defer manager.Close()

	message := &notify.Message{
		Type:    notify.MessageTypeEmail,
		To:      []string{"user@example.com"},
		Subject: "异步通知",
		Content: "这是一条异步发送的通知",
	}

	// 异步发送选项
	opts := &notify.SendOptions{
		Async: true,
		Callback: func(result *notify.SendResult) {
			if result.Success {
				fmt.Printf("异步发送成功: %s\n", result.MessageID)
			} else {
				fmt.Printf("异步发送失败: %v\n", result.Error)
			}
		},
	}

	result, _ := manager.Send(context.Background(), message, opts)
	fmt.Printf("异步发送请求已提交: %s\n", result.MessageID)

	// 等待异步发送完成
	time.Sleep(time.Second * 2)
}

// Example_batchSend 批量发送示例
func Example_batchSend() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.OutlookSMTP("your-outlook@outlook.com", "password"),
		DefaultFrom: "your-outlook@outlook.com",
	}

	manager, _ := notify.NewManager(config)
	defer manager.Close()

	// 创建多条消息
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
		{
			Type:    notify.MessageTypeEmail,
			To:      []string{"user3@example.com"},
			Subject: "通知 3",
			Content: "这是第三条通知",
		},
	}

	// 批量发送
	results, _ := manager.SendBatch(context.Background(), messages)

	// 统计结果
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	fmt.Printf("批量发送完成，成功: %d/%d\n", successCount, len(messages))
}

// Example_retryStrategy 重试策略示例
func Example_retryStrategy() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.GmailSMTP("user@gmail.com", "password"),
		DefaultFrom: "user@gmail.com",
	}

	manager, _ := notify.NewManager(config)
	defer manager.Close()

	message := &notify.Message{
		Type:    notify.MessageTypeEmail,
		To:      []string{"recipient@example.com"},
		Subject: "重试测试",
		Content: "测试重试机制",
	}

	// 自定义重试选项
	opts := &notify.SendOptions{
		Retry:         5,                    // 重试 5 次
		RetryInterval: time.Second * 10,     // 每次重试间隔 10 秒
		Timeout:       time.Minute,          // 总超时 1 分钟
	}

	result, err := manager.Send(context.Background(), message, opts)
	if err != nil {
		fmt.Printf("发送失败（已重试 %d 次）: %v\n", opts.Retry, err)
	} else {
		fmt.Printf("发送成功，耗时: %v\n", result.Cost)
	}
}

// Example_globalManager 全局管理器示例
func Example_globalManager() {
	// 1. 初始化全局管理器
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.GmailSMTP("user@gmail.com", "password"),
		DefaultFrom: "user@gmail.com",
	}

	if err := notify.InitGlobal(config); err != nil {
		log.Fatal(err)
	}

	// 2. 使用全局函数发送
	message := &notify.Message{
		Type:    notify.MessageTypeEmail,
		To:      []string{"user@example.com"},
		Subject: "全局通知",
		Content: "使用全局管理器发送",
	}

	result, _ := notify.Send(context.Background(), message)
	fmt.Printf("全局发送成功: %v\n", result.Success)
}

// Example_stats 统计信息示例
func Example_stats() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.GmailSMTP("user@gmail.com", "password"),
		DefaultFrom: "user@gmail.com",
	}

	manager, _ := notify.NewManager(config)
	defer manager.Close()

	// 发送一些邮件
	for i := 0; i < 10; i++ {
		message := &notify.Message{
			Type:    notify.MessageTypeEmail,
			To:      []string{fmt.Sprintf("user%d@example.com", i)},
			Subject: "测试邮件",
			Content: "统计测试",
		}
		_, _ = manager.Send(context.Background(), message)
	}

	// 获取统计信息
	stats := manager.GetStats(notify.MessageTypeEmail)
	fmt.Printf("总发送: %d\n", stats.TotalSent)
	fmt.Printf("成功: %d\n", stats.TotalSuccess)
	fmt.Printf("失败: %d\n", stats.TotalFailed)
	fmt.Printf("重试: %d\n", stats.TotalRetry)
	fmt.Printf("最后发送: %v\n", stats.LastSentAt.Format("2006-01-02 15:04:05"))
}

// Example_customTemplate 自定义模板示例
func Example_customTemplate() {
	config := notify.DefaultConfig()
	config.Email = &notify.EmailConfig{
		SMTP:        notify.GmailSMTP("user@gmail.com", "password"),
		DefaultFrom: "user@gmail.com",
	}

	manager, _ := notify.NewManager(config)
	defer manager.Close()

	// 注册自定义模板
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

	tm := manager.GetTemplateManager()
	_ = tm.RegisterHTMLTemplate("task_reminder", customTemplate)

	// 使用自定义模板
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

	result, _ := manager.Send(context.Background(), message)
	fmt.Printf("自定义模板邮件发送成功: %v\n", result.Success)
}

