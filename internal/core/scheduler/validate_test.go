package scheduler

import (
	"testing"
)

func TestValidateCronSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		withSeconds bool
		wantErr     bool
	}{
		// 6 字段格式测试（带秒）
		{
			name:        "valid 6-field every second",
			spec:        "* * * * * *",
			withSeconds: true,
			wantErr:     false,
		},
		{
			name:        "valid 6-field every 5 seconds",
			spec:        "*/5 * * * * *",
			withSeconds: true,
			wantErr:     false,
		},
		{
			name:        "valid 6-field daily at 2am",
			spec:        "0 0 2 * * *",
			withSeconds: true,
			wantErr:     false,
		},
		{
			name:        "valid 6-field every 30 minutes",
			spec:        "0 */30 * * * *",
			withSeconds: true,
			wantErr:     false,
		},

		// 5 字段格式测试（标准）
		{
			name:        "valid 5-field every hour",
			spec:        "0 * * * *",
			withSeconds: false,
			wantErr:     false,
		},
		{
			name:        "valid 5-field daily at midnight",
			spec:        "0 0 * * *",
			withSeconds: false,
			wantErr:     false,
		},
		{
			name:        "valid 5-field every monday 9am",
			spec:        "0 9 * * 1",
			withSeconds: false,
			wantErr:     false,
		},

		// 预定义格式测试
		{
			name:        "predefined @hourly",
			spec:        "@hourly",
			withSeconds: false,
			wantErr:     false,
		},
		{
			name:        "predefined @daily",
			spec:        "@daily",
			withSeconds: false,
			wantErr:     false,
		},
		{
			name:        "predefined @every 5m",
			spec:        "@every 5m",
			withSeconds: false,
			wantErr:     false,
		},

		// 错误格式测试
		{
			name:        "empty spec",
			spec:        "",
			withSeconds: false,
			wantErr:     true,
		},
		{
			name:        "wrong field count - 6 fields for 5-field mode",
			spec:        "0 0 0 * * *",
			withSeconds: false,
			wantErr:     true,
		},
		{
			name:        "wrong field count - 5 fields for 6-field mode",
			spec:        "0 0 * * *",
			withSeconds: true,
			wantErr:     true,
		},
		{
			name:        "invalid cron syntax",
			spec:        "invalid cron",
			withSeconds: false,
			wantErr:     true,
		},
		{
			name:        "invalid field value",
			spec:        "0 0 99 * *",
			withSeconds: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronSpec(tt.spec, tt.withSeconds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCronSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		job         *JobConfig
		withSeconds bool
		wantErr     bool
	}{
		{
			name: "valid job config",
			job: &JobConfig{
				Name:    "test-job",
				Spec:    "0 0 * * *",
				Enabled: true,
			},
			withSeconds: false,
			wantErr:     false,
		},
		{
			name: "disabled job - skip validation",
			job: &JobConfig{
				Name:    "",
				Spec:    "",
				Enabled: false,
			},
			withSeconds: false,
			wantErr:     false,
		},
		{
			name: "empty job name",
			job: &JobConfig{
				Name:    "",
				Spec:    "0 0 * * *",
				Enabled: true,
			},
			withSeconds: false,
			wantErr:     true,
		},
		{
			name: "empty cron spec",
			job: &JobConfig{
				Name:    "test-job",
				Spec:    "",
				Enabled: true,
			},
			withSeconds: false,
			wantErr:     true,
		},
		{
			name: "invalid cron spec",
			job: &JobConfig{
				Name:    "test-job",
				Spec:    "invalid",
				Enabled: true,
			},
			withSeconds: false,
			wantErr:     true,
		},
		{
			name: "negative timeout",
			job: &JobConfig{
				Name:    "test-job",
				Spec:    "0 0 * * *",
				Enabled: true,
				Timeout: -1,
			},
			withSeconds: false,
			wantErr:     true,
		},
		{
			name: "negative retry count",
			job: &JobConfig{
				Name:       "test-job",
				Spec:       "0 0 * * *",
				Enabled:    true,
				RetryCount: -1,
			},
			withSeconds: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate(tt.withSeconds)
			if (err != nil) != tt.wantErr {
				t.Errorf("JobConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

