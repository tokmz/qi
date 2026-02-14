package i18n

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Translator 翻译器接口
type Translator interface {
	// T 获取翻译（支持变量替换）
	T(ctx context.Context, key string, args ...any) string

	// Tn 获取翻译（支持复数形式）
	Tn(ctx context.Context, key, plural string, n int, args ...any) string

	// GetLanguage 获取当前语言
	GetLanguage(ctx context.Context) string

	// AvailableLanguages 获取可用语言列表
	AvailableLanguages() []string

	// HasKey 检查 key 是否存在
	HasKey(key string) bool

	// Preload 预加载指定语言
	Preload(languages ...string) error
}

// translator 翻译器实现
type translator struct {
	config       *Config
	loader       Loader
	translations map[string]map[string]string
	mu           sync.RWMutex
	languages    []string
	defaultLang  string
	varLeft      string
	varRight     string
}

// 确保 translator 实现 Translator 接口
var _ Translator = (*translator)(nil)

// New 创建翻译器
func New(cfg *Config) (Translator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	loader := cfg.Loader
	if loader == nil {
		loader = &JSONLoader{
			Dir:     cfg.Dir,
			Pattern: cfg.Pattern,
		}
	}

	t := &translator{
		config:       cfg,
		loader:       loader,
		translations: make(map[string]map[string]string),
		languages:    cfg.Languages,
		defaultLang:  cfg.DefaultLanguage,
		varLeft:      cfg.VarLeft,
		varRight:     cfg.VarRight,
	}

	if !cfg.Lazy {
		if err := t.Preload(cfg.Languages...); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// NewWithOptions 使用 Options 模式创建翻译器
func NewWithOptions(opts ...Option) (Translator, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return New(cfg)
}

// MustNew 创建翻译器（失败时 panic）
func MustNew(cfg *Config) Translator {
	t, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// MustNewWithOptions 使用 Options 创建翻译器（失败时 panic）
func MustNewWithOptions(opts ...Option) Translator {
	t, err := NewWithOptions(opts...)
	if err != nil {
		panic(err)
	}
	return t
}

// Preload 预加载指定语言
func (t *translator) Preload(languages ...string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.loadLanguages(languages...)
}

// loadLanguages 加载语言（调用方需持有写锁）
func (t *translator) loadLanguages(languages ...string) error {
	for _, lang := range languages {
		if _, ok := t.translations[lang]; ok {
			continue
		}

		data, err := t.loader.Load(context.Background(), t.config.Dir, []string{lang})
		if err != nil {
			return err
		}

		if translations, ok := data[lang]; ok {
			t.translations[lang] = translations
		}
	}
	return nil
}

// ensureLoaded 确保语言已加载（懒加载支持）
func (t *translator) ensureLoaded(lang string) {
	t.mu.RLock()
	_, ok := t.translations[lang]
	t.mu.RUnlock()
	if ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	// 双重检查
	if _, ok := t.translations[lang]; ok {
		return
	}
	_ = t.loadLanguages(lang)
}

// T 获取翻译（支持变量替换）
func (t *translator) T(ctx context.Context, key string, args ...any) string {
	lang := t.GetLanguage(ctx)
	t.ensureLoaded(lang)
	return t.translate(lang, key, args...)
}

// Tn 获取翻译（支持复数形式）
func (t *translator) Tn(ctx context.Context, key, plural string, n int, args ...any) string {
	useKey := key
	if n != 1 {
		useKey = plural
	}
	args = append([]any{"Count", n}, args...)
	return t.T(ctx, useKey, args...)
}

// translate 执行翻译
func (t *translator) translate(lang, key string, args ...any) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	translations, ok := t.translations[lang]
	if !ok {
		if lang != t.defaultLang {
			translations, ok = t.translations[t.defaultLang]
		}
		if !ok {
			return key
		}
	}

	value, ok := translations[key]
	if !ok {
		return key
	}

	if len(args) > 0 {
		value = t.replaceVariables(value, args...)
	}

	return value
}

// replaceVariables 替换变量
func (t *translator) replaceVariables(value string, args ...any) string {
	if len(args)%2 != 0 {
		return value
	}

	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		placeholder := t.varLeft + "." + key + t.varRight
		value = strings.ReplaceAll(value, placeholder, fmt.Sprintf("%v", args[i+1]))
	}

	return value
}

// GetLanguage 获取当前语言
func (t *translator) GetLanguage(ctx context.Context) string {
	if lang, ok := ctx.Value(LanguageKey).(string); ok && lang != "" {
		return lang
	}
	return t.defaultLang
}

// AvailableLanguages 获取可用语言列表
func (t *translator) AvailableLanguages() []string {
	return slices.Clone(t.languages)
}

// HasKey 检查 key 是否存在
func (t *translator) HasKey(key string) bool {
	t.ensureLoaded(t.defaultLang)

	t.mu.RLock()
	defer t.mu.RUnlock()

	translations, ok := t.translations[t.defaultLang]
	if !ok {
		return false
	}

	_, ok = translations[key]
	return ok
}
