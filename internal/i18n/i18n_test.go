package i18n

import (
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

// newTestBundle 构造一个带预设翻译的 Bundle，供多个测试复用。
func newTestBundle(t *testing.T) *Bundle {
	t.Helper()
	fsys := fstest.MapFS{
		"zh.json": &fstest.MapFile{Data: []byte(`{
			"welcome":     "欢迎, {name}！",
			"items_count": {"zero": "没有项目", "one": "{count} 个项目", "other": "{count} 个项目"},
			"only_zh":     "仅中文"
		}`)},
		"zh-cn.json": &fstest.MapFile{Data: []byte(`{
			"greeting": "你好"
		}`)},
		"en.json": &fstest.MapFile{Data: []byte(`{
			"welcome":     "Welcome, {name}!",
			"items_count": {"zero": "No items", "one": "{count} item", "other": "{count} items"}
		}`)},
	}
	b := NewBundle(WithFallback("en"))
	if err := b.LoadFS(fsys, "*.json"); err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	return b
}

// ===== Bundle =====

func TestBundle_LoadFS_unknownGlob(t *testing.T) {
	b := NewBundle()
	err := b.LoadFS(fstest.MapFS{}, "*.json")
	if err == nil {
		t.Fatal("expected error for empty glob match")
	}
}

func TestBundle_LoadFS_badJSON(t *testing.T) {
	fsys := fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`not json`)},
	}
	b := NewBundle()
	if err := b.LoadFS(fsys, "*.json"); err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestBundle_LoadFS_merge(t *testing.T) {
	// 同一 tag 两次加载，后加载覆盖同 key
	fsys := fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`{"hello": "Hello"}`)},
	}
	b := NewBundle()
	if err := b.LoadFS(fsys, "*.json"); err != nil {
		t.Fatal(err)
	}
	fsys2 := fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`{"hello": "Hi", "bye": "Bye"}`)},
	}
	if err := b.LoadFS(fsys2, "*.json"); err != nil {
		t.Fatal(err)
	}
	t.Run("overwritten key", func(t *testing.T) {
		if got := b.For("en").T("hello"); got != "Hi" {
			t.Errorf("got %q, want %q", got, "Hi")
		}
	})
	t.Run("new key", func(t *testing.T) {
		if got := b.For("en").T("bye"); got != "Bye" {
			t.Errorf("got %q, want %q", got, "Bye")
		}
	})
}

// ===== Translator.T =====

func TestTranslator_T_basic(t *testing.T) {
	b := newTestBundle(t)
	got := b.For("zh").T("welcome", "name", "Alice")
	if got != "欢迎, Alice！" {
		t.Errorf("got %q", got)
	}
}

func TestTranslator_T_missingKey(t *testing.T) {
	b := newTestBundle(t)
	if got := b.For("zh").T("nonexistent"); got != "nonexistent" {
		t.Errorf("expected key fallback, got %q", got)
	}
}

func TestTranslator_T_missingArg(t *testing.T) {
	b := newTestBundle(t)
	// {name} 未提供，保留原样
	got := b.For("zh").T("welcome")
	if !strings.Contains(got, "{name}") {
		t.Errorf("expected {name} preserved, got %q", got)
	}
}

func TestTranslator_T_oddArgs(t *testing.T) {
	b := newTestBundle(t)
	// 奇数个 args，最后一个忽略
	got := b.For("zh").T("welcome", "name", "Alice", "extra")
	if got != "欢迎, Alice！" {
		t.Errorf("got %q", got)
	}
}

func TestTranslator_T_nonStringKey(t *testing.T) {
	b := newTestBundle(t)
	// args 中非 string key 跳过
	got := b.For("zh").T("welcome", 123, "Alice")
	if !strings.Contains(got, "{name}") {
		t.Errorf("expected {name} preserved, got %q", got)
	}
}

func TestTranslator_T_caseInsensitive(t *testing.T) {
	b := newTestBundle(t)
	got1 := b.For("zh-CN").T("greeting")
	got2 := b.For("zh-cn").T("greeting")
	if got1 != got2 {
		t.Errorf("case sensitivity mismatch: %q vs %q", got1, got2)
	}
	if got1 != "你好" {
		t.Errorf("got %q, want %q", got1, "你好")
	}
}

// ===== Translator.N =====

func TestTranslator_N_forms(t *testing.T) {
	b := newTestBundle(t)
	cases := []struct {
		count int
		want  string
	}{
		{0, "没有项目"},
		{1, "1 个项目"},
		{5, "5 个项目"},
	}
	for _, tc := range cases {
		got := b.For("zh").N("items_count", tc.count)
		if got != tc.want {
			t.Errorf("count=%d: got %q, want %q", tc.count, got, tc.want)
		}
	}
}

