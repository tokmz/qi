# internal/i18n 设计文档

## 概述

`internal/i18n` 是 qi 框架内部国际化支持包，仅供根包 `qi` 消费，不对外暴露。

根包通过中间件和 `Context` 方法向用户提供 i18n 能力。

---

## 目标

- 从 HTTP 请求中自动检测语言（Accept-Language / 查询参数 / Cookie / Header）
- 支持参数插值：`"欢迎, {name}！"`
- 支持 zero / one / other 三种复数形式（count-based）
- 缺失 key 时按回退链降级，最终返回 key 本身
- 翻译文件使用 JSON，支持 `embed.FS` 和运行时目录两种加载方式
- 零外部依赖（仅标准库）

**不支持的场景（明确边界）：**
- 不支持阿拉伯语等需要 6 种复数形式的语言（可通过自定义 `PluralFunc` + 扩展 entry 实现）
- glob 加载不支持 `**` 递归通配（Go 标准库 `fs.Glob` 限制）

---

## 目录结构

```
internal/i18n/
├── DESIGN.md          本文档
├── i18n.go            Bundle（核心入口）
├── locale.go          Locale（单语言翻译集合）
├── translator.go      Translator（绑定语言的翻译执行器）
├── detector.go        语言检测策略
├── interpolate.go     参数插值
├── plural.go          复数规则
├── loader.go          翻译文件加载（embed.FS / os.DirFS）
└── i18n_test.go       单元测试
```

---

## 核心类型

### Bundle

```go
// Bundle 持有所有语言的翻译数据，全生命周期共享一个实例。
//
// 线程安全约定：
//   - LoadFS / LoadDir 仅在服务启动阶段调用，调用期间不得并发读取。
//   - 加载完成后所有操作只读，For / ForRequest 并发安全。
//   - 不提供运行时热重载；需要热重载请替换整个 Bundle 实例（原子指针）。
type Bundle struct {
    mu          sync.RWMutex
    locales     map[string]*Locale  // 小写 language tag → Locale
    fallback    string              // 默认回退语言，如 "en"
    detector    Detector            // 请求语言检测策略
    pluralRules map[string]PluralFunc
}

func NewBundle(opts ...BundleOption) *Bundle

// LoadFS 从 fs.FS 加载匹配 glob 的翻译文件。
// glob 仅支持单层通配符（如 "locales/*.json"），不支持 **。
// 多次调用时，后加载的同 key 值覆盖先前值。
func (b *Bundle) LoadFS(fsys fs.FS, glob string) error

// LoadDir 从操作系统目录加载翻译文件，等价于 LoadFS(os.DirFS(dir), "*.json")。
func (b *Bundle) LoadDir(dir string) error

// For 根据语言 tag 返回 Translator，tag 大小写不敏感。
// 找不到对应语言时降级到 fallback，仍找不到时返回 nopTranslator（T 返回 key 原文）。
func (b *Bundle) For(lang string) *Translator

// ForRequest 从 http.Request 检测语言后调用 For。
// Detector 返回空字符串时自动降级到 Bundle.fallback。
func (b *Bundle) ForRequest(r *http.Request) *Translator
```

### Locale

```go
// Locale 持有单一语言的所有翻译 key-value，创建后只读。
type Locale struct {
    tag      string           // 小写 BCP 47 language tag，如 "zh-cn"
    messages map[string]entry // key → entry
}

// entry 同时支持纯字符串和复数对象两种翻译形式。
// JSON 解析使用 json.RawMessage 做两步解析：
//   1. 尝试解析为 string → 存入 Other 字段
//   2. 失败则解析为复数对象
type entry struct {
    Zero  string // 数量为 0 时（可选）
    One   string // 数量为 1 时（单数）
    Other string // 其他数量 / 纯字符串形式
}
```

**翻译文件格式（`zh.json`）：**

```json
{
  "welcome":     "欢迎, {name}！",
  "items_count": {
    "zero":  "没有项目",
    "one":   "{count} 个项目",
    "other": "{count} 个项目"
  }
}
```

