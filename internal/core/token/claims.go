package token

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// claimsContextKey Claims 在 Context 中的键
type claimsContextKey struct{}

// Claims JWT Claims
type Claims struct {
	// UserID 用户ID
	UserID string `json:"user_id"`

	// DeviceID 设备ID（可选）
	DeviceID string `json:"device_id,omitempty"`

	// TokenType 令牌类型（access/refresh）
	TokenType string `json:"token_type"`

	// CustomClaims 自定义字段
	CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`

	// JWT 标准字段
	jwt.RegisteredClaims
}

// NewClaims 创建新的 Claims
func NewClaims(userID string, tokenType TokenType, expiration int64, issuer string, audience []string) *Claims {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiration) * time.Second)
	
	return &Claims{
		UserID:    userID,
		TokenType: tokenType.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			Audience:  audience,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
	}
}

// Get 获取自定义字段
func (c *Claims) Get(key string) interface{} {
	if c.CustomClaims == nil {
		return nil
	}
	return c.CustomClaims[key]
}

// Set 设置自定义字段
func (c *Claims) Set(key string, value interface{}) {
	if c.CustomClaims == nil {
		c.CustomClaims = make(map[string]interface{})
	}
	c.CustomClaims[key] = value
}

// GetString 获取字符串字段
func (c *Claims) GetString(key string) string {
	if v := c.Get(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt 获取整数字段
func (c *Claims) GetInt(key string) int {
	if v := c.Get(key); v != nil {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

// GetInt64 获取 int64 字段
func (c *Claims) GetInt64(key string) int64 {
	if v := c.Get(key); v != nil {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}

// GetBool 获取布尔字段
func (c *Claims) GetBool(key string) bool {
	if v := c.Get(key); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetFloat64 获取浮点数字段
func (c *Claims) GetFloat64(key string) float64 {
	if v := c.Get(key); v != nil {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

// GetStringSlice 获取字符串切片字段
func (c *Claims) GetStringSlice(key string) []string {
	if v := c.Get(key); v != nil {
		if slice, ok := v.([]string); ok {
			return slice
		}
		// 尝试从 []interface{} 转换
		if slice, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// Validate 验证 Claims
func (c *Claims) Validate() error {
	if c.UserID == "" {
		return ErrUserIDRequired
	}
	if c.TokenType == "" {
		return ErrInvalidTokenType
	}
	if c.TokenType != TokenTypeAccess.String() && c.TokenType != TokenTypeRefresh.String() {
		return ErrInvalidTokenType
	}
	return nil
}

// IsAccessToken 判断是否为访问令牌
func (c *Claims) IsAccessToken() bool {
	return c.TokenType == TokenTypeAccess.String()
}

// IsRefreshToken 判断是否为刷新令牌
func (c *Claims) IsRefreshToken() bool {
	return c.TokenType == TokenTypeRefresh.String()
}

// GetClaimsFromContext 从 Context 获取 Claims
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*Claims)
	return claims, ok
}

// SetClaimsToContext 将 Claims 设置到 Context
func SetClaimsToContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

