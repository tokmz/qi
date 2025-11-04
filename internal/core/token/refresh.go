package token

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RefreshTokenManager 刷新令牌管理器
type RefreshTokenManager struct {
	rdb    *redis.Client
	prefix string
	logger Logger
}

// newRefreshTokenManager 创建新的刷新令牌管理器
func newRefreshTokenManager(rdb *redis.Client, prefix string, logger Logger) *RefreshTokenManager {
	return &RefreshTokenManager{
		rdb:    rdb,
		prefix: prefix,
		logger: logger,
	}
}

// Store 存储刷新令牌信息
func (r *RefreshTokenManager) Store(ctx context.Context, info *RefreshTokenInfo) error {
	key := refreshTokenKey(r.prefix, info.TokenID)
	
	// 序列化令牌信息
	data, err := json.Marshal(info)
	if err != nil {
		r.logger.Error("Failed to marshal refresh token info", "tokenID", info.TokenID, "error", err)
		return err
	}

	// 计算 TTL
	ttl := time.Until(info.ExpiresAt)
	if ttl <= 0 {
		return ErrTokenExpired
	}

	// 存储到 Redis
	err = r.rdb.Set(ctx, key, data, ttl).Err()
	if err != nil {
		r.logger.Error("Failed to store refresh token", "tokenID", info.TokenID, "error", err)
		return err
	}

	// 如果有设备ID，也存储设备关联信息
	if info.DeviceID != "" {
		deviceKey := deviceTokenKey(r.prefix, info.UserID, info.DeviceID)
		err = r.rdb.Set(ctx, deviceKey, info.TokenID, ttl).Err()
		if err != nil {
			r.logger.Error("Failed to store device token mapping", "deviceID", info.DeviceID, "error", err)
			return err
		}
	}

	r.logger.Debug("Refresh token stored", "tokenID", info.TokenID, "userID", info.UserID, "deviceID", info.DeviceID)
	return nil
}

// Get 获取刷新令牌信息
func (r *RefreshTokenManager) Get(ctx context.Context, tokenID string) (*RefreshTokenInfo, error) {
	key := refreshTokenKey(r.prefix, tokenID)
	
	data, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTokenNotFound
		}
		r.logger.Error("Failed to get refresh token", "tokenID", tokenID, "error", err)
		return nil, err
	}

	var info RefreshTokenInfo
	if err := json.Unmarshal(data, &info); err != nil {
		r.logger.Error("Failed to unmarshal refresh token info", "tokenID", tokenID, "error", err)
		return nil, err
	}

	return &info, nil
}

// Delete 删除刷新令牌
func (r *RefreshTokenManager) Delete(ctx context.Context, tokenID string) error {
	// 先获取令牌信息
	info, err := r.Get(ctx, tokenID)
	if err != nil {
		if err == ErrTokenNotFound {
			return nil // 已经不存在，不需要删除
		}
		return err
	}

	// 删除令牌
	key := refreshTokenKey(r.prefix, tokenID)
	err = r.rdb.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error("Failed to delete refresh token", "tokenID", tokenID, "error", err)
		return err
	}

	// 如果有设备ID，也删除设备关联信息
	if info.DeviceID != "" {
		deviceKey := deviceTokenKey(r.prefix, info.UserID, info.DeviceID)
		if err := r.rdb.Del(ctx, deviceKey).Err(); err != nil {
			r.logger.Warn("Failed to delete device token mapping", "deviceID", info.DeviceID, "error", err)
		}
	}

	r.logger.Debug("Refresh token deleted", "tokenID", tokenID)
	return nil
}

// DeleteByUser 删除用户的所有刷新令牌
func (r *RefreshTokenManager) DeleteByUser(ctx context.Context, userID string) error {
	pattern := userTokensKey(r.prefix, userID)
	
	iter := r.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	count := 0
	
	for iter.Next(ctx) {
		if err := r.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			r.logger.Error("Failed to delete user token", "key", iter.Val(), "error", err)
			continue
		}
		count++
	}

	if err := iter.Err(); err != nil {
		r.logger.Error("Failed to scan user tokens", "userID", userID, "error", err)
		return err
	}

	r.logger.Debug("User refresh tokens deleted", "userID", userID, "count", count)
	return nil
}

// DeleteByDevice 删除设备的刷新令牌
func (r *RefreshTokenManager) DeleteByDevice(ctx context.Context, userID, deviceID string) error {
	deviceKey := deviceTokenKey(r.prefix, userID, deviceID)
	
	// 获取令牌ID
	tokenID, err := r.rdb.Get(ctx, deviceKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // 设备令牌不存在
		}
		r.logger.Error("Failed to get device token", "deviceID", deviceID, "error", err)
		return err
	}

	// 删除令牌
	if err := r.Delete(ctx, tokenID); err != nil {
		return err
	}

	r.logger.Debug("Device refresh token deleted", "userID", userID, "deviceID", deviceID)
	return nil
}

// GetByDevice 通过设备获取刷新令牌信息
func (r *RefreshTokenManager) GetByDevice(ctx context.Context, userID, deviceID string) (*RefreshTokenInfo, error) {
	deviceKey := deviceTokenKey(r.prefix, userID, deviceID)
	
	tokenID, err := r.rdb.Get(ctx, deviceKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTokenNotFound
		}
		r.logger.Error("Failed to get device token", "deviceID", deviceID, "error", err)
		return nil, err
	}

	return r.Get(ctx, tokenID)
}

// Exists 检查刷新令牌是否存在
func (r *RefreshTokenManager) Exists(ctx context.Context, tokenID string) (bool, error) {
	key := refreshTokenKey(r.prefix, tokenID)
	
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to check refresh token existence", "tokenID", tokenID, "error", err)
		return false, err
	}

	return exists > 0, nil
}

// UpdateExpiration 更新刷新令牌过期时间
func (r *RefreshTokenManager) UpdateExpiration(ctx context.Context, tokenID string, newExpiration time.Time) error {
	// 获取现有令牌信息
	info, err := r.Get(ctx, tokenID)
	if err != nil {
		return err
	}

	// 更新过期时间
	info.ExpiresAt = newExpiration

	// 重新存储
	return r.Store(ctx, info)
}

