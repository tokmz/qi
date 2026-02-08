package regexp

import (
	"regexp"
	"strings"
	"sync"
)

// 预编译的正则表达式
var (
	// 数字
	NumPattern         = `^[0-9]+$`
	Num1To9Pattern     = `^[1-9]$`
	Num1To9ZeroPattern = `^[1-9][0-9]*$`
	PositiveIntPattern = `^[1-9][0-9]*$`
	NegativeIntPattern = `^-[1-9][0-9]*$`
	IntegerPattern     = `^-?[1-9][0-9]*$`
	FloatPattern       = `^-?[0-9]+\.[0-9]+$`

	// 字母
	AlphaPattern      = `^[a-zA-Z]+$`
	AlphaLowerPattern = `^[a-z]+$`
	AlphaUpperPattern = `^[A-Z]+$`
	AlphaNumPattern   = `^[a-zA-Z0-9]+$`

	// 中文
	ChinesePattern     = `^[\u4e00-\u9fa5]+$`
	ChineseNamePattern = `^[\u4e00-\u9fa5]{2,10}$`

	// 邮箱
	EmailPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

	// 手机号（中国）
	PhoneCNPattern = `^1[3-9]\d{9}$`

	// 电话号码（中国）
	TelephonePattern = `^(0\d{2,3}-)?\d{7,8}$`

	// 身份证（中国）
	IDCardPattern = `^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[1-2]\d|3[0-1])\d{3}(\d|X|x)$`

	// URL
	URLPattern      = `^https?://[^\s]+$`
	URLHttpPattern  = `^http://[^\s]+$`
	URLHttpsPattern = `^https://[^\s]+$`

	// IP 地址
	IPv4Pattern = `^(\d{1,3}\.){3}\d{1,3}$`
	IPv6Pattern = `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`

	// 邮政编码（中国）
	ZipCodePattern = `^\d{6}$`

	// 银行卡（中国）
	BankCardPattern = `^\d{16,19}$`

	// 护照
	PassportPattern = `^[a-zA-Z0-9]{6,20}$`

	// QQ 号
	QQPattern = `^[1-9]\d{4,10}$`

	// 微信
	WeChatPattern = `^[a-zA-Z][a-zA-Z0-9_-]{5,19}$`

	// 域名
	DomainPattern = `^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)*\.[a-zA-Z]{2,}$`

	// MAC 地址
	MACPattern = `^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`

	// 十六进制颜色
	HexColorPattern = `^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`

	// 日期格式
	DateYYYYMMDDPattern     = `^\d{4}-\d{2}-\d{2}$`
	DateYYYYMMDDChinesePattern = `^\d{4}年\d{1,2}月\d{1,2}日$`
	DateMMDDPattern         = `^\d{2}-\d{2}$`

	// 时间格式
	TimeHHMMSSPattern     = `^\d{2}:\d{2}:\d{2}$`
	TimeHHMMSS24Pattern   = `^([01]?\d|2[0-3]):[0-5]?\d:[0-5]?\d$`
	TimeHHMMPattern       = `^\d{2}:\d{2}$`

	// 日期时间
	DateTimePattern = `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`

	// 标签
	TagPattern = `^[\u4e00-\u9fa5a-zA-Z0-9_-]{1,10}$`

	// 用户名
	UsernamePattern = `^[a-zA-Z][a-zA-Z0-9_]{3,15}$`

	// 密码（允许更多特殊字符，符合现代安全标准）
	// 注意：不限制字符集，只检查长度和复杂度
	PasswordPattern       = `^[\x20-\x7E]{6,}$` // 允许所有可打印 ASCII 字符
	PasswordSimplePattern = `^[\x20-\x7E]{6,}$`

	// base64
	Base64Pattern = `^[A-Za-z0-9+/=]+$`

	// 十六进制
	HexPattern = `^[0-9a-fA-F]+$`

	// 社会信用代码（中国）
	CreditCodePattern = `^[0-9A-Z]{18}$`
)

// 编译缓存
var (
	cache      = make(map[string]*regexp.Regexp)
	cacheMu    sync.RWMutex
	maxCache   = 100 // 最大缓存数量（可通过 SetMaxCacheSize 修改）
	cacheOrder []string
)

// SetMaxCacheSize 设置正则表达式缓存的最大数量
func SetMaxCacheSize(size int) {
	if size <= 0 {
		size = 100
	}
	cacheMu.Lock()
	maxCache = size
	cacheMu.Unlock()
}