规则：
- value 为字符串时等价于只有 `other` 字段的复数对象
- `zero` / `one` 可省略，省略时降级到 `other`

### Translator

```go
// Translator 绑定到具体语言，提供翻译方法，并发安全（只读）。
// 通过 Bundle.For() 或 Bundle.ForRequest() 获取，不要自行构造。
type Translator struct {
    bundle *Bundle
    lang   string // 已解析的小写 language tag
}

// T 翻译 key，args 为交替的 string key-value 对。
//
//   t.T("welcome", "name", "Alice")     → "欢迎, Alice！"
//   t.T("welcome", "name", "Alice", "x") → 最后 "x" 无配对，忽略
//
// args 中的 key 必须为 string 类型；value 通过 fmt.Sprint 转为字符串。
// 找不到 key 时返回 key 原文，未提供的 {placeholder} 保留原样。
func (t *Translator) T(key string, args ...any) string

// N 带数量的复数翻译，count 用于选择 PluralForm，同时作为插值参数 {count}。
//
//   t.N("items_count", 3)  → "3 个项目"
//   t.N("items_count", 1)  → "1 个项目"
func (t *Translator) N(key string, count int, args ...any) string

// Lang 返回当前语言 tag（小写）。
func (t *Translator) Lang() string
```

---

## 语言检测（Detector）

```go
// Detector 从请求中提取首选语言 tag。
// 返回空字符串表示检测失败，Bundle.ForRequest 将降级到 fallback。
type Detector interface {
    Detect(r *http.Request) string
}
```

内置实现（按优先级顺序，ChainDetector 依次尝试，首个非空值胜出）：

| 实现 | 读取位置 | 说明 |
|------|----------|---------|
| `QueryDetector` | `?lang=zh` | 查询参数，key 可配置，默认 `lang` |
| `CookieDetector` | Cookie `lang` | Cookie 名可配置，默认 `lang` |
| `HeaderDetector` | `X-Lang: zh` | 自定义请求头，Header 名可配置 |
| `AcceptDetector` | `Accept-Language` | 标准 HTTP 协商，解析 q 权重，取最高分 |
| `ChainDetector` | — | 组合多个 Detector |

默认策略 `DefaultDetector()`：Query → Cookie → X-Lang → Accept-Language。

---

## 复数规则（PluralFunc）

```go
// PluralForm 枚举当前支持的复数形式。
// 当前仅支持 zero / one / other，满足英语、中文、日语等语言。
// 阿拉伯语等需要 two/few/many 形式的语言，需自定义 PluralFunc
// 并相应扩展翻译文件中的 entry 字段（框架层不感知额外字段）。
type PluralForm string

const (
    PluralZero  PluralForm = "zero"
    PluralOne   PluralForm = "one"
    PluralOther PluralForm = "other"
)

// PluralFunc 根据数量返回对应 PluralForm
type PluralFunc func(count int) PluralForm
```

内置规则：
- `SimplePluralFunc`：`count==0 → zero`，`count==1 → one`，其余 `other`
- 通过 `WithPluralRule(lang, fn)` 注册自定义规则，lang 大小写不敏感

---

## 参数插值

使用 `{key}` 占位符，运行时字符串替换，**不使用 `fmt.Sprintf`**（避免格式字符串注入风险）。

```
"欢迎回来, {name}！你有 {count} 条消息。"
    args: "name", "Alice", "count", 5
    → "欢迎回来, Alice！你有 5 条消息。"
```

规则：
- 未提供的 `{key}` 保留原样，不替换
- `args` 长度为奇数时最后一个参数忽略
- `args` 中 key 必须为 `string`；value 通过 `fmt.Sprint` 转为字符串
- `N()` 自动将 `count` 注入为插值参数 `{count}`，无需手动传递

---

## 翻译文件加载

### 文件命名约定

```
locales/
├── zh.json        # 中文（作为 zh-* 的回退）
├── zh-CN.json     # 中文（大陆）
├── en.json        # 英文
└── en-US.json     # 英文（美国）
```

