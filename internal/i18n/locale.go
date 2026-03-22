package i18n

import (
	"encoding/json"
	"fmt"
)

// entry 持有单条翻译的所有复数形式。
// JSON 解析使用两步策略：先尝试 string，失败再解析为对象。
type entry struct {
	Zero  string // 数量为 0 时（可选，省略时降级到 Other）
	One   string // 数量为 1 时（可选，省略时降级到 Other）
	Other string // 其他数量 / 纯字符串形式
}

// get 根据 PluralForm 返回对应文本，找不到时降级到 Other。
// 若 Other 也为空（翻译文件缺少该字段），返回空字符串，
// Bundle.lookup 将视此为 key 缺失并继续回退链。
func (e entry) get(form PluralForm) string {
	switch form {
	case PluralZero:
		if e.Zero != "" {
			return e.Zero
		}
	case PluralOne:
		if e.One != "" {
			return e.One
		}
	}
	return e.Other
}

// Locale 持有单一语言的所有翻译 key-value。
// 由 Bundle 在持有写锁时通过 merge 填充，读取时并发安全。
type Locale struct {
	tag      string           // 小写 BCP 47 language tag，如 "zh-cn"
	messages map[string]entry // key → entry
}

func newLocale(tag string) *Locale {
	return &Locale{
		tag:      tag,
		messages: make(map[string]entry),
	}
}

// merge 将 data 中的翻译合并进 Locale，同 key 后加载的覆盖先前值。
func (l *Locale) merge(data map[string]json.RawMessage) error {
	for key, raw := range data {
		e, err := parseEntry(raw)
		if err != nil {
			return fmt.Errorf("i18n: locale %q key %q: %w", l.tag, key, err)
		}
		l.messages[key] = e
	}
	return nil
}

// lookup 返回指定 key 对应 PluralForm 的文本，找不到 key 时返回空字符串。
func (l *Locale) lookup(key string, form PluralForm) string {
	e, ok := l.messages[key]
	if !ok {
		return ""
	}
	return e.get(form)
}

// parseEntry 将 json.RawMessage 解析为 entry。
// 优先尝试纯字符串形式，失败则解析为复数对象。
func parseEntry(raw json.RawMessage) (entry, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return entry{Other: s}, nil
	}
	var obj struct {
		Zero  string `json:"zero"`
		One   string `json:"one"`
		Other string `json:"other"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return entry{}, fmt.Errorf("expected string or plural object: %w", err)
	}
	return entry{Zero: obj.Zero, One: obj.One, Other: obj.Other}, nil
}
