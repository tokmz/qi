package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tokmz/qi"
)

// CORSConfig CORS 中间件配置
type CORSConfig struct {
	// AllowOrigins 允许的源列表（默认 ["*"]）
	// 支持精确匹配和通配符，如 "https://*.example.com"
	AllowOrigins []string

	// AllowMethods 允许的 HTTP 方法（默认 GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS）
	AllowMethods []string

	// AllowHeaders 允许的请求头（默认 Origin, Content-Type, Accept, Authorization）
	AllowHeaders []string

	// ExposeHeaders 允许前端访问的响应头
	ExposeHeaders []string

	// AllowCredentials 是否允许携带凭证（Cookie 等）
	// 注意：为 true 时 AllowOrigins 不能为 ["*"]
	AllowCredentials bool

	// MaxAge 预检请求缓存时间（默认 12 小时）
	MaxAge time.Duration
}

// DefaultCORSConfig 返回默认配置（允许所有源）
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders:    nil,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// CORS 创建 CORS 中间件
func CORS(cfgs ...*CORSConfig) qi.HandlerFunc {
	cfg := DefaultCORSConfig()
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}

	// 预计算
	allowAllOrigins := len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*"
	allowMethods := strings.Join(cfg.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(int(cfg.MaxAge.Seconds()))

	// AllowCredentials 与 AllowOrigins: ["*"] 冲突校验
	if cfg.AllowCredentials && allowAllOrigins {
		panic("github.com/tokmz/qi/middleware: CORS AllowCredentials cannot be used with AllowOrigins [\"*\"]")
	}

	// 构建通配符和精确匹配集合
	var wildcardOrigins []string
	exactOrigins := make(map[string]bool)
	if !allowAllOrigins {
		for _, origin := range cfg.AllowOrigins {
			if strings.Contains(origin, "*") {
				wildcardOrigins = append(wildcardOrigins, origin)
			} else {
				exactOrigins[origin] = true
			}
		}
	}

	return func(c *qi.Context) {
		origin := c.GetHeader("Origin")

		// 无 Origin 头，非跨域请求
		if origin == "" {
			c.Next()
			return
		}

		// 检查 Origin 是否允许
		var allowed bool
		if allowAllOrigins {
			allowed = true
		} else {
			allowed = matchOrigin(origin, exactOrigins, wildcardOrigins)
		}

		if !allowed {
			c.Next()
			return
		}

		// 设置 CORS 响应头
		if allowAllOrigins {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if exposeHeaders != "" {
			c.Header("Access-Control-Expose-Headers", exposeHeaders)
		}

		// 预检请求
		if c.Request().Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)
			c.Header("Access-Control-Max-Age", maxAge)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// matchOrigin 检查 origin 是否匹配
func matchOrigin(origin string, exact map[string]bool, wildcards []string) bool {
	// 精确匹配
	if exact[origin] {
		return true
	}

	// 通配符匹配
	for _, pattern := range wildcards {
		if matchWildcard(origin, pattern) {
			return true
		}
	}

	return false
}

// matchWildcard 通配符匹配
// 支持 "https://*.example.com" 格式
func matchWildcard(origin, pattern string) bool {
	// 按 * 分割
	parts := strings.SplitN(pattern, "*", 2)
	if len(parts) != 2 {
		return origin == pattern
	}

	prefix := parts[0]
	suffix := parts[1]

	if !strings.HasPrefix(origin, prefix) {
		return false
	}
	if !strings.HasSuffix(origin, suffix) {
		return false
	}
	// 确保中间部分不为空
	return len(origin) > len(prefix)+len(suffix)
}
