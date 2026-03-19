package convert

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"time"
)

// ===== 错误定义 =====

var (
	ErrInvalidType   = errors.New("invalid type")
	ErrParseFailed  = errors.New("parse failed")
	ErrEmptyString  = errors.New("empty string")
	ErrOutOfRange  = errors.New("value out of range")
)

// ===== 基础类型转换 =====

// ToString 转为字符串
func ToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.Itoa(int(val))
	case int16:
		return strconv.Itoa(int(val))
	case int32:
		return strconv.Itoa(int(val))
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []byte:
		return string(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	default:
		return ""
	}
}

// ToInt 转为 int
func ToInt(v any) (int, error) {
	if v == nil {
		return 0, ErrInvalidType
	}
	switch val := v.(type) {
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, ErrParseFailed
		}
		return i, nil
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		// 根据系统位数检查范围
		if bits.UintSize == 32 {
			// 32 位系统
			if val < math.MinInt32 || val > math.MaxInt32 {
				return 0, ErrOutOfRange
			}
		} else {
			// 64 位系统
			if val < int64(math.MinInt) || val > int64(math.MaxInt) {
				return 0, ErrOutOfRange
			}
		}
		return int(val), nil
	case uint:
		if val > math.MaxInt {
			return 0, ErrOutOfRange
		}
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		// 根据系统位数检查范围
		if bits.UintSize == 32 {
			// 32 位系统
			if val > math.MaxInt32 {
				return 0, ErrOutOfRange
			}
		} else {
			// 64 位系统
			if uint64(val) > uint64(math.MaxInt) {
				return 0, ErrOutOfRange
			}
		}
		return int(val), nil
	case uint64:
		if val > math.MaxInt {
			return 0, ErrOutOfRange
		}
		return int(val), nil
	case float32:
		if val > math.MaxInt || val < math.MinInt {
			return 0, ErrOutOfRange
		}
		return int(val), nil
	case float64:
		if val > math.MaxInt || val < math.MinInt {
			return 0, ErrOutOfRange
		}
		return int(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, ErrInvalidType
	}
}

// ToInt64 转为 int64
func ToInt64(v any) (int64, error) {
	if v == nil {
		return 0, ErrInvalidType
	}
	switch val := v.(type) {
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, ErrParseFailed
		}
		return i, nil
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		if val > math.MaxInt64 {
			return 0, ErrOutOfRange
		}
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, ErrInvalidType
	}
}

// ToFloat64 转为 float64
func ToFloat64(v any) (float64, error) {
	if v == nil {
		return 0, ErrInvalidType
	}
	switch val := v.(type) {
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, ErrParseFailed
		}
		return f, nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, ErrInvalidType
	}
}

// ToBool 转为 bool
func ToBool(v any) (bool, error) {
	if v == nil {
		return false, ErrInvalidType
	}
	switch val := v.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(val))
		return s == "1" || s == "true" || s == "yes" || s == "on", nil
	case int:
		return val != 0, nil
	case int8:
		return val != 0, nil
	case int16:
		return val != 0, nil
	case int32:
		return val != 0, nil
	case int64:
		return val != 0, nil
	case uint:
		return val != 0, nil
	case uint8:
		return val != 0, nil
	case uint16:
		return val != 0, nil
	case uint32:
		return val != 0, nil
	case uint64:
		return val != 0, nil
	case float32:
		return val != 0, nil
	case float64:
		return val != 0, nil
	case bool:
		return val, nil
	default:
		return false, ErrInvalidType
	}
}

// ToBytes 转为 []byte
func ToBytes(v any) ([]byte, error) {
	if v == nil {
		return nil, ErrInvalidType
	}
	switch val := v.(type) {
	case string:
		return []byte(val), nil
	case []byte:
		return val, nil
	default:
		return nil, ErrInvalidType
	}
}

// ToUint 转为 uint
func ToUint(v any) (uint, error) {
	if v == nil {
		return 0, ErrInvalidType
	}
	switch val := v.(type) {
	case string:
		i, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return 0, ErrParseFailed
		}
		return uint(i), nil
	case int:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case int8:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case int16:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case int32:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case int64:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case uint:
		return val, nil
	case uint8:
		return uint(val), nil
	case uint16:
		return uint(val), nil
	case uint32:
		return uint(val), nil
	case uint64:
		return uint(val), nil
	case float32:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case float64:
		if val < 0 {
			return 0, ErrOutOfRange
		}
		return uint(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, ErrInvalidType
	}
}

// ===== 字符串转换 =====

// StringToInt 字符串转 int
func StringToInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmptyString
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, ErrParseFailed
	}
	return i, nil
}

