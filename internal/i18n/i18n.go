package i18n

import (
	"io/fs"
	"net/http"
	"strings"
	"sync"
)

// Bundle 持有所有语言的翻译数据，全生命周期共享一个实例。
//
// 线程安全约定：
//   - LoadFS / LoadDir 仅在服务启动阶段调用，调用期间不得并发读取。
//   - 加载完成后所有操作只读，For / ForRequest 并发安全。
//   - 不提供运行时热重载；需要热重载请替换整个 Bundle 实例（原子指针）。
type Bundle struct {
	mu          sync.RWMutex
	locales     map[string]*Locale  // 小写 language tag → Locale
	fallback    string              // 默认回退语言，默认 "en"
	detector    Detector            // 请求语言检测策略
	pluralRules map[string]PluralFunc
}

// BundleOption 是 Bundle 的函数式配置选项。
type BundleOption func(*Bundle)

// WithFallback 设置回退语言，默认 "en"。lang 大小写不敏感。
func WithFallback(lang string) BundleOption {
	return func(b *Bundle) {
		b.fallback = strings.ToLower(lang)
	}
}

// WithDetector 替换默认语言检测策略。
func WithDetector(d Detector) BundleOption {
	return func(b *Bundle) {
		if d != nil {
			b.detector = d
		}
	}
}

// WithPluralRule 注册语言的复数规则，lang 大小写不敏感。
func WithPluralRule(lang string, fn PluralFunc) BundleOption {
	return func(b *Bundle) {
		if fn != nil {
			b.pluralRules[strings.ToLower(lang)] = fn
		}
	}
}

// NewBundle 创建 Bundle，应用 opts 后返回。
func NewBundle(opts ...BundleOption) *Bundle {
	b := &Bundle{
		locales:     make(map[string]*Locale),
		fallback:    "en",
		detector:    DefaultDetector(),
		pluralRules: make(map[string]PluralFunc),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// LoadFS 从 fs.FS 加载匹配 glob 的翻译文件。
// glob 仅支持单层通配符（如 "locales/*.json"），不支持 ** 递归。
// 多次调用时，后加载的同 key 值覆盖先前值。
// 解析在加锁前完成，加锁后原子写入，避免部分更新污染 Bundle 状态。
func (b *Bundle) LoadFS(fsys fs.FS, glob string) error {
	// 1. 在锁外完成所有 IO 与解析，失败直接返回，Bundle 不受影响
	files, err := loadFromFS(fsys, glob)
	if err != nil {
		return err
	}
	// 2. 预构建所有 Locale，失败同样不影响 Bundle
	newLocales := make(map[string]*Locale, len(files))
	for tag, data := range files {
		l := newLocale(tag)
		if err := l.merge(data); err != nil {
			return err
		}
		newLocales[tag] = l
	}
	// 3. 加锁后原子合并：已有 Locale 逐 key 覆盖，新 tag 直接写入
	b.mu.Lock()
	defer b.mu.Unlock()
	for tag, src := range newLocales {
		if dst, ok := b.locales[tag]; ok {
			for k, e := range src.messages {
				dst.messages[k] = e
			}
		} else {
			b.locales[tag] = src
		}
	}
	return nil
}

// LoadDir 从操作系统目录加载翻译文件，等价于 LoadFS(os.DirFS(dir), "*.json")。
func (b *Bundle) LoadDir(dir string) error {
	files, err := loadFromDir(dir)
	if err != nil {
		return err
	}
	newLocales := make(map[string]*Locale, len(files))
	for tag, data := range files {
		l := newLocale(tag)
		if err := l.merge(data); err != nil {
			return err
		}
		newLocales[tag] = l
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for tag, src := range newLocales {
		if dst, ok := b.locales[tag]; ok {
			for k, e := range src.messages {
				dst.messages[k] = e
			}
		} else {
			b.locales[tag] = src
		}
	}
	return nil
}

// For 根据语言 tag 返回 Translator，tag 大小写不敏感。
// 找不到对应语言时回退到 Bundle.fallback；仍找不到时 Translator.T 返回 key 原文。
func (b *Bundle) For(lang string) *Translator {
	return &Translator{bundle: b, lang: strings.ToLower(lang)}
}

// ForRequest 从 http.Request 检测语言后调用 For。
// Detector 返回空字符串时自动降级到 Bundle.fallback。
func (b *Bundle) ForRequest(r *http.Request) *Translator {
	lang := b.detector.Detect(r)
	if lang == "" {
		lang = b.fallback
	}
	return b.For(lang)
}

// lookup 按回退链查找 key：lang → 父语言 → fallback → ""。
func (b *Bundle) lookup(lang, key string, form PluralForm) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	parent := parentLang(lang)

	// 1. 精确匹配
	if l, ok := b.locales[lang]; ok {
		if msg := l.lookup(key, form); msg != "" {
			return msg
		}
	}

	// 2. 父语言（BCP 47 第一段）
	if parent != lang {
		if l, ok := b.locales[parent]; ok {
			if msg := l.lookup(key, form); msg != "" {
				return msg
			}
		}
	}

	// 3. fallback
	if b.fallback != lang && b.fallback != parent {
		if l, ok := b.locales[b.fallback]; ok {
			if msg := l.lookup(key, form); msg != "" {
				return msg
			}
		}
	}

	return ""
}

// pluralForm 根据 lang 和 count 返回 PluralForm，未注册规则时使用 SimplePluralFunc。
func (b *Bundle) pluralForm(lang string, count int) PluralForm {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if fn, ok := b.pluralRules[lang]; ok {
		return fn(count)
	}
	if parent := parentLang(lang); parent != lang {
		if fn, ok := b.pluralRules[parent]; ok {
			return fn(count)
		}
	}
	return SimplePluralFunc(count)
}

// parentLang 返回 BCP 47 tag 的第一段（仅切一层）。
// "zh-cn" → "zh"，"en" → "en"，"zh-hant-tw" → "zh"。
func parentLang(lang string) string {
	if idx := strings.IndexByte(lang, '-'); idx > 0 {
		return lang[:idx]
	}
	return lang
}
