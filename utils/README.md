# Utils

常用工具包集合，提供字符串处理、切片操作、类型转换、时间处理、指针操作和正则表达式等功能。

## 安装

```bash
go get qi/v2/utils
```

---

## Strings - 字符串处理

提供常用的字符串操作功能。

```go
import "qi/v2/utils/strings"
```

| 函数 | 说明 |
|------|------|
| **基础操作** | |
| `IsEmpty(s string) bool` | 检查字符串是否为空 |
| `IsNotEmpty(s string) bool` | 检查字符串是否非空 |
| `IsBlank(s string) bool` | 检查是否为空或只包含空白字符 |
| `IsNotBlank(s string) bool` | 检查是否非空且包含非空白字符 |
| `Default(s, defaultValue string) string` | 空字符串返回默认值 |
| **大小写转换** | |
| `ToUpper/ToLower/ToTitle(s string) string` | 转为大写/小写/标题格式 |
| `ToCamel(s string) string` | 转为 camelCase |
| `ToSnake(s string) string` | 转为 snake_case |
| `ToKebab(s string) string` | 转为 kebab-case |
| **裁剪** | |
| `Trim/TrimLeft/TrimRight(s string) string` | 去除两端/左/右空白 |
| `TrimPrefix/TrimSuffix(s, prefix/suffix string) string` | 去除前缀/后缀 |
| **查找** | |
| `Contains/HasPrefix/HasSuffix(s, substr string) bool` | 包含/前缀/后缀判断 |
| `Index/LastIndex(s, substr string) int` | 子串位置/最后位置 |
| **分割与合并** | |
| `Split/SplitN(s, sep string, n int) []string` | 分割字符串 |
| `SplitAny(s, seps string) []string` | 按任意字符分割 |
| `Join(parts []string, sep string) string` | 合并字符串 |
| **替换** | |
| `Replace/ReplaceAll(s, old, new string, n int) string` | 替换子串 |
| `ReplaceChars(s string, oldNew ...rune) string` | 替换指定字符 |
| **统计** | |
| `Count(s, substr string) int` | 子串出现次数 |
| `Len(s string) int` | 字符数（UTF-8） |
| `WordCount(s string) int` | 单词数 |
| **类型判断** | |
| `IsAlpha/IsNumber/IsAlphaNumber(s string) bool` | 字母/数字/字母数字判断 |
| `IsEmail/IsPhone/IsURL/IsIP(s string) bool` | 邮箱/手机号/URL/IP 判断 |
| `IsChinese(s string) bool` | 包含中文字符 |
| **截取** | |
| `Substring(s string, start, length int) string` | 截取字符串 |
| `Truncate(s string, maxLen int, suffix string) string` | 截断并添加省略号 |
| `TakeLeft/TakeRight(s string, n int) string` | 获取左侧/右侧 n 个字符 |
| **重复与填充** | |
| `Repeat(s string, count int) string` | 重复字符串 |
| `PadLeft/PadRight(s string, width int, padStr string) string` | 左/右填充 |
| `Center(s string, width int, padStr string) string` | 居中填充 |
| **转换** | |
| `Reverse(s string) string` | 反转字符串 |
| `ToBytes/ToString(b []byte) []byte/string` | 字节数组互转 |
| **拼音相关** | |
| `PinyinFirstChar/PinyinAbbr(s string) string` | 拼音首字母/缩写 |
| **其他** | |
| `Nl2Br/Br2Nl(s string) string` | 换行符与 `<br>` 互转 |
| `Quote/Unquote(s string) string` | 添加/去除引号 |

```go
fmt.Println(strings.ToCamel("hello_world"))    // "helloWorld"
fmt.Println(strings.ToSnake("helloWorld"))      // "hello_world"
fmt.Println(strings.IsEmail("test@example.com")) // true
```

---

## Array - 切片工具

提供常用的切片操作函数，基于泛型实现。

```go
import "qi/v2/utils/array"
```