// get 获取或编译正则表达式（带 LRU 缓存）
func get(pattern string) *regexp.Regexp {
	cacheMu.RLock()
	if re, ok := cache[pattern]; ok {
		cacheMu.RUnlock()
		return re
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	// Double-check locking
	if re, ok := cache[pattern]; ok {
		return re
	}

	// 检查缓存大小，超过限制则删除最旧的
	if len(cache) >= maxCache {
		// 删除最旧的缓存项
		oldest := cacheOrder[0]
		delete(cache, oldest)
		cacheOrder = cacheOrder[1:]
	}

	re := regexp.MustCompile(pattern)
	cache[pattern] = re
	cacheOrder = append(cacheOrder, pattern)
	return re
}

// ===== 匹配验证 =====

// IsMatch 检查字符串是否匹配正则
func IsMatch(s string, pattern string) bool {
	return get(pattern).MatchString(s)
}

// IsMatchNum 检查是否为数字
func IsMatchNum(s string) bool {
	return get(NumPattern).MatchString(s)
}

// IsMatchEmail 检查是否为邮箱
func IsMatchEmail(s string) bool {
	return get(EmailPattern).MatchString(s)
}

// IsMatchPhone 检查是否为手机号（中国）
func IsMatchPhone(s string) bool {
	return get(PhoneCNPattern).MatchString(s)
}

// IsMatchChinese 检查是否为纯中文
func IsMatchChinese(s string) bool {
	return get(ChinesePattern).MatchString(s)
}

// IsMatchURL 检查是否为 URL
func IsMatchURL(s string) bool {
	return get(URLPattern).MatchString(s)
}

// IsMatchIP 检查是否为 IP 地址
func IsMatchIP(s string) bool {
	return get(IPv4Pattern).MatchString(s)
}

// IsMatchIDCard 检查是否为身份证号（中国）
func IsMatchIDCard(s string) bool {
	return get(IDCardPattern).MatchString(s)
}

// ===== 提取 =====

// FindString 提取第一个匹配项
func FindString(s string, pattern string) string {
	return get(pattern).FindString(s)
}

// FindStringSubmatch 提取第一个匹配项及其子匹配
func FindStringSubmatch(s string, pattern string) []string {
	return get(pattern).FindStringSubmatch(s)
}

// FindAllString 提取所有匹配项
func FindAllString(s string, pattern string, n int) []string {
	return get(pattern).FindAllString(s, n)
}

// FindNumber 提取第一个数字
func FindNumber(s string) string {
	return FindString(s, NumPattern)
}

// FindAllNumbers 提取所有数字
func FindAllNumbers(s string) []string {
	return FindAllString(s, NumPattern, -1)
}

// FindChinese 提取所有中文
func FindChinese(s string) string {
	return FindString(s, ChinesePattern)
}

// FindEmail 提取邮箱
func FindEmail(s string) string {
	return FindString(s, EmailPattern)
}

// FindPhone 提取手机号
func FindPhone(s string) string {
	return FindString(s, PhoneCNPattern)
}

// FindURL 提取 URL
func FindURL(s string) string {
	return FindString(s, URLPattern)
}

// FindAllURLs 提取所有 URL
func FindAllURLs(s string) []string {
	return FindAllString(s, URLPattern, -1)
}

// FindAllEmails 提取所有邮箱
func FindAllEmails(s string) []string {
	return FindAllString(s, EmailPattern, -1)
}

// FindAllPhones 提取所有手机号
func FindAllPhones(s string) []string {
	return FindAllString(s, PhoneCNPattern, -1)
}

// ===== 替换 =====

// ReplaceString 替换匹配项
func ReplaceString(s string, pattern string, repl string) string {
	return get(pattern).ReplaceAllString(s, repl)
}

// ReplaceStringFunc 替换匹配项（使用函数）
func ReplaceStringFunc(s string, pattern string, repl func(string) string) string {
	return get(pattern).ReplaceAllStringFunc(s, repl)
}

// ReplaceNumber 替换数字为指定字符
func ReplaceNumber(s string, repl string) string {
	return ReplaceString(s, NumPattern, repl)
}

// ReplaceChinese 替换中文为指定字符
func ReplaceChinese(s string, repl string) string {
	return ReplaceString(s, ChinesePattern, repl)
}

// ===== 分割 =====

// Split 分割字符串
func Split(s string, pattern string) []string {
	return get(pattern).Split(s, -1)
}

// SplitBySpace 按空白字符分割
func SplitBySpace(s string) []string {
	return strings.Fields(s)
}

// SplitByComma 按逗号分割
func SplitByComma(s string) []string {
	return Split(s, `,`)
}

// SplitBySemicolon 按分号分割
func SplitBySemicolon(s string) []string {
	return Split(s, `;`)
}

// SplitByNewline 按换行符分割
func SplitByNewline(s string) []string {
	return Split(s, `\r?\n`)
}

// ===== 数量统计 =====

// Count 统计匹配数量
func Count(s string, pattern string) int {
	return len(get(pattern).FindAllStringIndex(s, -1))
}

// CountNumbers 统计数字数量
func CountNumbers(s string) int {
	return Count(s, NumPattern)
}

// CountChinese 统计中文数量
func CountChinese(s string) int {
	return Count(s, ChinesePattern)
}

// CountEmails 统计邮箱数量
func CountEmails(s string) int {
	return Count(s, EmailPattern)
}

// CountPhones 统计手机号数量
func CountPhones(s string) int {
	return Count(s, PhoneCNPattern)
}

// CountURLs 统计 URL 数量
func CountURLs(s string) int {
	return Count(s, URLPattern)
}

// ===== 位置信息 =====

// FindStringIndex 返回匹配的起止位置
func FindStringIndex(s string, pattern string) []int {
	return get(pattern).FindStringIndex(s)
}

// FindAllStringIndex 返回所有匹配的起止位置
func FindAllStringIndex(s string, pattern string) [][]int {
	return get(pattern).FindAllStringIndex(s, -1)
}

// FindStringIndexNumber 返回第一个数字的位置
func FindStringIndexNumber(s string) []int {
	return FindStringIndex(s, NumPattern)
}

// ===== 验证函数 =====

// ValidateEmail 验证邮箱
func ValidateEmail(email string) bool {
	return IsMatchEmail(email)
}

// ValidatePhone 验证手机号（中国）
func ValidatePhone(phone string) bool {
	return IsMatchPhone(phone)
}

// ValidateURL 验证 URL
func ValidateURL(url string) bool {
	return IsMatchURL(url)
}

// ValidateIP 验证 IP 地址
func ValidateIP(ip string) bool {
	if !IsMatchIP(ip) {
		return false
	}
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		// 检查前导零
		if len(p) > 1 && strings.HasPrefix(p, "0") {
			return false
		}
		// 验证数值范围 0-255
		num := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
			num = num*10 + int(c-'0')
		}
		if num > 255 {
			return false
		}
	}
	return true
}

