package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

// loadFromFS 遍历 fsys 中匹配 glob 的文件，返回 tag → raw messages 映射。
// glob 仅支持单层通配符（如 "locales/*.json"），不支持 ** 递归。
func loadFromFS(fsys fs.FS, glob string) (map[string]map[string]json.RawMessage, error) {
	matches, err := fs.Glob(fsys, glob)
	if err != nil {
		return nil, fmt.Errorf("i18n: glob %q: %w", glob, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("i18n: no files matched glob %q", glob)
	}

	result := make(map[string]map[string]json.RawMessage, len(matches))
	for _, p := range matches {
		tag, data, err := parseFile(fsys, p)
		if err != nil {
			return nil, err
		}
		if existing, ok := result[tag]; ok {
			// 多次命中同一 tag：后加载覆盖同 key
			for k, v := range data {
				existing[k] = v
			}
		} else {
			result[tag] = data
		}
	}
	return result, nil
}

// parseFile 打开并解析单个翻译文件，从文件名推导 tag。
// 使用 path.Base（非 filepath.Base）以兼容 fs.FS 始终使用 / 的路径规范。
// 文件名去掉 .json 后缀后统一转小写作为 language tag。
func parseFile(fsys fs.FS, name string) (tag string, data map[string]json.RawMessage, err error) {
	base := path.Base(name)
	tag = strings.ToLower(strings.TrimSuffix(base, ".json"))

	f, err := fsys.Open(name)
	if err != nil {
		return "", nil, fmt.Errorf("i18n: open %q: %w", name, err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&data); err != nil {
		return "", nil, fmt.Errorf("i18n: parse %q: %w", name, err)
	}
	return tag, data, nil
}

// loadFromDir 从操作系统目录加载翻译文件，等价于 loadFromFS(os.DirFS(dir), "*.json")。
func loadFromDir(dir string) (map[string]map[string]json.RawMessage, error) {
	return loadFromFS(os.DirFS(dir), "*.json")
}
