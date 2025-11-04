package authz_test

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"qi/internal/core/authz"
	"qi/internal/middleware"
)

// Example_singleTenant 单租户模式示例
func Example_singleTenant() {
	// 1. 创建配置
	config := &authz.Config{
		Mode: authz.ModeSingle,
		Single: authz.SingleConfig{
			ModelPath:  "testdata/model.conf",
			PolicyPath: "testdata/policy.csv",
		},
		Adapter: authz.AdapterConfig{
			Type: authz.AdapterTypeFile,
		},
		AutoLoad:         false,
		AutoLoadInterval: 60,
		EnableLog:        true,
	}

	// 2. 创建权限管理器
	manager, err := authz.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// 3. 获取单租户控制器
	single := manager.Single()

	// 4. 添加策略：admin 角色可以访问所有用户资源
	_ = single.AddPolicy("admin", "/api/v1/users", "*")
	_ = single.AddPolicy("admin", "/api/v1/projects", "*")

	// 5. 添加策略：user 角色只能读取
	_ = single.AddPolicy("user", "/api/v1/users", "GET")
	_ = single.AddPolicy("user", "/api/v1/projects", "GET")

	// 6. 为用户分配角色
	_ = single.AddRoleForUser("alice", "admin")
	_ = single.AddRoleForUser("bob", "user")

	// 7. 检查权限
	allowed, _ := single.CheckPermission("alice", "/api/v1/users", "POST")
	fmt.Printf("Alice can POST users: %v\n", allowed)

	allowed, _ = single.CheckPermission("bob", "/api/v1/users", "POST")
	fmt.Printf("Bob can POST users: %v\n", allowed)

	// 8. 获取用户角色
	roles, _ := single.GetRolesForUser("alice")
	fmt.Printf("Alice's roles: %v\n", roles)

	// 9. 获取用户权限
	permissions, _ := single.GetPermissionsForUser("alice")
	fmt.Printf("Alice's permissions count: %d\n", len(permissions))
}

// Example_multiTenant 多租户模式示例
func Example_multiTenant() {
	// 1. 创建配置
	config := &authz.Config{
		Mode: authz.ModeMulti,
		Multi: authz.MultiConfig{
			ModelPath:  "testdata/model_tenant.conf",
			PolicyPath: "testdata/policy_tenant.csv",
		},
		Adapter: authz.AdapterConfig{
			Type: authz.AdapterTypeFile,
		},
		AutoLoad:         false,
		AutoLoadInterval: 60,
		EnableLog:        true,
	}

	// 2. 创建权限管理器
	manager, err := authz.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// 3. 获取多租户控制器
	multi := manager.Multi()

	// 4. 为租户1添加策略
	_ = multi.AddPolicy("tenant_001", "admin", "/api/v1/users", "*")
	_ = multi.AddPolicy("tenant_001", "admin", "/api/v1/projects", "*")
	_ = multi.AddPolicy("tenant_001", "member", "/api/v1/projects", "GET")

	// 5. 为租户2添加策略
	_ = multi.AddPolicy("tenant_002", "admin", "/api/v1/users", "*")
	_ = multi.AddPolicy("tenant_002", "member", "/api/v1/projects", "GET")

	// 6. 为租户用户分配角色
	_ = multi.AddRoleForUser("tenant_001", "alice", "admin")
	_ = multi.AddRoleForUser("tenant_001", "bob", "member")
	_ = multi.AddRoleForUser("tenant_002", "charlie", "admin")

	// 7. 检查租户用户权限
	allowed, _ := multi.CheckPermission("tenant_001", "alice", "/api/v1/projects", "POST")
	fmt.Printf("Tenant1-Alice can POST projects: %v\n", allowed)

	allowed, _ = multi.CheckPermission("tenant_001", "bob", "/api/v1/projects", "POST")
	fmt.Printf("Tenant1-Bob can POST projects: %v\n", allowed)

	// 8. Alice 不能访问 tenant_002 的资源
	allowed, _ = multi.CheckPermission("tenant_002", "alice", "/api/v1/projects", "GET")
	fmt.Printf("Tenant2-Alice can GET projects: %v\n", allowed)

	// 9. 获取租户用户角色
	roles, _ := multi.GetRolesForUser("tenant_001", "alice")
	fmt.Printf("Tenant1-Alice's roles: %v\n", roles)

	// 10. 获取租户的所有策略
	policies, _ := multi.GetPoliciesForTenant("tenant_001")
	fmt.Printf("Tenant1 policies count: %d\n", len(policies))
}

