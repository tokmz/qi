package token

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// 全局令牌管理器
var (
	globalManager *Manager
	globalMu      sync.RWMutex
)

// Manager 令牌管理器
type Manager struct {
	config    *Config
	validator *Validator
	blacklist *Blacklist
	refresh   *RefreshTokenManager
	logger    Logger
	closed    bool
	mu        sync.RWMutex

	// 清理定时器
	cleanupTimer *time.Ticker
	cleanupStop  chan struct{}
}

// New 创建新的令牌管理器
func New(cfg *Config, logger Logger) (*Manager, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 使用默认日志器
	if logger == nil {
		logger = &DefaultLogger{}
	}

	// 创建管理器
	m := &Manager{
		config:      cfg,
		validator:   newValidator(cfg, logger),
		blacklist:   newBlacklist(cfg.Redis.Client, cfg.Redis.BlacklistPrefix, logger),
		refresh:     newRefreshTokenManager(cfg.Redis.Client, cfg.Redis.KeyPrefix, logger),
		logger:      logger,
		cleanupStop: make(chan struct{}),
	}

	// 启动自动清理
	if cfg.Cleanup.Enabled {
		m.startCleanup()
	}

	logger.Info("Token manager created successfully")
	return m, nil
}

// InitGlobal 初始化全局令牌管理器
func InitGlobal(cfg *Config, rdb *redis.Client, logger Logger) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	// 设置 Redis 客户端
	if cfg.Redis.Client == nil {
		if rdb == nil {
			return ErrRedisClientRequired
		}
		cfg.Redis.Client = rdb
	}

	manager, err := New(cfg, logger)
	if err != nil {
		return err
	}

	globalManager = manager
	return nil
}

// GetGlobal 获取全局令牌管理器
func GetGlobal() *Manager {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalManager == nil {
		panic(ErrManagerNotInitialized)
	}

	return globalManager
}

// Close 关闭令牌管理器
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrManagerAlreadyClosed
	}

	// 停止清理定时器
	if m.cleanupTimer != nil {
		m.cleanupTimer.Stop()
		close(m.cleanupStop)
	}

	m.closed = true
	m.logger.Info("Token manager closed")
	return nil
}

// GenerateTokenPair 生成令牌对（访问令牌 + 刷新令牌）
func (m *Manager) GenerateTokenPair(ctx context.Context, userID string, customClaims map[string]interface{}) (*TokenPair, error) {
	return m.GenerateTokenPairWithDevice(ctx, userID, "", customClaims)
}

// GenerateTokenPairWithDevice 生成带设备信息的令牌对
func (m *Manager) GenerateTokenPairWithDevice(ctx context.Context, userID, deviceID string, customClaims map[string]interface{}) (*TokenPair, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	// 生成访问令牌
	accessToken, accessClaims, err := m.generateAccessToken(userID, deviceID, customClaims)
	if err != nil {
		return nil, err
	}

	// 生成刷新令牌
	refreshToken, refreshClaims, err := m.generateRefreshToken(userID, deviceID, customClaims)
	if err != nil {
		return nil, err
	}

	// 存储刷新令牌信息
	refreshInfo := &RefreshTokenInfo{
		TokenID:   refreshClaims.ID,
		UserID:    userID,
		DeviceID:  deviceID,
		ExpiresAt: refreshClaims.ExpiresAt.Time,
		CreatedAt: time.Now(),
	}

	if err := m.refresh.Store(ctx, refreshInfo); err != nil {
		return nil, err
	}

	// 更新设备信息
	if deviceID != "" {
		if err := m.updateDeviceInfo(ctx, userID, deviceID, customClaims); err != nil {
			m.logger.Warn("Failed to update device info", "deviceID", deviceID, "error", err)
		}
	}

	pair := &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    accessClaims.ExpiresAt.Time,
		ExpiresIn:    int64(time.Until(accessClaims.ExpiresAt.Time).Seconds()),
	}

	m.logger.Debug("Token pair generated", "userID", userID, "deviceID", deviceID)
	return pair, nil
}

