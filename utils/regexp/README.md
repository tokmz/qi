# Regexp

正则表达式工具包，提供常用正则验证和操作功能。

## 安装

```bash
go get qi/pkg/regexp
```

## 预定义正则常量

### 数字

| 常量 | 值 | 说明 |
|------|-----|------|
| `NumPattern` | `^[0-9]+$` | 数字 |
| `PositiveIntPattern` | `^[1-9][0-9]*$` | 正整数 |
| `NegativeIntPattern` | `^-[1-9][0-9]*$` | 负整数 |
| `IntegerPattern` | `^-?[1-9][0-9]*$` | 整数 |
| `FloatPattern` | `^-?[0-9]+\.[0-9]+$` | 浮点数 |

### 字母

| 常量 | 值 | 说明 |
|------|-----|------|
| `AlphaPattern` | `^[a-zA-Z]+$` | 纯字母 |
| `AlphaLowerPattern` | `^[a-z]+$` | 小写字母 |
| `AlphaUpperPattern` | `^[A-Z]+$` | 大写字母 |
| `AlphaNumPattern` | `^[a-zA-Z0-9]+$` | 字母和数字 |

### 中文

| 常量 | 值 | 说明 |
|------|-----|------|
| `ChinesePattern` | `^[\u4e00-\u9fa5]+$` | 纯中文 |
| `ChineseNamePattern` | `^[\u4e00-\u9fa5]{2,10}$` | 中文姓名 |

### 联系方式

| 常量 | 值 | 说明 |
|------|-----|------|
| `EmailPattern` | `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$` | 邮箱 |
| `PhoneCNPattern` | `^1[3-9]\d{9}$` | 手机号（中国） |
| `TelephonePattern` | `^(0\d{2,3}-)?\d{7,8}$` | 电话号码（中国） |
| `IDCardPattern` | 身份证号正则 | 身份证号（中国） |
| `QQPattern` | `^[1-9]\d{4,10}$` | QQ 号 |
| `WeChatPattern` | `^[a-zA-Z][a-zA-Z0-9_-]{5,19}$` | 微信 |

### 网络

| 常量 | 值 | 说明 |
|------|-----|------|
| `URLPattern` | `^https?://[^\s]+$` | URL |
| `IPv4Pattern` | `^(\d{1,3}\.){3}\d{1,3}$` | IPv4 地址 |
| `DomainPattern` | 域名正则 | 域名 |
| `MACPattern` | `^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$` | MAC 地址 |
| `HexColorPattern` | `^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$` | 十六进制颜色 |

### 日期时间

| 常量 | 值 | 说明 |
|------|-----|------|
| `DateYYYYMMDDPattern` | `^\d{4}-\d{2}-\d{2}$` | 日期 YYYY-MM-DD |
| `DateTimePattern` | `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$` | 日期时间 |
| `TimeHHMMSSPattern` | `^\d{2}:\d{2}:\d{2}$` | 时间 HH:MM:SS |

### 其他

| 常量 | 值 | 说明 |
|------|-----|------|
| `UsernamePattern` | `^[a-zA-Z][a-zA-Z0-9_]{3,15}$` | 用户名 |
| `PasswordPattern` | `^[a-zA-Z0-9_@#!]{6,20}$` | 密码 |
| `ZipCodePattern` | `^\d{6}$` | 邮政编码（中国） |
| `BankCardPattern` | `^\d{16,19}$` | 银行卡号 |
| `CreditCodePattern` | `^[0-9A-Z]{18}$` | 社会信用代码 |

## 匹配验证

| 函数 | 说明 |
|------|------|
| `IsMatch(s string, pattern string) bool` | 检查字符串是否匹配正则 |
| `IsMatchNum(s string) bool` | 检查是否为数字 |
| `IsMatchEmail(s string) bool` | 检查是否为邮箱 |
| `IsMatchPhone(s string) bool` | 检查是否为手机号（中国） |
| `IsMatchChinese(s string) bool` | 检查是否为纯中文 |
| `IsMatchURL(s string) bool` | 检查是否为 URL |
| `IsMatchIP(s string) bool` | 检查是否为 IP 地址 |
| `IsMatchIDCard(s string) bool` | 检查是否为身份证号（中国） |

## 提取

