package config_test

import (
	"fmt"
	"log"
	"time"

	"qi/internal/core/config"
)

// Example_basic 基础使用示例
func Example_basic() {
	// 1. 创建配置选项
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
		ConfigType: "yaml",
	}

	// 2. 创建配置管理器
	logger := &config.DefaultLogger{}
	mgr, err := config.New(opts, logger)
	if err != nil {
		log.Fatal(err)
	}

	// 3. 读取配置
	appName := mgr.GetString("app.name")
	port := mgr.GetInt("server.port")
	debug := mgr.GetBool("app.debug")

	fmt.Printf("App: %s, Port: %d, Debug: %v\n", appName, port, debug)
}

// Example_withEnvironment 环境变量示例
func Example_withEnvironment() {
	opts := &config.Options{
		ConfigName:    "config",
		ConfigType:    "yaml",
		ConfigPaths:   []string{".", "./configs"},
		AutomaticEnv:  true,
		EnvPrefix:     "APP",
		AllowEmptyEnv: false,
	}

	mgr, _ := config.New(opts, nil) // 不使用日志

	// 读取配置（会自动从环境变量 APP_SERVER_PORT 读取）
	port := mgr.GetInt("server.port")
	fmt.Printf("Port: %d\n", port)
}

// Example_autoReload 自动重载示例
func Example_autoReload() {
	opts := &config.Options{
		ConfigFile:     "./testdata/config.yaml",
		ConfigType:     "yaml",
		AutoReload:     true,
		ReloadDebounce: 500 * time.Millisecond,
	}

	logger := &config.DefaultLogger{}
	mgr, _ := config.New(opts, logger)

	// 注册配置变化回调
	mgr.OnChange(func(event *config.ChangeEvent) {
		fmt.Printf("Config changed: %s at %v\n", event.Name, event.Time)

		// 重新读取配置
		appName := mgr.GetString("app.name")
		fmt.Printf("New app name: %s\n", appName)
	})

	// 等待配置变化...
	time.Sleep(10 * time.Second)
}

// Example_unmarshal 解析到结构体示例
func Example_unmarshal() {
	// 定义配置结构体
	type ServerConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	type AppConfig struct {
		Name   string       `mapstructure:"name"`
		Debug  bool         `mapstructure:"debug"`
		Server ServerConfig `mapstructure:"server"`
	}

	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 解析整个配置
	var appCfg AppConfig
	if err := mgr.Unmarshal(&appCfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("App: %s, Server: %s:%d\n",
		appCfg.Name, appCfg.Server.Host, appCfg.Server.Port)
}

// Example_unmarshalKey 解析指定键示例
func Example_unmarshalKey() {
	type ServerConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 只解析 server 部分
	var serverCfg ServerConfig
	if err := mgr.UnmarshalKey("server", &serverCfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server: %s:%d\n", serverCfg.Host, serverCfg.Port)
}

// Example_globalManager 全局配置管理器示例
func Example_globalManager() {
	// 初始化全局配置管理器
	opts := config.DefaultOptions()
	opts.ConfigFile = "./testdata/config.yaml"

	if err := config.InitGlobal(opts, &config.DefaultLogger{}); err != nil {
		log.Fatal(err)
	}

	// 在任何地方获取全局配置管理器
	mgr := config.GetGlobal()

	// 读取配置
	appName := mgr.GetString("app.name")
	fmt.Printf("App: %s\n", appName)
}

// Example_multipleFormats 多种配置格式示例
func Example_multipleFormats() {
	// YAML 格式
	yamlOpts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
		ConfigType: "yaml",
	}
	yamlMgr, _ := config.New(yamlOpts, nil)
	fmt.Println("YAML:", yamlMgr.GetString("app.name"))

	// JSON 格式
	jsonOpts := &config.Options{
		ConfigFile: "./testdata/config.json",
		ConfigType: "json",
	}
	jsonMgr, _ := config.New(jsonOpts, nil)
	fmt.Println("JSON:", jsonMgr.GetString("app.name"))

	// TOML 格式
	tomlOpts := &config.Options{
		ConfigFile: "./testdata/config.toml",
		ConfigType: "toml",
	}
	tomlMgr, _ := config.New(tomlOpts, nil)
	fmt.Println("TOML:", tomlMgr.GetString("app.name"))
}

// Example_setDefaults 设置默认值示例
func Example_setDefaults() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 设置默认值
	mgr.SetDefault("server.timeout", 30)
	mgr.SetDefault("server.maxConnections", 1000)

	// 读取配置（如果配置文件中没有，使用默认值）
	timeout := mgr.GetInt("server.timeout")
	maxConn := mgr.GetInt("server.maxConnections")

	fmt.Printf("Timeout: %d, MaxConn: %d\n", timeout, maxConn)
}

