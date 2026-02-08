package strings

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// 预编译的正则表达式缓存
var (
	emailRegex *regexp.Regexp
	phoneRegex *regexp.Regexp
	urlRegex   *regexp.Regexp
	ipRegex    *regexp.Regexp
	regexOnce  sync.Once
)

// initRegex 初始化正则表达式（只执行一次）
func initRegex() {
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	urlRegex = regexp.MustCompile(`^https?://[^\s]+$`)
	ipRegex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
}

// ===== 基础操作 =====

// IsEmpty 检查字符串是否为空
func IsEmpty(s string) bool {
	return len(s) == 0
}

// IsNotEmpty 检查字符串是否非空
func IsNotEmpty(s string) bool {
	return len(s) > 0
}

// IsBlank 检查字符串是否为空或只包含空白字符
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotBlank 检查字符串是否非空且包含非空白字符
func IsNotBlank(s string) bool {
	return strings.TrimSpace(s) != ""
}

// Default 如果字符串为空，返回默认值
func Default(s string, defaultValue string) string {
	if IsBlank(s) {
		return defaultValue
	}
	return s
}

// DefaultIfEmpty 如果字符串为空，返回默认值
func DefaultIfEmpty(s string, defaultValue string) string {
	if IsEmpty(s) {
		return defaultValue
	}
	return s
}

// ===== 大小写转换 =====

// ToUpper 转为大写
func ToUpper(s string) string {
	return strings.ToUpper(s)
}

// ToLower 转为小写
func ToLower(s string) string {
	return strings.ToLower(s)
}

// ToTitle 转为标题格式
func ToTitle(s string) string {
	return strings.ToTitle(s)
}

