package middleware

import (
	"fmt"

	"qi"
	"qi/pkg/i18n"
)

// I18nConfig 国际化中间件配置
type I18nConfig struct {
	// I18n 国际化实例，为空则使用全局实例
	I18n *i18n.I18n

	// GetLanguage 从请求中获取语言的函数
	// 默认从以下位置按顺序获取：
	// 1. Query 参数 "lang"
	// 2. Header "Accept-Language"
	// 3. Cookie "lang"
	GetLanguage func(*qi.Context) string
}

// I18n 返回国际化中间件
func I18n() qi.HandlerFunc {
	return I18nWithConfig(I18nConfig{})
}

// I18nWithConfig 返回带配置的国际化中间件
func I18nWithConfig(config I18nConfig) qi.HandlerFunc {
	// 使用配置的实例，为空则使用全局实例
	instance := config.I18n
	if instance == nil {
		instance = i18n.Global()
	}

	// 默认语言获取函数
	getLanguage := config.GetLanguage
	if getLanguage == nil {
		getLanguage = defaultGetLanguage(instance)
	}

	return func(c *qi.Context) {
		// 获取语言
		lang := getLanguage(c)

		// 验证语言是否支持
		if !instance.IsSupported(lang) {
			lang = instance.GetDefaultLanguage()
		}

		// 设置到上下文
		qi.SetContextLanguage(c, lang)

		c.Next()
	}
}

// defaultGetLanguage 默认语言获取函数
func defaultGetLanguage(instance *i18n.I18n) func(*qi.Context) string {
	return func(c *qi.Context) string {
		// 1. 尝试从 Query 参数获取
		if lang := c.Query("lang"); lang != "" {
			return lang
		}

		// 2. 尝试从 Header 获取
		if lang := c.GetHeader("Accept-Language"); lang != "" {
			// 解析 Accept-Language（取第一个）
			// 例如：zh-CN,zh;q=0.9,en;q=0.8 -> zh-CN
			if parsed := parseAcceptLanguage(lang); parsed != "" {
				return parsed
			}
		}

		// 3. 尝试从 Cookie 获取
		if lang, err := c.Cookie("lang"); err == nil && lang != "" {
			return lang
		}

		// 4. 返回默认语言
		return instance.GetDefaultLanguage()
	}
}

// parseAcceptLanguage 解析 Accept-Language 头，返回 quality value 最高的语言
// 例如：zh-CN,zh;q=0.9,en;q=0.8 -> zh-CN
// 例如：en;q=0.8,zh-CN;q=0.9 -> zh-CN
func parseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}

	type langQuality struct {
		lang    string
		quality float64
	}

	var languages []langQuality
	parts := splitAndTrim(header, ',')

	for _, part := range parts {
		// 解析每个语言项：zh-CN;q=0.9 或 zh-CN
		langParts := splitAndTrim(part, ';')
		if len(langParts) == 0 {
			continue
		}

		lang := langParts[0]
		quality := 1.0 // 默认 quality 为 1.0

		// 解析 q 值
		for i := 1; i < len(langParts); i++ {
			qPart := langParts[i]
			if len(qPart) > 2 && qPart[0] == 'q' && qPart[1] == '=' {
				if q, err := parseQuality(qPart[2:]); err == nil {
					quality = q
				}
			}
		}

		languages = append(languages, langQuality{lang: lang, quality: quality})
	}

	// 找到 quality 最高的语言
	if len(languages) == 0 {
		return ""
	}

	best := languages[0]
	for i := 1; i < len(languages); i++ {
		if languages[i].quality > best.quality {
			best = languages[i]
		}
	}

	return best.lang
}

// splitAndTrim 分割字符串并去除空格
func splitAndTrim(s string, sep rune) []string {
	var result []string
	var current []rune

	for _, c := range s {
		if c == sep {
			if len(current) > 0 {
				result = append(result, string(trim(current)))
				current = current[:0]
			}
		} else {
			current = append(current, c)
		}
	}

	if len(current) > 0 {
		result = append(result, string(trim(current)))
	}

	return result
}

// trim 去除首尾空格
func trim(s []rune) []rune {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}

// parseQuality 解析 quality value（0.0 - 1.0）
func parseQuality(s string) (float64, error) {
	// 简单解析：0.9, 1, 0.123 等
	if s == "" {
		return 0, fmt.Errorf("empty quality")
	}

	var result float64
	var decimal float64 = 0.1
	var beforeDot = true

	for _, c := range s {
		if c == '.' {
			if !beforeDot {
				return 0, fmt.Errorf("invalid quality: multiple dots")
			}
			beforeDot = false
			continue
		}

		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid quality: non-digit character")
		}

		digit := float64(c - '0')
		if beforeDot {
			result = result*10 + digit
		} else {
			result += digit * decimal
			decimal *= 0.1
		}
	}

	if result > 1.0 {
		return 1.0, nil
	}

	return result, nil
}
