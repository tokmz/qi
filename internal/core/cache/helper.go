package cache

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

// buildKey 构建完整的缓存键
func buildKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + key
}

// randomExpiration 生成随机过期时间
// baseTTL: 基础过期时间
// jitter: 抖动比例 (0.0 - 1.0)，例如 0.2 表示 ±20%
func randomExpiration(baseTTL time.Duration, jitter float64) time.Duration {
	if jitter <= 0 || jitter > 1 {
		return baseTTL
	}

	// 计算抖动范围
	jitterDuration := time.Duration(float64(baseTTL) * jitter)

	// 生成随机抖动 (-jitterDuration ~ +jitterDuration)
	randomJitter, err := randomInt64(int64(jitterDuration * 2))
	if err != nil {
		return baseTTL
	}

	return baseTTL + time.Duration(randomJitter) - jitterDuration
}

// randomInt64 生成随机 int64
func randomInt64(max int64) (int64, error) {
	if max <= 0 {
		return 0, nil
	}
	
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	
	return n.Int64(), nil
}

// calculateHitRate 计算命中率
func calculateHitRate(hits, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// isNullValue 检查是否为空值标记
func isNullValue(data []byte) bool {
	return len(data) == 0 || string(data) == "null" || string(data) == "nil"
}

// nullValueMarker 空值标记
func nullValueMarker() []byte {
	return []byte("__NULL__")
}

// min 返回两个数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// calculateExpiration 计算过期时间
func calculateExpiration(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{} // 永不过期
	}
	return time.Now().Add(ttl)
}

// isExpired 检查是否过期
func isExpired(expiresAt time.Time) bool {
	if expiresAt.IsZero() {
		return false // 永不过期
	}
	return time.Now().After(expiresAt)
}

// timeUntil 计算剩余时间
func timeUntil(expiresAt time.Time) time.Duration {
	if expiresAt.IsZero() {
		return time.Duration(math.MaxInt64) // 永不过期
	}
	
	ttl := time.Until(expiresAt)
	if ttl < 0 {
		return 0
	}
	return ttl
}

// batchKeys 将键列表分批
func batchKeys(keys []string, batchSize int) [][]string {
	if batchSize <= 0 {
		batchSize = 100
	}

	var batches [][]string
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batches = append(batches, keys[i:end])
	}

	return batches
}

