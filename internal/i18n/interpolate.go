package i18n

import (
	"fmt"
	"strings"
)

// interpolate 将 s 中的 {key} 占位符替换为 args 中对应的值。
// args 为交替的 string key-value 对："name", "Alice", "count", 5。
// 规则：
//   - args 长度为奇数时最后一个参数忽略
//   - args 中 key 必须为 string 类型，否则跳过该对
//   - value 通过 fmt.Sprint 转为字符串
//   - 未提供的 {key} 保留原样
func interpolate(s string, args []any) string {
	if len(args) < 2 || !strings.Contains(s, "{") {
		return s
	}
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		val := fmt.Sprint(args[i+1])
		s = strings.ReplaceAll(s, "{"+key+"}", val)
	}
	return s
}
