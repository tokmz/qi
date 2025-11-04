package token

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// generateTokenID 生成唯一的令牌ID（JTI）
func generateTokenID() string {
	return uuid.New().String()
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// calculateTTL 计算 TTL（生存时间）
func calculateTTL(expiresAt time.Time) int64 {
	ttl := time.Until(expiresAt).Seconds()
	if ttl < 0 {
		return 0
	}
	return int64(ttl)
}

// isExpired 检查是否过期
func isExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

// redisKey 生成 Redis 键
func redisKey(prefix, key string) string {
	return prefix + key
}

// userTokensKey 生成用户令牌键
func userTokensKey(prefix, userID string) string {
	return redisKey(prefix, "user:"+userID+":*")
}

// deviceTokenKey 生成设备令牌键
func deviceTokenKey(prefix, userID, deviceID string) string {
	return redisKey(prefix, "user:"+userID+":device:"+deviceID)
}

// refreshTokenKey 生成刷新令牌键
func refreshTokenKey(prefix, tokenID string) string {
	return redisKey(prefix, "refresh:"+tokenID)
}

// blacklistKey 生成黑名单键
func blacklistKey(prefix, tokenID string) string {
	return redisKey(prefix, tokenID)
}

// deviceInfoKey 生成设备信息键
func deviceInfoKey(prefix, userID, deviceID string) string {
	return redisKey(prefix, "device:"+userID+":"+deviceID)
}

// userDevicesPattern 生成用户设备模式
func userDevicesPattern(prefix, userID string) string {
	return redisKey(prefix, "device:"+userID+":*")
}

