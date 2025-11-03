package scheduler

import (
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
)

// validateCronSpec 验证 cron 表达式格式
func validateCronSpec(spec string, withSeconds bool) error {
	// 去除首尾空格
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ErrInvalidCronSpec
	}

	// 检查是否是预定义的特殊字符串
	predefined := map[string]bool{
		"@yearly":   true,
		"@annually": true,
		"@monthly":  true,
		"@weekly":   true,
		"@daily":    true,
		"@midnight": true,
		"@hourly":   true,
	}

	if predefined[strings.ToLower(spec)] {
		return nil
	}

	// 检查是否是 @every 格式
	if strings.HasPrefix(strings.ToLower(spec), "@every ") {
		return nil
	}

	// 验证字段数量
	fields := strings.Fields(spec)
	expectedFields := 5
	if withSeconds {
		expectedFields = 6
	}

	if len(fields) != expectedFields {
		if withSeconds {
			return fmt.Errorf("%w: expected 6 fields (second minute hour day month weekday), got %d",
				ErrInvalidCronSpec, len(fields))
		}
		return fmt.Errorf("%w: expected 5 fields (minute hour day month weekday), got %d",
			ErrInvalidCronSpec, len(fields))
	}

	// 使用 cron 库验证表达式
	var parser cron.Parser
	if withSeconds {
		parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	} else {
		parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	}

	if _, err := parser.Parse(spec); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCronSpec, err)
	}

	return nil
}

// ValidateCronSpec 公开的验证函数，用于外部验证 cron 表达式
func ValidateCronSpec(spec string, withSeconds bool) error {
	return validateCronSpec(spec, withSeconds)
}
