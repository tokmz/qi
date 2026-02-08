package datetime

import (
	"fmt"
	"strconv"
	"time"
)

// 时间常量
const (
	DateFormat     = "2006-01-02"
	DateTimeFormat = "2006-01-02 15:04:05"
	TimeFormat     = "15:04:05"
	RFC3339Format  = "2006-01-02T15:04:05Z07:00"
)

// ========== 格式化 ==========

// FormatDate 格式化为日期
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// FormatDateTime 格式化为日期时间
func FormatDateTime(t time.Time) string {
	return t.Format(DateTimeFormat)
}

// FormatTime 格式化为时间
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// FormatRFC3339 格式化为 RFC3339
func FormatRFC3339(t time.Time) string {
	return t.Format(RFC3339Format)
}

// FormatUnix 格式化为 Unix 时间戳
func FormatUnix(ts int64) string {
	return fmt.Sprintf("%d", ts)
}

// FormatDuration 格式化为人类可读时长
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %.0fs", int(d.Minutes()), d.Seconds())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	return fmt.Sprintf("%dd %dh", days, int(d.Hours())%24)
}

// ========== 解析 ==========

// ParseDate 解析日期
func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateFormat, s)
}

// ParseDateTime 解析日期时间
func ParseDateTime(s string) (time.Time, error) {
	return time.Parse(DateTimeFormat, s)
}

// ParseRFC3339 解析 RFC3339
func ParseRFC3339(s string) (time.Time, error) {
	return time.Parse(RFC3339Format, s)
}

// ParseUnix 解析 Unix 时间戳
func ParseUnix(s string) (time.Time, error) {
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

// ParseFromTimestamp 从毫秒/秒时间戳解析
func ParseFromTimestamp(ts int64, isMs bool) time.Time {
	if isMs {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}

// ========== 计算 ==========

// StartOfDay 获取当天开始时间
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取当天结束时间
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek 获取周开始时间（周一）
func StartOfWeek(t time.Time) time.Time {
	t = StartOfDay(t)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -weekday+1)
}

// EndOfWeek 获取周结束时间（周日）
func EndOfWeek(t time.Time) time.Time {
	return StartOfWeek(t).AddDate(0, 0, 7).Add(-time.Nanosecond)
}

// StartOfMonth 获取月开始时间
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取月结束时间
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// StartOfYear 获取年开始时间
func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// EndOfYear 获取年结束时间
func EndOfYear(t time.Time) time.Time {
	return StartOfYear(t).AddDate(1, 0, 0).Add(-time.Nanosecond)
}

// ========== 判断 ==========

// IsToday 是否是今天
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() &&
		t.Month() == now.Month() &&
		t.Day() == now.Day()
}

// IsYesterday 是否是昨天
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() &&
		t.Month() == yesterday.Month() &&
		t.Day() == yesterday.Day()
}

// IsTomorrow 是否是明天
func IsTomorrow(t time.Time) bool {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return t.Year() == tomorrow.Year() &&
		t.Month() == tomorrow.Month() &&
		t.Day() == tomorrow.Day()
}

// IsThisWeek 是否是本周
func IsThisWeek(t time.Time) bool {
	now := time.Now()
	thisWeekStart, thisWeekEnd := GetWeekRange(now)
	return (t.After(thisWeekStart) || t.Equal(thisWeekStart)) &&
		(t.Before(thisWeekEnd) || t.Equal(thisWeekEnd))
}

// IsThisMonth 是否是本月
func IsThisMonth(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month()
}

// IsThisYear 是否是本年
func IsThisYear(t time.Time) bool {
	return t.Year() == time.Now().Year()
}

// IsWeekend 是否是周末
func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// IsLeapYear 是否是闰年
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// ========== 转换 ==========

// ToUnix 转换为 Unix 时间戳（秒）
func ToUnix(t time.Time) int64 {
	return t.Unix()
}

// ToUnixMs 转换为 Unix 时间戳（毫秒）
func ToUnixMs(t time.Time) int64 {
	return t.UnixMilli()
}

// ToRFC3339 转换为 RFC3339 字符串
func ToRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// TimestampToTime 将时间戳转换为时间
func TimestampToTime(ts int64) time.Time {
	return time.Unix(ts, 0)
}

// ========== 时间段 ==========

// Age 计算年龄
func Age(birthday time.Time) int {
	now := time.Now()
	age := now.Year() - birthday.Year()
	if now.YearDay() < birthday.YearDay() {
		age--
	}
	return age
}

// DaysBetween 计算两个日期相隔天数
// 注意：使用日期差而不是小时差，避免夏令时问题
func DaysBetween(start, end time.Time) int {
	start = StartOfDay(start)
	end = StartOfDay(end)

	// 使用时间差计算天数，避免循环性能问题
	duration := end.Sub(start)
	days := int(duration.Hours() / 24)

	// 处理夏令时导致的小数天问题
	if duration.Hours() > 0 && duration.Hours() < 24 {
		days = 0
	} else if duration.Hours() < 0 && duration.Hours() > -24 {
		days = 0
	}

	return days
}

// HoursBetween 计算两个时间相隔小时数
func HoursBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours())
}

// ========== 时间范围 ==========

// GetDayRange 获取当天时间范围
func GetDayRange(t time.Time) (start, end time.Time) {
	return StartOfDay(t), EndOfDay(t)
}

// GetWeekRange 获取周时间范围
func GetWeekRange(t time.Time) (start, end time.Time) {
	return StartOfWeek(t), EndOfWeek(t)
}

