package token

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 测试辅助函数
func setupTestManager(t *testing.T) (*Manager, *redis.Client) {
	// 使用 Redis 测试数据库
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试 Redis 连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	cfg := DefaultConfig()
	cfg.SecretKey = "test-secret-key-must-be-at-least-32-characters-long!"
	cfg.Redis.Client = rdb
	cfg.AccessToken.Expiration = 5 * time.Minute
	cfg.RefreshToken.Expiration = 1 * time.Hour

	manager, err := New(cfg, &NoopLogger{})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	return manager, rdb
}

func cleanupTestManager(t *testing.T, manager *Manager, rdb *redis.Client) {
	ctx := context.Background()
	
	// 清理测试数据
	rdb.FlushDB(ctx)
	
	// 关闭管理器
	if err := manager.Close(); err != nil {
		t.Errorf("Failed to close manager: %v", err)
	}
	
	// 关闭 Redis 连接
	if err := rdb.Close(); err != nil {
		t.Errorf("Failed to close redis: %v", err)
	}
}

func TestGenerateTokenPair(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()
	userID := "test-user-123"

	pair, err := manager.GenerateTokenPair(ctx, userID, map[string]interface{}{
		"username": "testuser",
		"role":     "admin",
	})

	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("Access token is empty")
	}

	if pair.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}

	if pair.TokenType != "Bearer" {
		t.Errorf("Expected token type Bearer, got %s", pair.TokenType)
	}

	if pair.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}
}

func TestVerifyAccessToken(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()
	userID := "test-user-123"

	// 生成令牌
	pair, err := manager.GenerateTokenPair(ctx, userID, map[string]interface{}{
		"username": "testuser",
		"role":     "admin",
	})
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 验证访问令牌
	claims, err := manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to verify access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}

	if claims.GetString("username") != "testuser" {
		t.Error("Username claim not found or incorrect")
	}

	if claims.GetString("role") != "admin" {
		t.Error("Role claim not found or incorrect")
	}
}

func TestRefreshToken(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()
	userID := "test-user-123"

	// 生成初始令牌对
	originalPair, err := manager.GenerateTokenPair(ctx, userID, nil)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 等待一小段时间以确保时间戳不同
	time.Sleep(time.Second)

	// 刷新令牌
	newPair, err := manager.RefreshToken(ctx, originalPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if newPair.AccessToken == originalPair.AccessToken {
		t.Error("New access token should be different from original")
	}

	// 验证新的访问令牌
	claims, err := manager.VerifyAccessToken(ctx, newPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to verify new access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
}

func TestRevokeToken(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()
	userID := "test-user-123"

	// 生成令牌
	pair, err := manager.GenerateTokenPair(ctx, userID, nil)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 验证令牌有效
	_, err = manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("Token should be valid before revocation: %v", err)
	}

	// 撤销令牌
	err = manager.RevokeToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// 验证令牌已被撤销
	_, err = manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != ErrTokenBlacklisted {
		t.Errorf("Expected ErrTokenBlacklisted, got %v", err)
	}
}

func TestTokenWithDevice(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()
	userID := "test-user-123"
	deviceID := "device-abc"

	// 为设备生成令牌
	pair, err := manager.GenerateTokenPairWithDevice(ctx, userID, deviceID, map[string]interface{}{
		"device_name": "iPhone 13",
	})
	if err != nil {
		t.Fatalf("Failed to generate token pair with device: %v", err)
	}

	// 验证令牌
	claims, err := manager.VerifyAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to verify access token: %v", err)
	}

	if claims.DeviceID != deviceID {
		t.Errorf("Expected device ID %s, got %s", deviceID, claims.DeviceID)
	}

	// 获取设备信息
	devices, err := manager.GetUserDevices(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user devices: %v", err)
	}

	if len(devices) == 0 {
		t.Error("Expected at least one device")
	}

	// 撤销设备令牌
	err = manager.RevokeDeviceToken(ctx, userID, deviceID)
	if err != nil {
		t.Fatalf("Failed to revoke device token: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*Config)
		expectError  error
	}{
		{
			name: "valid config",
			modifyConfig: func(cfg *Config) {
				cfg.SecretKey = "valid-secret-key-at-least-32-characters-long!"
				cfg.Redis.Client = redis.NewClient(&redis.Options{
					Addr: "localhost:6379",
					DB:   15,
				})
			},
			expectError: nil,
		},
		{
			name: "secret key too short",
			modifyConfig: func(cfg *Config) {
				cfg.SecretKey = "short"
			},
			expectError: ErrSecretKeyTooShort,
		},
		{
			name: "missing secret key",
			modifyConfig: func(cfg *Config) {
				cfg.SecretKey = ""
			},
			expectError: ErrInvalidConfig,
		},
		{
			name: "missing redis client",
			modifyConfig: func(cfg *Config) {
				cfg.SecretKey = "valid-secret-key-at-least-32-characters-long!"
				cfg.Redis.Client = nil
			},
			expectError: ErrRedisClientRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modifyConfig(cfg)

			err := cfg.Validate()
			if err != tt.expectError {
				t.Errorf("Expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

func TestClaimsHelpers(t *testing.T) {
	claims := &Claims{
		CustomClaims: map[string]interface{}{
			"string_field": "value",
			"int_field":    42,
			"bool_field":   true,
			"float_field":  3.14,
			"slice_field":  []string{"a", "b", "c"},
		},
	}

	// Test GetString
	if claims.GetString("string_field") != "value" {
		t.Error("GetString failed")
	}

	// Test GetInt
	if claims.GetInt("int_field") != 42 {
		t.Error("GetInt failed")
	}

	// Test GetBool
	if !claims.GetBool("bool_field") {
		t.Error("GetBool failed")
	}

	// Test GetFloat64
	if claims.GetFloat64("float_field") != 3.14 {
		t.Error("GetFloat64 failed")
	}

	// Test GetStringSlice
	slice := claims.GetStringSlice("slice_field")
	if len(slice) != 3 || slice[0] != "a" {
		t.Error("GetStringSlice failed")
	}

	// Test non-existent field
	if claims.GetString("nonexistent") != "" {
		t.Error("GetString should return empty for nonexistent field")
	}
}

func TestHealth(t *testing.T) {
	manager, rdb := setupTestManager(t)
	defer cleanupTestManager(t, manager, rdb)

	ctx := context.Background()

	err := manager.Health(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