// GenerateAccessToken 仅生成访问令牌
func (m *Manager) GenerateAccessToken(ctx context.Context, userID string, customClaims map[string]interface{}) (string, error) {
	if userID == "" {
		return "", ErrUserIDRequired
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return "", ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	token, _, err := m.generateAccessToken(userID, "", customClaims)
	return token, err
}

// GenerateRefreshToken 仅生成刷新令牌
func (m *Manager) GenerateRefreshToken(ctx context.Context, userID string, customClaims map[string]interface{}) (string, error) {
	if userID == "" {
		return "", ErrUserIDRequired
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return "", ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	token, _, err := m.generateRefreshToken(userID, "", customClaims)
	return token, err
}

// generateAccessToken 内部方法：生成访问令牌
func (m *Manager) generateAccessToken(userID, deviceID string, customClaims map[string]interface{}) (string, *Claims, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.AccessToken.Expiration)

	claims := &Claims{
		UserID:       userID,
		DeviceID:     deviceID,
		TokenType:    TokenTypeAccess.String(),
		CustomClaims: customClaims,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.AccessToken.Issuer,
			Subject:   userID,
			Audience:  m.config.AccessToken.Audience,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
	}

	token, err := m.validator.generateToken(claims)
	if err != nil {
		return "", nil, err
	}

	return token, claims, nil
}

// generateRefreshToken 内部方法：生成刷新令牌
func (m *Manager) generateRefreshToken(userID, deviceID string, customClaims map[string]interface{}) (string, *Claims, error) {
	now := time.Now()
	expiresAt := now.Add(m.config.RefreshToken.Expiration)

	claims := &Claims{
		UserID:       userID,
		DeviceID:     deviceID,
		TokenType:    TokenTypeRefresh.String(),
		CustomClaims: customClaims,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.RefreshToken.Issuer,
			Subject:   userID,
			Audience:  m.config.RefreshToken.Audience,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
	}

	token, err := m.validator.generateToken(claims)
	if err != nil {
		return "", nil, err
	}

	return token, claims, nil
}

// VerifyAccessToken 验证访问令牌
func (m *Manager) VerifyAccessToken(ctx context.Context, tokenStr string) (*Claims, error) {
	return m.verifyToken(ctx, tokenStr, TokenTypeAccess)
}

// VerifyRefreshToken 验证刷新令牌
func (m *Manager) VerifyRefreshToken(ctx context.Context, tokenStr string) (*Claims, error) {
	return m.verifyToken(ctx, tokenStr, TokenTypeRefresh)
}

// VerifyToken 验证令牌（自动判断类型）
func (m *Manager) VerifyToken(ctx context.Context, tokenStr string) (*Claims, error) {
	return m.verifyToken(ctx, tokenStr, "")
}

// verifyToken 内部方法：验证令牌
func (m *Manager) verifyToken(ctx context.Context, tokenStr string, expectedType TokenType) (*Claims, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrManagerAlreadyClosed
	}
	m.mu.RUnlock()

	// 解析令牌
	claims, err := m.validator.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}

	// 检查令牌类型
	if expectedType != "" && claims.TokenType != expectedType.String() {
		return nil, ErrTokenTypeMismatch
	}

	// 检查是否在黑名单中
	isBlacklisted, err := m.blacklist.IsBlacklisted(ctx, claims.ID)
	if err != nil {
		m.logger.Error("Failed to check blacklist", "tokenID", claims.ID, "error", err)
		return nil, err
	}
	if isBlacklisted {
		return nil, ErrTokenBlacklisted
	}

	// 如果是刷新令牌，检查是否在存储中
	if claims.IsRefreshToken() {
		exists, err := m.refresh.Exists(ctx, claims.ID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrTokenNotFound
		}
	}

	return claims, nil
}

// RefreshToken 刷新令牌（使用刷新令牌获取新的访问令牌）
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 验证刷新令牌
	claims, err := m.VerifyRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// 生成新的令牌对
	return m.GenerateTokenPairWithDevice(ctx, claims.UserID, claims.DeviceID, claims.CustomClaims)
}

// RefreshAndRevokeOld 刷新令牌并撤销旧令牌
func (m *Manager) RefreshAndRevokeOld(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 验证刷新令牌
	claims, err := m.VerifyRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// 撤销旧的刷新令牌
	if err := m.revokeRefreshToken(ctx, claims); err != nil {
		m.logger.Warn("Failed to revoke old refresh token", "tokenID", claims.ID, "error", err)
	}

	// 生成新的令牌对
	return m.GenerateTokenPairWithDevice(ctx, claims.UserID, claims.DeviceID, claims.CustomClaims)
}

// RevokeToken 撤销令牌
func (m *Manager) RevokeToken(ctx context.Context, tokenStr string) error {
	// 解析令牌
	claims, err := m.validator.parseToken(tokenStr)
	if err != nil {
		return err
	}

	// 根据类型撤销
	if claims.IsRefreshToken() {
		return m.revokeRefreshToken(ctx, claims)
	}
	return m.revokeAccessToken(ctx, claims)
}