// ToCamel 转为 camelCase 格式
func ToCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var result strings.Builder
	upperNext := false

	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			upperNext = true
			continue
		}
		if upperNext {
			result.WriteRune(unicode.ToUpper(r))
			upperNext = false
		} else if i == 0 {
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToSnake 转为 snake_case 格式
func ToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToKebab 转为 kebab-case 格式
func ToKebab(s string) string {
	snake := ToSnake(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// ===== 裁剪 =====

// Trim 去除两端空白
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// TrimLeft 去除左侧空白
func TrimLeft(s string) string {
	return strings.TrimLeft(s, " \t\n\r")
}

// TrimRight 去除右侧空白
func TrimRight(s string) string {
	return strings.TrimRight(s, " \t\n\r")
}

// TrimPrefix 去除前缀
func TrimPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

// TrimSuffix 去除后缀
func TrimSuffix(s, suffix string) string {
	return strings.TrimSuffix(s, suffix)
}

// ===== 查找 =====

// Contains 检查是否包含子串
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsAny 检查是否包含任意字符
func ContainsAny(s, chars string) bool {
	return strings.ContainsAny(s, chars)
}

// ContainsRune 检查是否包含字符
func ContainsRune(s string, r rune) bool {
	return strings.ContainsRune(s, r)
}

// HasPrefix 检查前缀
func HasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// HasSuffix 检查后缀
func HasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// Index 查找子串位置
func Index(s, substr string) int {
	return strings.Index(s, substr)
}

// LastIndex 查找最后出现位置
func LastIndex(s, substr string) int {
	return strings.LastIndex(s, substr)
}

// ===== 分割与合并 =====

// Split 分割字符串
func Split(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, sep)
}

// SplitN 分割指定次数
func SplitN(s, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

// SplitAny 按任意字符分割
func SplitAny(s, seps string) []string {
	if s == "" {
		return []string{}
	}
	return strings.FieldsFunc(s, func(r rune) bool {
		for _, sep := range seps {
			if r == rune(sep) {
				return true
			}
		}
		return false
	})
}

// Join 合并字符串
func Join(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

// ===== 替换 =====

// Replace 替换子串
func Replace(s, old, new string, n int) string {
	return strings.Replace(s, old, new, n)
}

// ReplaceAll 替换所有子串
func ReplaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// ReplaceChars 替换指定字符
func ReplaceChars(s string, oldNew ...rune) string {
	if len(oldNew)%2 != 0 {
		return s
	}
	result := make([]rune, len(s))
	for i, r := range s {
		replaced := false
		for j := 0; j < len(oldNew); j += 2 {
			if r == oldNew[j] {
				result[i] = oldNew[j+1]
				replaced = true
				break
			}
		}
		if !replaced {
			result[i] = r
		}
	}
	return string(result)
}

// ===== 统计 =====

// Count 计算子串出现次数
func Count(s, substr string) int {
	return strings.Count(s, substr)
}

// Len 计算字符数（UTF-8）
func Len(s string) int {
	return strings.Count(s, "") - 1
}

// WordCount 计算单词数
func WordCount(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

// ===== 大小写判断 =====

// IsUpper 检查是否全大写
func IsUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return s != ""
}

// IsLower 检查是否全小写
func IsLower(s string) bool {
	for _, r := range s {
		if !unicode.IsLower(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return s != ""
}

// IsTitle 检查是否标题格式
func IsTitle(s string) bool {
	return strings.ToTitle(s) == s && !IsUpper(s)
}

// ===== 类型判断 =====

// IsAlpha 检查是否只包含字母
func IsAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return s != ""
}

// IsNumber 检查是否只包含数字
func IsNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// IsAlphaNumber 检查是否只包含字母和数字
func IsAlphaNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// IsEmail 检查是否为邮箱格式
func IsEmail(s string) bool {
	regexOnce.Do(initRegex)
	return emailRegex.MatchString(s)
}

// IsPhone 检查是否为手机号格式（中国）
func IsPhone(s string) bool {
	regexOnce.Do(initRegex)
	return phoneRegex.MatchString(s)
}

// IsURL 检查是否为 URL 格式
func IsURL(s string) bool {
	regexOnce.Do(initRegex)
	return urlRegex.MatchString(s)
}

// IsIP 检查是否为 IP 地址
func IsIP(s string) bool {
	regexOnce.Do(initRegex)
	return ipRegex.MatchString(s)
}

// IsChinese 检查是否包含中文字符
func IsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// ===== 截取 =====

// Substring 截取字符串
func Substring(s string, start, length int) string {
	if start < 0 {
		start = 0
	}
	if length <= 0 {
		return ""
	}
	rs := []rune(s)

	// 检查 start 是否超出范围
	if start >= len(rs) {
		return ""
	}

	end := start + length
	if end > len(rs) {
		end = len(rs)
	}
	if start >= end {
		return ""
	}
	return string(rs[start:end])
}

// Truncate 截断字符串（添加省略号）
func Truncate(s string, maxLen int, suffix string) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	rs := []rune(s)
	if maxLen > len(rs) {
		maxLen = len(rs)
	}
	return string(rs[:maxLen]) + suffix
}

// TakeLeft 获取左侧 n 个字符
func TakeLeft(s string, n int) string {
	if n <= 0 {
		return ""
	}
	rs := []rune(s)
	if n >= len(rs) {
		return s
	}
	return string(rs[:n])
}

// TakeRight 获取右侧 n 个字符
func TakeRight(s string, n int) string {
	if n <= 0 {
		return ""
	}
	rs := []rune(s)
	if n >= len(rs) {
		return s
	}
	return string(rs[len(rs)-n:])
}

// ===== 重复与填充 =====

// Repeat 重复字符串
func Repeat(s string, count int) string {
	return strings.Repeat(s, count)
}

// PadLeft 左侧填充
func PadLeft(s string, width int, padStr string) string {
	if padStr == "" {
		padStr = " "
	}
	if len(s) >= width {
		return s
	}
	return strings.Repeat(padStr, width-len(s)) + s
}

// PadRight 右侧填充
func PadRight(s string, width int, padStr string) string {
	if padStr == "" {
		padStr = " "
	}
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(padStr, width-len(s))
}

// Center 居中填充
func Center(s string, width int, padStr string) string {
	if padStr == "" {
		padStr = " "
	}
	if len(s) >= width {
		return s
	}
	left := (width - len(s)) / 2
	right := width - len(s) - left
	return strings.Repeat(padStr, left) + s + strings.Repeat(padStr, right)
}

// ===== 转换 =====

// Reverse 反转字符串
func Reverse(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

// ToBytes 转为字节数组
func ToBytes(s string) []byte {
	return []byte(s)
}

// ToString 转为字符串
func ToString(b []byte) string {
	return string(b)
}

// ===== 拼音相关 =====

// IsPinyin 检查是否为有效的拼音（简拼或全拼）
func IsPinyin(s string) bool {
	if s == "" {
		return false
	}
	// 简拼：2-6 位字母
	// 全拼：验证基本字母范围
	pattern := `^[a-zA-Z]{2,20}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

// PinyinFirstChar 获取拼音首字母
func PinyinFirstChar(s string) string {
	if s == "" {
		return ""
	}
	return string(unicode.ToUpper(rune(s[0])))
}

// PinyinAbbr 获取拼音缩写（每字首字母）
func PinyinAbbr(s string) string {
	parts := SplitAny(s, " -_")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, string(unicode.ToUpper(rune(part[0]))))
		}
	}
	return Join(result, "")
}

// ===== 其他 =====

// Nl2Br 换行符转为 <br>
func Nl2Br(s string) string {
	return strings.ReplaceAll(s, "\n", "<br>")
}

// Br2Nl <br> 转为换行符
func Br2Nl(s string) string {
	return strings.ReplaceAll(s, "<br>", "\n")
}

// Quote 添加引号
func Quote(s string) string {
	return "\"" + s + "\""
}

// Unquote 去除引号
func Unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// Unique 去重相邻重复字符
func Unique(s string) string {
	if s == "" {
		return ""
	}
	result := make([]rune, 0, len(s))
	var prev rune
	for _, r := range s {
		if r != prev || len(result) == 0 {
			result = append(result, r)
			prev = r
		}
	}
	return string(result)
}

// WordInitials 获取单词首字母大写
func WordInitials(s string) string {
	parts := strings.Fields(s)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, strings.ToUpper(string(part[0])))
		}
	}
	return Join(result, "")
}
