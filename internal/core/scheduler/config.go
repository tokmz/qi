// Package scheduler 提供基于 Cron v3 的定时任务调度功能
package scheduler

import (
	"time"
)

// Config 定时任务调度器配置
type Config struct {
	// 是否启用定时任务
	Enabled bool `mapstructure:"enabled" json:"enabled"`

	// 是否启用秒字段（6 字段格式）
	// true: 秒 分 时 日 月 周（6 字段）
	// false: 分 时 日 月 周（5 字段，标准格式）
	WithSeconds bool `mapstructure:"with_seconds" json:"with_seconds"`

	// 时区设置
	// 例如: "Asia/Shanghai", "UTC", "America/New_York"
	Timezone string `mapstructure:"timezone" json:"timezone"`

	// 任务列表
	Jobs []JobConfig `mapstructure:"jobs" json:"jobs"`

	// 任务执行超时时间（可选）
	DefaultTimeout time.Duration `mapstructure:"default_timeout" json:"default_timeout"`

	// 是否启用任务并发控制（防止同一任务重复执行）
	SkipIfStillRunning bool `mapstructure:"skip_if_still_running" json:"skip_if_still_running"`

	// 是否启用 panic 恢复
	RecoverPanic bool `mapstructure:"recover_panic" json:"recover_panic"`

	// 任务执行失败重试次数
	RetryCount int `mapstructure:"retry_count" json:"retry_count"`

	// 任务执行失败重试间隔
	RetryInterval time.Duration `mapstructure:"retry_interval" json:"retry_interval"`
}

// JobConfig 任务配置
type JobConfig struct {
	// 任务名称（唯一标识）
	Name string `mapstructure:"name" json:"name"`

	// Cron 表达式
	// 5 字段格式: "分 时 日 月 周"
	// 6 字段格式: "秒 分 时 日 月 周"
	Spec string `mapstructure:"spec" json:"spec"`

	// 是否启用该任务
	Enabled bool `mapstructure:"enabled" json:"enabled"`

	// 任务描述
	Description string `mapstructure:"description" json:"description"`

	// 任务超时时间（覆盖默认超时）
	Timeout time.Duration `mapstructure:"timeout" json:"timeout"`

	// 任务执行失败重试次数（覆盖默认重试次数）
	RetryCount int `mapstructure:"retry_count" json:"retry_count"`

	// 任务参数（可选，JSON 格式）
	Params map[string]interface{} `mapstructure:"params" json:"params"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:            true,
		WithSeconds:        true,
		Timezone:           "Asia/Shanghai",
		DefaultTimeout:     30 * time.Minute,
		SkipIfStillRunning: true,
		RecoverPanic:       true,
		RetryCount:         0,
		RetryInterval:      5 * time.Second,
		Jobs:               []JobConfig{},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	// 验证时区
	if c.Timezone == "" {
		return ErrInvalidTimezone
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return ErrInvalidTimezone
	}

	// 验证任务配置
	jobNames := make(map[string]bool)
	for _, job := range c.Jobs {
		// 检查任务名称是否重复
		if jobNames[job.Name] {
			return ErrDuplicateJobName
		}
		jobNames[job.Name] = true

		// 验证任务配置
		if err := job.Validate(c.WithSeconds); err != nil {
			return err
		}
	}

	return nil
}

// Validate 验证任务配置
func (j *JobConfig) Validate(withSeconds bool) error {
	if !j.Enabled {
		return nil
	}

	if j.Name == "" {
		return ErrInvalidJobName
	}

	if j.Spec == "" {
		return ErrInvalidCronSpec
	}

	// 简单验证 cron 表达式格式
	// 实际解析由 cron 库完成
	return nil
}

// GetEnabledJobs 获取所有启用的任务
func (c *Config) GetEnabledJobs() []JobConfig {
	var enabled []JobConfig
	for _, job := range c.Jobs {
		if job.Enabled {
			enabled = append(enabled, job)
		}
	}
	return enabled
}

// GetJobByName 根据名称获取任务配置
func (c *Config) GetJobByName(name string) *JobConfig {
	for i := range c.Jobs {
		if c.Jobs[i].Name == name {
			return &c.Jobs[i]
		}
	}
	return nil
}

