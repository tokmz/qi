# Qi i18n 国际化包

为 Qi 框架提供简洁、开箱即用的国际化支持。

## 安装

```go
import "qi/pkg/i18n"
```

## 快速开始

### 1. 创建翻译文件

```
locales/
├── zh-CN.json
└── en-US.json
```

```json
// locales/zh-CN.json
{
    "hello": "你好 {{.Name}}",
    "user": {
        "login": "登录",
        "logout": "退出登录"
    }
}
```

```json
// locales/en-US.json
{
    "hello": "Hello {{.Name}}",
    "user": {
        "login": "Login",
        "logout": "Logout"
    }
}
```

### 2. 创建翻译器

```go
trans, err := i18n.NewWithOptions(
    i18n.WithDir("./locales"),
    i18n.WithDefaultLanguage("zh-CN"),
    i18n.WithLanguages("zh-CN", "en-US"),
)
if err != nil {
    panic(err)
}
```

### 3. 使用翻译

```go
ctx := context.Background()

// 默认语言翻译
trans.T(ctx, "hello", "Name", "Alice") // "你好 Alice"

// 切换语言
ctx = i18n.WithLanguage(ctx, "en-US")
trans.T(ctx, "hello", "Name", "Alice") // "Hello Alice"

// 嵌套键访问
trans.T(ctx, "user.login") // "Login"
```

## API

### 创建翻译器

```go
// 使用 Options 模式
trans, err := i18n.NewWithOptions(opts ...Option)

// 使用 Config
trans, err := i18n.New(cfg *Config)

// Must 版本（失败时 panic）
trans := i18n.MustNew(cfg *Config)
trans := i18n.MustNewWithOptions(opts ...Option)
```

### Translator 接口

```go
type Translator interface {
    // T 翻译文本，支持变量替换
    // 变量格式：{{.Name}}，参数为 key-value 对
    T(ctx context.Context, key string, args ...any) string

    // Tn 复数形式翻译
    // n == 1 使用 key，n != 1 使用 plural
    Tn(ctx context.Context, key, plural string, n int, args ...any) string

    // GetLanguage 获取当前语言
    GetLanguage(ctx context.Context) string

    // AvailableLanguages 获取支持的语言列表
    AvailableLanguages() []string

    // HasKey 检查翻译键是否存在
    HasKey(key string) bool

    // Preload 预加载指定语言
    Preload(languages ...string) error
}
```

### 配置选项

```go
i18n.WithDefaultLanguage("zh-CN")     // 默认语言
i18n.WithLanguages("zh-CN", "en-US")  // 支持的语言列表
i18n.WithDir("./locales")             // 翻译文件目录
i18n.WithPattern("{lang}.json")       // 文件名模式
i18n.WithLazy(true)                   // 懒加载（默认开启）
i18n.WithLoader(loader)               // 自定义加载器
i18n.WithVarDelimiters("{{", "}}")    // 变量分隔符
```

### Helper 函数

```go
// 设置语言到 Context
ctx = i18n.WithLanguage(ctx, "en-US")

// 从 Context 获取语言
lang := i18n.GetLanguageFromContext(ctx)
```

## 功能说明

### 变量替换

翻译模板中使用 `{{.Name}}` 格式的占位符，调用时传入 key-value 对：

```go
// 模板: "欢迎 {{.Name}}，你有 {{.Count}} 条消息"
trans.T(ctx, "welcome", "Name", "Alice", "Count", 5)
// 输出: "欢迎 Alice，你有 5 条消息"
```

### 复数形式

```json
{
    "item_one": "{{.Count}} item",
    "item_other": "{{.Count}} items"
}
```

```go
trans.Tn(ctx, "item_one", "item_other", 1)  // "1 item"
trans.Tn(ctx, "item_one", "item_other", 5)  // "5 items"
```

`Tn` 会自动将 `n` 作为 `{{.Count}}` 变量注入。

### 嵌套 JSON

支持嵌套结构，使用点号访问：

```json
{
    "user": {
        "profile": {
            "title": "个人资料"
        }
    }
}
```

```go
trans.T(ctx, "user.profile.title") // "个人资料"
```

### 懒加载

默认开启懒加载，语言文件在首次访问时自动加载。也可以关闭懒加载或手动预加载：

```go
// 关闭懒加载，启动时加载所有语言
i18n.WithLazy(false)

// 手动预加载指定语言
trans.Preload("zh-CN", "en-US")
```

### 语言回退

当请求的语言中找不到翻译键时，自动回退到默认语言。如果默认语言也找不到，返回 key 本身。

### 自定义加载器

实现 `Loader` 接口可以从任意数据源加载翻译：

```go
type Loader interface {
    Load(ctx context.Context, dir string, languages []string) (map[string]map[string]string, error)
}
```

## 与 Qi 框架集成

```go
func main() {
    trans, err := i18n.NewWithOptions(
        i18n.WithDir("./locales"),
        i18n.WithDefaultLanguage("zh-CN"),
        i18n.WithLanguages("zh-CN", "en-US"),
    )
    if err != nil {
        panic(err)
    }

    engine := qi.Default()
    r := engine.RouterGroup()

    r.GET("/hello", func(c *qi.Context) {
        msg := trans.T(c.RequestContext(), "hello", "Name", "Alice")
        c.Success(msg)
    })

    // 泛型路由
    qi.Handle[HelloReq, HelloResp](r.POST, "/hello", func(c *qi.Context, req *HelloReq) (*HelloResp, error) {
        msg := trans.T(c.RequestContext(), "hello", "Name", req.Name)
        return &HelloResp{Message: msg}, nil
    })

    engine.Run(":8080")
}
```

## 资源文件规范

- 文件格式：JSON
- 命名规范：BCP 47 语言标签，如 `zh-CN.json`、`en-US.json`
- 文件名模式可通过 `WithPattern` 自定义，默认 `{lang}.json`
- 支持嵌套结构，内部自动扁平化为点号分隔的 key

## 文件结构

```
pkg/i18n/
├── config.go       # 配置和 Options
├── errors.go       # 错误定义
├── helper.go       # Context 辅助函数
├── loader.go       # Loader 接口和 JSONLoader
├── translator.go   # Translator 接口和实现
└── i18n_test.go    # 单元测试
```
