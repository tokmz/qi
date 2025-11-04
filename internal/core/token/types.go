package token

import (
	"time"
)

// SigningMethod 签名方法
type SigningMethod string

const (
	// SigningMethodHS256 HMAC SHA256
	SigningMethodHS256 SigningMethod = "HS256"
	// SigningMethodHS384 HMAC SHA384
	SigningMethodHS384 SigningMethod = "HS384"
	// SigningMethodHS512 HMAC SHA512
	SigningMethodHS512 SigningMethod = "HS512"
	// SigningMethodRS256 RSA SHA256
	SigningMethodRS256 SigningMethod = "RS256"
	// SigningMethodRS384 RSA SHA384
	SigningMethodRS384 SigningMethod = "RS384"
	// SigningMethodRS512 RSA SHA512
	SigningMethodRS512 SigningMethod = "RS512"
)

// String 返回签名方法的字符串表示
func (s SigningMethod) String() string {
	return string(s)
}

// IsHMAC 判断是否为 HMAC 算法
func (s SigningMethod) IsHMAC() bool {
	switch s {
	case SigningMethodHS256, SigningMethodHS384, SigningMethodHS512:
		return true
	default:
		return false
	}
}

// IsRSA 判断是否为 RSA 算法
func (s SigningMethod) IsRSA() bool {
	switch s {
	case SigningMethodRS256, SigningMethodRS384, SigningMethodRS512:
		return true
	default:
		return false
	}
}

// TokenType 令牌类型
type TokenType string

const (
	// TokenTypeAccess 访问令牌
	TokenTypeAccess TokenType = "access"
	// TokenTypeRefresh 刷新令牌
	TokenTypeRefresh TokenType = "refresh"
)

// String 返回令牌类型的字符串表示
func (t TokenType) String() string {
	return string(t)
}

// TokenPair 令牌对（访问令牌 + 刷新令牌）
type TokenPair struct {
	// AccessToken 访问令牌
	AccessToken string `json:"access_token"`

	// RefreshToken 刷新令牌
	RefreshToken string `json:"refresh_token"`

	// TokenType 令牌类型（默认为 "Bearer"）
	TokenType string `json:"token_type"`

	// ExpiresAt 过期时间
	ExpiresAt time.Time `json:"expires_at"`

	// ExpiresIn 过期秒数（从现在开始）
	ExpiresIn int64 `json:"expires_in"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	// DeviceID 设备ID
	DeviceID string `json:"device_id"`

	// UserID 用户ID
	UserID string `json:"user_id"`

	// LastActive 最后活跃时间
	LastActive time.Time `json:"last_active"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// CustomInfo 自定义信息
	CustomInfo map[string]interface{} `json:"custom_info,omitempty"`
}

// RefreshTokenInfo 刷新令牌信息
type RefreshTokenInfo struct {
	// TokenID 令牌ID（JTI）
	TokenID string `json:"token_id"`

	// UserID 用户ID
	UserID string `json:"user_id"`

	// DeviceID 设备ID（可选）
	DeviceID string `json:"device_id,omitempty"`

	// ExpiresAt 过期时间
	ExpiresAt time.Time `json:"expires_at"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// BlacklistEntry 黑名单条目
type BlacklistEntry struct {
	// TokenID 令牌ID（JTI）
	TokenID string `json:"token_id"`

	// UserID 用户ID
	UserID string `json:"user_id"`

	// RevokedAt 撤销时间
	RevokedAt time.Time `json:"revoked_at"`

	// TTL 过期时间（秒）
	TTL int64 `json:"ttl"`
}