| 函数 | 说明 |
|------|------|
| `FindString(s string, pattern string) string` | 提取第一个匹配项 |
| `FindStringSubmatch(s string, pattern string) []string` | 提取第一个匹配项及其子匹配 |
| `FindAllString(s string, pattern string, n int) []string` | 提取所有匹配项 |
| `FindNumber(s string) string` | 提取第一个数字 |
| `FindAllNumbers(s string) []string` | 提取所有数字 |
| `FindChinese(s string) string` | 提取中文 |
| `FindEmail(s string) string` | 提取邮箱 |
| `FindPhone(s string) string` | 提取手机号 |
| `FindURL(s string) string` | 提取 URL |
| `FindAllURLs(s string) []string` | 提取所有 URL |
| `FindAllEmails(s string) []string` | 提取所有邮箱 |
| `FindAllPhones(s string) []string` | 提取所有手机号 |

## 替换

| 函数 | 说明 |
|------|------|
| `ReplaceString(s string, pattern, repl string) string` | 替换匹配项 |
| `ReplaceStringFunc(s string, pattern string, repl func(string) string) string` | 替换匹配项（使用函数） |
| `ReplaceNumber(s string, repl string) string` | 替换数字 |
| `ReplaceChinese(s string, repl string) string` | 替换中文 |

## 分割

| 函数 | 说明 |
|------|------|
| `Split(s string, pattern string) []string` | 分割字符串 |
| `SplitBySpace(s string) []string` | 按空白字符分割 |
| `SplitByComma(s string) []string` | 按逗号分割 |
| `SplitBySemicolon(s string) []string` | 按分号分割 |
| `SplitByNewline(s string) []string` | 按换行符分割 |

## 数量统计

| 函数 | 说明 |
|------|------|
| `Count(s string, pattern string) int` | 统计匹配数量 |
| `CountNumbers(s string) int` | 统计数字数量 |
| `CountChinese(s string) int` | 统计中文数量 |
| `CountEmails(s string) int` | 统计邮箱数量 |
| `CountPhones(s string) int` | 统计手机号数量 |
| `CountURLs(s string) int` | 统计 URL 数量 |

## 位置信息

| 函数 | 说明 |
|------|------|
| `FindStringIndex(s string, pattern string) []int` | 返回匹配的起止位置 |
| `FindAllStringIndex(s string, pattern string) [][]int` | 返回所有匹配的起止位置 |

## 验证函数

| 函数 | 说明 |
|------|------|
| `ValidateEmail(email string) bool` | 验证邮箱 |
| `ValidatePhone(phone string) bool` | 验证手机号（中国） |
| `ValidateURL(url string) bool` | 验证 URL |
| `ValidateIP(ip string) bool` | 验证 IP 地址 |
| `ValidateIDCard(idCard string) bool` | 验证身份证号（中国） |
| `ValidateChineseName(name string) bool` | 验证中文姓名 |
| `ValidateUsername(username string) bool` | 验证用户名 |
| `ValidatePassword(password string, level int) bool` | 验证密码强度 |
| `ValidateCreditCode(code string) bool` | 验证社会信用代码 |
| `ValidateBankCard(card string) bool` | 验证银行卡号 |
| `ValidateZipCode(zipCode string) bool` | 验证邮政编码 |

## 使用示例

```go
package main

import (
	"fmt"

	"qi/pkg/regexp"
)

func main() {
	s := "我的邮箱是 test@example.com，手机号是 13800138000"

	// 验证
	fmt.Println(regexp.ValidateEmail("test@example.com")) // true
	fmt.Println(regexp.ValidatePhone("13800138000"))     // true
	fmt.Println(regexp.ValidateIP("192.168.1.1"))        // true

	// 匹配验证
	fmt.Println(regexp.IsMatchNum("12345"))              // true
	fmt.Println(regexp.IsMatchChinese("你好"))            // true
	fmt.Println(regexp.IsMatchURL("https://example.com")) // true

	// 提取
	email := regexp.FindEmail(s)
	fmt.Println(email) // "test@example.com"

	phone := regexp.FindPhone(s)
	fmt.Println(phone) // "13800138000"

	// 提取所有
	emails := regexp.FindAllEmails("a@b.com, c@d.com")
	fmt.Println(emails) // ["a@b.com" "c@d.com"]

	// 替换
	hidden := regexp.ReplaceNumber("密码是 123456", "*")
	fmt.Println(hidden) // "密码是 ******"

	// 分割
	parts := regexp.SplitByComma("a,b,c,d")
	fmt.Println(parts) // ["a" "b" "c" "d"]

	// 统计
	count := regexp.CountEmails("a@b.com, c@d.com, e@f.com")
	fmt.Println(count) // 3

	// 验证密码强度（1=简单，2=中等，3=强）
	fmt.Println(regexp.ValidatePassword("123456", 1))    // true
	fmt.Println(regexp.ValidatePassword("abc123", 2))   // false
	fmt.Println(regexp.ValidatePassword("Abc123!@#", 3)) // true
}
```
