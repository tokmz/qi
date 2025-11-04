package authz

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	// Skipper 定义跳过中间件的函数
	Skipper func(c *gin.Context) bool

	// 未授权时的响应状态码
	UnauthorizedCode int

	// 禁止访问时的响应状态码
	ForbiddenCode int

	// 租户ID提取函数（多租户模式）
	TenantExtractor func(c *gin.Context) string

	// 用户ID提取函数
	UserExtractor func(c *gin.Context) string

	// 资源提取函数（默认使用请求路径）
	ResourceExtractor func(c *gin.Context) string

	// 操作提取函数（默认使用请求方法）
	ActionExtractor func(c *gin.Context) string

	// 错误处理函数
	ErrorHandler func(c *gin.Context, err error)
}

// DefaultMiddlewareConfig 返回默认中间件配置
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		Skipper: func(c *gin.Context) bool {
			return false
		},
		UnauthorizedCode: 401,
		ForbiddenCode:    403,
		TenantExtractor: func(c *gin.Context) string {
			// 默认从 JWT claims 中获取
			if tenantID, exists := c.Get("tenant_id"); exists {
				if tid, ok := tenantID.(string); ok {
					return tid
				}
			}
			// 或从请求头获取
			return c.GetHeader("X-Tenant-ID")
		},
		UserExtractor: func(c *gin.Context) string {
			// 默认从 JWT claims 中获取
			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok {
					return uid
				}
			}
			// 或从请求头获取
			return c.GetHeader("X-User-ID")
		},
		ResourceExtractor: func(c *gin.Context) string {
			return c.Request.URL.Path
		},
		ActionExtractor: func(c *gin.Context) string {
			return c.Request.Method
		},
		ErrorHandler: func(c *gin.Context, err error) {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error",
				"error":   err.Error(),
			})
		},
	}
}

// Middleware 创建权限检查中间件
func Middleware(manager *Manager, config ...*MiddlewareConfig) gin.HandlerFunc {
	cfg := DefaultMiddlewareConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		// 跳过检查
		if cfg.Skipper != nil && cfg.Skipper(c) {
			c.Next()
			return
		}

		// 提取参数
		userID := cfg.UserExtractor(c)
		resource := cfg.ResourceExtractor(c)
		action := cfg.ActionExtractor(c)

		// 检查用户ID
		if userID == "" {
			c.JSON(cfg.UnauthorizedCode, gin.H{
				"code":    cfg.UnauthorizedCode,
				"message": "unauthorized: missing user id",
			})
			c.Abort()
			return
		}

		var allowed bool
		var err error

		// 根据模式检查权限
		if manager.GetMode() == ModeSingle {
			allowed, err = manager.Single().CheckPermission(userID, resource, action)
		} else {
			tenantID := cfg.TenantExtractor(c)
			if tenantID == "" {
				c.JSON(cfg.UnauthorizedCode, gin.H{
					"code":    cfg.UnauthorizedCode,
					"message": "unauthorized: missing tenant id",
				})
				c.Abort()
				return
			}
			allowed, err = manager.Multi().CheckPermission(tenantID, userID, resource, action)
		}

		if err != nil {
			if cfg.ErrorHandler != nil {
				cfg.ErrorHandler(c, err)
			}
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(cfg.ForbiddenCode, gin.H{
				"code":    cfg.ForbiddenCode,
				"message": "forbidden: permission denied",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GlobalMiddleware 使用全局管理器创建中间件
func GlobalMiddleware(config ...*MiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := GetGlobal()
		if manager == nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error: enforcer not initialized",
			})
			c.Abort()
			return
		}

		Middleware(manager, config...)(c)
	}
}

// SkipperFunc 跳过检查的辅助函数

// SkipPrefixes 跳过指定前缀的路径
func SkipPrefixes(prefixes ...string) func(c *gin.Context) bool {
	return func(c *gin.Context) bool {
		path := c.Request.URL.Path
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		return false
	}
}

// SkipPaths 跳过指定的路径
func SkipPaths(paths ...string) func(c *gin.Context) bool {
	pathMap := make(map[string]bool)
	for _, p := range paths {
		pathMap[p] = true
	}
	return func(c *gin.Context) bool {
		return pathMap[c.Request.URL.Path]
	}
}

