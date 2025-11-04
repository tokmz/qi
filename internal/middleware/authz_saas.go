package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"qi/internal/core/authz"
)

// SaaSAuthzMiddleware SaaS 系统权限中间件
// 自动识别是平台请求还是租户请求
func SaaSAuthzMiddleware(saasManager *authz.SaaSManager, config ...*AuthzMiddlewareConfig) gin.HandlerFunc {
	cfg := DefaultAuthzMiddlewareConfig()
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

		// 判断是平台请求还是租户请求
		if strings.HasPrefix(resource, "/platform/") {
			// 平台请求：使用平台权限检查
			allowed, err = saasManager.CheckPlatformPermission(userID, resource, action)
		} else {
			// 租户请求：使用租户权限检查
			tenantID := cfg.TenantExtractor(c)
			if tenantID == "" {
				c.JSON(cfg.UnauthorizedCode, gin.H{
					"code":    cfg.UnauthorizedCode,
					"message": "unauthorized: missing tenant id",
				})
				c.Abort()
				return
			}
			allowed, err = saasManager.CheckTenantPermission(tenantID, userID, resource, action)
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

// SaaSRequirePlatformAdmin 要求平台管理员权限
func SaaSRequirePlatformAdmin(saasManager *authz.SaaSManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		isPlatformAdmin, err := saasManager.IsPlatformAdmin(uid)
		if err != nil {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "internal server error",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		if !isPlatformAdmin {
			c.JSON(403, gin.H{
				"code":    403,
				"message": "forbidden: platform admin required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SaaSRequireTenantRole 要求租户特定角色
func SaaSRequireTenantRole(saasManager *authz.SaaSManager, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{
				"code":    401,
				"message": "unauthorized: missing user id",
			})
			c.Abort()
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(401, gin.H{
				"code":    401,
				"message": "unauthorized: missing tenant id",
			})
			c.Abort()
			return
		}

		uid := userID.(string)
		tid := tenantID.(string)

		hasRole, err := saasManager.GetManager().Multi().HasRoleForUser(tid, uid, role)
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
				"message": "forbidden: required tenant role not found",
				"role":    role,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

