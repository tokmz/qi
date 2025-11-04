package token

import (
	"crypto/rsa"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 令牌管理器配置
type Config struct {
	// SecretKey 签名密钥（HMAC 算法使用）
	// 建议至少 32 个字符
	SecretKey string

	// RSAPrivateKey RSA 私钥（RS* 算法使用）
	RSAPrivateKey *rsa.PrivateKey

	// RSAPublicKey RSA 公钥（RS* 算法使用）
	RSAPublicKey *rsa.PublicKey

	// AccessToken 访问令牌配置
	AccessToken TokenConfig

	// RefreshToken 刷新令牌配置
	RefreshToken TokenConfig

	// SigningMethod 签名算法
	SigningMethod SigningMethod

	// Redis Redis 配置
	Redis RedisConfig

	// Cleanup 清理配置
	Cleanup CleanupConfig
}

// TokenConfig 令牌配置
type TokenConfig struct {
	// Expiration 过期时间
	Expiration time.Duration

	// Issuer 签发者
	Issuer string

	// Audience 受众
	Audience []string

	// Subject 主题
	Subject string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	// Client Redis 客户端
	Client *redis.Client

	// KeyPrefix 键前缀
	KeyPrefix string

	// BlacklistPrefix 黑名单前缀
	BlacklistPrefix string
}

// CleanupConfig 清理配置
type CleanupConfig struct {
	// Enabled 是否启用自动清理
	Enabled bool

	// Interval 清理间隔
	Interval time.Duration

	// BatchSize 批量清理大小
	BatchSize int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		AccessToken: TokenConfig{
			Expiration: 15 * time.Minute, // 默认 15 分钟
			Issuer:     "qi-token",
		},
		RefreshToken: TokenConfig{
			Expiration: 7 * 24 * time.Hour, // 默认 7 天
			Issuer:     "qi-token",
		},
		SigningMethod: SigningMethodHS256,
		Redis: RedisConfig{
			KeyPrefix:       "token:",
			BlacklistPrefix: "blacklist:",
		},
		Cleanup: CleanupConfig{
			Enabled:   true,
			Interval:  1 * time.Hour,
			BatchSize: 1000,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证签名方法
	if c.SigningMethod == "" {
		return ErrInvalidConfig
	}

	// 验证密钥
	if c.SigningMethod.IsHMAC() {
		if c.SecretKey == "" {
			return ErrInvalidConfig
		}
		if len(c.SecretKey) < 32 {
			return ErrSecretKeyTooShort
		}
	}

	// 验证 RSA 密钥
	if c.SigningMethod.IsRSA() {
		if c.RSAPrivateKey == nil || c.RSAPublicKey == nil {
			return ErrRSAKeyRequired
		}
	}

	// 验证 Redis 客户端
	if c.Redis.Client == nil {
		return ErrRedisClientRequired
	}

	// 验证令牌配置
	if c.AccessToken.Expiration <= 0 {
		return ErrInvalidConfig
	}
	if c.RefreshToken.Expiration <= 0 {
		return ErrInvalidConfig
	}

	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	
	// 复制切片
	if len(c.AccessToken.Audience) > 0 {
		clone.AccessToken.Audience = make([]string, len(c.AccessToken.Audience))
		copy(clone.AccessToken.Audience, c.AccessToken.Audience)
	}
	
	if len(c.RefreshToken.Audience) > 0 {
		clone.RefreshToken.Audience = make([]string, len(c.RefreshToken.Audience))
		copy(clone.RefreshToken.Audience, c.RefreshToken.Audience)
	}
	
	return &clone
}