// SkipMethods 跳过指定的 HTTP 方法
func SkipMethods(methods ...string) func(c *gin.Context) bool {
	methodMap := make(map[string]bool)
	for _, m := range methods {
		methodMap[strings.ToUpper(m)] = true
	}
	return func(c *gin.Context) bool {
		return methodMap[c.Request.Method]
	}
}

// CombineSkippers 组合多个跳过函数
func CombineSkippers(skippers ...func(c *gin.Context) bool) func(c *gin.Context) bool {
	return func(c *gin.Context) bool {
		for _, skipper := range skippers {
			if skipper != nil && skipper(c) {
				return true
			}
		}
		return false
	}
}

// RequireRole 要求用户拥有指定角色的中间件
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := GetGlobal()
		if manager == nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error: enforcer not initialized",
			})
			c.Abort()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{
				"code":    401,
				"message": "unauthorized: missing user id",
			})
			c.Abort()
			return
		}

		uid := userID.(string)
		var hasRole bool
		var err error

		if manager.GetMode() == ModeSingle {
			hasRole, err = manager.Single().HasRoleForUser(uid, role)
		} else {
			tenantID, exists := c.Get("tenant_id")
			if !exists {
				c.JSON(401, gin.H{
					"code":    401,
					"message": "unauthorized: missing tenant id",
				})
				c.Abort()
				return
			}
			tid := tenantID.(string)
			hasRole, err = manager.Multi().HasRoleForUser(tid, uid, role)
		}

		if err != nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(403, gin.H{
				"code":    403,
				"message": "forbidden: required role not found",
				"role":    role,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole 要求用户拥有任意一个指定角色的中间件
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := GetGlobal()
		if manager == nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error: enforcer not initialized",
			})
			c.Abort()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{
				"code":    401,
				"message": "unauthorized: missing user id",
			})
			c.Abort()
			return
		}

		uid := userID.(string)
		var userRoles []string
		var err error

		if manager.GetMode() == ModeSingle {
			userRoles, err = manager.Single().GetRolesForUser(uid)
		} else {
			tenantID, exists := c.Get("tenant_id")
			if !exists {
				c.JSON(401, gin.H{
					"code":    401,
					"message": "unauthorized: missing tenant id",
				})
				c.Abort()
				return
			}
			tid := tenantID.(string)
			userRoles, err = manager.Multi().GetRolesForUser(tid, uid)
		}

		if err != nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// 检查是否拥有任意一个角色
		hasAnyRole := false
		for _, role := range roles {
			for _, userRole := range userRoles {
				if userRole == role {
					hasAnyRole = true
					break
				}
			}
			if hasAnyRole {
				break
			}
		}

		if !hasAnyRole {
			c.JSON(403, gin.H{
				"code":    403,
				"message": "forbidden: none of required roles found",
				"roles":   roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllRoles 要求用户拥有所有指定角色的中间件
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := GetGlobal()
		if manager == nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error: enforcer not initialized",
			})
			c.Abort()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{
				"code":    401,
				"message": "unauthorized: missing user id",
			})
			c.Abort()
			return
		}

		uid := userID.(string)

		for _, role := range roles {
			var hasRole bool
			var err error

			if manager.GetMode() == ModeSingle {
				hasRole, err = manager.Single().HasRoleForUser(uid, role)
			} else {
				tenantID, exists := c.Get("tenant_id")
				if !exists {
					c.JSON(401, gin.H{
						"code":    401,
						"message": "unauthorized: missing tenant id",
					})
					c.Abort()
					return
				}
				tid := tenantID.(string)
				hasRole, err = manager.Multi().HasRoleForUser(tid, uid, role)
			}

			if err != nil {
				c.JSON(500, gin.H{
					"code":    500,
					"message": "internal server error",
					"error":   err.Error(),
				})
				c.Abort()
				return
			}

			if !hasRole {
				c.JSON(403, gin.H{
					"code":    403,
					"message": "forbidden: required role not found",
					"role":    role,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