func TestTranslator_N_en(t *testing.T) {
	b := newTestBundle(t)
	cases := []struct {
		count int
		want  string
	}{
		{0, "No items"},
		{1, "1 item"},
		{2, "2 items"},
	}
	for _, tc := range cases {
		got := b.For("en").N("items_count", tc.count)
		if got != tc.want {
			t.Errorf("count=%d: got %q, want %q", tc.count, got, tc.want)
		}
	}
}

func TestTranslator_N_missingKey(t *testing.T) {
	b := newTestBundle(t)
	if got := b.For("zh").N("nonexistent", 3); got != "nonexistent" {
		t.Errorf("expected key fallback, got %q", got)
	}
}

// ===== 回退链 =====

func TestFallback_parentLang(t *testing.T) {
	b := newTestBundle(t)
	// zh-cn 无 "only_zh"，应回退到 zh
	got := b.For("zh-cn").T("only_zh")
	if got != "仅中文" {
		t.Errorf("got %q, want %q", got, "仅中文")
	}
}

func TestFallback_toFallbackLang(t *testing.T) {
	b := newTestBundle(t)
	// fr 无翻译，回退到 fallback "en"
	got := b.For("fr").T("welcome", "name", "Alice")
	if got != "Welcome, Alice!" {
		t.Errorf("got %q, want %q", got, "Welcome, Alice!")
	}
}

func TestFallback_ultimateKeyFallback(t *testing.T) {
	b := newTestBundle(t)
	// 所有语言都无此 key，返回 key 原文
	if got := b.For("fr").T("unknown_key"); got != "unknown_key" {
		t.Errorf("got %q, want %q", got, "unknown_key")
	}
}

