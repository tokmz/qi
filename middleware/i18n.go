package middleware

import (
	"fmt"
	"strings"

	"github.com/tokmz/qi"
	"github.com/tokmz/qi/pkg/i18n"
)

// I18nConfig 中间件配置
type I18nConfig struct {
	// 查询参数名（默认：lang）
	QueryKey string
	// Cookie 名（默认：language）
	CookieKey string
	// Header 名（默认：Accept-Language）
	HeaderKey string
	// 是否将语言写入 Cookie
	SetCookie bool
	// Cookie 过期时间（秒）
	CookieMaxAge int
}

// DefaultI18nConfig 返回默认配置
func DefaultI18nConfig() *I18nConfig {
	return &I18nConfig{
		QueryKey:     "lang",
		CookieKey:    "language",
		HeaderKey:    "Accept-Language",
		SetCookie:    false,
		CookieMaxAge: 86400 * 30,
	}
}

// I18n 创建 i18n 中间件
// 从请求中识别语言，设置到 qi.Context 和 request context 中
func I18n(translator i18n.Translator, cfgs ...*I18nConfig) qi.HandlerFunc {
	cfg := DefaultI18nConfig()
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	available := make(map[string]bool)
	for _, lang := range translator.AvailableLanguages() {
		available[lang] = true
	}

	return func(c *qi.Context) {
		lang := resolveLanguage(c, cfg, available)

		// 未识别到语言时使用翻译器默认语言
		if lang == "" {
			lang = translator.GetLanguage(c.RequestContext())
		}

		// 设置到 qi.Context（供 helper.go 的 GetContextLanguage 使用）
		qi.SetContextLanguage(c, lang)

		// 设置到 request context（供 i18n.Translator.T 使用）
		ctx := i18n.WithLanguage(c.RequestContext(), lang)
		c.SetRequestContext(ctx)

		// 可选：写入 Cookie（仅当 lang 值合法时）
		if cfg.SetCookie && c.Query(cfg.QueryKey) != "" && isValidLang(lang) {
			c.Header("Set-Cookie", fmt.Sprintf("%s=%s; Path=/; Max-Age=%d; HttpOnly; SameSite=Lax", cfg.CookieKey, lang, cfg.CookieMaxAge))
		}

		c.Next()
	}
}

// resolveLanguage 从请求中解析语言
// 优先级：Query > Cookie > Accept-Language
func resolveLanguage(c *qi.Context, cfg *I18nConfig, available map[string]bool) string {
	// 1. Query 参数
	if lang := c.Query(cfg.QueryKey); lang != "" && available[lang] {
		return lang
	}

	// 2. Cookie
	if lang := c.GetHeader("Cookie"); lang != "" {
		if parsed := parseCookieValue(lang, cfg.CookieKey); parsed != "" && available[parsed] {
			return parsed
		}
	}

	// 3. Accept-Language
	if accept := c.GetHeader(cfg.HeaderKey); accept != "" {
		if lang := matchAcceptLanguage(accept, available); lang != "" {
			return lang
		}
	}

	return ""
}

// matchAcceptLanguage 解析 Accept-Language 并匹配可用语言
// 格式: "zh-CN,zh;q=0.9,en;q=0.8"
func matchAcceptLanguage(accept string, available map[string]bool) string {
	for _, part := range strings.Split(accept, ",") {
		lang := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])

		// 精确匹配
		if available[lang] {
			return lang
		}

		// 前缀匹配：zh -> zh-CN
		prefix := strings.SplitN(lang, "-", 2)[0]
		for supported := range available {
			if strings.HasPrefix(supported, prefix) {
				return supported
			}
		}
	}
	return ""
}

// parseCookieValue 从 Cookie header 中提取指定 key 的值
func parseCookieValue(cookie, key string) string {
	for _, part := range strings.Split(cookie, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}

// isValidLang 校验语言标签是否合法（防止 Cookie 注入）
func isValidLang(lang string) bool {
	for _, c := range lang {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return len(lang) > 0
}
