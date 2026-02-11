package logger

// RotateConfig 文件轮转配置
type RotateConfig struct {
	Filename   string // 日志文件路径
	MaxSize    int    // 单文件最大大小（MB，默认 100MB）
	MaxAge     int    // 文件保留天数（默认 30 天）
	MaxBackups int    // 最多保留文件数（默认 10 个）
	LocalTime  bool   // 使用本地时间（默认 true）
	Compress   bool   // 是否压缩（默认 false）
}

// setDefaults 设置默认值
func (r *RotateConfig) setDefaults() {
	if r.MaxSize == 0 {
		r.MaxSize = 100 // 100MB
	}
	if r.MaxAge == 0 {
		r.MaxAge = 30 // 30 天
	}
	if r.MaxBackups == 0 {
		r.MaxBackups = 10 // 10 个文件
	}
	r.LocalTime = true
}
