# Config Package

基于 [Viper](https://github.com/spf13/viper) 的配置管理包，提供配置文件加载、热监控、保护模式自动恢复等功能。

## 功能特性

- 支持 YAML/JSON/TOML 等多种配置格式
- Options 模式配置，API 简洁
- 配置文件热监控（基于 fsnotify）
- 保护模式：检测到文件被篡改后自动恢复原始内容
- 非保护模式：文件变更后自动重载并触发回调
- 泛型 `Get[T]` 类型安全读取
- 环境变量绑定与覆盖
- 结构体反序列化
- 全局默认实例
- 并发安全

## 快速开始

```go
import "github.com/tokmz/qi/pkg/config"

cfg := config.New(
    config.WithConfigFile("config.yaml"),
)
if err := cfg.Load(); err != nil {
    log.Fatal(err)
}
defer cfg.Close()

name := cfg.GetString("app.name")
port := cfg.GetInt("server.port")
```

## 配置选项

```go
cfg := config.New(
    // 配置文件
    config.WithConfigFile("config.yaml"),          // 指定文件路径
    config.WithConfigName("config"),               // 文件名（不含扩展名）
    config.WithConfigType("yaml"),                 // 文件类型
    config.WithConfigPaths(".", "/etc/myapp"),      // 搜索路径

    // 默认值
    config.WithDefaults(map[string]any{
        "server.port": 8080,
        "app.debug":   false,
    }),

    // 环境变量
    config.WithEnvPrefix("MYAPP"),                                // 前缀
    config.WithEnvKeyReplacer(strings.NewReplacer(".", "_")),     // 键名替换

    // 监控
    config.WithAutoWatch(true),                    // 自动开启文件监控
    config.WithProtected(true),                    // 保护模式
    config.WithOnChange(func() {                   // 变更回调（非保护模式）
        log.Println("配置已更新")
    }),
    config.WithOnError(func(err error) {           // 错误回调
        log.Printf("配置错误: %v", err)
    }),
)
```

## 读取配置

```go
// 基础类型
cfg.GetString("app.name")
cfg.GetInt("server.port")
cfg.GetInt64("app.id")
cfg.GetFloat64("metrics.rate")
cfg.GetBool("app.debug")
cfg.GetDuration("server.timeout")

// 复合类型
cfg.GetStringSlice("app.tags")
cfg.GetStringMap("database")
cfg.GetStringMapString("labels")

// 泛型读取
name := config.Get[string](cfg, "app.name")
debug := config.Get[bool](cfg, "app.debug")

// 检查键是否存在
cfg.IsSet("app.name")

// 获取所有配置
cfg.AllSettings()
```

## 结构体反序列化

```go
type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Name     string `mapstructure:"name"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
}

// 反序列化指定 key
var dbCfg DatabaseConfig
cfg.UnmarshalKey("database", &dbCfg)

// 反序列化全部配置
var appCfg AppConfig
cfg.Unmarshal(&appCfg)
```

## 子配置

```go
// 获取 server 下的所有配置
serverCfg := cfg.Sub("server")
if serverCfg != nil {
    addr := serverCfg.GetString("addr")
    timeout := serverCfg.GetDuration("timeout")
}
```

注意：`Sub()` 返回的是只读轻量实例，不继承监控、保护模式等属性。

## 配置文件监控

### 非保护模式

文件变更后自动重载配置，触发 `OnChange` 回调：

```go
cfg := config.New(
    config.WithConfigFile("config.yaml"),
    config.WithAutoWatch(true),
    config.WithOnChange(func() {
        log.Println("配置已重载")
        // 重新读取需要的配置值
    }),
)
```

### 保护模式

文件被外部修改后，自动恢复为加载时的原始内容：

```go
cfg := config.New(
    config.WithConfigFile("config.yaml"),
    config.WithProtected(true),
    config.WithAutoWatch(true),
    config.WithOnError(func(err error) {
        log.Printf("恢复失败: %v", err)
    }),
)
```

恢复机制使用临时文件 + 原子替换，确保文件完整性。

### 动态切换

```go
cfg.SetProtected(true)   // 开启保护
cfg.SetProtected(false)  // 关闭保护
cfg.IsProtected()        // 查询状态

cfg.StartWatch()         // 手动开启监控
cfg.StopWatch()          // 停止监控
```

## 环境变量

环境变量会覆盖配置文件中的值：

```go
cfg := config.New(
    config.WithConfigFile("config.yaml"),
    config.WithEnvPrefix("MYAPP"),
    config.WithEnvKeyReplacer(strings.NewReplacer(".", "_")),
)
cfg.Load()

// 配置文件: app.name = "myapp"
// 环境变量: MYAPP_APP_NAME = "from-env"
// 结果: cfg.GetString("app.name") == "from-env"
```

## 全局默认实例

```go
// 设置全局实例
config.SetDefault(cfg)

// 获取全局实例（未设置时自动创建空实例）
globalCfg := config.Default()
```

## 错误定义

| 错误 | 错误码 | HTTP 状态码 | 说明 |
|------|--------|-------------|------|
| `ErrConfigNotFound` | 3001 | 500 | 配置文件未找到 |
| `ErrConfigReadFailed` | 3003 | 500 | 配置读取失败 |

## API 参考

### 构造函数

- `New(opts ...Option) *Config` — 创建配置管理器
- `Default() *Config` — 获取全局默认实例
- `SetDefault(c *Config)` — 设置全局默认实例

### 读取方法

- `GetString(key) string`
- `GetInt(key) int` / `GetInt64(key) int64` / `GetFloat64(key) float64`
- `GetBool(key) bool`
- `GetDuration(key) time.Duration`
- `GetStringSlice(key) []string`
- `GetStringMap(key) map[string]any`
- `GetStringMapString(key) map[string]string`
- `Get[T any](c *Config, key string) T` — 泛型读取
- `IsSet(key) bool`
- `AllSettings() map[string]any`

### 写入方法

- `Set(key string, value any)`

### 反序列化

- `Unmarshal(rawVal any) error`
- `UnmarshalKey(key string, rawVal any) error`
- `Sub(key string) *Config`

### 生命周期

- `Load() error` — 加载配置文件
- `Close()` — 关闭管理器

### 监控与保护

- `StartWatch() error` / `StopWatch()`
- `SetProtected(bool)` / `IsProtected() bool`

### 高级

- `Viper() *viper.Viper` — 获取底层 viper 实例（绕过并发锁，需自行保证线程安全）
