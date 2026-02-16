package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tokmz/qi/pkg/config"
)

// DatabaseConfig 数据库配置结构体
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

func main() {
	// ============ 1. 基本配置加载 ============
	fmt.Println("=== 基本配置加载 ===")

	cfg := config.New(
		config.WithConfigFile("example/config/config.yaml"),
		config.WithDefaults(map[string]any{
			"app.name":    "default-app",
			"server.addr": ":3000",
		}),
	)

	if err := cfg.Load(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	defer cfg.Close()

	// ============ 2. 各种类型的配置读取 ============
	fmt.Println("\n=== 配置读取 ===")

	fmt.Printf("app.name:       %s\n", cfg.GetString("app.name"))
	fmt.Printf("app.debug:      %v\n", cfg.GetBool("app.debug"))
	fmt.Printf("database.port:  %d\n", cfg.GetInt("database.port"))
	fmt.Printf("log.output:     %v\n", cfg.GetStringSlice("log.output"))
	fmt.Printf("features:       %v\n", cfg.GetStringSlice("features"))

	// 泛型 Get
	name := config.Get[string](cfg, "app.name")
	fmt.Printf("泛型 Get:       %s\n", name)

	// 检查 key 是否存在
	fmt.Printf("app.name 存在:  %v\n", cfg.IsSet("app.name"))
	fmt.Printf("app.foo 存在:   %v\n", cfg.IsSet("app.foo"))

	// ============ 3. 结构体反序列化 ============
	fmt.Println("\n=== 结构体反序列化 ===")

	var dbCfg DatabaseConfig
	if err := cfg.UnmarshalKey("database", &dbCfg); err != nil {
		log.Fatalf("反序列化失败: %v", err)
	}
	fmt.Printf("数据库: %s:%d/%s (user=%s, maxOpen=%d)\n",
		dbCfg.Host, dbCfg.Port, dbCfg.Name, dbCfg.User, dbCfg.MaxOpenConns)

	// ============ 4. 子配置 ============
	fmt.Println("\n=== 子配置 ===")

	serverCfg := cfg.Sub("server")
	if serverCfg != nil {
		fmt.Printf("server.addr:          %s\n", serverCfg.GetString("addr"))
		fmt.Printf("server.read_timeout:  %s\n", serverCfg.GetDuration("read_timeout"))
	}

	// ============ 5. 环境变量绑定 ============
	fmt.Println("\n=== 环境变量绑定 ===")

	envCfg := config.New(
		config.WithConfigFile("example/config/config.yaml"),
		config.WithEnvPrefix("QI"),
		config.WithEnvKeyReplacer(strings.NewReplacer(".", "_")),
	)
	if err := envCfg.Load(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置环境变量后，环境变量值会覆盖配置文件
	os.Setenv("QI_APP_NAME", "qi-from-env")
	fmt.Printf("app.name (env): %s\n", envCfg.GetString("app.name"))
	os.Unsetenv("QI_APP_NAME")

	// ============ 6. 配置文件监控（非保护模式） ============
	fmt.Println("\n=== 配置文件监控（非保护模式） ===")

	watchCfg := config.New(
		config.WithConfigFile("example/config/config.yaml"),
		config.WithAutoWatch(true),
		config.WithOnChange(func() {
			fmt.Println("[回调] 检测到配置文件变更，已自动重新加载")
		}),
		config.WithOnError(func(err error) {
			fmt.Printf("[错误] %v\n", err)
		}),
	)
	if err := watchCfg.Load(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	defer watchCfg.Close()

	fmt.Println("非保护模式已启动，修改 config.yaml 将触发回调")

	// ============ 7. 配置文件保护模式 ============
	fmt.Println("\n=== 配置文件保护模式 ===")

	protectedCfg := config.New(
		config.WithConfigFile("example/config/config.yaml"),
		config.WithProtected(true),
		config.WithAutoWatch(true),
		config.WithOnError(func(err error) {
			fmt.Printf("[保护模式错误] %v\n", err)
		}),
	)
	if err := protectedCfg.Load(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	defer protectedCfg.Close()

	fmt.Println("保护模式已启动，修改 config.yaml 将自动恢复原始内容")
	fmt.Println("可以尝试手动修改 example/config/config.yaml 观察效果")

	// 动态切换保护模式
	fmt.Println("\n=== 动态切换保护模式 ===")
	fmt.Printf("当前保护状态: %v\n", protectedCfg.IsProtected())
	protectedCfg.SetProtected(false)
	fmt.Printf("关闭保护后:   %v\n", protectedCfg.IsProtected())
	protectedCfg.SetProtected(true)
	fmt.Printf("重新开启后:   %v\n", protectedCfg.IsProtected())

	// ============ 8. 全局默认实例 ============
	fmt.Println("\n=== 全局默认实例 ===")

	config.SetDefault(cfg)
	globalCfg := config.Default()
	fmt.Printf("全局实例 app.name: %s\n", globalCfg.GetString("app.name"))

	fmt.Println("\n配置管理演示完成。等待 5 秒后退出（期间可修改配置文件观察效果）...")
	time.Sleep(5 * time.Second)
}
