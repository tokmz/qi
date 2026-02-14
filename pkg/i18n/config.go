package i18n

// Config i18n 配置
type Config struct {
	// 默认语言（默认：zh-CN）
	DefaultLanguage string
	// 支持的语言列表
	Languages []string
	// 翻译文件目录
	Dir string
	// 文件名模式（默认：{lang}.json）
	Pattern string
	// 自定义加载器
	Loader Loader
	// 是否启用懒加载（默认：true）
	Lazy bool
	// 变量替换分隔符（默认：{{ 和 }}）
	VarLeft  string
	VarRight string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DefaultLanguage: "zh-CN",
		Languages:       []string{"zh-CN", "en-US"},
		Pattern:         "{lang}.json",
		Lazy:            true,
		VarLeft:         "{{",
		VarRight:        "}}",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.DefaultLanguage == "" {
		c.DefaultLanguage = "zh-CN"
	}
	if c.Pattern == "" {
		c.Pattern = "{lang}.json"
	}
	if len(c.Languages) == 0 {
		c.Languages = []string{c.DefaultLanguage}
	}
	if c.VarLeft == "" {
		c.VarLeft = "{{"
	}
	if c.VarRight == "" {
		c.VarRight = "}}"
	}
	return nil
}

// Option 配置选项
type Option func(*Config)

// WithDefaultLanguage 设置默认语言
func WithDefaultLanguage(lang string) Option {
	return func(c *Config) {
		c.DefaultLanguage = lang
	}
}

// WithLanguages 设置支持的语言列表
func WithLanguages(langs ...string) Option {
	return func(c *Config) {
		c.Languages = langs
	}
}

// WithDir 设置翻译文件目录
func WithDir(dir string) Option {
	return func(c *Config) {
		c.Dir = dir
	}
}

// WithPattern 设置文件名模式
func WithPattern(pattern string) Option {
	return func(c *Config) {
		c.Pattern = pattern
	}
}

// WithLoader 设置加载器
func WithLoader(loader Loader) Option {
	return func(c *Config) {
		c.Loader = loader
	}
}

// WithLazy 设置是否启用懒加载
func WithLazy(lazy bool) Option {
	return func(c *Config) {
		c.Lazy = lazy
	}
}

// WithVarDelimiters 设置变量替换分隔符
func WithVarDelimiters(left, right string) Option {
	return func(c *Config) {
		c.VarLeft = left
		c.VarRight = right
	}
}