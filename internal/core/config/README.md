## Config - 配置管理

基于 Viper 的配置管理包，支持多种配置格式、自动重载、类型安全和并发安全的配置读取。

## 功能特性

- ✅ **多格式支持**: JSON, YAML, TOML, INI, Properties 等格式
- ✅ **自动重载**: 监听配置文件变化并自动重新加载
- ✅ **变化回调**: 支持注册配置变化回调函数
- ✅ **类型安全**: 提供完整的类型安全读取方法
- ✅ **并发安全**: 所有读写操作都是线程安全的
- ✅ **环境变量**: 支持从环境变量读取配置
- ✅ **默认值**: 支持设置配置默认值
- ✅ **结构体映射**: 支持将配置解析到结构体
- ✅ **防抖机制**: 配置重载防抖，避免频繁触发
- ✅ **全局实例**: 支持单例模式的全局配置管理器

## 目录结构

```
config/
├── config.go          # 核心配置管理器
├── types.go           # 类型定义
├── errors.go          # 错误定义
├── logger.go          # 日志接口和实现
├── example_test.go    # 示例代码
└── README.md          # 文档（本文件）
```

## 快速开始

### 1. 基础使用

```go
package main

import (
    "log"
    "qi/internal/core/config"
)

func main() {
    // 创建配置选项
    opts := &config.Options{
        ConfigFile: "./config.yaml",
        ConfigType: "yaml",
    }

    // 创建配置管理器
    logger := &config.DefaultLogger{}
    mgr, err := config.New(opts, logger)
    if err != nil {
        log.Fatal(err)
    }

    // 读取配置
    appName := mgr.GetString("app.name")
    port := mgr.GetInt("server.port")
    debug := mgr.GetBool("app.debug")

    log.Printf("App: %s, Port: %d, Debug: %v", appName, port, debug)
}
```

### 2. 配置文件示例

#### YAML 格式 (config.yaml)
```yaml
app:
  name: "My Application"
  version: "1.0.0"
  debug: true
  tags:
    - "api"
    - "backend"

server:
  host: "0.0.0.0"
  port: 8080
  timeout: 30s
  
database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  username: "root"
  password: "password"
  dbname: "mydb"
```

#### JSON 格式 (config.json)
```json
{
  "app": {
    "name": "My Application",
    "version": "1.0.0",
    "debug": true
  },
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  }
}
```

#### TOML 格式 (config.toml)
```toml
[app]
name = "My Application"
version = "1.0.0"
debug = true

[server]
host = "0.0.0.0"
port = 8080
```

### 3. 自动重载配置

```go
opts := &config.Options{
    ConfigFile:     "./config.yaml",
    ConfigType:     "yaml",
    AutoReload:     true,                    // 启用自动重载
    ReloadDebounce: 500 * time.Millisecond, // 防抖时间
}

mgr, _ := config.New(opts, &config.DefaultLogger{})

// 注册配置变化回调
mgr.OnChange(func(event *config.ChangeEvent) {
    log.Printf("Config changed at %v", event.Time)
    
    // 重新读取配置
    newPort := mgr.GetInt("server.port")
    log.Printf("New port: %d", newPort)
    
    // 执行其他操作，如重启服务等
    restartServer(newPort)
})
```

### 4. 环境变量支持

```go
opts := &config.Options{
    ConfigName:    "config",
    ConfigType:    "yaml",
    ConfigPaths:   []string{".", "./configs"},
    AutomaticEnv:  true,  // 启用自动读取环境变量
    EnvPrefix:     "APP", // 环境变量前缀
    AllowEmptyEnv: false,
}

mgr, _ := config.New(opts, nil)

// 读取配置
// 会先从配置文件读取 server.port
// 如果环境变量 APP_SERVER_PORT 存在，则使用环境变量的值
port := mgr.GetInt("server.port")
```

### 5. 解析到结构体

```go
// 定义配置结构体
type Config struct {
    App struct {
        Name    string   `mapstructure:"name"`
        Version string   `mapstructure:"version"`
        Debug   bool     `mapstructure:"debug"`
        Tags    []string `mapstructure:"tags"`
    } `mapstructure:"app"`
    
    Server struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"server"`
    
    Database struct {
        Driver   string `mapstructure:"driver"`
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        Username string `mapstructure:"username"`
        Password string `mapstructure:"password"`
        DBName   string `mapstructure:"dbname"`
    } `mapstructure:"database"`
}

// 解析配置
var cfg Config
if err := mgr.Unmarshal(&cfg); err != nil {
    log.Fatal(err)
}

