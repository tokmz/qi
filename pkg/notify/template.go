package notify

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"
	texttemplate "text/template"
)

// TemplateManager 模板管理器
type TemplateManager struct {
	config        *TemplateConfig
	htmlTemplates map[string]*template.Template
	textTemplates map[string]*texttemplate.Template
	mu            sync.RWMutex
}

// NewTemplateManager 创建模板管理器
func NewTemplateManager(config *TemplateConfig) (*TemplateManager, error) {
	if config == nil {
		config = &TemplateConfig{
			TemplateDir: "templates",
		}
	}

	tm := &TemplateManager{
		config:        config,
		htmlTemplates: make(map[string]*template.Template),
		textTemplates: make(map[string]*texttemplate.Template),
	}

	// 加载模板
	if err := tm.LoadTemplates(); err != nil {
		return nil, err
	}

	return tm, nil
}

// LoadTemplates 加载所有模板
func (tm *TemplateManager) LoadTemplates() error {
	if tm.config.TemplateDir == "" {
		return nil
	}

	// 检查目录是否存在
	if _, err := os.Stat(tm.config.TemplateDir); os.IsNotExist(err) {
		// 目录不存在时不报错，允许使用内存模板
		return nil
	}

	// 遍历模板目录
	return filepath.Walk(tm.config.TemplateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 只处理 .html 和 .txt 文件
		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".txt" {
			return nil
		}

		// 读取模板内容
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// 模板名称（相对路径，去除扩展名）
		relPath, _ := filepath.Rel(tm.config.TemplateDir, path)
		name := relPath[:len(relPath)-len(ext)]

		// 解析模板
		if ext == ".html" {
			tmpl, err := template.New(name).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse html template %s failed: %w", name, err)
			}
			tm.mu.Lock()
			tm.htmlTemplates[name] = tmpl
			tm.mu.Unlock()
		} else if ext == ".txt" {
			tmpl, err := texttemplate.New(name).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse text template %s failed: %w", name, err)
			}
			tm.mu.Lock()
			tm.textTemplates[name] = tmpl
			tm.mu.Unlock()
		}

		return nil
	})
}

// Render 渲染模板
func (tm *TemplateManager) Render(name string, data interface{}) (string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 优先查找 HTML 模板
	if tmpl, ok := tm.htmlTemplates[name]; ok {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	// 查找文本模板
	if tmpl, ok := tm.textTemplates[name]; ok {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
}

// RegisterHTMLTemplate 注册 HTML 模板（用于运行时添加模板）
func (tm *TemplateManager) RegisterHTMLTemplate(name, content string) error {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return err
	}

	tm.mu.Lock()
	tm.htmlTemplates[name] = tmpl
	tm.mu.Unlock()

	return nil
}

// RegisterTextTemplate 注册文本模板
func (tm *TemplateManager) RegisterTextTemplate(name, content string) error {
	tmpl, err := texttemplate.New(name).Parse(content)
	if err != nil {
		return err
	}

	tm.mu.Lock()
	tm.textTemplates[name] = tmpl
	tm.mu.Unlock()

	return nil
}

// HasTemplate 检查模板是否存在
func (tm *TemplateManager) HasTemplate(name string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, htmlOk := tm.htmlTemplates[name]
	_, textOk := tm.textTemplates[name]

	return htmlOk || textOk
}

// ListTemplates 列出所有模板名称
func (tm *TemplateManager) ListTemplates() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var names []string
	for name := range tm.htmlTemplates {
		names = append(names, name+" (html)")
	}
	for name := range tm.textTemplates {
		names = append(names, name+" (text)")
	}
	return names
}

// RemoveTemplate 删除模板
func (tm *TemplateManager) RemoveTemplate(name string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.htmlTemplates, name)
	delete(tm.textTemplates, name)
}

// ReloadTemplates 重新加载模板（用于开发环境热更新）
func (tm *TemplateManager) ReloadTemplates() error {
	tm.mu.Lock()
	tm.htmlTemplates = make(map[string]*template.Template)
	tm.textTemplates = make(map[string]*texttemplate.Template)
	tm.mu.Unlock()

	return tm.LoadTemplates()
}