// revokeAccessToken 撤销访问令牌
func (m *Manager) revokeAccessToken(ctx context.Context, claims *Claims) error {
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil // 已过期，无需撤销
	}

	entry := &BlacklistEntry{
		TokenID:   claims.ID,
		UserID:    claims.UserID,
		RevokedAt: time.Now(),
		TTL:       int64(ttl.Seconds()),
	}

	return m.blacklist.Add(ctx, entry.TokenID, entry.UserID, ttl)
}

// revokeRefreshToken 撤销刷新令牌
func (m *Manager) revokeRefreshToken(ctx context.Context, claims *Claims) error {
	// 删除刷新令牌
	if err := m.refresh.Delete(ctx, claims.ID); err != nil {
		return err
	}

	// 同时加入黑名单
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		entry := &BlacklistEntry{
			TokenID:   claims.ID,
			UserID:    claims.UserID,
			RevokedAt: time.Now(),
			TTL:       int64(ttl.Seconds()),
		}
		return m.blacklist.Add(ctx, entry.TokenID, entry.UserID, ttl)
	}

	return nil
}

// RevokeAllUserTokens 撤销用户的所有令牌
func (m *Manager) RevokeAllUserTokens(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrUserIDRequired
	}

	// 删除所有刷新令牌
	if err := m.refresh.DeleteByUser(ctx, userID); err != nil {
		m.logger.Error("Failed to delete user refresh tokens", "userID", userID, "error", err)
		return err
	}

	m.logger.Info("All user tokens revoked", "userID", userID)
	return nil
}

// RevokeDeviceToken 撤销特定设备的令牌
func (m *Manager) RevokeDeviceToken(ctx context.Context, userID, deviceID string) error {
	if userID == "" {
		return ErrUserIDRequired
	}
	if deviceID == "" {
		return ErrDeviceNotFound
	}

	// 删除设备的刷新令牌
	if err := m.refresh.DeleteByDevice(ctx, userID, deviceID); err != nil {
		m.logger.Error("Failed to delete device token", "userID", userID, "deviceID", deviceID, "error", err)
		return err
	}

	// 删除设备信息
	if err := m.deleteDeviceInfo(ctx, userID, deviceID); err != nil {
		m.logger.Warn("Failed to delete device info", "deviceID", deviceID, "error", err)
	}

	m.logger.Info("Device token revoked", "userID", userID, "deviceID", deviceID)
	return nil
}

// RevokeTokens 批量撤销令牌
func (m *Manager) RevokeTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	for _, tokenStr := range tokens {
		if err := m.RevokeToken(ctx, tokenStr); err != nil {
			m.logger.Warn("Failed to revoke token", "error", err)
			continue
		}
	}

	m.logger.Info("Tokens revoked in batch", "count", len(tokens))
	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (m *Manager) IsBlacklisted(ctx context.Context, tokenStr string) (bool, error) {
	claims, err := m.validator.parseToken(tokenStr)
	if err != nil {
		return false, err
	}

	return m.blacklist.IsBlacklisted(ctx, claims.ID)
}

// GetUserDevices 获取用户的所有设备
func (m *Manager) GetUserDevices(ctx context.Context, userID string) ([]*DeviceInfo, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	pattern := userDevicesPattern(m.config.Redis.KeyPrefix, userID)
	iter := m.config.Redis.Client.Scan(ctx, 0, pattern, 0).Iterator()

	var devices []*DeviceInfo

	for iter.Next(ctx) {
		data, err := m.config.Redis.Client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			m.logger.Warn("Failed to get device info", "key", iter.Val(), "error", err)
			continue
		}

		var device DeviceInfo
		if err := json.Unmarshal(data, &device); err != nil {
			m.logger.Warn("Failed to unmarshal device info", "key", iter.Val(), "error", err)
			continue
		}

		devices = append(devices, &device)
	}

	if err := iter.Err(); err != nil {
		m.logger.Error("Failed to scan user devices", "userID", userID, "error", err)
		return nil, err
	}

	return devices, nil
}

