# DateTime

时间处理工具包，提供时间格式化、解析、计算等功能。

## 安装

```bash
go get qi/pkg/datetime
```

## 时间常量

| 常量 | 值 | 说明 |
|------|-----|------|
| `DateFormat` | `2006-01-02` | 日期格式 |
| `DateTimeFormat` | `2006-01-02 15:04:05` | 日期时间格式 |
| `TimeFormat` | `15:04:05` | 时间格式 |
| `RFC3339Format` | `2006-01-02T15:04:05Z07:00` | RFC3339 格式 |

## 格式化

| 函数 | 说明 |
|------|------|
| `FormatDate(t time.Time) string` | 格式化为日期 `2006-01-02` |
| `FormatDateTime(t time.Time) string` | 格式化为日期时间 `2006-01-02 15:04:05` |
| `FormatTime(t time.Time) string` | 格式化为时间 `15:04:05` |
| `FormatRFC3339(t time.Time) string` | 格式化为 RFC3339 |
| `FormatUnix(ts int64) string` | 格式化为 Unix 时间戳字符串 |
| `FormatDuration(d time.Duration) string` | 格式化为人类可读时长 |

## 解析

| 函数 | 说明 |
|------|------|
| `ParseDate(s string) (time.Time, error)` | 解析日期字符串 |
| `ParseDateTime(s string) (time.Time, error)` | 解析日期时间字符串 |
| `ParseRFC3339(s string) (time.Time, error)` | 解析 RFC3339 字符串 |
| `ParseUnix(s string) (time.Time, error)` | 解析 Unix 时间戳字符串 |
| `ParseFromTimestamp(ts int64, isMs bool) time.Time` | 从时间戳解析 |

## 计算

| 函数 | 说明 |
|------|------|
| `StartOfDay(t time.Time) time.Time` | 获取当天开始时间 |
| `EndOfDay(t time.Time) time.Time` | 获取当天结束时间 |
| `StartOfWeek(t time.Time) time.Time` | 获取周开始时间（周一） |
| `EndOfWeek(t time.Time) time.Time` | 获取周结束时间（周日） |
| `StartOfMonth(t time.Time) time.Time` | 获取月开始时间 |
| `EndOfMonth(t time.Time) time.Time` | 获取月结束时间 |
| `StartOfYear(t time.Time) time.Time` | 获取年开始时间 |
| `EndOfYear(t time.Time) time.Time` | 获取年结束时间 |

## 判断

| 函数 | 说明 |
|------|------|
| `IsToday(t time.Time) bool` | 是否是今天 |
| `IsYesterday(t time.Time) bool` | 是否是昨天 |
| `IsTomorrow(t time.Time) bool` | 是否是明天 |
| `IsThisWeek(t time.Time) bool` | 是否是本周 |
| `IsThisMonth(t time.Time) bool` | 是否是本月 |
| `IsThisYear(t time.Time) bool` | 是否是本年 |
| `IsWeekend(t time.Time) bool` | 是否是周末 |
| `IsLeapYear(year int) bool` | 是否是闰年 |

## 转换

| 函数 | 说明 |
|------|------|
| `ToUnix(t time.Time) int64` | 转换为 Unix 时间戳（秒） |
| `ToUnixMs(t time.Time) int64` | 转换为 Unix 时间戳（毫秒） |
| `ToRFC3339(t time.Time) string` | 转换为 RFC3339 字符串 |
| `TimestampToTime(ts int64) time.Time` | 将时间戳转换为时间 |

## 时间段

| 函数 | 说明 |
|------|------|
| `Age(birthday time.Time) int` | 计算年龄 |
| `DaysBetween(start, end time.Time) int` | 计算两个日期相隔天数 |
| `HoursBetween(start, end time.Time) int` | 计算两个时间相隔小时数 |

## 时间范围

| 函数 | 说明 |
|------|------|
| `GetDayRange(t time.Time) (start, end time.Time)` | 获取当天时间范围 |
| `GetWeekRange(t time.Time) (start, end time.Time)` | 获取周时间范围 |
| `GetMonthRange(t time.Time) (start, end time.Time)` | 获取月时间范围 |
| `GetYearRange(t time.Time) (start, end time.Time)` | 获取年时间范围 |

## 时间偏移

| 函数 | 说明 |
|------|------|
| `AddDays(t time.Time, days int) time.Time` | 增加天数 |
| `AddHours(t time.Time, hours int) time.Time` | 增加小时 |
| `AddMinutes(t time.Time, minutes int) time.Time` | 增加分钟 |
| `AddSeconds(t time.Time, seconds int) time.Time` | 增加秒 |
| `AddMonths(t time.Time, months int) time.Time` | 增加月 |
| `AddYears(t time.Time, years int) time.Time` | 增加年 |

## 比较

| 函数 | 说明 |
|------|------|
| `IsAfter(t, after time.Time) bool` | 检查 t 是否在 after 之后 |
| `IsBefore(t, before time.Time) bool` | 检查 t 是否在 before 之前 |
| `IsBetween(t, start, end time.Time) bool` | 检查 t 是否在 start 和 end 之间 |

## 其他

| 函数 | 说明 |
|------|------|
| `NowUnix() int64` | 获取当前 Unix 时间戳（秒） |
| `NowUnixMs() int64` | 获取当前 Unix 时间戳（毫秒） |
| `NowRFC3339() string` | 获取当前 RFC3339 时间 |
| `Since(t time.Time) time.Duration` | 获取从指定时间到现在的间隔 |
| `Until(t time.Time) time.Duration` | 获取从现在到指定时间的间隔 |

## 相对时间显示

| 函数 | 说明 |
|------|------|
| `TimeAgo(t time.Time) string` | 显示多久之前（中文） |
| `TimeAgoEn(t time.Time) string` | 显示多久之前（英文） |
| `TimeLeft(t time.Time) string` | 显示距离某个时间还有多久（中文） |
| `TimeLeftEn(t time.Time) string` | 显示距离某个时间还有多久（英文） |
| `DurationText(d time.Duration) string` | 显示时间间隔文本（中文） |
| `DurationTextEn(d time.Duration) string` | 显示时间间隔文本（英文） |

## 使用示例

```go
package main

import (
	"fmt"
	"time"

	"qi/pkg/datetime"
)

func main() {
	now := time.Now()

	// 格式化
	fmt.Println(datetime.FormatDate(now))      // 2024-01-15
	fmt.Println(datetime.FormatDateTime(now))  // 2024-01-15 10:30:00
	fmt.Println(datetime.FormatDuration(3662000000000)) // 1h 1m

	// 解析
	t, _ := datetime.ParseDate("2024-01-15")
	fmt.Println(t)

	// 时间范围
	start, end := datetime.GetMonthRange(now)
	fmt.Printf("本月: %v 到 %v\n", start, end)

	// 判断
	fmt.Println(datetime.IsToday(now))   // true
	fmt.Println(datetime.IsWeekend(now)) // false

	// 相对时间
	fmt.Println(datetime.TimeAgo(time.Now().Add(-5 * time.Minute))) // 5分钟前
	fmt.Println(datetime.TimeLeft(time.Now().Add(1 * time.Hour)))   // 1小时后

	// 相对时间（英文）
	fmt.Println(datetime.TimeAgoEn(time.Now().Add(-30 * time.Second))) // just now
}
```