// StringToInt64 字符串转 int64
func StringToInt64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmptyString
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ErrParseFailed
	}
	return i, nil
}

// StringToFloat64 字符串转 float64
func StringToFloat64(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmptyString
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, ErrParseFailed
	}
	return f, nil
}

// StringToBool 字符串转 bool
func StringToBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false, ErrEmptyString
	}
	return s == "1" || s == "true" || s == "yes" || s == "on", nil
}

// StringToUint 字符串转 uint
func StringToUint(s string) (uint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmptyString
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, ErrParseFailed
	}
	return uint(i), nil
}

// StringToIntArray 字符串按分隔符转 int 数组
func StringToIntArray(s string, sep string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyString
	}
	parts := SplitString(s, sep)
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		i, err := strconv.Atoi(p)
		if err != nil {
			return nil, ErrParseFailed
		}
		result = append(result, i)
	}
	return result, nil
}

// StringToStringArray 字符串按分隔符转字符串数组
func StringToStringArray(s string, sep string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyString
	}
	parts := SplitString(s, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, nil
}

// SplitString 分割字符串
func SplitString(s string, sep string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, sep)
}

// ===== 时间转换 =====

// ToTime 转为 time.Time
func ToTime(v any) (time.Time, error) {
	switch val := v.(type) {
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return time.Time{}, ErrEmptyString
		}
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			"2006-01-02T15:04:05Z07:00",
			time.RFC3339,
			time.RFC3339Nano,
		}
		for _, format := range formats {
			if parsedTime, err := time.Parse(format, s); err == nil {
				return parsedTime, nil
			}
		}
		return time.Time{}, ErrParseFailed
	case int64:
		return time.Unix(val, 0), nil
	case int:
		return time.Unix(int64(val), 0), nil
	case time.Time:
		return val, nil
	default:
		return time.Time{}, ErrInvalidType
	}
}

// TimeToTimestamp time.Time 转为时间戳（秒）
func TimeToTimestamp(t time.Time) int64 {
	return t.Unix()
}

// TimeToTimestampMs time.Time 转为时间戳（毫秒）
func TimeToTimestampMs(t time.Time) int64 {
	return t.UnixMilli()
}

// TimestampToTime 时间戳转 time.Time
func TimestampToTime(ts int64) time.Time {
	return time.Unix(ts, 0)
}

// TimestampMsToTime 毫秒时间戳转 time.Time
func TimestampMsToTime(ts int64) time.Time {
	return time.UnixMilli(ts)
}

// NowTimestamp 获取当前时间戳（秒）
func NowTimestamp() int64 {
	return time.Now().Unix()
}

// NowTimestampMs 获取当前时间戳（毫秒）
func NowTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// ===== Base64 转换 =====

// ToBase64 编码为 Base64
func ToBase64(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return base64.StdEncoding.EncodeToString([]byte(val)), nil
	case []byte:
		return base64.StdEncoding.EncodeToString(val), nil
	default:
		return "", ErrInvalidType
	}
}

// FromBase64 解码 Base64
func FromBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyString
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrParseFailed
	}
	return data, nil
}

// ToURLBase64 编码为 URL Base64
func ToURLBase64(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return base64.URLEncoding.EncodeToString([]byte(val)), nil
	case []byte:
		return base64.URLEncoding.EncodeToString(val), nil
	default:
		return "", ErrInvalidType
	}
}

// FromURLBase64 解码 URL Base64
func FromURLBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyString
	}
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrParseFailed
	}
	return data, nil
}

// ToRawURLBase64 编码为 Raw URL Base64（无填充）
func ToRawURLBase64(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return base64.RawURLEncoding.EncodeToString([]byte(val)), nil
	case []byte:
		return base64.RawURLEncoding.EncodeToString(val), nil
	default:
		return "", ErrInvalidType
	}
}

// FromRawURLBase64 解码 Raw URL Base64
func FromRawURLBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyString
	}
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrParseFailed
	}
	return data, nil
}

// ===== Hex 转换 =====

// ToHex 转为十六进制字符串
func ToHex(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return hex.EncodeToString([]byte(val)), nil
	case []byte:
		return hex.EncodeToString(val), nil
	default:
		return "", ErrInvalidType
	}
}

// FromHex 从十六进制字符串解析
func FromHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrEmptyString
	}
	if len(s)%2 != 0 {
		return nil, ErrParseFailed
	}
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, ErrParseFailed
	}
	return data, nil
}

// HexToInt 十六进制字符串转 int64
func HexToInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmptyString
	}
	i, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, ErrParseFailed
	}
	return i, nil
}

// IntToHex int64 转十六进制字符串
func IntToHex(i int64) string {
	return strconv.FormatInt(i, 16)
}
