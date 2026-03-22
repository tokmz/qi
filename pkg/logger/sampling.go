package logger

// SamplingConfig 采样配置
type SamplingConfig struct {
	Initial    int // 每秒前 N 条日志必定记录
	Thereafter int // 之后每 M 条记录 1 条
}

// setDefaults 设置默认值
func (s *SamplingConfig) setDefaults() {
	if s.Initial == 0 {
		s.Initial = 100
	}
	if s.Thereafter == 0 {
		s.Thereafter = 100
	}
}
