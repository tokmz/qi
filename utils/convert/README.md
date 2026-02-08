# Convert

类型转换工具包，提供各种类型之间的转换功能。

## 安装

```bash
go get qi/pkg/convert
```

## 错误定义

```go
var (
    ErrInvalidType = errors.New("invalid type")
    ErrParseFailed = errors.New("parse failed")
    ErrEmptyString = errors.New("empty string")
    ErrOutOfRange = errors.New("value out of range")
)
```

## 基础类型转换

| 函数 | 说明 |
|------|------|
| `ToString(v any) string` | 任意类型转为字符串 |
| `ToInt(v any) (int, error)` | 任意类型转为 int（支持溢出检查） |
| `ToInt64(v any) (int64, error)` | 任意类型转为 int64 |
| `ToUint(v any) (uint, error)` | 任意类型转为 uint |
| `ToFloat64(v any) (float64, error)` | 任意类型转为 float64 |
| `ToBool(v any) (bool, error)` | 任意类型转为 bool |
| `ToBytes(v any) ([]byte, error)` | 任意类型转为 []byte |

## 字符串转换

| 函数 | 说明 |
|------|------|
| `StringToInt(s string) (int, error)` | 字符串转 int |
| `StringToInt64(s string) (int64, error)` | 字符串转 int64 |
| `StringToUint(s string) (uint, error)` | 字符串转 uint |
| `StringToFloat64(s string) (float64, error)` | 字符串转 float64 |
| `StringToBool(s string) (bool, error)` | 字符串转 bool |
| `StringToIntArray(s, sep string) ([]int, error)` | 字符串按分隔符转 int 数组 |
| `StringToStringArray(s, sep string) ([]string, error)` | 字符串按分隔符转字符串数组 |
| `SplitString(s, sep string) []string` | 分割字符串 |

## 时间转换

| 函数 | 说明 |
|------|------|
| `ToTime(v any) (time.Time, error)` | 任意类型转为 time.Time |
| `TimeToTimestamp(t time.Time) int64` | time.Time 转为时间戳（秒） |
| `TimeToTimestampMs(t time.Time) int64` | time.Time 转为时间戳（毫秒） |
| `TimestampToTime(ts int64) time.Time` | 时间戳转 time.Time |
| `TimestampMsToTime(ts int64) time.Time` | 毫秒时间戳转 time.Time |
| `NowTimestamp() int64` | 获取当前时间戳（秒） |
| `NowTimestampMs() int64` | 获取当前时间戳（毫秒） |

## Base64 转换

| 函数 | 说明 |
|------|------|
| `ToBase64(v any) (string, error)` | 编码为 Base64 |
| `FromBase64(s string) ([]byte, error)` | 解码 Base64 |
| `ToURLBase64(v any) (string, error)` | 编码为 URL Base64 |
| `FromURLBase64(s string) ([]byte, error)` | 解码 URL Base64 |
| `ToRawURLBase64(v any) (string, error)` | 编码为 Raw URL Base64 |
| `FromRawURLBase64(s string) ([]byte, error)` | 解码 Raw URL Base64 |

## Hex 转换

| 函数 | 说明 |
|------|------|
| `ToHex(v any) (string, error)` | 编码为十六进制字符串 |
| `FromHex(s string) ([]byte, error)` | 解码十六进制字符串 |
| `HexToInt(s string) (int64, error)` | 十六进制字符串转 int64 |
| `IntToHex(i int64) string` | int64 转十六进制字符串 |

## 使用示例

```go
package main

import (
	"fmt"
	"log"

	"qi/pkg/convert"
)

func main() {
	// 基础类型转换
	str := convert.ToString(123)
	fmt.Println(str) // "123"

	n, err := convert.ToInt("456")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n) // 456

	n64, err := convert.ToInt64("789")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n64) // 789

	f, err := convert.ToFloat64("3.14")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(f) // 3.14

	b, err := convert.ToBool("true")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(b) // true

	data, err := convert.ToBytes("hello")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data)) // "hello"

	// 字符串转换
	n, err = convert.StringToInt("100")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n) // 100

	f, err = convert.StringToFloat64("2.5")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(f) // 2.5

	b, err = convert.StringToBool("true")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(b) // true

	arr, err := convert.StringToIntArray("1,2,3", ",")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(arr) // [1 2 3]

	// 时间转换
	t, err := convert.ToTime("2024-01-15")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(convert.TimeToTimestamp(t)) // 1705305600

	// Base64 编码解码
	encoded, err := convert.ToBase64("hello")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(encoded) // "aGVsbG8="

	decoded, err := convert.FromBase64(encoded)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(decoded)) // "hello"

	// Hex 编码解码
	hexStr := convert.IntToHex(255)
	fmt.Println(hexStr) // "ff"

	i64, err := convert.HexToInt("ff")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(i64) // 255
}
```
