# Token - JWT 令牌管理

基于 JWT v5 和 Redis v9 的令牌管理包，提供完整的 Access Token 和 Refresh Token 双令牌机制，支持令牌黑名单、自动刷新、并发安全的令牌管理功能。

## 功能特性

### 核心特性

- ✅ **双令牌机制**: Access Token + Refresh Token，安全性更高
- ✅ **JWT v5**: 基于最新的 golang-jwt/jwt/v5 实现
- ✅ **Redis v9**: 使用 Redis 存储刷新令牌和黑名单
- ✅ **令牌黑名单**: 支持令牌撤销和注销
- ✅ **自动刷新**: 基于 Refresh Token 自动刷新 Access Token
- ✅ **并发安全**: 所有操作都是线程安全的
- ✅ **灵活配置**: 支持自定义过期时间、签名算法等
- ✅ **Claims 扩展**: 支持自定义 Claims 字段
- ✅ **多签名算法**: 支持 HS256、HS384、HS512、RS256、RS384、RS512
- ✅ **类型安全**: 完整的类型定义和错误处理
- ✅ **完整日志**: 详细的日志记录
- ✅ **高质量代码**: 清晰的结构，完整的中文注释

### 技术栈

- [JWT v5](https://github.com/golang-jwt/jwt) - JSON Web Token 实现
- [Redis v9](https://github.com/redis/go-redis) - Redis 客户端

## 目录结构

```
token/
├── config.go          # 配置定义
├── errors.go          # 错误定义
├── types.go           # 类型定义
├── manager.go         # 令牌管理器核心实现
├── claims.go          # Claims 定义和处理
├── blacklist.go       # 黑名单管理
├── refresh.go         # 刷新令牌管理
├── logger.go          # 日志接口和实现
├── helper.go          # 辅助函数
├── validator.go       # 令牌验证器
├── example_test.go    # 示例代码
└── README.md          # 文档（本文件）
```

## 快速开始

### 1. 基础使用

```go
package main

import (
    "context"
    "log"
    "time"
    "qi/internal/core/token"
    "github.com/redis/go-redis/v9"
)

func main() {
    // 创建 Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
    })

    // 创建配置
    cfg := &token.Config{
        // 签名密钥
        SecretKey: "your-secret-key-min-32-chars-long",
        
        // Access Token 配置
        AccessToken: token.TokenConfig{
            Expiration: 15 * time.Minute,  // 15分钟过期
            Issuer:     "qi-service",
        },
        
        // Refresh Token 配置
        RefreshToken: token.TokenConfig{
            Expiration: 7 * 24 * time.Hour,  // 7天过期
            Issuer:     "qi-service",
        },
        
        // 签名算法
        SigningMethod: token.SigningMethodHS256,
        
        // Redis 配置
        Redis: token.RedisConfig{
            Client:          rdb,
            KeyPrefix:       "token:",
            BlacklistPrefix: "blacklist:",
        },
    }

    // 创建令牌管理器
    logger := &token.DefaultLogger{}
    manager, err := token.New(cfg, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    // 生成令牌对
    ctx := context.Background()
    userID := "user-123"
    
    pair, err := manager.GenerateTokenPair(ctx, userID, map[string]interface{}{
        "username": "alice",
        "role":     "admin",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Access Token: %s", pair.AccessToken)
    log.Printf("Refresh Token: %s", pair.RefreshToken)
    log.Printf("Expires At: %v", pair.ExpiresAt)

    // 验证 Access Token
    claims, err := manager.VerifyAccessToken(ctx, pair.AccessToken)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User ID: %s", claims.UserID)
    log.Printf("Username: %s", claims.Get("username"))

    // 刷新令牌
    newPair, err := manager.RefreshToken(ctx, pair.RefreshToken)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("New Access Token: %s", newPair.AccessToken)

    // 撤销令牌（登出）
    err = manager.RevokeToken(ctx, pair.AccessToken)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. 多设备管理

```go
// 为每个设备生成独立的令牌
deviceID := "device-abc123"

pair, err := manager.GenerateTokenPairWithDevice(ctx, userID, deviceID, map[string]interface{}{
    "username": "alice",
    "device":   "iOS 16.0",
})

// 获取用户所有设备的令牌
devices, err := manager.GetUserDevices(ctx, userID)
for _, device := range devices {
    log.Printf("Device: %s, Last Active: %v", device.DeviceID, device.LastActive)
}

// 撤销特定设备的令牌
err = manager.RevokeDeviceToken(ctx, userID, deviceID)

// 撤销用户所有设备的令牌（强制全部登出）
err = manager.RevokeAllUserTokens(ctx, userID)
```

### 3. 使用 RSA 签名

```go
// 生成 RSA 密钥对
privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
publicKey := &privateKey.PublicKey

// 配置
cfg := &token.Config{
    SigningMethod: token.SigningMethodRS256,
    RSAPrivateKey: privateKey,
    RSAPublicKey:  publicKey,
    // ... 其他配置
}

manager, err := token.New(cfg, logger)
```

### 4. 自定义 Claims

```go
// 生成令牌时添加自定义字段
pair, err := manager.GenerateTokenPair(ctx, userID, map[string]interface{}{
    "username":    "alice",
    "email":       "alice@example.com",
    "role":        "admin",
    "department":  "IT",
    "permissions": []string{"read", "write", "delete"},
})

// 验证后读取自定义字段
claims, err := manager.VerifyAccessToken(ctx, pair.AccessToken)

username := claims.Get("username").(string)
email := claims.Get("email").(string)
permissions := claims.Get("permissions").([]interface{})
```

## 配置说明

### 完整配置示例

```yaml
token:
  # 签名密钥（HMAC 算法使用）
  secret_key: "your-secret-key-at-least-32-characters-long-for-security"
  
  # Access Token 配置
  access_token:
    expiration: 15m        # 过期时间
    issuer: "qi-service"   # 签发者
    audience: ["web", "mobile"]  # 受众
  
  # Refresh Token 配置
  refresh_token:
    expiration: 168h       # 7天
    issuer: "qi-service"
    audience: ["web", "mobile"]
  
  # 签名算法: HS256, HS384, HS512, RS256, RS384, RS512
  signing_method: HS256
  
  # RSA 密钥文件路径（RS* 算法使用）
  rsa_private_key_file: "configs/keys/private.pem"
  rsa_public_key_file: "configs/keys/public.pem"
  
  # Redis 配置
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
    key_prefix: "token:"
    blacklist_prefix: "blacklist:"
    
  # 令牌清理配置
  cleanup:
    enabled: true
    interval: 1h           # 清理间隔
    batch_size: 1000       # 批量清理大小
```

### Config 结构体

```go
type Config struct {
    // 签名密钥（HMAC 算法）
    SecretKey string
    
    // RSA 私钥（RS* 算法）
    RSAPrivateKey *rsa.PrivateKey
    
    // RSA 公钥（RS* 算法）
    RSAPublicKey *rsa.PublicKey
    
    // Access Token 配置
    AccessToken TokenConfig
    
    // Refresh Token 配置
    RefreshToken TokenConfig
    
    // 签名算法
    SigningMethod SigningMethod
    
    // Redis 配置
    Redis RedisConfig
    
    // 清理配置
    Cleanup CleanupConfig
}

type TokenConfig struct {
    // 过期时间
    Expiration time.Duration
    
    // 签发者
    Issuer string
    
    // 受众
    Audience []string
    
    // 主题
    Subject string
}

type RedisConfig struct {
    // Redis 客户端
    Client *redis.Client
    
    // 键前缀
    KeyPrefix string
    
    // 黑名单前缀
    BlacklistPrefix string
}

type CleanupConfig struct {
    // 是否启用自动清理
    Enabled bool
    
    // 清理间隔
    Interval time.Duration
    
    // 批量清理大小
    BatchSize int
}
```

## API 文档

### 令牌管理器

```go
// 创建令牌管理器
New(cfg *Config, logger Logger) (*Manager, error)

// 初始化全局令牌管理器
InitGlobal(cfg *Config, rdb *redis.Client, logger Logger) error

// 获取全局令牌管理器
GetGlobal() *Manager

// 关闭令牌管理器
Close() error
```

### 令牌生成

```go
// 生成令牌对（Access Token + Refresh Token）
GenerateTokenPair(ctx context.Context, userID string, customClaims map[string]interface{}) (*TokenPair, error)

// 生成带设备信息的令牌对
GenerateTokenPairWithDevice(ctx context.Context, userID, deviceID string, customClaims map[string]interface{}) (*TokenPair, error)

// 仅生成 Access Token
GenerateAccessToken(ctx context.Context, userID string, customClaims map[string]interface{}) (string, error)

// 仅生成 Refresh Token
GenerateRefreshToken(ctx context.Context, userID string, customClaims map[string]interface{}) (string, error)
```

### 令牌验证

```go
// 验证 Access Token
VerifyAccessToken(ctx context.Context, tokenStr string) (*Claims, error)

// 验证 Refresh Token
VerifyRefreshToken(ctx context.Context, tokenStr string) (*Claims, error)

// 验证令牌（自动判断类型）
VerifyToken(ctx context.Context, tokenStr string) (*Claims, error)

// 检查令牌是否在黑名单
IsBlacklisted(ctx context.Context, tokenStr string) (bool, error)
```

### 令牌刷新

```go
// 刷新令牌（使用 Refresh Token 获取新的 Access Token）
RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

// 刷新并撤销旧令牌
RefreshAndRevokeOld(ctx context.Context, refreshToken string) (*TokenPair, error)
```

### 令牌撤销

```go
// 撤销单个令牌
RevokeToken(ctx context.Context, tokenStr string) error

// 撤销用户的所有令牌
RevokeAllUserTokens(ctx context.Context, userID string) error

// 撤销特定设备的令牌
RevokeDeviceToken(ctx context.Context, userID, deviceID string) error

// 批量撤销令牌
RevokeTokens(ctx context.Context, tokens []string) error
```

### 设备管理

```go
// 获取用户所有设备
GetUserDevices(ctx context.Context, userID string) ([]*DeviceInfo, error)

// 获取设备信息
GetDeviceInfo(ctx context.Context, userID, deviceID string) (*DeviceInfo, error)

// 更新设备最后活跃时间
UpdateDeviceActivity(ctx context.Context, userID, deviceID string) error

// 删除设备
DeleteDevice(ctx context.Context, userID, deviceID string) error
```

### Claims 操作

```go
// 从 Context 获取 Claims
GetClaimsFromContext(ctx context.Context) (*Claims, bool)

// 将 Claims 设置到 Context
SetClaimsToContext(ctx context.Context, claims *Claims) context.Context
```

### 辅助方法

```go
// 获取配置（只读）
GetConfig() *Config

// 解析令牌但不验证（用于调试）
ParseTokenWithoutValidation(tokenStr string) (*Claims, error)

// 获取令牌信息（包含所有字段）
GetTokenInfo(tokenStr string) (map[string]interface{}, error)

// 验证令牌结构是否合法
ValidateTokenStructure(tokenStr string) error

// 清理过期令牌
CleanupExpiredTokens(ctx context.Context) error

// 获取统计信息
GetStats(ctx context.Context) (*Stats, error)

// 健康检查
Health(ctx context.Context) error
```

### Claims 结构

```go
type Claims struct {
    // 用户ID
    UserID string `json:"user_id"`
    
    // 设备ID
    DeviceID string `json:"device_id,omitempty"`
    
    // 令牌类型（access/refresh）
    TokenType string `json:"token_type"`
    
    // 令牌ID（用于撤销）
    TokenID string `json:"jti"`
    
    // 自定义字段
    CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`
    
    // JWT 标准字段
    jwt.RegisteredClaims
}

// 创建 Claims
NewClaims(userID string, tokenType TokenType, expiration int64, issuer string, audience []string) *Claims

// 获取自定义字段
Get(key string) interface{}

// 设置自定义字段
Set(key string, value interface{})

// 获取字符串字段
GetString(key string) string

// 获取整数字段
GetInt(key string) int

// 获取 int64 字段
GetInt64(key string) int64

// 获取布尔字段
GetBool(key string) bool

// 获取浮点数字段
GetFloat64(key string) float64

// 获取字符串切片字段
GetStringSlice(key string) []string

// 验证 Claims
Validate() error

// 判断是否为访问令牌
IsAccessToken() bool

// 判断是否为刷新令牌
IsRefreshToken() bool
```

### TokenPair 结构

```go
type TokenPair struct {
    // Access Token
    AccessToken string `json:"access_token"`
    
    // Refresh Token
    RefreshToken string `json:"refresh_token"`
    
    // 令牌类型
    TokenType string `json:"token_type"`
    
    // 过期时间
    ExpiresAt time.Time `json:"expires_at"`
    
    // 过期秒数
    ExpiresIn int64 `json:"expires_in"`
}
```

### 辅助类型

```go
// 签名方法
type SigningMethod string

const (
    SigningMethodHS256 SigningMethod = "HS256"  // HMAC SHA256
    SigningMethodHS384 SigningMethod = "HS384"  // HMAC SHA384
    SigningMethodHS512 SigningMethod = "HS512"  // HMAC SHA512
    SigningMethodRS256 SigningMethod = "RS256"  // RSA SHA256
    SigningMethodRS384 SigningMethod = "RS384"  // RSA SHA384
    SigningMethodRS512 SigningMethod = "RS512"  // RSA SHA512
)

// 令牌类型
type TokenType string

const (
    TokenTypeAccess  TokenType = "access"   // 访问令牌
    TokenTypeRefresh TokenType = "refresh"  // 刷新令牌
)

// 设备信息
type DeviceInfo struct {
    DeviceID   string                 `json:"device_id"`
    UserID     string                 `json:"user_id"`
    LastActive time.Time              `json:"last_active"`
    CreatedAt  time.Time              `json:"created_at"`
    CustomInfo map[string]interface{} `json:"custom_info,omitempty"`
}

// 刷新令牌信息
type RefreshTokenInfo struct {
    TokenID   string    `json:"token_id"`
    UserID    string    `json:"user_id"`
    DeviceID  string    `json:"device_id,omitempty"`
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
}

// 黑名单条目
type BlacklistEntry struct {
    TokenID   string    `json:"token_id"`
    UserID    string    `json:"user_id"`
    RevokedAt time.Time `json:"revoked_at"`
    TTL       int64     `json:"ttl"`
}

// 统计信息
type Stats struct {
    BlacklistCount int64         `json:"blacklist_count"`
    Uptime         time.Duration `json:"uptime"`
}
```

### Logger 接口

```go
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}

// 提供的实现
// - NoopLogger: 无操作日志（默认）
// - DefaultLogger: 标准输出日志
// - DebugLogger: 调试日志
```

### 错误定义

```go
var (
    // 配置错误
    ErrInvalidConfig       = errors.New("invalid token config")
    ErrSecretKeyTooShort   = errors.New("secret key must be at least 32 characters")
    ErrRSAKeyRequired      = errors.New("RSA private and public keys are required for RS* algorithms")
    ErrRedisClientRequired = errors.New("redis client is required")
    
    // 令牌验证错误
    ErrTokenExpired          = errors.New("token has expired")
    ErrTokenInvalid          = errors.New("invalid token")
    ErrTokenBlacklisted      = errors.New("token has been revoked")
    ErrTokenNotFound         = errors.New("token not found")
    ErrTokenTypeMismatch     = errors.New("token type mismatch")
    ErrInvalidTokenType      = errors.New("invalid token type")
    ErrSigningMethodMismatch = errors.New("signing method mismatch")
    
    // 业务错误
    ErrRefreshTokenRequired = errors.New("refresh token is required")
    ErrAccessTokenRequired  = errors.New("access token is required")
    ErrUserIDRequired       = errors.New("user ID is required")
    ErrDeviceNotFound       = errors.New("device not found")
    ErrInvalidClaims        = errors.New("invalid claims")
    
    // 管理器状态错误
    ErrManagerNotInitialized = errors.New("token manager not initialized")
    ErrManagerAlreadyClosed  = errors.New("token manager already closed")
)
```

## 最佳实践

### 1. 密钥安全

```go
// ✅ 推荐：从环境变量读取
secretKey := os.Getenv("JWT_SECRET_KEY")
if len(secretKey) < 32 {
    log.Fatal("Secret key must be at least 32 characters")
}

// ✅ 推荐：使用 RSA 密钥对（生产环境）
// 生成密钥：
// openssl genrsa -out private.pem 2048
// openssl rsa -in private.pem -pubout -out public.pem

// ❌ 不推荐：硬编码密钥
secretKey := "hardcoded-secret"  // 不安全！
```

### 2. 令牌过期时间

```go
// Access Token: 短期有效（15分钟 - 1小时）
cfg.AccessToken.Expiration = 15 * time.Minute

// Refresh Token: 长期有效（7天 - 30天）
cfg.RefreshToken.Expiration = 7 * 24 * time.Hour

// 敏感操作可以使用更短的过期时间
cfg.AccessToken.Expiration = 5 * time.Minute  // 5分钟
```

### 3. 令牌刷新策略

```go
// 策略1: 滑动窗口（推荐）
// Access Token 过期前可以用 Refresh Token 刷新
// 刷新后旧的 Refresh Token 仍然有效

pair, err := manager.RefreshToken(ctx, refreshToken)

// 策略2: 一次性刷新
// 刷新后旧的 Refresh Token 立即失效
pair, err := manager.RefreshAndRevokeOld(ctx, refreshToken)
```

### 4. 登出处理

```go
// 单设备登出
func logout(ctx context.Context, accessToken string) error {
    return manager.RevokeToken(ctx, accessToken)
}

// 全设备登出
func logoutAll(ctx context.Context, userID string) error {
    return manager.RevokeAllUserTokens(ctx, userID)
}

// 其他设备登出
func logoutOtherDevices(ctx context.Context, userID, currentDeviceID string) error {
    devices, err := manager.GetUserDevices(ctx, userID)
    if err != nil {
        return err
    }
    
    for _, device := range devices {
        if device.DeviceID != currentDeviceID {
            manager.RevokeDeviceToken(ctx, userID, device.DeviceID)
        }
    }
    return nil
}
```

### 5. 错误处理

```go
claims, err := manager.VerifyAccessToken(ctx, tokenStr)
if err != nil {
    switch {
    case errors.Is(err, token.ErrTokenExpired):
        // 令牌过期，提示刷新
        return errors.New("token expired, please refresh")
        
    case errors.Is(err, token.ErrTokenInvalid):
        // 令牌无效
        return errors.New("invalid token")
        
    case errors.Is(err, token.ErrTokenBlacklisted):
        // 令牌已撤销
        return errors.New("token revoked")
        
    case errors.Is(err, token.ErrTokenNotFound):
        // 令牌不存在
        return errors.New("token not found")
        
    default:
        // 其他错误
        return fmt.Errorf("token verification failed: %w", err)
    }
}
```

### 6. Redis 连接池优化

```go
rdb := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     10,              // 连接池大小
    MinIdleConns: 5,               // 最小空闲连接
    MaxRetries:   3,               // 最大重试次数
    DialTimeout:  5 * time.Second, // 连接超时
    ReadTimeout:  3 * time.Second, // 读取超时
    WriteTimeout: 3 * time.Second, // 写入超时
})
```

### 7. 并发安全

```go
// 令牌管理器是并发安全的
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        userID := fmt.Sprintf("user-%d", id)
        pair, _ := manager.GenerateTokenPair(ctx, userID, nil)
        claims, _ := manager.VerifyAccessToken(ctx, pair.AccessToken)
        // ...
    }(i)
}
wg.Wait()
```

### 8. 自定义 Claims 验证

```go
// 验证自定义字段
claims, err := manager.VerifyAccessToken(ctx, tokenStr)
if err != nil {
    return err
}

// 检查角色
role := claims.GetString("role")
if role != "admin" {
    return errors.New("insufficient permissions")
}

// 检查权限
permissions := claims.Get("permissions").([]interface{})
hasPermission := false
for _, p := range permissions {
    if p.(string) == "write" {
        hasPermission = true
        break
    }
}
if !hasPermission {
    return errors.New("no write permission")
}
```

## 使用场景

### 1. Web 应用认证

```go
// 用户登录
func login(username, password string) (*TokenPair, error) {
    // 验证用户名密码
    user := authenticateUser(username, password)
    
    // 生成令牌
    return manager.GenerateTokenPair(ctx, user.ID, map[string]interface{}{
        "username": user.Username,
        "email":    user.Email,
        "role":     user.Role,
    })
}

// API 请求
func apiRequest(accessToken string) error {
    claims, err := manager.VerifyAccessToken(ctx, accessToken)
    if err != nil {
        return err
    }
    
    // 使用 claims 中的用户信息
    userID := claims.UserID
    // ...
    return nil
}
```

### 2. 移动应用多设备管理

```go
// 移动端登录
func mobileLogin(username, password, deviceID, deviceInfo string) (*TokenPair, error) {
    user := authenticateUser(username, password)
    
    return manager.GenerateTokenPairWithDevice(ctx, user.ID, deviceID, map[string]interface{}{
        "username":    user.Username,
        "device_info": deviceInfo,
        "platform":    "iOS",
    })
}

// 查看登录设备列表
func listDevices(userID string) ([]*DeviceInfo, error) {
    return manager.GetUserDevices(ctx, userID)
}

// 踢出特定设备
func kickDevice(userID, deviceID string) error {
    return manager.RevokeDeviceToken(ctx, userID, deviceID)
}
```

### 3. 微服务间认证

```go
// Service A: 生成服务间调用令牌
func generateServiceToken(serviceID string) (string, error) {
    return manager.GenerateAccessToken(ctx, serviceID, map[string]interface{}{
        "service": "service-a",
        "role":    "internal",
    })
}

// Service B: 验证服务间调用令牌
func verifyServiceToken(tokenStr string) error {
    claims, err := manager.VerifyAccessToken(ctx, tokenStr)
    if err != nil {
        return err
    }
    
    if claims.GetString("role") != "internal" {
        return errors.New("not a service token")
    }
    
    return nil
}
```

### 4. SSO 单点登录

```go
// 中央认证服务
func ssoLogin(username, password string) (*TokenPair, error) {
    user := authenticateUser(username, password)
    
    // 生成可跨服务使用的令牌
    return manager.GenerateTokenPair(ctx, user.ID, map[string]interface{}{
        "username": user.Username,
        "tenant":   user.TenantID,
        "services": []string{"service-a", "service-b", "service-c"},
    })
}

// 各个子服务验证
func verifyInService(tokenStr string) (*Claims, error) {
    claims, err := manager.VerifyAccessToken(ctx, tokenStr)
    if err != nil {
        return nil, err
    }
    
    // 检查是否有权访问当前服务
    services := claims.Get("services").([]interface{})
    // ... 验证逻辑
    
    return claims, nil
}
```

## 性能优化

### 1. Redis 优化

```go
// 使用 Pipeline 批量操作
func revokeMultipleTokens(tokens []string) error {
    pipe := rdb.Pipeline()
    
    for _, token := range tokens {
        pipe.Set(ctx, "blacklist:"+token, "1", 24*time.Hour)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}

// 使用 Scan 代替 Keys
func getAllUserTokens(userID string) ([]string, error) {
    var tokens []string
    iter := rdb.Scan(ctx, 0, "token:"+userID+":*", 100).Iterator()
    for iter.Next(ctx) {
        tokens = append(tokens, iter.Val())
    }
    return tokens, iter.Err()
}
```

### 2. 令牌缓存

```go
// 在内存中缓存验证结果（短时间）
type CachedManager struct {
    *Manager
    cache *sync.Map
}

func (cm *CachedManager) VerifyAccessToken(ctx context.Context, tokenStr string) (*Claims, error) {
    // 先查缓存
    if val, ok := cm.cache.Load(tokenStr); ok {
        cached := val.(*cacheItem)
        if time.Now().Before(cached.expireAt) {
            return cached.claims, nil
        }
        cm.cache.Delete(tokenStr)
    }
    
    // 缓存未命中，验证令牌
    claims, err := cm.Manager.VerifyAccessToken(ctx, tokenStr)
    if err != nil {
        return nil, err
    }
    
    // 缓存结果（1分钟）
    cm.cache.Store(tokenStr, &cacheItem{
        claims:   claims,
        expireAt: time.Now().Add(1 * time.Minute),
    })
    
    return claims, nil
}
```

### 3. 定期清理过期令牌

```go
// 启用自动清理
cfg.Cleanup.Enabled = true
cfg.Cleanup.Interval = 1 * time.Hour
cfg.Cleanup.BatchSize = 1000

// 手动清理
func cleanupExpiredTokens() error {
    return manager.CleanupExpiredTokens(ctx)
}
```

## 安全建议

### 1. HTTPS Only

```go
// 生产环境必须使用 HTTPS
// 令牌应该通过 HTTPS 传输，防止中间人攻击
// 如果使用 Cookie 存储，应启用 Secure 和 HttpOnly 标志
```

### 2. CSRF 防护

```go
// 在令牌中包含 CSRF token
pair, err := manager.GenerateTokenPair(ctx, userID, map[string]interface{}{
    "csrf_token": generateCSRFToken(),
})

// 验证时检查 CSRF token
claims, _ := manager.VerifyAccessToken(ctx, tokenStr)
if claims.GetString("csrf_token") != requestCSRFToken {
    return errors.New("CSRF token mismatch")
}
```

### 3. IP 绑定（可选）

```go
// 生成令牌时记录 IP
clientIP := getClientIP()  // 从请求中获取客户端 IP
pair, err := manager.GenerateTokenPair(ctx, userID, map[string]interface{}{
    "ip": clientIP,
})

// 验证时检查 IP
claims, _ := manager.VerifyAccessToken(ctx, tokenStr)
currentIP := getClientIP()
if claims.GetString("ip") != currentIP {
    return errors.New("IP address changed")
}
```

### 4. 限流

```go
// 限制令牌生成频率
func rateLimitTokenGeneration(userID string) error {
    key := "rate:token:gen:" + userID
    count, _ := rdb.Incr(ctx, key).Result()
    
    if count == 1 {
        rdb.Expire(ctx, key, 1*time.Minute)
    }
    
    if count > 10 {  // 每分钟最多10次
        return errors.New("rate limit exceeded")
    }
    
    return nil
}
```

## 故障排查

### 1. 令牌验证失败

```go
// 启用详细日志
logger := &token.DebugLogger{}
manager, _ := token.New(cfg, logger)

// 检查令牌格式
parts := strings.Split(tokenStr, ".")
if len(parts) != 3 {
    log.Println("Invalid token format")
}

// 检查签名算法
// 确保生成和验证使用相同的算法
```

### 2. Redis 连接问题

```go
// 测试 Redis 连接
if err := rdb.Ping(ctx).Err(); err != nil {
    log.Fatal("Redis connection failed:", err)
}

// 检查 Redis 键
keys, _ := rdb.Keys(ctx, "token:*").Result()
log.Printf("Token keys count: %d", len(keys))
```

### 3. 令牌过期时间不正确

```go
// 检查系统时间
log.Printf("Server time: %v", time.Now())

// 检查令牌的过期时间
claims, _ := manager.VerifyAccessToken(ctx, tokenStr)
log.Printf("Token expires at: %v", claims.ExpiresAt)
log.Printf("Time until expiry: %v", time.Until(claims.ExpiresAt.Time))
```

## 依赖

在 `go.mod` 中添加：

```go
require (
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/redis/go-redis/v9 v9.7.0
    github.com/google/uuid v1.6.0
)
```

## 参考资源

- [JWT 官方网站](https://jwt.io/)
- [RFC 7519 - JWT 规范](https://datatracker.ietf.org/doc/html/rfc7519)
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- [Redis Go Client](https://github.com/redis/go-redis)
- [OWASP JWT 安全最佳实践](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)

## 许可证

MIT License

