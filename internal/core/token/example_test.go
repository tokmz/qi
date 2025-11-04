package token_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"qi/internal/core/token"

	"github.com/redis/go-redis/v9"
)

// Example_basic 基础使用示例
func Example_basic() {
	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	// 创建配置
	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	// 创建令牌管理器
	manager, err := token.New(cfg, &token.DefaultLogger{})
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 生成令牌对
	pair, err := manager.GenerateTokenPair(ctx, "user-123", map[string]interface{}{
		"username": "alice",
		"role":     "admin",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Access Token: %s\n", pair.AccessToken[:20]+"...")
	fmt.Printf("Refresh Token: %s\n", pair.RefreshToken[:20]+"...")

	// 验证访问令牌
	claims, err := manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User ID: %s\n", claims.UserID)
	fmt.Printf("Username: %s\n", claims.GetString("username"))
}

// Example_withDevice 带设备信息的示例
func Example_withDevice() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 为设备生成令牌
	pair, err := manager.GenerateTokenPairWithDevice(
		ctx,
		"user-123",
		"device-abc",
		map[string]interface{}{
			"device_name": "iPhone 13",
			"device_os":   "iOS 16.0",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Device token generated: %s\n", pair.AccessToken[:20]+"...")

	// 获取用户的所有设备
	devices, err := manager.GetUserDevices(ctx, "user-123")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total devices: %d\n", len(devices))

	// 撤销特定设备的令牌
	err = manager.RevokeDeviceToken(ctx, "user-123", "device-abc")
	if err != nil {
		log.Fatal(err)
	}
}

// Example_refreshToken 刷新令牌示例
func Example_refreshToken() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 生成令牌对
	pair, err := manager.GenerateTokenPair(ctx, "user-123", nil)
	if err != nil {
		log.Fatal(err)
	}

	// 使用刷新令牌获取新的访问令牌
	newPair, err := manager.RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("New Access Token: %s\n", newPair.AccessToken[:20]+"...")
	fmt.Printf("Expires in: %d seconds\n", newPair.ExpiresIn)
}

// Example_revokeToken 撤销令牌示例
func Example_revokeToken() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 生成令牌
	pair, err := manager.GenerateTokenPair(ctx, "user-123", nil)
	if err != nil {
		log.Fatal(err)
	}

	// 撤销令牌
	err = manager.RevokeToken(ctx, pair.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	// 验证被撤销的令牌（应该失败）
	_, err = manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != nil {
		fmt.Printf("Token verification failed: %v\n", err)
	}

	// 撤销用户的所有令牌
	err = manager.RevokeAllUserTokens(ctx, "user-123")
	if err != nil {
		log.Fatal(err)
	}
}

// Example_customClaims 自定义 Claims 示例
func Example_customClaims() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 生成带自定义字段的令牌
	pair, err := manager.GenerateTokenPair(ctx, "user-123", map[string]interface{}{
		"username":    "alice",
		"email":       "alice@example.com",
		"role":        "admin",
		"permissions": []string{"read", "write", "delete"},
		"department":  "IT",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 验证并读取自定义字段
	claims, err := manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Username: %s\n", claims.GetString("username"))
	fmt.Printf("Email: %s\n", claims.GetString("email"))
	fmt.Printf("Role: %s\n", claims.GetString("role"))
	fmt.Printf("Department: %s\n", claims.GetString("department"))

	// 读取权限列表
	permissions := claims.GetStringSlice("permissions")
	fmt.Printf("Permissions: %v\n", permissions)
}

// Example_basicManager 基础管理器示例
func Example_basicManager() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	// 创建管理器
	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	pair, err := manager.GenerateTokenPair(ctx, "user-123", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Token generated: %s\n", pair.AccessToken[:20]+"...")
}

// Example_errorHandling 错误处理示例
func Example_errorHandling() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 尝试验证无效的令牌
	invalidToken := "invalid.token.string"
	_, err = manager.VerifyAccessToken(ctx, invalidToken)
	
	// 检查具体的错误类型
	switch err {
	case token.ErrTokenInvalid:
		fmt.Println("Token is invalid")
	case token.ErrTokenExpired:
		fmt.Println("Token has expired")
	case token.ErrTokenBlacklisted:
		fmt.Println("Token has been revoked")
	default:
		fmt.Printf("Unexpected error: %v\n", err)
	}
}

// Example_configuration 配置示例
func Example_configuration() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	// 创建自定义配置
	cfg := &token.Config{
		SecretKey: "your-secret-key-must-be-at-least-32-characters-long!",
		
		// 访问令牌配置
		AccessToken: token.TokenConfig{
			Expiration: 30 * time.Minute, // 30 分钟
			Issuer:     "my-service",
			Audience:   []string{"web", "mobile"},
		},
		
		// 刷新令牌配置
		RefreshToken: token.TokenConfig{
			Expiration: 30 * 24 * time.Hour, // 30 天
			Issuer:     "my-service",
			Audience:   []string{"web", "mobile"},
		},
		
		// 签名算法
		SigningMethod: token.SigningMethodHS256,
		
		// Redis 配置
		Redis: token.RedisConfig{
			Client:          rdb,
			KeyPrefix:       "myapp:token:",
			BlacklistPrefix: "myapp:blacklist:",
		},
		
		// 清理配置
		Cleanup: token.CleanupConfig{
			Enabled:   true,
			Interval:  2 * time.Hour,
			BatchSize: 1000,
		},
	}

	manager, err := token.New(cfg, &token.DefaultLogger{})
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	fmt.Println("Manager created with custom configuration")
}

// Example_healthCheck 健康检查示例
func Example_healthCheck() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	cfg := token.DefaultConfig()
	cfg.SecretKey = "your-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb

	manager, err := token.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 健康检查
	if err := manager.Health(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("Service is healthy")
	}

	// 获取统计信息
	stats, err := manager.GetStats(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Blacklist count: %d\n", stats.BlacklistCount)
}

