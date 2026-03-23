package qi

import "io"

// LoggerConfig 请求日志中间件配置
type LoggerConfig struct {
	Output    io.Writer // 日志输出目标，nil 时默认 os.Stdout
	SkipPaths []string  // 跳过日志记录的路径，如 ["/ping", "/health"]
}

// WithLogger 配置请求日志中间件。
// 输出格式：[QI] 2006/01/02 - 15:04:05 |  200 |       917ns |  127.0.0.1 | GET     "/path" trace_id
func WithLogger(cfg *LoggerConfig) Option {
	return func(c *Config) {
		c.loggerConfig = cfg
	}
}