// ValidateIDCard 验证身份证号（中国）
func ValidateIDCard(idCard string) bool {
	return IsMatchIDCard(idCard)
}

// ValidateChineseName 验证中文姓名
func ValidateChineseName(name string) bool {
	return get(ChineseNamePattern).MatchString(name)
}

// ValidateUsername 验证用户名
func ValidateUsername(username string) bool {
	return get(UsernamePattern).MatchString(username)
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string, level int) bool {
	if len(password) < 6 {
		return false
	}

	// 基础字符检查
	if !get(PasswordPattern).MatchString(password) {
		return false
	}

	switch level {
	case 1: // 简单：6位以上
		return true
	case 2: // 中等：6位以上，包含数字和字母
		hasNum := get(`[0-9]`).MatchString(password)
		hasAlpha := get(`[a-zA-Z]`).MatchString(password)
		return hasNum && hasAlpha
	case 3: // 强：8位以上，包含大小写和数字
		if len(password) < 8 {
			return false
		}
		hasNum := get(`[0-9]`).MatchString(password)
		hasLower := get(`[a-z]`).MatchString(password)
		hasUpper := get(`[A-Z]`).MatchString(password)
		return hasNum && hasLower && hasUpper
	default:
		return true
	}
}

// ValidateCreditCode 验证社会信用代码（中国）
func ValidateCreditCode(code string) bool {
	return get(CreditCodePattern).MatchString(code)
}

// ValidateBankCard 验证银行卡号（中国）
func ValidateBankCard(card string) bool {
	return get(BankCardPattern).MatchString(card)
}

// ValidateZipCode 验证邮政编码（中国）
func ValidateZipCode(zipCode string) bool {
	return get(ZipCodePattern).MatchString(zipCode)
}