// Example_globalManager 全局管理器示例
func Example_globalManager() {
	// 1. 初始化全局管理器
	config := authz.DefaultConfig()
	config.Mode = authz.ModeSingle
	config.AutoLoad = false

	if err := authz.InitGlobal(config, nil); err != nil {
		log.Fatal(err)
	}

	// 2. 使用全局快捷函数
	_ = authz.AddRoleForUser("alice", "admin")

	// 3. 检查权限
	allowed, _ := authz.CheckPermission("alice", "/api/v1/users", "GET")
	fmt.Printf("Alice can GET users: %v\n", allowed)

	// 4. 获取全局管理器
	manager := authz.GetGlobal()
	fmt.Printf("Manager mode: %s\n", manager.GetMode())
}

// Example_ginMiddleware Gin 中间件示例
func Example_ginMiddleware() {
	// 1. 初始化全局管理器
	config := &authz.Config{
		Mode: authz.ModeSingle,
		Single: authz.SingleConfig{
			ModelPath:  "testdata/model.conf",
			PolicyPath: "testdata/policy.csv",
		},
		Adapter: authz.AdapterConfig{
			Type: authz.AdapterTypeFile,
		},
		EnableLog: true,
	}

	if err := authz.InitGlobal(config, nil); err != nil {
		log.Fatal(err)
	}

	// 2. 设置测试数据
	_ = authz.AddRoleForUser("alice", "admin")
	manager := authz.GetGlobal()
	_ = manager.Single().AddPolicy("admin", "/api/v1/users", "*")

	// 3. 创建 Gin 应用
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 4. 自定义中间件配置
	middlewareConfig := middleware.DefaultAuthzMiddlewareConfig()
	middlewareConfig.Skipper = middleware.AuthzCombineSkippers(
		middleware.AuthzSkipPaths("/health", "/login"),
		middleware.AuthzSkipPrefixes("/public"),
	)
	middlewareConfig.UserExtractor = func(c *gin.Context) string {
		// 从上下文或请求头获取用户ID
		if userID, exists := c.Get("user_id"); exists {
			return userID.(string)
		}
		return c.GetHeader("X-User-ID")
	}

	// 5. 使用权限中间件
	r.Use(middleware.AuthzGlobalMiddleware(middlewareConfig))

	// 6. 定义路由
	r.GET("/api/v1/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "users list"})
	})

	r.POST("/api/v1/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "user created"})
	})

	// 7. 需要特定角色的路由
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthzRequireRole("admin"))
	{
		admin.GET("/dashboard", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "admin dashboard"})
		})
	}

	fmt.Println("Gin application configured with authz middleware")
}

