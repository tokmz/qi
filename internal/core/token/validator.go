package token

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Validator 令牌验证器
type Validator struct {
	config *Config
	logger Logger
}

// newValidator 创建新的验证器
func newValidator(config *Config, logger Logger) *Validator {
	return &Validator{
		config: config,
		logger: logger,
	}
}

// parseToken 解析令牌
func (v *Validator) parseToken(tokenStr string) (*Claims, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		return v.getSigningKey(token)
	})

	if err != nil {
		v.logger.Error("Failed to parse token", "error", err)
		return nil, ErrTokenInvalid
	}

	// 检查令牌是否有效
	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	// 提取 Claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	// 验证 Claims
	if err := claims.Validate(); err != nil {
		return nil, err
	}

	return claims, nil
}

// generateToken 生成令牌
func (v *Validator) generateToken(claims *Claims) (string, error) {
	// 创建 JWT token
	token := jwt.NewWithClaims(v.getSigningMethod(), claims)

	// 获取签名密钥
	signingKey, err := v.getSigningKeyForGenerate()
	if err != nil {
		return "", err
	}

	// 签名并获取完整的编码后的令牌字符串
	tokenStr, err := token.SignedString(signingKey)
	if err != nil {
		v.logger.Error("Failed to sign token", "error", err)
		return "", err
	}

	return tokenStr, nil
}

// getSigningMethod 获取签名方法
func (v *Validator) getSigningMethod() jwt.SigningMethod {
	switch v.config.SigningMethod {
	case SigningMethodHS256:
		return jwt.SigningMethodHS256
	case SigningMethodHS384:
		return jwt.SigningMethodHS384
	case SigningMethodHS512:
		return jwt.SigningMethodHS512
	case SigningMethodRS256:
		return jwt.SigningMethodRS256
	case SigningMethodRS384:
		return jwt.SigningMethodRS384
	case SigningMethodRS512:
		return jwt.SigningMethodRS512
	default:
		return jwt.SigningMethodHS256
	}
}

// getSigningKey 获取验证密钥（用于解析令牌）
func (v *Validator) getSigningKey(token *jwt.Token) (interface{}, error) {
	// 检查签名方法
	expectedMethod := v.getSigningMethod()
	if token.Method.Alg() != expectedMethod.Alg() {
		return nil, fmt.Errorf("%w: expected %s, got %s", 
			ErrSigningMethodMismatch, expectedMethod.Alg(), token.Method.Alg())
	}

	// 根据算法返回相应的密钥
	if v.config.SigningMethod.IsHMAC() {
		return []byte(v.config.SecretKey), nil
	}

	if v.config.SigningMethod.IsRSA() {
		return v.config.RSAPublicKey, nil
	}

	return nil, ErrInvalidConfig
}

// getSigningKeyForGenerate 获取签名密钥（用于生成令牌）
func (v *Validator) getSigningKeyForGenerate() (interface{}, error) {
	// 根据算法返回相应的密钥
	if v.config.SigningMethod.IsHMAC() {
		return []byte(v.config.SecretKey), nil
	}

	if v.config.SigningMethod.IsRSA() {
		return v.config.RSAPrivateKey, nil
	}

	return nil, ErrInvalidConfig
}