| 函数 | 说明 |
|------|------|
| **基础操作** | |
| `SliceContains[T comparable](slice []T, item T) bool` | 检查是否包含元素 |
| `SliceRemove/SliceRemoveAtIndex` | 移除元素/指定索引 |
| `SliceUnique/SliceReverse/SliceShuffle` | 去重/反转/打乱 |
| `SliceChunk(slice []T, size int) [][]T` | 分块 |
| `SliceSplit/SliceJoin` | 字符串分割/连接 |
| **集合操作** | |
| `SliceIntersect(slice1, slice2 []T) []T` | 交集 |
| `SliceUnion(slice1, slice2 []T) []T` | 并集 |
| `SliceDifference(slice1, slice2 []T) []T` | 差集 |
| **函数式操作** | |
| `SliceFilter(slice []T, predicate func(T) bool) []T` | 过滤 |
| `SliceMap(slice []T, mapper func(T) R) []R` | 映射 |
| `SliceReduce(slice []T, initial R, reducer func(R, T) R) R` | 规约 |
| `SliceFind/SliceFindIndex` | 查找元素/索引 |
| `SliceCount/SliceEvery/SliceSome` | 统计/全满足/任一满足 |
| **判断** | |
| `SliceEqual(slice1, slice2 []T) bool` | 相等判断 |
| `SliceIsEmpty/SliceIsNotEmpty` | 空/非空判断 |
| **获取元素** | |
| `SliceFirst/SliceLast(slice []T) (T, bool)` | 获取首/尾元素 |
| **统计计算** | |
| `SliceMin/SliceMax(slice []T) (T, bool)` | 最小值/最大值 |
| `SliceSum/SliceAverage(slice []T) T/(T, bool)` | 求和/平均值 |

```go
nums := []int{1, 2, 3, 4, 5}
evens := array.SliceFilter(nums, func(n int) bool { return n%2 == 0 })
doubles := array.SliceMap(nums, func(n int) int { return n * 2 })
sum := array.SliceReduce(nums, 0, func(acc, n int) int { return acc + n })
```

---

## Convert - 类型转换

提供各种类型之间的转换功能。

```go
import "qi/v2/utils/convert"
```

| 函数 | 说明 |
|------|------|
| **基础类型** | |
| `ToString(v any) string` | 任意类型转为字符串 |
| `ToInt/ToInt64/ToUint(v any) (int/int64/uint, error)` | 转为整数 |
| `ToFloat64(v any) (float64, error)` | 转为浮点数 |
| `ToBool(v any) (bool, error)` | 转为布尔 |
| `ToBytes(v any) ([]byte, error)` | 转为字节数组 |
| **字符串转换** | |
| `StringToInt/StringToFloat64/StringToBool` | 字符串转对应类型 |
| `StringToIntArray/StringToStringArray(s, sep string)` | 按分隔符转数组 |
| **时间转换** | |
| `ToTime(v any) (time.Time, error)` | 任意类型转时间 |
| `TimeToTimestamp/TimeToTimestampMs(t time.Time) int64` | 时间转时间戳 |
| `TimestampToTime/TimestampMsToTime(ts int64) time.Time` | 时间戳转时间 |
| `NowTimestamp/NowTimestampMs() int64` | 当前时间戳 |
| **Base64** | |
| `ToBase64/FromBase64(v any) (string/[]byte, error)` | Base64 编码/解码 |
| `ToURLBase64/FromURLBase64` | URL Base64 |
| `ToRawURLBase64/FromRawURLBase64` | Raw URL Base64 |
| **Hex** | |
| `ToHex/FromHex(v any) (string/[]byte, error)` | Hex 编码/解码 |
| `HexToInt/IntToHex(s string) (int64/string)` | Hex 与整数互转 |

```go
str := convert.ToString(123)                  // "123"
encoded, _ := convert.ToBase64("hello")      // "aGVsbG8="
hexStr := convert.IntToHex(255)               // "ff"
```

---

## DateTime - 时间处理

提供时间格式化、解析、计算等功能。

```go
import "qi/v2/utils/datetime"
```

**时间常量**：`DateFormat` (2006-01-02), `DateTimeFormat` (2006-01-02 15:04:05), `TimeFormat`, `RFC3339Format`

