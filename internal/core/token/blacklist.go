package token

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Blacklist 黑名单管理器
type Blacklist struct {
	rdb    *redis.Client
	prefix string
	logger Logger
}

// newBlacklist 创建新的黑名单管理器
func newBlacklist(rdb *redis.Client, prefix string, logger Logger) *Blacklist {
	return &Blacklist{
		rdb:    rdb,
		prefix: prefix,
		logger: logger,
	}
}

// Add 添加令牌到黑名单
func (b *Blacklist) Add(ctx context.Context, tokenID string, userID string, ttl time.Duration) error {
	key := blacklistKey(b.prefix, tokenID)
	
	// 存储到 Redis，设置过期时间
	err := b.rdb.Set(ctx, key, userID, ttl).Err()
	if err != nil {
		b.logger.Error("Failed to add token to blacklist", "tokenID", tokenID, "error", err)
		return err
	}

	b.logger.Debug("Token added to blacklist", "tokenID", tokenID, "userID", userID, "ttl", ttl)
	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (b *Blacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := blacklistKey(b.prefix, tokenID)
	
	exists, err := b.rdb.Exists(ctx, key).Result()
	if err != nil {
		b.logger.Error("Failed to check blacklist", "tokenID", tokenID, "error", err)
		return false, err
	}

	return exists > 0, nil
}

// Remove 从黑名单中移除令牌
func (b *Blacklist) Remove(ctx context.Context, tokenID string) error {
	key := blacklistKey(b.prefix, tokenID)
	
	err := b.rdb.Del(ctx, key).Err()
	if err != nil {
		b.logger.Error("Failed to remove token from blacklist", "tokenID", tokenID, "error", err)
		return err
	}

	b.logger.Debug("Token removed from blacklist", "tokenID", tokenID)
	return nil
}

// AddBatch 批量添加令牌到黑名单
func (b *Blacklist) AddBatch(ctx context.Context, entries []*BlacklistEntry) error {
	if len(entries) == 0 {
		return nil
	}

	pipe := b.rdb.Pipeline()

	for _, entry := range entries {
		key := blacklistKey(b.prefix, entry.TokenID)
		ttl := time.Duration(entry.TTL) * time.Second
		pipe.Set(ctx, key, entry.UserID, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		b.logger.Error("Failed to add tokens to blacklist in batch", "count", len(entries), "error", err)
		return err
	}

	b.logger.Debug("Tokens added to blacklist in batch", "count", len(entries))
	return nil
}

// RemoveBatch 批量从黑名单中移除令牌
func (b *Blacklist) RemoveBatch(ctx context.Context, tokenIDs []string) error {
	if len(tokenIDs) == 0 {
		return nil
	}

	keys := make([]string, len(tokenIDs))
	for i, tokenID := range tokenIDs {
		keys[i] = blacklistKey(b.prefix, tokenID)
	}

	err := b.rdb.Del(ctx, keys...).Err()
	if err != nil {
		b.logger.Error("Failed to remove tokens from blacklist in batch", "count", len(tokenIDs), "error", err)
		return err
	}

	b.logger.Debug("Tokens removed from blacklist in batch", "count", len(tokenIDs))
	return nil
}

// Clear 清空黑名单
func (b *Blacklist) Clear(ctx context.Context) error {
	pattern := b.prefix + "*"
	
	// 使用 SCAN 命令迭代删除
	iter := b.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	
	count := 0
	for iter.Next(ctx) {
		if err := b.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			b.logger.Error("Failed to delete blacklist key", "key", iter.Val(), "error", err)
			continue
		}
		count++
	}

	if err := iter.Err(); err != nil {
		b.logger.Error("Failed to scan blacklist keys", "error", err)
		return err
	}

	b.logger.Info("Blacklist cleared", "count", count)
	return nil
}

// Count 获取黑名单数量
func (b *Blacklist) Count(ctx context.Context) (int64, error) {
	pattern := b.prefix + "*"
	
	var count int64
	iter := b.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	
	for iter.Next(ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		b.logger.Error("Failed to count blacklist entries", "error", err)
		return 0, err
	}

	return count, nil
}

