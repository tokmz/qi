package cache

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// unlockScript 仅当 token 匹配时删除 key，保证解锁原子性
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`

const (
	lockRetryBase = 10 * time.Millisecond  // 初始重试间隔
	lockRetryMax  = 200 * time.Millisecond // 最大重试间隔
)

type redisLocker struct {
	client redis.UniversalClient
	prefix string
}

func (l *redisLocker) k(key string) string {
	if l.prefix == "" {
		return "lock:" + key
	}
	return l.prefix + "lock:" + key
}

func (l *redisLocker) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, func(), error) {
	token := uuid.NewString()
	ok, err := l.client.SetNX(ctx, l.k(key), token, ttl).Result()
	if err != nil {
		return false, nil, fmt.Errorf("cache: lock setnx: %w", err)
	}
	if !ok {
		return false, nil, nil
	}
	unlock := func() {
		_ = l.client.Eval(context.Background(), unlockScript, []string{l.k(key)}, token).Err()
	}
	return true, unlock, nil
}

func (l *redisLocker) Lock(ctx context.Context, key string, ttl time.Duration) (func(), error) {
	backoff := lockRetryBase
	for {
		ok, unlock, err := l.TryLock(ctx, key, ttl)
		if err != nil {
			return nil, err
		}
		if ok {
			return unlock, nil
		}
		// 指数退避 + ±25% 随机抖动，避免惊群
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		wait := backoff + jitter - backoff/4
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("cache: lock %q: %w", key, ctx.Err())
		case <-time.After(wait):
		}
		// 间隔翻倍，上限 lockRetryMax
		backoff *= 2
		if backoff > lockRetryMax {
			backoff = lockRetryMax
		}
	}
}