| 函数 | 说明 |
|------|------|
| **格式化** | |
| `FormatDate/FormatDateTime/FormatTime/FormatRFC3339(t time.Time) string` | 格式化时间 |
| `FormatUnix(ts int64) string` | Unix 时间戳转字符串 |
| `FormatDuration(d time.Duration) string` | 时长转人类可读字符串 |
| **解析** | |
| `ParseDate/ParseDateTime/ParseRFC3339(s string) (time.Time, error)` | 解析时间字符串 |
| `ParseUnix/ParseFromTimestamp(ts int64, isMs bool) time.Time` | 解析时间戳 |
| **计算** | |
| `StartOfDay/EndOfDay/StartOfWeek/EndOfWeek` | 日/周开始/结束时间 |
| `StartOfMonth/EndOfMonth/StartOfYear/EndOfYear` | 月/年开始/结束时间 |
| **判断** | |
| `IsToday/IsYesterday/IsTomorrow/IsThisWeek/IsThisMonth/IsThisYear` | 是否是今天/昨天/明天/本周/本月/本年 |
| `IsWeekend/IsLeapYear(year int) bool` | 周末/闰年判断 |
| **转换** | |
| `ToUnix/ToUnixMs(t time.Time) int64` | 转 Unix 时间戳 |
| `TimestampToTime(ts int64) time.Time` | 时间戳转时间 |
| **时间段** | |
| `Age(birthday time.Time) int` | 计算年龄 |
| `DaysBetween/HoursBetween(start, end time.Time) int` | 相隔天数/小时数 |
| **时间范围** | |
| `GetDayRange/GetWeekRange/GetMonthRange/GetYearRange(t time.Time)` | 获取日/周/月/年的时间范围 |
| **时间偏移** | |
| `AddDays/Hours/Minutes/Seconds/Months/Years(t time.Time, n int) time.Time` | 增加天数/小时/分钟/秒/月/年 |
| **比较** | |
| `IsAfter/IsBefore(t, other time.Time) bool` | 之后/之前判断 |
| `IsBetween(t, start, end time.Time) bool` | 是否在范围内 |
| **其他** | |
| `NowUnix/NowUnixMs() int64` | 当前 Unix 时间戳 |
| `Since/Until(t time.Time) time.Duration` | 从/到指定时间的间隔 |
| **相对时间** | |
| `TimeAgo/TimeAgoEn(t time.Time) string` | 多久之前（中/英文） |
| `TimeLeft/TimeLeftEn(t time.Time) string` | 距离某个时间还有多久（中/英文） |
| `DurationText/DurationTextEn(d time.Duration) string` | 时间间隔文本（中/英文） |

```go
fmt.Println(datetime.FormatDate(now))                      // 2024-01-15
fmt.Println(datetime.TimeAgo(time.Now().Add(-5*time.Minute))) // 5分钟前
```

---

## Pointer - 指针操作

提供指针的创建、转换、判空等常用操作。

```go
import "qi/v2/utils/pointer"
```

| 函数 | 说明 |
|------|------|
| **基础转换** | |
| `Of[T any](v T) *T` | 返回值的指针 |
| `Get[T any](p *T) T` | 获取指针的值，nil 返回零值 |
| `GetOrDefault[T any](p *T, defaultValue T) T` | nil 返回默认值 |
| **判断** | |
| `IsNil/IsNotNil[T any](p *T) bool` | nil/非 nil 判断 |
| **条件取值** | |
| `Coalesce[T any](pointers ...*T) *T` | 返回第一个非 nil 指针 |
| **转换** | |
| `ToString/ToInt/ToInt64/ToFloat64/ToBool(p *T) T` | 指针转值 |
| `ToStringPtr/ToIntPtr(s string) *string/*int` | 字符串转指针 |
| **切片转换** | |
| `ToSlicePtr/ToSlice(ptrs []*T) []*T/[]T` | 切片与指针切片互转 |
| `FilterNotNil(ptrs []*T) []*T` | 过滤 nil 指针 |
| `MapPtr(slice []T, mapper func(T) *R) []*R` | 映射指针切片 |
| **Map 操作** | |
| `ToMapPtr(slice []V, keyFunc func(V) K) map[K]*V` | 切片转为 map（值为指针） |
| **比较** | |
| `Equal[T any](p1, p2 *T) bool` | 比较指针地址 |