func TestParentLang(t *testing.T) {
	cases := []struct{ in, want string }{
		{"zh-cn", "zh"},
		{"en-us", "en"},
		{"zh-hant-tw", "zh"},
		{"en", "en"},
	}
	for _, tc := range cases {
		if got := parentLang(tc.in); got != tc.want {
			t.Errorf("parentLang(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ===== 自定义复数规则 =====

func TestCustomPluralRule(t *testing.T) {
	fsys := fstest.MapFS{
		"pl.json": &fstest.MapFile{Data: []byte(`{
			"apples": {"one": "{count} jabłko", "other": "{count} jabłek"}
		}`)},
	}
	// 简化的波兰语复数规则（仅用于测试）
	plRule := func(count int) PluralForm {
		if count == 1 {
			return PluralOne
		}
		return PluralOther
	}
	b := NewBundle(WithPluralRule("pl", plRule))
	if err := b.LoadFS(fsys, "*.json"); err != nil {
		t.Fatal(err)
	}
	if got := b.For("pl").N("apples", 1); got != "1 jabłko" {
		t.Errorf("got %q", got)
	}
	if got := b.For("pl").N("apples", 5); got != "5 jabłek" {
		t.Errorf("got %q", got)
	}
}

// ===== Detector =====

func TestQueryDetector(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/?lang=zh", nil)
	if got := (QueryDetector{}).Detect(r); got != "zh" {
		t.Errorf("got %q", got)
	}
}

func TestCookieDetector(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "lang", Value: "zh-CN"})
	if got := (CookieDetector{}).Detect(r); got != "zh-CN" {
		t.Errorf("got %q", got)
	}
}

func TestHeaderDetector(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Lang", "en")
	if got := (HeaderDetector{}).Detect(r); got != "en" {
		t.Errorf("got %q", got)
	}
}

func TestAcceptDetector(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	if got := (AcceptDetector{}).Detect(r); got != "zh-CN" {
		t.Errorf("got %q", got)
	}
}

func TestChainDetector_priority(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/?lang=zh", nil)
	r.Header.Set("Accept-Language", "en")
	det := Chain(QueryDetector{}, AcceptDetector{})
	if got := det.Detect(r); got != "zh" {
		t.Errorf("got %q, want query result", got)
	}
}

func TestChainDetector_fallsThrough(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "en")
	det := Chain(QueryDetector{}, AcceptDetector{})
	if got := det.Detect(r); got != "en" {
		t.Errorf("got %q, want accept-language result", got)
	}
}

func TestForRequest_emptyDetector(t *testing.T) {
	b := newTestBundle(t)
	// 无任何语言信号，应降级到 fallback
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	trans := b.ForRequest(r)
	if trans.Lang() != "en" {
		t.Errorf("got lang %q, want %q", trans.Lang(), "en")
	}
}

// ===== interpolate =====

func TestInterpolate(t *testing.T) {
	cases := []struct {
		s    string
		args []any
		want string
	}{
		{"hello {name}", []any{"name", "Alice"}, "hello Alice"},
		{"no placeholders", []any{"name", "Alice"}, "no placeholders"},
		{"{a} {b}", []any{"a", "1", "b", "2"}, "1 2"},
		{"{x} {y}", []any{"x", "ok"}, "ok {y}"},   // {y} 无对应 arg，保留原样
		{"{n}", []any{"n", 42}, "42"},               // int value
	}
	for _, tc := range cases {
		if got := interpolate(tc.s, tc.args); got != tc.want {
			t.Errorf("interpolate(%q, %v) = %q, want %q", tc.s, tc.args, got, tc.want)
		}
	}
}

// ===== SimplePluralFunc =====

func TestSimplePluralFunc(t *testing.T) {
	cases := []struct {
		count int
		want  PluralForm
	}{
		{0, PluralZero},
		{1, PluralOne},
		{2, PluralOther},
		{100, PluralOther},
	}
	for _, tc := range cases {
		if got := SimplePluralFunc(tc.count); got != tc.want {
			t.Errorf("count=%d: got %q, want %q", tc.count, got, tc.want)
		}
	}
}

// ===== Translator.Lang =====

func TestTranslator_Lang(t *testing.T) {
	b := newTestBundle(t)
	if got := b.For("zh-CN").Lang(); got != "zh-cn" {
		t.Errorf("got %q, want %q", got, "zh-cn")
	}
}

// ===== AcceptDetector 边界 =====

func TestAcceptDetector_tooLong(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", strings.Repeat("a", 36))
	if got := (AcceptDetector{}).Detect(r); got != "" {
		t.Errorf("expected empty for oversized tag, got %q", got)
	}
}

func TestAcceptDetector_empty(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	if got := (AcceptDetector{}).Detect(r); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ===== entry.Other 为空时触发回退链 =====

func TestFallback_emptyOther(t *testing.T) {
	// fr.json 的 key 只有 one，other 为空 → 应回退到 fallback en
	fsys := fstest.MapFS{
		"fr.json": &fstest.MapFile{Data: []byte(`{"apples": {"one": "une pomme"}}`)},
		"en.json": &fstest.MapFile{Data: []byte(`{"apples": "{count} apples"}`)},
	}
	b := NewBundle(WithFallback("en"))
	if err := b.LoadFS(fsys, "*.json"); err != nil {
		t.Fatal(err)
	}
	// count=5 → PluralOther → fr entry.Other="" → 回退到 en
	if got := b.For("fr").N("apples", 5); got != "5 apples" {
		t.Errorf("got %q, want %q", got, "5 apples")
	}
}

// ===== 复数规则父语言继承 =====

func TestPluralRule_parentInheritance(t *testing.T) {
	fsys := fstest.MapFS{
		"zh-tw.json": &fstest.MapFile{Data: []byte(`{"items": {"other": "{count} 個"}}`)},
	}
	// 注册 zh 的规则，zh-tw 应继承
	customRule := func(count int) PluralForm { return PluralOther }
	b := NewBundle(WithPluralRule("zh", customRule))
	if err := b.LoadFS(fsys, "*.json"); err != nil {
		t.Fatal(err)
	}
	if got := b.For("zh-tw").N("items", 3); got != "3 個" {
		t.Errorf("got %q, want %q", got, "3 個")
	}
}

// ===== loader: path.Base 跨平台 =====

func TestParseFile_tagFromNestedPath(t *testing.T) {
	// fs.Glob 返回带目录前缀的路径，如 "locales/zh-CN.json"
	// path.Base 应正确提取 "zh-cn"
	fsys := fstest.MapFS{
		"locales/zh-CN.json": &fstest.MapFile{Data: []byte(`{"hi": "你好"}`)},
	}
	b := NewBundle()
	if err := b.LoadFS(fsys, "locales/*.json"); err != nil {
		t.Fatal(err)
	}
	if got := b.For("zh-CN").T("hi"); got != "你好" {
		t.Errorf("got %q, want %q", got, "你好")
	}
}

func TestLoadFS_interface(t *testing.T) {
	var _ fs.FS = fstest.MapFS{}
	fsys := fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`{"hi": "Hi"}`)},
	}
	b := NewBundle()
	if err := b.LoadFS(fsys, "*.json"); err != nil {
		t.Fatal(err)
	}
	if got := b.For("en").T("hi"); got != "Hi" {
		t.Errorf("got %q", got)
	}
}
