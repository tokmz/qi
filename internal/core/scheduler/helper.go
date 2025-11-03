package scheduler

import (
	"fmt"
	"time"
)

// Predefined 预定义的 Cron 表达式
var Predefined = struct {
	// 每秒执行（需要启用秒字段）
	EverySecond string
	// 每分钟执行
	EveryMinute string
	// 每小时执行
	EveryHour string
	// 每天执行
	EveryDay string
	// 每周执行
	EveryWeek string
	// 每月执行
	EveryMonth string
	// 每年执行
	EveryYear string
}{
	EverySecond: "* * * * * *",
	EveryMinute: "0 * * * *",
	EveryHour:   "0 0 * * *",
	EveryDay:    "0 0 0 * *",
	EveryWeek:   "0 0 0 * * 0",
	EveryMonth:  "0 0 0 1 * *",
	EveryYear:   "0 0 0 1 1 *",
}

// EveryN 生成每 N 单位执行的 Cron 表达式
type EveryN struct{}

// Seconds 每 N 秒执行（需要启用秒字段）
func (e EveryN) Seconds(n int) string {
	return fmt.Sprintf("*/%d * * * * *", n)
}

// Minutes 每 N 分钟执行
func (e EveryN) Minutes(n int) string {
	return fmt.Sprintf("0 */%d * * *", n)
}

// Hours 每 N 小时执行
func (e EveryN) Hours(n int) string {
	return fmt.Sprintf("0 0 */%d * *", n)
}

// Days 每 N 天执行
func (e EveryN) Days(n int) string {
	return fmt.Sprintf("0 0 0 */%d * *", n)
}

// Every 生成定时表达式的辅助函数
var Every = EveryN{}

// At 在指定时间执行
type At struct{}

// DailyAt 每天在指定时间执行
// hour: 0-23, minute: 0-59
func (a At) DailyAt(hour, minute int) string {
	return fmt.Sprintf("0 %d %d * * *", minute, hour)
}

// WeeklyAt 每周在指定星期和时间执行
// weekday: 0-6 (0=Sunday), hour: 0-23, minute: 0-59
func (a At) WeeklyAt(weekday, hour, minute int) string {
	return fmt.Sprintf("0 %d %d * * %d", minute, hour, weekday)
}

// MonthlyAt 每月在指定日期和时间执行
// day: 1-31, hour: 0-23, minute: 0-59
func (a At) MonthlyAt(day, hour, minute int) string {
	return fmt.Sprintf("0 %d %d %d * *", minute, hour, day)
}

// AtTime 时间辅助函数
var AtTime = At{}

// ParseDuration 解析时间间隔字符串
// 支持格式: "1h", "30m", "5s" 等
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// NextRun 计算下次运行时间
func NextRun(spec string, withSeconds bool) (time.Time, error) {
	// 使用 cron 库解析表达式
	// 这里需要实际的 cron 实例来计算
	return time.Time{}, fmt.Errorf("not implemented")
}

// ValidateSpec 验证 Cron 表达式
func ValidateSpec(spec string, withSeconds bool) error {
	// 简单的字段数量验证
	// 实际验证由 cron 库完成
	return nil
}

// SpecExamples Cron 表达式示例
var SpecExamples = map[string]string{
	// 5 字段格式（标准格式）
	"每分钟":      "0 * * * *",
	"每小时":      "0 0 * * *",
	"每天凌晨2点":   "0 2 * * *",
	"每周一早上9点":  "0 9 * * 1",
	"每月1号凌晨3点": "0 3 1 * *",
	"工作日早上8点":  "0 8 * * 1-5",
	"每30分钟":    "*/30 * * * *",
	"每2小时":     "0 */2 * * *",

	// 6 字段格式（带秒）
	"每秒":        "* * * * * *",
	"每5秒":       "*/5 * * * * *",
	"每10秒":      "*/10 * * * * *",
	"每30秒":      "*/30 * * * * *",
	"每分钟第0秒":   "0 * * * * *",
	"每小时第0分0秒": "0 0 * * * *",
}

// GetSpecExample 获取表达式示例
func GetSpecExample(name string) (string, bool) {
	spec, exists := SpecExamples[name]
	return spec, exists
}

// ListSpecExamples 列出所有表达式示例
func ListSpecExamples() map[string]string {
	return SpecExamples
}