// Example_runtimeSet 运行时设置配置示例
func Example_runtimeSet() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 运行时设置配置（不会写入文件）
	mgr.Set("app.name", "NewAppName")
	mgr.Set("server.port", 9090)

	// 读取配置
	appName := mgr.GetString("app.name")
	port := mgr.GetInt("server.port")

	fmt.Printf("App: %s, Port: %d\n", appName, port)
}

// Example_checkKey 检查配置键示例
func Example_checkKey() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 检查配置键是否存在
	if mgr.IsSet("app.name") {
		fmt.Println("app.name exists")
	}

	if !mgr.IsSet("app.nonexistent") {
		fmt.Println("app.nonexistent does not exist")
	}

	// 获取所有配置键
	allKeys := mgr.AllKeys()
	fmt.Printf("Total keys: %d\n", len(allKeys))
}

// Example_typeConversion 类型转换示例
func Example_typeConversion() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 字符串
	name := mgr.GetString("app.name")
	fmt.Printf("String: %s\n", name)

	// 整数
	port := mgr.GetInt("server.port")
	fmt.Printf("Int: %d\n", port)

	// 布尔值
	debug := mgr.GetBool("app.debug")
	fmt.Printf("Bool: %v\n", debug)

	// 时间间隔
	timeout := mgr.GetDuration("server.timeout")
	fmt.Printf("Duration: %v\n", timeout)

	// 字符串切片
	tags := mgr.GetStringSlice("app.tags")
	fmt.Printf("StringSlice: %v\n", tags)

	// 映射
	settings := mgr.GetStringMap("database")
	fmt.Printf("StringMap: %v\n", settings)
}

// Example_concurrentAccess 并发访问示例
func Example_concurrentAccess() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
	}

	mgr, _ := config.New(opts, nil)

	// 并发读取配置（线程安全）
	for i := 0; i < 10; i++ {
		go func(n int) {
			appName := mgr.GetString("app.name")
			port := mgr.GetInt("server.port")
			fmt.Printf("Goroutine %d: %s:%d\n", n, appName, port)
		}(i)
	}

	time.Sleep(1 * time.Second)
}

// Example_multipleCallbacks 多个回调示例
func Example_multipleCallbacks() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
		AutoReload: true,
	}

	mgr, _ := config.New(opts, &config.DefaultLogger{})

	// 注册多个回调
	mgr.OnChange(func(event *config.ChangeEvent) {
		fmt.Println("Callback 1: Config changed")
	})

	mgr.OnChange(func(event *config.ChangeEvent) {
		fmt.Println("Callback 2: Reloading application settings")
	})

	mgr.OnChange(func(event *config.ChangeEvent) {
		fmt.Println("Callback 3: Notifying services")
	})

	time.Sleep(10 * time.Second)
}

// Example_manualReload 手动重载示例
func Example_manualReload() {
	opts := &config.Options{
		ConfigFile: "./testdata/config.yaml",
		AutoReload: false, // 不自动重载
	}

	mgr, _ := config.New(opts, &config.DefaultLogger{})

	// 读取初始配置
	initialName := mgr.GetString("app.name")
	fmt.Printf("Initial name: %s\n", initialName)

	// ... 等待一段时间，配置文件被修改 ...
	time.Sleep(5 * time.Second)

	// 手动重载配置
	if err := mgr.Reload(); err != nil {
		log.Printf("Reload failed: %v", err)
	}

	// 读取新配置
	newName := mgr.GetString("app.name")
	fmt.Printf("New name: %s\n", newName)
}