// Example_multiTenantGin 多租户 Gin 中间件示例
func Example_multiTenantGin() {
	// 1. 初始化多租户全局管理器
	config := &authz.Config{
		Mode: authz.ModeMulti,
		Multi: authz.MultiConfig{
			ModelPath:  "testdata/model_tenant.conf",
			PolicyPath: "testdata/policy_tenant.csv",
		},
		Adapter: authz.AdapterConfig{
			Type: authz.AdapterTypeFile,
		},
		EnableLog: true,
	}

	if err := authz.InitGlobal(config, nil); err != nil {
		log.Fatal(err)
	}

	// 2. 设置测试数据
	_ = authz.AddTenantRoleForUser("tenant_001", "alice", "admin")
	manager := authz.GetGlobal()
	_ = manager.Multi().AddPolicy("tenant_001", "admin", "/api/v1/projects", "*")

	// 3. 创建 Gin 应用
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 4. 多租户中间件配置
	middlewareConfig := middleware.DefaultAuthzMiddlewareConfig()
	middlewareConfig.TenantExtractor = func(c *gin.Context) string {
		// 从 JWT 或请求头获取租户ID
		if tenantID, exists := c.Get("tenant_id"); exists {
			return tenantID.(string)
		}
		return c.GetHeader("X-Tenant-ID")
	}

	// 5. 使用权限中间件
	r.Use(middleware.AuthzGlobalMiddleware(middlewareConfig))

	// 6. 定义路由
	r.GET("/api/v1/projects", func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		c.JSON(200, gin.H{
			"message":   "projects list",
			"tenant_id": tenantID,
		})
	})

	fmt.Println("Multi-tenant Gin application configured")
}

// Example_advancedUsage 高级用法示例
func Example_advancedUsage() {
	// 1. 创建配置
	config := authz.DefaultConfig()
	config.Mode = authz.ModeSingle
	config.AutoLoad = false

	// 2. 创建管理器
	manager, err := authz.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	single := manager.Single()

	// 3. 批量添加策略
	policies := [][]string{
		{"admin", "/api/v1/users", "*"},
		{"admin", "/api/v1/projects", "*"},
		{"admin", "/api/v1/tasks", "*"},
		{"user", "/api/v1/projects", "GET"},
		{"user", "/api/v1/tasks", "GET"},
		{"user", "/api/v1/tasks", "POST"},
	}

	for _, p := range policies {
		_ = single.AddPolicy(p[0], p[1], p[2])
	}

	// 4. 批量分配角色
	userRoles := map[string]string{
		"alice":   "admin",
		"bob":     "user",
		"charlie": "user",
	}

	for user, role := range userRoles {
		_ = single.AddRoleForUser(user, role)
	}

	// 5. 统一的权限检查接口
	ctx := context.Background()
	result, err := manager.Enforce(ctx, &authz.EnforceRequest{
		UserID:   "alice",
		Resource: "/api/v1/users",
		Action:   "DELETE",
	})

	if err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Printf("Alice can DELETE users: %v\n", result.Allowed)

	// 6. 获取所有角色
	allRoles, _ := single.GetAllRoles()
	fmt.Printf("All roles: %v\n", allRoles)

	// 7. 获取所有资源
	allObjects, _ := single.GetAllObjects()
	fmt.Printf("All resources count: %d\n", len(allObjects))

	// 8. 重新加载策略
	if err := manager.LoadPolicy(); err != nil {
		log.Printf("Load policy error: %v", err)
	}

	// 9. 保存策略
	if err := manager.SavePolicy(); err != nil {
		log.Printf("Save policy error: %v", err)
	}
}

// Example_roleManagement 角色管理示例
func Example_roleManagement() {
	config := authz.DefaultConfig()
	config.AutoLoad = false

	manager, err := authz.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	single := manager.Single()

	// 1. 创建角色层级（admin 继承 user）
	_ = single.AddPolicy("user", "/api/v1/projects", "GET")
	_ = single.AddPolicy("admin", "/api/v1/projects", "*")
	_ = single.AddPolicy("admin", "/api/v1/users", "*")

	// 2. 分配角色
	_ = single.AddRoleForUser("alice", "admin")
	_ = single.AddRoleForUser("bob", "user")

	// 3. 检查角色
	hasRole, _ := single.HasRoleForUser("alice", "admin")
	fmt.Printf("Alice has admin role: %v\n", hasRole)

	// 4. 获取角色的所有用户
	users, _ := single.GetUsersForRole("admin")
	fmt.Printf("Admin users: %v\n", users)

	// 5. 移除角色
	_ = single.RemoveRoleForUser("bob", "user")

	// 6. 删除角色
	_ = single.DeleteRole("user")

	fmt.Println("Role management completed")
}

