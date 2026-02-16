package qi

import (
	"context"
	"strings"

	"github.com/tokmz/qi/pkg/i18n"
)

const contextKeyTranslator = "__qi_translator"

// i18nMiddleware 创建 i18n 中间件
// 语言检测优先级：Query(lang) > Header(X-Language) > Accept-Language > 默认语言
func i18nMiddleware(t i18n.Translator) HandlerFunc {
	return func(c *Context) {
		lang := c.Query("lang")
		if lang == "" {
			lang = c.GetHeader("X-Language")
		}
		if lang == "" {
			lang = parseAcceptLanguage(c.GetHeader("Accept-Language"))
		}
		if lang != "" {
			SetContextLanguage(c, lang)
		}
		c.Set(contextKeyTranslator, t)
		c.Next()
	}
}

// parseAcceptLanguage 解析 Accept-Language 头，取第一个语言标签
func parseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}
	// 取第一个语言标签（逗号分隔），忽略权重
	parts := strings.SplitN(header, ",", 2)
	lang := strings.TrimSpace(parts[0])
	// 去掉权重部分（如 "en-US;q=0.9" -> "en-US"）
	if idx := strings.IndexByte(lang, ';'); idx >= 0 {
		lang = strings.TrimSpace(lang[:idx])
	}
	return lang
}

// T 获取翻译（支持变量替换）
// 如果未启用 i18n，直接返回 key
func (c *Context) T(key string, args ...any) string {
	t, ok := c.Get(contextKeyTranslator)
	if !ok || t == nil {
		return key
	}
	return t.(i18n.Translator).T(c.i18nContext(), key, args...)
}

// Tn 获取翻译（支持复数形式）
// 如果未启用 i18n，直接返回 key
func (c *Context) Tn(key, plural string, n int, args ...any) string {
	t, ok := c.Get(contextKeyTranslator)
	if !ok || t == nil {
		return key
	}
	return t.(i18n.Translator).Tn(c.i18nContext(), key, plural, n, args...)
}

// i18nContext 构建带语言信息的 context.Context
func (c *Context) i18nContext() context.Context {
	ctx := context.Background()
	if lang := GetContextLanguage(c); lang != "" {
		ctx = i18n.WithLanguage(ctx, lang)
	}
	return ctx
}
