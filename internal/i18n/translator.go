package i18n

// Translator 绑定到具体语言，提供翻译方法，并发安全（只读）。
// 通过 Bundle.For() 或 Bundle.ForRequest() 获取，不要自行构造。
type Translator struct {
	bundle *Bundle
	lang   string // 小写 language tag
}

// T 翻译 key，args 为交替的 string key-value 对。
//
//	t.T("welcome", "name", "Alice")  → "欢迎, Alice！"
//
// 找不到 key 时返回 key 原文，未提供的 {placeholder} 保留原样。
func (t *Translator) T(key string, args ...any) string {
	msg := t.bundle.lookup(t.lang, key, PluralOther)
	if msg == "" {
		return key
	}
	return interpolate(msg, args)
}

// N 带数量的复数翻译。count 用于选择 PluralForm，并自动注入为 {count} 插值参数。
//
//	t.N("items_count", 3)  → "3 个项目"
//	t.N("items_count", 1)  → "1 个项目"
func (t *Translator) N(key string, count int, args ...any) string {
	form := t.bundle.pluralForm(t.lang, count)
	msg := t.bundle.lookup(t.lang, key, form)
	if msg == "" {
		return key
	}
	// count 前置注入为 {count}；用户 args 中相同 key 无效（{count} 已被替换）
	allArgs := make([]any, 0, 2+len(args))
	allArgs = append(allArgs, "count", count)
	allArgs = append(allArgs, args...)
	return interpolate(msg, allArgs)
}

// Lang 返回当前语言 tag（小写）。
func (t *Translator) Lang() string {
	return t.lang
}
