package i18n

import "context"

// ContextKey Context Key 类型
type ContextKey string

// LanguageKey Context 中存储语言的 Key
const LanguageKey ContextKey = "i18n_language"

// GetLanguageFromContext 从 Context 获取语言
func GetLanguageFromContext(ctx context.Context) string {
	if lang, ok := ctx.Value(LanguageKey).(string); ok {
		return lang
	}
	return ""
}

// WithLanguage 创建带语言的 Context
func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, LanguageKey, lang)
}