// GetMonthRange 获取月时间范围
func GetMonthRange(t time.Time) (start, end time.Time) {
	return StartOfMonth(t), EndOfMonth(t)
}

// GetYearRange 获取年时间范围
func GetYearRange(t time.Time) (start, end time.Time) {
	return StartOfYear(t), EndOfYear(t)
}

// ========== 时间偏移 ==========

// AddDays 增加天数
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddHours 增加小时
func AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// AddMinutes 增加分钟
func AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// AddSeconds 增加秒
func AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// AddMonths 增加月
func AddMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// AddYears 增加年
func AddYears(t time.Time, years int) time.Time {
	return t.AddDate(years, 0, 0)
}

// ========== 比较 ==========

// IsAfter 检查 t 是否在 after 之后
func IsAfter(t, after time.Time) bool {
	return t.After(after)
}

// IsBefore 检查 t 是否在 before 之前
func IsBefore(t, before time.Time) bool {
	return t.Before(before)
}

// IsBetween 检查 t 是否在 start 和 end 之间
func IsBetween(t, start, end time.Time) bool {
	return (t.After(start) || t.Equal(start)) && (t.Before(end) || t.Equal(end))
}

// ========== 其他 ==========

// NowUnix 获取当前 Unix 时间戳（秒）
func NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixMs 获取当前 Unix 时间戳（毫秒）
func NowUnixMs() int64 {
	return time.Now().UnixMilli()
}

// NowRFC3339 获取当前 RFC3339 时间
func NowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

// Since 获取从指定时间到现在的时间间隔
func Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Until 获取从现在到指定时间的时间间隔
func Until(t time.Time) time.Duration {
	return time.Until(t)
}

// ========== 相对时间显示 ==========

// TimeAgo 显示多久之前（中文）
// 例如：刚刚、5秒前、3分钟前、2小时前、昨天、3天前、2周前、5个月前、2年前
func TimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		// 未来时间
		return FormatDateTime(t)
	}

	seconds := int(diff.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 5:
		return "刚刚"
	case seconds < 60:
		return fmt.Sprintf("%d秒前", seconds)
	case minutes < 60:
		return fmt.Sprintf("%d分钟前", minutes)
	case hours < 24:
		return fmt.Sprintf("%d小时前", hours)
	case days == 1:
		return "昨天"
	case days < 7:
		return fmt.Sprintf("%d天前", days)
	case weeks < 4:
		return fmt.Sprintf("%d周前", weeks)
	case months < 12:
		return fmt.Sprintf("%d个月前", months)
	default:
		return fmt.Sprintf("%d年前", years)
	}
}

// TimeAgoEn 显示多久之前（英文）
// 例如：just now, 5 seconds ago, 3 minutes ago, 2 hours ago, yesterday, 3 days ago
func TimeAgoEn(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		return FormatDateTime(t)
	}

	seconds := int(diff.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	switch {
	case seconds == 1:
		return "1 second ago"
	case seconds < 5:
		return "just now"
	case seconds < 60:
		return fmt.Sprintf("%d seconds ago", seconds)
	case minutes == 1:
		return "1 minute ago"
	case minutes < 60:
		return fmt.Sprintf("%d minutes ago", minutes)
	case hours == 1:
		return "1 hour ago"
	case hours < 24:
		return fmt.Sprintf("%d hours ago", hours)
	case days == 1:
		return "yesterday"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	default:
		return FormatDate(t)
	}
}

// TimeLeft 显示距离某个时间还有多久（中文）
// 例如：5秒后、3分钟后、2小时后、3天后、2周后、5个月后、2年后
func TimeLeft(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)

	if diff < 0 {
		return "已过期"
	}

	seconds := int(diff.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 60:
		return fmt.Sprintf("%d秒后", seconds)
	case minutes < 60:
		return fmt.Sprintf("%d分钟后", minutes)
	case hours < 24:
		return fmt.Sprintf("%d小时后", hours)
	case days < 7:
		return fmt.Sprintf("%d天后", days)
	case weeks < 4:
		return fmt.Sprintf("%d周后", weeks)
	case months < 12:
		return fmt.Sprintf("%d个月后", months)
	default:
		return fmt.Sprintf("%d年后", years)
	}
}

// TimeLeftEn 显示距离某个时间还有多久（英文）
func TimeLeftEn(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)

	if diff < 0 {
		return "expired"
	}

	seconds := int(diff.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	switch {
	case seconds == 1:
		return "1 second"
	case seconds < 60:
		return fmt.Sprintf("%d seconds", seconds)
	case minutes == 1:
		return "1 minute"
	case minutes < 60:
		return fmt.Sprintf("%d minutes", minutes)
	case hours == 1:
		return "1 hour"
	case hours < 24:
		return fmt.Sprintf("%d hours", hours)
	case days == 1:
		return "1 day"
	default:
		return fmt.Sprintf("%d days", days)
	}
}

// DurationText 显示时间间隔文本（中文）
// 例如：2小时30分45秒
func DurationText(d time.Duration) string {
	seconds := int(d.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d天", days))
		hours = hours % 24
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d小时", hours))
		minutes = minutes % 60
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d分", minutes))
		seconds = seconds % 60
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d秒", seconds))
	}

	return joinStrings(parts...)
}

// DurationTextEn 显示时间间隔文本（英文）
// 例如：2h 30m 45s
func DurationTextEn(d time.Duration) string {
	seconds := int(d.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		hours = hours % 24
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		minutes = minutes % 60
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
		seconds = seconds % 60
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return joinStrings(parts...)
}

// joinStrings 连接字符串
func joinStrings(parts ...string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}
