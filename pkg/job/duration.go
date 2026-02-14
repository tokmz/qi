package job

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration 包装 time.Duration，提供友好的 JSON 序列化。
// JSON 输出为可读字符串（如 "5s"、"1m30s"），反序列化兼容字符串和纳秒数字。
type Duration time.Duration

// MarshalJSON 序列化为可读字符串，如 "5s"、"1h30m"
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON 反序列化，兼容字符串（"5s"）和数字（纳秒）
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch val := v.(type) {
	case string:
		dur, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid duration string %q: %w", val, err)
		}
		*d = Duration(dur)
	case float64:
		*d = Duration(time.Duration(int64(val)))
	default:
		return fmt.Errorf("invalid duration type: %T", v)
	}
	return nil
}

// Unwrap 返回底层 time.Duration
func (d Duration) Unwrap() time.Duration {
	return time.Duration(d)
}

// String 实现 fmt.Stringer
func (d Duration) String() string {
	return time.Duration(d).String()
}