```go
p := pointer.Of(42)
fmt.Println(pointer.Get(p))                        // 42
result := pointer.Coalesce(p1, pointer.Of(20), p3)
```

---

## Regexp - 正则表达式

提供常用正则验证和操作功能。

```go
import "qi/v2/utils/regexp"
```

### 预定义正则常量

| 类型 | 常量 |
|------|------|
| **数字** | `NumPattern`, `PositiveIntPattern`, `NegativeIntPattern`, `IntegerPattern`, `FloatPattern` |
| **字母** | `AlphaPattern`, `AlphaLowerPattern`, `AlphaUpperPattern`, `AlphaNumPattern` |
| **中文** | `ChinesePattern`, `ChineseNamePattern` |
| **联系方式** | `EmailPattern`, `PhoneCNPattern`, `TelephonePattern`, `IDCardPattern`, `QQPattern`, `WeChatPattern` |
| **网络** | `URLPattern`, `IPv4Pattern`, `DomainPattern`, `MACPattern`, `HexColorPattern` |
| **日期时间** | `DateYYYYMMDDPattern`, `DateTimePattern`, `TimeHHMMSSPattern` |
| **其他** | `UsernamePattern`, `PasswordPattern`, `ZipCodePattern`, `BankCardPattern`, `CreditCodePattern` |

| 函数 | 说明 |
|------|------|
| **匹配验证** | |
| `IsMatch(s, pattern string) bool` | 检查是否匹配正则 |
| `IsMatchNum/IsMatchEmail/IsMatchPhone/IsMatchChinese/IsMatchURL/IsMatchIP` | 常用匹配验证 |
| **提取** | |
| `FindString(s, pattern string) string` | 提取第一个匹配项 |
| `FindStringSubmatch(s, pattern string) []string` | 提取匹配项及子匹配 |
| `FindAllString(s, pattern string, n int) []string` | 提取所有匹配项 |
| `FindNumber/FindAllNumbers/FindChinese` | 提取数字/所有数字/中文 |
| `FindEmail/FindPhone/FindURL` | 提取邮箱/手机号/URL |
| `FindAllEmails/FindAllPhones/FindAllURLs` | 提取所有邮箱/手机号/URL |
| **替换** | |
| `ReplaceString(s, pattern, repl string) string` | 替换匹配项 |
| `ReplaceStringFunc(s, pattern string, f func(string) string) string` | 函数替换 |
| `ReplaceNumber/ReplaceChinese(s, repl string) string` | 替换数字/中文 |
| **分割** | |
| `Split(s, pattern string) []string` | 按正则分割 |
| `SplitBySpace/ByComma/BySemicolon/ByNewline(s string) []string` | 按指定字符分割 |
| **统计** | |
| `Count(s, pattern string) int` | 匹配数量 |
| `CountNumbers/CountChinese/CountEmails/CountPhones/CountURLs(s string) int` | 各类型数量统计 |
| **位置** | |
| `FindStringIndex(s, pattern string) []int` | 匹配的起止位置 |
| `FindAllStringIndex(s, pattern string) [][]int` | 所有匹配的起止位置 |
| **验证函数** | |
| `ValidateEmail/ValidatePhone/ValidateURL/ValidateIP` | 邮箱/手机号/URL/IP 验证 |
| `ValidateIDCard/ValidateChineseName` | 身份证/中文姓名验证 |
| `ValidateUsername/ValidatePassword(password string, level int)` | 用户名/密码强度验证 |
| `ValidateCreditCode/ValidateBankCard/ValidateZipCode` | 信用代码/银行卡/邮编验证 |

```go
fmt.Println(regexp.ValidateEmail("test@example.com")) // true
email := regexp.FindEmail("我的邮箱是 test@example.com")
hidden := regexp.ReplaceNumber("密码是 123456", "*") // "密码是 ******"
```
