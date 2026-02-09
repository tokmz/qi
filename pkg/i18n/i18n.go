package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Config 国际化配置
type Config struct {
	// DefaultLanguage 默认语言
	DefaultLanguage string

	// FallbackLanguage 回退语言（翻译不存在时使用）
	FallbackLanguage string

	// SupportedLanguages 支持的语言列表
	SupportedLanguages []string
}

// I18n 国际化实例
type I18n struct {
	config       Config
	translations map[string]map[string]string // [language][key]value
	mu           sync.RWMutex
}

var (
	// globalI18n 全局实例
	globalI18n *I18n
	mu         sync.RWMutex
	once       sync.Once
)

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		DefaultLanguage:    "en",
		FallbackLanguage:   "en",
		SupportedLanguages: []string{"en", "zh-CN"},
	}
}

// New 创建国际化实例
func New(config Config) *I18n {
	return &I18n{
		config:       config,
		translations: make(map[string]map[string]string),
	}
}

// Init 初始化全局实例
func Init(config Config) {
	mu.Lock()
	defer mu.Unlock()
	globalI18n = New(config)
}

// Global 获取全局实例（并发安全，使用 sync.Once 确保只初始化一次）
func Global() *I18n {
	mu.RLock()
	if globalI18n != nil {
		i := globalI18n
		mu.RUnlock()
		return i
	}
	mu.RUnlock()

	// 未初始化，使用 sync.Once 确保只初始化一次
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		if globalI18n == nil {
			globalI18n = New(DefaultConfig())
		}
	})

	return globalI18n
}

// LoadFromMap 从 map 加载翻译（自动添加语言到支持列表）
func (i *I18n) LoadFromMap(lang string, translations map[string]string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 自动添加语言到支持列表
	if !i.isSupportedUnsafe(lang) {
		i.config.SupportedLanguages = append(i.config.SupportedLanguages, lang)
	}

	if i.translations[lang] == nil {
		i.translations[lang] = make(map[string]string)
	}

	for key, value := range translations {
		i.translations[lang][key] = value
	}

	return nil
}

// isSupportedUnsafe 检查语言是否支持（不加锁，内部使用）
func (i *I18n) isSupportedUnsafe(lang string) bool {
	for _, l := range i.config.SupportedLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// LoadFromJSON 从 JSON 字节加载翻译
func (i *I18n) LoadFromJSON(lang string, data []byte) error {
	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return i.LoadFromMap(lang, translations)
}

// LoadFromFile 从文件加载翻译
func (i *I18n) LoadFromFile(lang string, filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return i.LoadFromJSON(lang, data)
}

// T 翻译（支持格式化参数）
func (i *I18n) T(lang, key string, args ...any) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 尝试获取指定语言的翻译
	if langMap, ok := i.translations[lang]; ok {
		if value, ok := langMap[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(value, args...)
			}
			return value
		}
	}

	// 回退到默认语言
	if lang != i.config.FallbackLanguage {
		if langMap, ok := i.translations[i.config.FallbackLanguage]; ok {
			if value, ok := langMap[key]; ok {
				if len(args) > 0 {
					return fmt.Sprintf(value, args...)
				}
				return value
			}
		}
	}

	// 翻译不存在，返回 key
	return key
}

// Translate 翻译（T 的别名）
func (i *I18n) Translate(lang, key string, args ...any) string {
	return i.T(lang, key, args...)
}

// MustT 翻译（不存在时 panic）
func (i *I18n) MustT(lang, key string, args ...any) string {
	result := i.T(lang, key, args...)

	// 如果返回的是 key 本身，说明翻译不存在
	if result == key {
		i.mu.RLock()
		defer i.mu.RUnlock()

		// 再次确认翻译确实不存在（避免 key 恰好等于翻译值的情况）
		if _, ok := i.translations[lang]; !ok {
			panic(fmt.Sprintf("translation not found: lang=%s, key=%s", lang, key))
		}
		if _, ok := i.translations[lang][key]; !ok {
			if lang != i.config.FallbackLanguage {
				if _, ok := i.translations[i.config.FallbackLanguage]; !ok {
					panic(fmt.Sprintf("translation not found: lang=%s, key=%s", lang, key))
				}
				if _, ok := i.translations[i.config.FallbackLanguage][key]; !ok {
					panic(fmt.Sprintf("translation not found: lang=%s, key=%s", lang, key))
				}
			} else {
				panic(fmt.Sprintf("translation not found: lang=%s, key=%s", lang, key))
			}
		}
	}

	return result
}

// AddLanguage 添加支持的语言
func (i *I18n) AddLanguage(lang string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 检查是否已存在
	for _, l := range i.config.SupportedLanguages {
		if l == lang {
			return
		}
	}

	i.config.SupportedLanguages = append(i.config.SupportedLanguages, lang)
}

// IsSupported 检查语言是否支持
func (i *I18n) IsSupported(lang string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, l := range i.config.SupportedLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// GetSupportedLanguages 获取支持的语言列表
func (i *I18n) GetSupportedLanguages() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	langs := make([]string, len(i.config.SupportedLanguages))
	copy(langs, i.config.SupportedLanguages)
	return langs
}

// GetDefaultLanguage 获取默认语言
func (i *I18n) GetDefaultLanguage() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.config.DefaultLanguage
}

// ============ 全局快捷方法 ============

// T 全局翻译
func T(lang, key string, args ...any) string {
	return Global().T(lang, key, args...)
}

// Translate 全局翻译（T 的别名）
func Translate(lang, key string, args ...any) string {
	return Global().T(lang, key, args...)
}

// MustT 全局翻译（不存在时 panic）
func MustT(lang, key string, args ...any) string {
	return Global().MustT(lang, key, args...)
}

// LoadFromMap 全局加载翻译
func LoadFromMap(lang string, translations map[string]string) error {
	return Global().LoadFromMap(lang, translations)
}

// LoadFromJSON 全局加载 JSON 翻译
func LoadFromJSON(lang string, data []byte) error {
	return Global().LoadFromJSON(lang, data)
}

// LoadFromFile 全局加载文件翻译
func LoadFromFile(lang string, filepath string) error {
	return Global().LoadFromFile(lang, filepath)
}

// AddLanguage 全局添加语言
func AddLanguage(lang string) {
	Global().AddLanguage(lang)
}

// IsSupported 全局检查语言支持
func IsSupported(lang string) bool {
	return Global().IsSupported(lang)
}

// GetSupportedLanguages 全局获取支持的语言列表
func GetSupportedLanguages() []string {
	return Global().GetSupportedLanguages()
}

// GetDefaultLanguage 全局获取默认语言
func GetDefaultLanguage() string {
	return Global().GetDefaultLanguage()
}