// 使用配置
log.Printf("App: %s v%s", cfg.App.Name, cfg.App.Version)
log.Printf("Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
```

### 6. 全局配置管理器

```go
// 在 main.go 中初始化
func init() {
    opts := config.DefaultOptions()
    opts.ConfigFile = "./config.yaml"
    
    if err := config.InitGlobal(opts, &config.DefaultLogger{}); err != nil {
        log.Fatal(err)
    }
}

// 在任何地方使用
func someFunction() {
    mgr := config.GetGlobal()
    port := mgr.GetInt("server.port")
    // ...
}
```

## 配置选项

### Options 结构体

```go
type Options struct {
    // ConfigFile 配置文件路径（优先级最高）
    ConfigFile string

    // ConfigName 配置文件名（不包含扩展名）
    ConfigName string

    // ConfigType 配置文件类型
    ConfigType string

    // ConfigPaths 配置文件搜索路径列表
    ConfigPaths []string

    // AutoReload 是否自动监听配置文件变化
    AutoReload bool

    // EnvPrefix 环境变量前缀
    EnvPrefix string

    // AutomaticEnv 是否自动读取环境变量
    AutomaticEnv bool

    // AllowEmptyEnv 是否允许空环境变量
    AllowEmptyEnv bool

    // ReloadDebounce 配置重载防抖时间
    ReloadDebounce time.Duration
}
```

### 默认配置

```go
opts := config.DefaultOptions()
// ConfigName:     "config"
// ConfigType:     "yaml"
// ConfigPaths:    []string{".", "./configs", "/etc/app"}
// AutoReload:     false
// ReloadDebounce: 500ms
```

## API 文档

### 配置管理器

```go
// 创建配置管理器
New(opts *Options, logger Logger) (*Manager, error)

// 初始化全局配置管理器
InitGlobal(opts *Options, logger Logger) error

// 获取全局配置管理器
GetGlobal() *Manager

// 手动重新加载配置
Reload() error

// 获取当前使用的配置文件路径
GetConfigFile() string
```

### 监听配置变化

```go
// 注册配置变化回调
OnChange(callback OnChangeCallback)

// 开始监听配置文件变化
StartWatching() error

// 停止监听配置文件变化
StopWatching() error

// 检查是否正在监听
IsWatching() bool
```

### 类型安全的读取方法

```go
// 基础类型
Get(key string) interface{}
GetString(key string) string
GetInt(key string) int
GetInt32(key string) int32
GetInt64(key string) int64
GetUint(key string) uint
GetUint32(key string) uint32
GetUint64(key string) uint64
GetFloat64(key string) float64
GetBool(key string) bool

// 时间相关
GetTime(key string) time.Time
GetDuration(key string) time.Duration

// 切片和映射
GetStringSlice(key string) []string
GetIntSlice(key string) []int
GetStringMap(key string) map[string]interface{}
GetStringMapString(key string) map[string]string
GetStringMapStringSlice(key string) map[string][]string

// 字节大小
GetSizeInBytes(key string) uint
```

### 设置配置

```go
// 设置运行时配置值（不会写入文件）
Set(key string, value interface{})

// 设置默认配置值
SetDefault(key string, value interface{})
```

### 结构体解析

```go
// 解析整个配置到结构体
Unmarshal(rawVal interface{}) error

// 解析指定键的配置到结构体
UnmarshalKey(key string, rawVal interface{}) error
```

### 配置检查

```go
// 检查配置键是否存在
IsSet(key string) bool

// 获取所有配置键
AllKeys() []string

// 获取所有配置
AllSettings() map[string]interface{}
```

## 使用场景

### 1. Web 应用配置

```go
type WebConfig struct {
    Server struct {
        Host         string        `mapstructure:"host"`
        Port         int           `mapstructure:"port"`
        ReadTimeout  time.Duration `mapstructure:"read_timeout"`
        WriteTimeout time.Duration `mapstructure:"write_timeout"`
    } `mapstructure:"server"`
    
    Database struct {
        DSN         string `mapstructure:"dsn"`
        MaxIdle     int    `mapstructure:"max_idle"`
        MaxOpen     int    `mapstructure:"max_open"`
        MaxLifetime string `mapstructure:"max_lifetime"`
    } `mapstructure:"database"`
}

// 加载配置
mgr, _ := config.New(opts, logger)
var cfg WebConfig
mgr.Unmarshal(&cfg)

// 使用配置启动服务器
server := &http.Server{
    Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
    ReadTimeout:  cfg.Server.ReadTimeout,
    WriteTimeout: cfg.Server.WriteTimeout,
}
```

### 2. 多环境配置

```go
// 开发环境
opts := &config.Options{
    ConfigFile: "./configs/config.dev.yaml",
}

// 生产环境
if os.Getenv("ENV") == "production" {
    opts.ConfigFile = "./configs/config.prod.yaml"
}

mgr, _ := config.New(opts, logger)
```

### 3. 配置热更新

```go
opts := &config.Options{
    ConfigFile: "./config.yaml",
    AutoReload: true,
}

mgr, _ := config.New(opts, logger)

// 监听日志级别变化
mgr.OnChange(func(event *config.ChangeEvent) {
    newLevel := mgr.GetString("log.level")
    updateLogLevel(newLevel)
})

// 监听数据库配置变化
mgr.OnChange(func(event *config.ChangeEvent) {
    if mgr.IsSet("database.max_open") {
        maxOpen := mgr.GetInt("database.max_open")
        db.SetMaxOpenConns(maxOpen)
    }
})
```

### 4. 服务发现配置

```go
type ServiceConfig struct {
    Name    string            `mapstructure:"name"`
    Version string            `mapstructure:"version"`
    Tags    []string          `mapstructure:"tags"`
    Meta    map[string]string `mapstructure:"meta"`
}

mgr, _ := config.New(opts, logger)

var svcCfg ServiceConfig
mgr.UnmarshalKey("service", &svcCfg)

// 注册服务
registerService(svcCfg)
```

## 最佳实践

### 1. 使用结构体管理配置

```go
// 推荐：定义配置结构体
type AppConfig struct {
    App      AppSettings      `mapstructure:"app"`
    Server   ServerSettings   `mapstructure:"server"`
    Database DatabaseSettings `mapstructure:"database"`
}

var cfg AppConfig
mgr.Unmarshal(&cfg)

// 不推荐：直接使用字符串键
port := mgr.GetInt("server.port") // 容易出错
```

### 2. 设置合理的默认值

```go
mgr.SetDefault("server.port", 8080)
mgr.SetDefault("server.host", "0.0.0.0")
mgr.SetDefault("app.debug", false)
mgr.SetDefault("log.level", "info")
```

### 3. 验证配置

```go
type Config struct {
    Server struct {
        Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
        Host string `mapstructure:"host" validate:"required"`
    } `mapstructure:"server"`
}

var cfg Config
if err := mgr.Unmarshal(&cfg); err != nil {
    log.Fatal(err)
}

// 使用 validator 验证
validate := validator.New()
if err := validate.Struct(cfg); err != nil {
    log.Fatal(err)
}
```

### 4. 合理使用回调

```go
// 好的做法：轻量级操作
mgr.OnChange(func(event *config.ChangeEvent) {
    log.Println("Config changed, reloading...")
    reloadLightweightSettings()
})

// 避免：在回调中执行耗时操作
mgr.OnChange(func(event *config.ChangeEvent) {
    // ❌ 不要在回调中执行重启服务等重量级操作
    // restartEntireApplication()
    
    // ✅ 应该通过通道通知主流程处理
    configChangeChan <- event
})
```

### 5. 并发安全

```go
// 配置管理器是并发安全的，可以在多个 goroutine 中使用
for i := 0; i < 10; i++ {
    go func() {
        port := mgr.GetInt("server.port")
        // 使用 port
    }()
}
```

## 支持的配置格式

### YAML
```yaml
app:
  name: "My App"
```

### JSON
```json
{
  "app": {
    "name": "My App"
  }
}
```

### TOML
```toml
[app]
name = "My App"
```

### INI
```ini
[app]
name = My App
```

### Properties
```properties
app.name=My App
```

## 环境变量

### 自动绑定

```go
opts := &config.Options{
    AutomaticEnv: true,
    EnvPrefix:    "MYAPP",
}
```

配置键 `server.port` 会自动绑定到环境变量 `MYAPP_SERVER_PORT`

### 优先级

1. 运行时设置 (Set)
2. 环境变量
3. 配置文件
4. 默认值 (SetDefault)

## 故障排查

### 1. 配置文件未找到

```go
// 检查配置文件路径
log.Println("Config file:", mgr.GetConfigFile())

// 添加多个搜索路径
opts.ConfigPaths = []string{".", "./configs", "/etc/app", "/app/configs"}
```

### 2. 配置未生效

```go
// 检查配置键是否存在
if !mgr.IsSet("server.port") {
    log.Println("server.port not found in config")
}

// 查看所有配置键
log.Println("All keys:", mgr.AllKeys())

// 查看所有配置
log.Println("All settings:", mgr.AllSettings())
```

### 3. 自动重载不工作

```go
// 检查监听状态
if !mgr.IsWatching() {
    log.Println("Config watching not started")
    mgr.StartWatching()
}

// 检查配置文件权限
// 确保应用有读取配置文件的权限
```

## 性能优化

### 1. 避免频繁读取

```go
// ❌ 不好：每次都读取
func handler(w http.ResponseWriter, r *http.Request) {
    timeout := mgr.GetDuration("server.timeout")
    // ...
}

// ✅ 好：读取一次，缓存使用
var serverTimeout time.Duration

func init() {
    serverTimeout = mgr.GetDuration("server.timeout")
    
    // 配置变化时更新
    mgr.OnChange(func(event *config.ChangeEvent) {
        serverTimeout = mgr.GetDuration("server.timeout")
    })
}

func handler(w http.ResponseWriter, r *http.Request) {
    // 使用缓存的值
    // ...
}
```

### 2. 使用结构体

```go
// 解析一次，多次使用
var cfg AppConfig
mgr.Unmarshal(&cfg)

// 更新时重新解析
mgr.OnChange(func(event *config.ChangeEvent) {
    mgr.Unmarshal(&cfg)
})
```

## 许可证

MIT License