// 内置常用模板

// WelcomeEmailTemplate 欢迎邮件模板
const WelcomeEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>欢迎</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #f9f9f9; padding: 30px; border-radius: 10px;">
        <h1 style="color: #333;">欢迎，{{.Username}}！</h1>
        <p style="color: #666; font-size: 16px;">
            感谢您注册 {{.AppName}}，我们很高兴您的加入。
        </p>
        <p style="color: #666; font-size: 16px;">
            您的账号信息：
        </p>
        <ul style="color: #666; font-size: 16px;">
            <li>用户名：{{.Username}}</li>
            <li>邮箱：{{.Email}}</li>
            <li>注册时间：{{.RegisterTime}}</li>
        </ul>
        <div style="margin-top: 30px; text-align: center;">
            <a href="{{.LoginURL}}" style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
                立即登录
            </a>
        </div>
        <p style="color: #999; font-size: 14px; margin-top: 30px;">
            如果您没有注册此账号，请忽略此邮件。
        </p>
    </div>
</body>
</html>
`

// VerificationEmailTemplate 验证邮件模板
const VerificationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>邮箱验证</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #f9f9f9; padding: 30px; border-radius: 10px;">
        <h1 style="color: #333;">验证您的邮箱</h1>
        <p style="color: #666; font-size: 16px;">
            您好，{{.Username}}！
        </p>
        <p style="color: #666; font-size: 16px;">
            请点击下面的按钮验证您的邮箱地址：
        </p>
        <div style="margin: 30px 0; text-align: center;">
            <a href="{{.VerifyURL}}" style="background-color: #28a745; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
                验证邮箱
            </a>
        </div>
        <p style="color: #666; font-size: 16px;">
            或者复制以下链接到浏览器打开：
        </p>
        <p style="color: #007bff; font-size: 14px; word-break: break-all;">
            {{.VerifyURL}}
        </p>
        <p style="color: #666; font-size: 14px;">
            验证码：<strong style="font-size: 24px; color: #333;">{{.Code}}</strong>
        </p>
        <p style="color: #999; font-size: 14px; margin-top: 30px;">
            此验证链接将在 {{.ExpireTime}} 后失效。
        </p>
        <p style="color: #999; font-size: 14px;">
            如果您没有请求验证，请忽略此邮件。
        </p>
    </div>
</body>
</html>
`

// ResetPasswordEmailTemplate 重置密码邮件模板
const ResetPasswordEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>重置密码</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #f9f9f9; padding: 30px; border-radius: 10px;">
        <h1 style="color: #333;">重置您的密码</h1>
        <p style="color: #666; font-size: 16px;">
            您好，{{.Username}}！
        </p>
        <p style="color: #666; font-size: 16px;">
            我们收到了重置密码的请求。如果这是您本人的操作，请点击下面的按钮重置密码：
        </p>
        <div style="margin: 30px 0; text-align: center;">
            <a href="{{.ResetURL}}" style="background-color: #dc3545; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
                重置密码
            </a>
        </div>
        <p style="color: #666; font-size: 16px;">
            或者复制以下链接到浏览器打开：
        </p>
        <p style="color: #007bff; font-size: 14px; word-break: break-all;">
            {{.ResetURL}}
        </p>
        <p style="color: #999; font-size: 14px; margin-top: 30px;">
            此重置链接将在 {{.ExpireTime}} 后失效。
        </p>
        <p style="color: #dc3545; font-size: 14px; font-weight: bold;">
            如果您没有请求重置密码，请立即联系我们。
        </p>
    </div>
</body>
</html>
`

// RegisterBuiltinTemplates 注册内置模板
func RegisterBuiltinTemplates(tm *TemplateManager) error {
	templates := map[string]string{
		"welcome":        WelcomeEmailTemplate,
		"verification":   VerificationEmailTemplate,
		"reset_password": ResetPasswordEmailTemplate,
	}

	for name, content := range templates {
		if err := tm.RegisterHTMLTemplate(name, content); err != nil {
			return err
		}
	}

	return nil
}
