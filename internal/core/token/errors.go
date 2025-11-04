package token

import "errors"

// 预定义错误
var (
	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("invalid token config")

	// ErrSecretKeyTooShort 密钥太短
	ErrSecretKeyTooShort = errors.New("secret key must be at least 32 characters")

	// ErrRSAKeyRequired RSA 密钥必需
	ErrRSAKeyRequired = errors.New("RSA private and public keys are required for RS* algorithms")

	// ErrRedisClientRequired Redis 客户端必需
	ErrRedisClientRequired = errors.New("redis client is required")

	// ErrTokenExpired 令牌已过期
	ErrTokenExpired = errors.New("token has expired")

	// ErrTokenInvalid 令牌无效
	ErrTokenInvalid = errors.New("invalid token")

	// ErrTokenBlacklisted 令牌已被撤销（在黑名单中）
	ErrTokenBlacklisted = errors.New("token has been revoked")

	// ErrTokenNotFound 令牌不存在
	ErrTokenNotFound = errors.New("token not found")

	// ErrTokenTypeMismatch 令牌类型不匹配
	ErrTokenTypeMismatch = errors.New("token type mismatch")

	// ErrInvalidTokenType 无效的令牌类型
	ErrInvalidTokenType = errors.New("invalid token type")

	// ErrRefreshTokenRequired 需要刷新令牌
	ErrRefreshTokenRequired = errors.New("refresh token is required")

	// ErrAccessTokenRequired 需要访问令牌
	ErrAccessTokenRequired = errors.New("access token is required")

	// ErrUserIDRequired 用户ID必需
	ErrUserIDRequired = errors.New("user ID is required")

	// ErrDeviceNotFound 设备不存在
	ErrDeviceNotFound = errors.New("device not found")

	// ErrInvalidClaims 无效的 Claims
	ErrInvalidClaims = errors.New("invalid claims")

	// ErrSigningMethodMismatch 签名方法不匹配
	ErrSigningMethodMismatch = errors.New("signing method mismatch")

	// ErrManagerNotInitialized 管理器未初始化
	ErrManagerNotInitialized = errors.New("token manager not initialized")

	// ErrManagerAlreadyClosed 管理器已关闭
	ErrManagerAlreadyClosed = errors.New("token manager already closed")
)
