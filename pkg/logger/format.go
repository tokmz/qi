package logger

// Format 日志格式
type Format string

const (
	// JSONFormat JSON 格式（生产环境推荐）
	JSONFormat Format = "json"
	// ConsoleFormat 控制台格式（开发环境推荐）
	ConsoleFormat Format = "console"
)

// String 返回格式名称
func (f Format) String() string {
	return string(f)
}

// IsValid 检查格式是否有效
func (f Format) IsValid() bool {
	return f == JSONFormat || f == ConsoleFormat
}
