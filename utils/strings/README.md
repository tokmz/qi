# String

字符串处理工具包，提供常用的字符串操作功能。

## 安装

```bash
go get qi/pkg/string
```

## 基础操作

| 函数 | 说明 |
|------|------|
| `IsEmpty(s string) bool` | 检查字符串是否为空 |
| `IsNotEmpty(s string) bool` | 检查字符串是否非空 |
| `IsBlank(s string) bool` | 检查字符串是否为空或只包含空白字符 |
| `IsNotBlank(s string) bool` | 检查字符串是否非空且包含非空白字符 |
| `Default(s, defaultValue string) string` | 如果字符串为空，返回默认值 |
| `DefaultIfEmpty(s, defaultValue string) string` | 如果字符串为空，返回默认值 |

## 大小写转换

| 函数 | 说明 |
|------|------|
| `ToUpper(s string) string` | 转为大写 |
| `ToLower(s string) string` | 转为小写 |
| `ToTitle(s string) string` | 转为标题格式 |
| `ToCamel(s string) string` | 转为 camelCase 格式 |
| `ToSnake(s string) string` | 转为 snake_case 格式 |
| `ToKebab(s string) string` | 转为 kebab-case 格式 |

## 裁剪

| 函数 | 说明 |
|------|------|
| `Trim(s string) string` | 去除两端空白 |
| `TrimLeft(s string) string` | 去除左侧空白 |
| `TrimRight(s string) string` | 去除右侧空白 |
| `TrimPrefix(s, prefix string) string` | 去除前缀 |
| `TrimSuffix(s, suffix string) string` | 去除后缀 |

## 查找

| 函数 | 说明 |
|------|------|
| `Contains(s, substr string) bool` | 检查是否包含子串 |
| `ContainsAny(s, chars string) bool` | 检查是否包含任意字符 |
| `ContainsRune(s string, r rune) bool` | 检查是否包含字符 |
| `HasPrefix(s, prefix string) bool` | 检查前缀 |
| `HasSuffix(s, suffix string) bool` | 检查后缀 |
| `Index(s, substr string) int` | 查找子串位置 |
| `LastIndex(s, substr string) int` | 查找最后出现位置 |

## 分割与合并

| 函数 | 说明 |
|------|------|
| `Split(s, sep string) []string` | 分割字符串 |
| `SplitN(s, sep string, n int) []string` | 分割指定次数 |
| `SplitAny(s, seps string) []string` | 按任意字符分割 |
| `Join(parts []string, sep string) string` | 合并字符串 |

## 替换

| 函数 | 说明 |
|------|------|
| `Replace(s, old, new string, n int) string` | 替换子串 |
| `ReplaceAll(s, old, new string) string` | 替换所有子串 |
| `ReplaceChars(s string, oldNew ...rune) string` | 替换指定字符 |

## 统计

| 函数 | 说明 |
|------|------|
| `Count(s, substr string) int` | 计算子串出现次数 |
| `Len(s string) int` | 计算字符数（UTF-8） |
| `WordCount(s string) int` | 计算单词数 |

## 大小写判断

| 函数 | 说明 |
|------|------|
| `IsUpper(s string) bool` | 检查是否全大写 |
| `IsLower(s string) bool` | 检查是否全小写 |
| `IsTitle(s string) bool` | 检查是否标题格式 |

## 类型判断

| 函数 | 说明 |
|------|------|
| `IsAlpha(s string) bool` | 检查是否只包含字母 |
| `IsNumber(s string) bool` | 检查是否只包含数字 |
| `IsAlphaNumber(s string) bool` | 检查是否只包含字母和数字 |
| `IsEmail(s string) bool` | 检查是否为邮箱格式 |
| `IsPhone(s string) bool` | 检查是否为手机号格式（中国） |
| `IsURL(s string) bool` | 检查是否为 URL 格式 |
| `IsIP(s string) bool` | 检查是否为 IP 地址 |
| `IsChinese(s string) bool` | 检查是否包含中文字符 |

## 截取

| 函数 | 说明 |
|------|------|
| `Substring(s string, start, length int) string` | 截取字符串 |
| `Truncate(s string, maxLen int, suffix string) string` | 截断字符串（添加省略号） |
| `TakeLeft(s string, n int) string` | 获取左侧 n 个字符 |
| `TakeRight(s string, n int) string` | 获取右侧 n 个字符 |

## 重复与填充

| 函数 | 说明 |
|------|------|
| `Repeat(s string, count int) string` | 重复字符串 |
| `PadLeft(s string, width int, padStr string) string` | 左侧填充 |
| `PadRight(s string, width int, padStr string) string` | 右侧填充 |
| `Center(s string, width int, padStr string) string` | 居中填充 |

## 转换

| 函数 | 说明 |
|------|------|
| `Reverse(s string) string` | 反转字符串 |
| `ToBytes(s string) []byte` | 转为字节数组 |
| `ToString(b []byte) string` | 转为字符串 |

## 拼音相关

| 函数 | 说明 |
|------|------|
| `IsPinyin(s string) bool` | 检查是否为有效的拼音 |
| `PinyinFirstChar(s string) string` | 获取拼音首字母 |
| `PinyinAbbr(s string) string` | 获取拼音缩写 |

## 其他

| 函数 | 说明 |
|------|------|
| `Nl2Br(s string) string` | 换行符转为 `<br>` |
| `Br2Nl(s string) string` | `<br>` 转为换行符 |
| `Quote(s string) string` | 添加引号 |
| `Unquote(s string) string` | 去除引号 |
| `Unique(s string) string` | 去重相邻重复字符 |
| `WordInitials(s string) string` | 获取单词首字母大写 |

## 使用示例

```go
package main

import (
	"fmt"

	"qi/pkg/string"
)

func main() {
	// 基础操作
	fmt.Println(string.IsEmpty(""))        // true
	fmt.Println(string.IsBlank("  "))     // true
	fmt.Println(string.Default("", "默认")) // "默认"

	// 大小写转换
	fmt.Println(string.ToCamel("hello_world")) // "helloWorld"
	fmt.Println(string.ToSnake("helloWorld"))   // "hello_world"
	fmt.Println(string.ToKebab("HelloWorld"))   // "hello-world"

	// 裁剪
	fmt.Println(string.Trim("  hello  "))      // "hello"
	fmt.Println(string.TrimPrefix("-hello", "-")) // "hello"

	// 查找
	fmt.Println(string.Contains("hello world", "world")) // true
	fmt.Println(string.HasPrefix("hello", "he"))        // true

	// 分割与合并
	parts := string.Split("a,b,c", ",")
	fmt.Println(parts) // ["a" "b" "c"]
	fmt.Println(string.Join(parts, "-")) // "a-b-c"

	// 截取
	fmt.Println(string.Substring("hello", 0, 3))    // "hel"
	fmt.Println(string.Truncate("hello world", 5, "...")) // "hello..."

	// 重复与填充
	fmt.Println(string.Repeat("a", 3))    // "aaa"
	fmt.Println(string.PadLeft("5", 3, "0")) // "005"
	fmt.Println(string.Center("ab", 5, "*")) // "*ab**"

	// 转换
	fmt.Println(string.Reverse("hello")) // "olleh"
	fmt.Println(string.WordInitials("hello world")) // "HW"

	// 类型判断
	fmt.Println(string.IsEmail("test@example.com")) // true
	fmt.Println(string.IsPhone("13800138000"))       // true
	fmt.Println(string.IsChinese("你好"))            // true
}
```