// GetDeviceInfo 获取设备信息
func (m *Manager) GetDeviceInfo(ctx context.Context, userID, deviceID string) (*DeviceInfo, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}
	if deviceID == "" {
		return nil, ErrDeviceNotFound
	}

	key := deviceInfoKey(m.config.Redis.KeyPrefix, userID, deviceID)
	data, err := m.config.Redis.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	var device DeviceInfo
	if err := json.Unmarshal(data, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

// UpdateDeviceActivity 更新设备最后活跃时间
func (m *Manager) UpdateDeviceActivity(ctx context.Context, userID, deviceID string) error {
	if userID == "" {
		return ErrUserIDRequired
	}
	if deviceID == "" {
		return ErrDeviceNotFound
	}

	device, err := m.GetDeviceInfo(ctx, userID, deviceID)
	if err != nil {
		return err
	}

	device.LastActive = time.Now()

	return m.updateDeviceInfo(ctx, userID, deviceID, device.CustomInfo)
}

// DeleteDevice 删除设备
func (m *Manager) DeleteDevice(ctx context.Context, userID, deviceID string) error {
	// 先删除设备的令牌
	if err := m.RevokeDeviceToken(ctx, userID, deviceID); err != nil {
		return err
	}

	return nil
}

// updateDeviceInfo 更新设备信息
func (m *Manager) updateDeviceInfo(ctx context.Context, userID, deviceID string, customInfo map[string]interface{}) error {
	key := deviceInfoKey(m.config.Redis.KeyPrefix, userID, deviceID)

	device := &DeviceInfo{
		DeviceID:   deviceID,
		UserID:     userID,
		LastActive: time.Now(),
		CreatedAt:  time.Now(),
		CustomInfo: customInfo,
	}

	data, err := json.Marshal(device)
	if err != nil {
		return err
	}

	// 设备信息的过期时间与刷新令牌相同
	ttl := m.config.RefreshToken.Expiration
	return m.config.Redis.Client.Set(ctx, key, data, ttl).Err()
}

// deleteDeviceInfo 删除设备信息
func (m *Manager) deleteDeviceInfo(ctx context.Context, userID, deviceID string) error {
	key := deviceInfoKey(m.config.Redis.KeyPrefix, userID, deviceID)
	return m.config.Redis.Client.Del(ctx, key).Err()
}

// CleanupExpiredTokens 清理过期的令牌
func (m *Manager) CleanupExpiredTokens(ctx context.Context) error {
	m.logger.Info("Starting token cleanup")

	count := 0

	// 清理过期的黑名单条目（Redis 会自动处理，这里主要是统计）
	blacklistCount, _ := m.blacklist.Count(ctx)
	m.logger.Debug("Blacklist entries", "count", blacklistCount)

	m.logger.Info("Token cleanup completed", "cleaned", count)
	return nil
}

// startCleanup 启动自动清理
func (m *Manager) startCleanup() {
	m.cleanupTimer = time.NewTicker(m.config.Cleanup.Interval)

	go func() {
		for {
			select {
			case <-m.cleanupTimer.C:
				ctx := context.Background()
				if err := m.CleanupExpiredTokens(ctx); err != nil {
					m.logger.Error("Failed to cleanup expired tokens", "error", err)
				}
			case <-m.cleanupStop:
				return
			}
		}
	}()

	m.logger.Info("Token cleanup started", "interval", m.config.Cleanup.Interval)
}

// GetConfig 获取配置（只读）
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Clone()
}

// ParseTokenWithoutValidation 解析令牌但不验证（用于调试）
func (m *Manager) ParseTokenWithoutValidation(tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	
	token, _, err := parser.ParseUnverified(tokenStr, &Claims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// GetTokenInfo 获取令牌信息（不验证有效性）
func (m *Manager) GetTokenInfo(tokenStr string) (map[string]interface{}, error) {
	claims, err := m.ParseTokenWithoutValidation(tokenStr)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"user_id":    claims.UserID,
		"device_id":  claims.DeviceID,
		"token_type": claims.TokenType,
		"token_id":   claims.ID,
		"issuer":     claims.Issuer,
		"subject":    claims.Subject,
		"audience":   claims.Audience,
		"expires_at": claims.ExpiresAt.Time,
		"issued_at":  claims.IssuedAt.Time,
		"not_before": claims.NotBefore.Time,
	}

	if claims.CustomClaims != nil {
		info["custom_claims"] = claims.CustomClaims
	}

	return info, nil
}

// ValidateTokenStructure 验证令牌结构（不验证签名和过期）
func (m *Manager) ValidateTokenStructure(tokenStr string) error {
	claims, err := m.ParseTokenWithoutValidation(tokenStr)
	if err != nil {
		return err
	}

	return claims.Validate()
}

// Stats 令牌管理器统计信息
type Stats struct {
	// BlacklistCount 黑名单数量
	BlacklistCount int64 `json:"blacklist_count"`
	
	// Uptime 运行时间
	Uptime time.Duration `json:"uptime"`
}

// GetStats 获取统计信息
func (m *Manager) GetStats(ctx context.Context) (*Stats, error) {
	blacklistCount, err := m.blacklist.Count(ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		BlacklistCount: blacklistCount,
	}, nil
}

// Health 健康检查
func (m *Manager) Health(ctx context.Context) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return fmt.Errorf("manager is closed")
	}
	m.mu.RUnlock()

	// 检查 Redis 连接
	if err := m.config.Redis.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