- 语言 tag 从文件名去掉 `.json` 后缀得到
- **存取时统一转小写**：`zh-CN.json` → tag `zh-cn`，调用方无需关心大小写
- `For("zh-CN")` 与 `For("zh-cn")` 等价

### 加载 API

```go
// embed.FS（推荐：编译进二进制，无运行时文件依赖）
// glob 仅支持单层通配，不支持 ** 递归
//go:embed locales/*.json
var localeFS embed.FS

b := i18n.NewBundle(i18n.WithFallback("en"))
b.LoadFS(localeFS, "locales/*.json")

// os.DirFS（开发阶段，无需重新编译即可修改翻译文件）
b.LoadDir("./locales")
```

---

## 回退链

查找 key 时按以下顺序降级：

```
请求语言 (zh-cn)
  → 父语言 (zh)          ← strings.SplitN(lang, "-", 2)[0]，仅取第一段
    → Bundle.fallback
      → key 原文（最终兜底，永不 panic）
```

**父语言切割规则：** 仅取 BCP 47 tag 第一段（`-` 分隔），不做多层递归。
例：`zh-hant-tw → zh`，`en-us → en`。此策略简单够用，框架不内置完整 BCP 47 父链。

---

## 线程安全

| 阶段 | 安全性 | 说明 |
|------|--------|------|
| `LoadFS` / `LoadDir` | 非并发安全 | 仅在服务启动阶段调用，完成前不得并发读取 |
| `For` / `ForRequest` | 并发安全 | 加载完成后全程只读，无锁竞争 |
| `Translator.T` / `.N` | 并发安全 | 无状态，纯函数式 |

`Bundle` 内部持有 `sync.RWMutex`，为未来热重载预留接口，当前实现加载阶段加写锁，读取阶段加读锁。

---

## 与根包 qi 的集成

### 中间件

```go
// I18n 返回中间件，将 *i18n.Translator 注入 Context。
// 必须在需要翻译的路由之前注册，否则 c.T() 将静默返回 key 原文。
func I18n(b *i18n.Bundle) qi.HandlerFunc {
    return func(c *qi.Context) {
        t := b.ForRequest(c.Request())
        c.Set("_i18n", t)
        c.Next()
    }
}
```

### Context 扩展方法

```go
// T 从当前请求 Context 获取翻译。
// 若未注册 I18n 中间件，静默返回 key 原文（设计决策：降级而非 panic）。
func (c *Context) T(key string, args ...any) string {
    if v, ok := c.Get("_i18n"); ok {
        if t, ok := v.(*i18n.Translator); ok {
            return t.T(key, args...)
        }
    }
    return key // 未注册中间件时的兜底行为
}
```

### 完整使用示例

```go
package main

import (
    "embed"

    "github.com/tokmz/qi"
    "github.com/tokmz/qi/internal/i18n"
)

//go:embed locales/*.json
var localeFS embed.FS

func main() {
    bundle := i18n.NewBundle(
        i18n.WithFallback("en"),
    )
    if err := bundle.LoadFS(localeFS, "locales/*.json"); err != nil {
        panic(err)
    }

    app := qi.New(qi.WithAddr(":8080"))
    app.Use(qi.I18n(bundle))

    app.GET("/hello", func(c *qi.Context) {
        msg := c.T("welcome", "name", "Alice")
        c.OK(msg)
    })

    app.Run()
}
```

---

## BundleOption 列表

```go
WithFallback(lang string)                   // 设置回退语言，默认 "en"
WithDetector(d Detector)                    // 替换默认检测策略
WithPluralRule(lang string, fn PluralFunc)  // 注册复数规则，lang 大小写不敏感
```

---

## 设计约束

| 约束 | 说明 |
|------|------|
| 不对外导出 | 包路径 `internal/i18n`，外部模块无法 import |
| 仅标准库 | 不引入任何第三方依赖 |
| 启动时快速失败 | `LoadFS` / `LoadDir` 返回 `error`，调用方决定是否 `panic` |
| 无全局状态 | 所有状态在 `Bundle` 实例内，支持多 Bundle 并存 |
| 静默降级 | 找不到 key / 未注册中间件时返回 key 原文，不 panic，不 log |