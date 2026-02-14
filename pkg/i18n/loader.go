package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader 翻译资源加载器接口
type Loader interface {
	// Load 加载翻译资源
	Load(ctx context.Context, dir string, languages []string) (map[string]map[string]string, error)
}

// JSONLoader JSON 文件加载器
type JSONLoader struct {
	Dir     string
	Pattern string
}

// Load 加载翻译资源
func (l *JSONLoader) Load(ctx context.Context, dir string, languages []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)

	for _, lang := range languages {
		filename := l.buildFilename(dir, lang)

		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue // 文件不存在，跳过
			}
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
		}

		result[lang] = flattenMap(raw, "")
	}

	return result, nil
}

// buildFilename 根据 Pattern 构建语言文件路径
func (l *JSONLoader) buildFilename(dir, lang string) string {
	pattern := l.Pattern
	if pattern == "" {
		pattern = "{lang}.json"
	}
	return filepath.Join(dir, strings.ReplaceAll(pattern, "{lang}", lang))
}

// flattenMap 将嵌套的 map 转换为扁平的 key-value
func flattenMap(m map[string]interface{}, prefix string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			result[key] = val
		case map[string]interface{}:
			for kk, vv := range flattenMap(val, key) {
				result[kk] = vv
			}
		default:
			result[key] = fmt.Sprintf("%v", v)
		}
	}
	return result
}
