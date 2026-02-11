package logger

import "go.uber.org/zap/zapcore"

// Option 配置选项函数
type Option func(*Config)

// WithLevel 设置日志级别
func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat 设置日志格式
func WithFormat(format Format) Option {
	return func(c *Config) {
		c.Format = format
	}
}

// WithConsoleOutput 启用控制台输出
func WithConsoleOutput() Option {
	return func(c *Config) {
		c.Console = true
	}
}

// WithFileOutput 设置文件输出
func WithFileOutput(filename string) Option {
	return func(c *Config) {
		c.File = filename
	}
}

// WithRotateOutput 设置文件轮转输出
func WithRotateOutput(config *RotateConfig) Option {
	return func(c *Config) {
		c.Rotate = config
	}
}

// WithSampling 设置采样配置
func WithSampling(config *SamplingConfig) Option {
	return func(c *Config) {
		c.Sampling = config
	}
}

// WithBufferSize 设置缓冲区大小
func WithBufferSize(size int) Option {
	return func(c *Config) {
		c.BufferSize = size
	}
}

// WithCaller 设置是否记录调用位置
func WithCaller(enable bool) Option {
	return func(c *Config) {
		c.EnableCaller = enable
	}
}

// WithStacktrace 设置是否记录堆栈
func WithStacktrace(enable bool) Option {
	return func(c *Config) {
		c.EnableStacktrace = enable
	}
}

// WithEncoderConfig 设置自定义 Encoder 配置
func WithEncoderConfig(config *zapcore.EncoderConfig) Option {
	return func(c *Config) {
		c.EncoderConfig = config
	}
}

// WithHook 添加 Hook
func WithHook(hook Hook) Option {
	return func(c *Config) {
		c.Hooks = append(c.Hooks, hook)
	}
}
