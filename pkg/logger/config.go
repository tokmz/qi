package logger

import "go.uber.org/zap/zapcore"

// Config 日志配置
type Config struct {
	// 基础配置
	Level  Level  // 日志级别（默认 InfoLevel）
	Format Format // 日志格式（json/console，默认 json）

	// 输出配置
	Console bool          // 是否输出到控制台（默认 true）
	File    string        // 文件路径（空则不输出到文件）
	Rotate  *RotateConfig // 轮转配置（nil 则不轮转）

	// 性能配置
	Sampling   *SamplingConfig // 采样配置（nil 则不采样）
	BufferSize int             // 缓冲区大小（默认 256KB）

	// 功能配置
	EnableCaller     bool // 是否记录调用位置（默认 true）
	EnableStacktrace bool // 是否记录堆栈（Error 及以上，默认 true）

	// 扩展配置
	EncoderConfig *zapcore.EncoderConfig // 自定义 Encoder 配置
	Hooks         []Hook                 // Hook 列表
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	if c.Level == 0 {
		c.Level = InfoLevel
	}
	if c.Format == "" {
		c.Format = JSONFormat
	}
	if c.BufferSize == 0 {
		c.BufferSize = 256 * 1024 // 256KB
	}
	// 默认启用控制台输出
	if !c.Console && c.File == "" && c.Rotate == nil {
		c.Console = true
	}
	// 默认启用 Caller 和 Stacktrace
	c.EnableCaller = true
	c.EnableStacktrace = true
}
