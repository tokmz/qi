package i18n

import (
	"context"
	"os"
	"testing"
)

func TestJSONLoader(t *testing.T) {
	dir := t.TempDir()

	zhCN := `{
		"hello": "你好",
		"user": {
			"name": "用户名",
			"login": "登录"
		}
	}`
	if err := os.WriteFile(dir+"/zh-CN.json", []byte(zhCN), 0644); err != nil {
		t.Fatal(err)
	}

	enUS := `{
		"hello": "Hello",
		"user": {
			"name": "Username",
			"login": "Login"
		}
	}`
	if err := os.WriteFile(dir+"/en-US.json", []byte(enUS), 0644); err != nil {
		t.Fatal(err)
	}

	loader := &JSONLoader{
		Dir:     dir,
		Pattern: "{lang}.json",
	}

	data, err := loader.Load(context.Background(), dir, []string{"zh-CN", "en-US"})
	if err != nil {
		t.Fatal(err)
	}

	if data["zh-CN"]["hello"] != "你好" {
		t.Errorf("expected 你好, got %s", data["zh-CN"]["hello"])
	}
	if data["zh-CN"]["user.name"] != "用户名" {
		t.Errorf("expected 用户名, got %s", data["zh-CN"]["user.name"])
	}
	if data["en-US"]["hello"] != "Hello" {
		t.Errorf("expected Hello, got %s", data["en-US"]["hello"])
	}
}

func TestTranslator(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/zh-CN.json", []byte(`{"hello": "你好 {{.Name}}"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/en-US.json", []byte(`{"hello": "Hello {{.Name}}"}`), 0644); err != nil {
		t.Fatal(err)
	}

	trans, err := NewWithOptions(
		WithDir(dir),
		WithDefaultLanguage("zh-CN"),
		WithLanguages("zh-CN", "en-US"),
		WithLazy(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	// 测试默认语言翻译
	ctx := context.Background()
	if got := trans.T(ctx, "hello", "Name", "Alice"); got != "你好 Alice" {
		t.Errorf("expected '你好 Alice', got '%s'", got)
	}

	// 测试切换语言
	ctx = WithLanguage(ctx, "en-US")
	if got := trans.T(ctx, "hello", "Name", "Alice"); got != "Hello Alice" {
		t.Errorf("expected 'Hello Alice', got '%s'", got)
	}

	// 测试 key 不存在时返回 key
	if got := trans.T(ctx, "nonexistent"); got != "nonexistent" {
		t.Errorf("expected 'nonexistent', got '%s'", got)
	}
}

func TestTranslatorPlural(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/en-US.json", []byte(`{
		"item_one": "{{.Count}} item",
		"item_other": "{{.Count}} items"
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	trans, err := NewWithOptions(
		WithDir(dir),
		WithDefaultLanguage("en-US"),
		WithLanguages("en-US"),
		WithLazy(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	if got := trans.Tn(ctx, "item_one", "item_other", 1); got != "1 item" {
		t.Errorf("expected '1 item', got '%s'", got)
	}
	if got := trans.Tn(ctx, "item_one", "item_other", 5); got != "5 items" {
		t.Errorf("expected '5 items', got '%s'", got)
	}
}

func TestLazyLoad(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/zh-CN.json", []byte(`{"hello": "你好"}`), 0644); err != nil {
		t.Fatal(err)
	}

	trans, err := NewWithOptions(
		WithDir(dir),
		WithDefaultLanguage("zh-CN"),
		WithLanguages("zh-CN"),
		WithLazy(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	// 懒加载模式下，T() 应自动加载语言文件
	ctx := context.Background()
	if got := trans.T(ctx, "hello"); got != "你好" {
		t.Errorf("expected '你好', got '%s'", got)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultLanguage = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.DefaultLanguage != "zh-CN" {
		t.Errorf("expected default language to be zh-CN, got %s", cfg.DefaultLanguage)
	}

	cfg = DefaultConfig()
	cfg.Languages = nil
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(cfg.Languages) == 0 {
		t.Error("expected languages to be set")
	}
}

func TestAvailableLanguages(t *testing.T) {
	trans, err := NewWithOptions(
		WithDefaultLanguage("zh-CN"),
		WithLanguages("zh-CN", "en-US", "ja-JP"),
	)
	if err != nil {
		t.Fatal(err)
	}

	langs := trans.AvailableLanguages()
	if len(langs) != 3 {
		t.Errorf("expected 3 languages, got %d", len(langs))
	}

	// 验证返回的是副本，修改不影响原始数据
	langs[0] = "modified"
	if trans.AvailableLanguages()[0] == "modified" {
		t.Error("AvailableLanguages should return a copy")
	}
}

func TestHasKey(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/zh-CN.json", []byte(`{"hello": "你好"}`), 0644); err != nil {
		t.Fatal(err)
	}

	trans, err := NewWithOptions(
		WithDir(dir),
		WithDefaultLanguage("zh-CN"),
		WithLanguages("zh-CN"),
		WithLazy(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !trans.HasKey("hello") {
		t.Error("expected key to exist")
	}
	if trans.HasKey("nonexistent") {
		t.Error("expected key to not exist")
	}
}
