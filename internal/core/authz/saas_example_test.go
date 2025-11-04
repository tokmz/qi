package authz_test

import (
	"fmt"
	"log"

	"qi/internal/core/authz"
)

// Example_saasUnifiedMode SaaS 系统使用统一多租户模式示例
func Example_saasUnifiedMode() {
	// 1. 创建配置（会自动设置为多租户模式）
	config := authz.DefaultConfig()
	config.AutoLoad = false

	// 2. 创建 SaaS 管理器
	saasManager, err := authz.NewSaaSManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer saasManager.Close()

	// 3. 初始化平台角色（系统启动时执行一次）
	if err := saasManager.InitializePlatformRoles(); err != nil {
		log.Fatal(err)
	}

	// 4. 创建平台管理员
	_ = saasManager.AssignPlatformRole("admin@platform.com", "super_admin")
	_ = saasManager.AssignPlatformRole("operator@platform.com", "platform_operator")

	// 5. 平台管理员创建租户
	_ = saasManager.CreateTenant("tenant_001")
	_ = saasManager.CreateTenant("tenant_002")

	// 6. 为租户分配管理员和成员
	_ = saasManager.AssignTenantRole("tenant_001", "alice@company-a.com", "tenant_admin")
	_ = saasManager.AssignTenantRole("tenant_001", "bob@company-a.com", "member")

	_ = saasManager.AssignTenantRole("tenant_002", "charlie@company-b.com", "tenant_admin")
	_ = saasManager.AssignTenantRole("tenant_002", "david@company-b.com", "viewer")

	// ===== 场景1: 平台管理员访问平台功能 =====
	allowed, _ := saasManager.CheckPlatformPermission(
		"admin@platform.com",
		"/platform/tenants",
		"POST",
	)
	fmt.Printf("平台管理员可以创建租户: %v\n", allowed)

	allowed, _ = saasManager.CheckPlatformPermission(
		"operator@platform.com",
		"/platform/tenants",
		"POST",
	)
	fmt.Printf("平台运营人员可以创建租户: %v\n", allowed)

	// ===== 场景2: 租户管理员管理自己的租户 =====
	allowed, _ = saasManager.CheckTenantPermission(
		"tenant_001",
		"alice@company-a.com",
		"/api/v1/users",
		"POST",
	)
	fmt.Printf("租户A管理员可以创建用户: %v\n", allowed)

	// ===== 场景3: 租户隔离 - Alice 不能访问租户B =====
	allowed, _ = saasManager.CheckTenantPermission(
		"tenant_002",
		"alice@company-a.com",
		"/api/v1/projects",
		"GET",
	)
	fmt.Printf("租户A管理员可以访问租户B: %v\n", allowed)

	// ===== 场景4: 租户成员权限 =====
	allowed, _ = saasManager.CheckTenantPermission(
		"tenant_001",
		"bob@company-a.com",
		"/api/v1/tasks",
		"POST",
	)
	fmt.Printf("租户A成员可以创建任务: %v\n", allowed)

	// ===== 场景5: 租户访客权限 =====
	allowed, _ = saasManager.CheckTenantPermission(
		"tenant_002",
		"david@company-b.com",
		"/api/v1/projects",
		"POST",
	)
	fmt.Printf("租户B访客可以创建项目: %v\n", allowed)

	// ===== 场景6: 检查是否是平台管理员 =====
	isPlatformAdmin, _ := saasManager.IsPlatformAdmin("admin@platform.com")
	fmt.Printf("admin@platform.com 是平台管理员: %v\n", isPlatformAdmin)

	isPlatformAdmin, _ = saasManager.IsPlatformAdmin("alice@company-a.com")
	fmt.Printf("alice@company-a.com 是平台管理员: %v\n", isPlatformAdmin)
}

// Example_saasDualMode SaaS 系统使用双 Manager 模式示例（不推荐，仅供参考）
func Example_saasDualMode() {
	// 方案二：创建两个独立的 Manager

	// 1. 创建平台管理器（单租户模式）
	platformConfig := authz.DefaultConfig()
	platformConfig.Mode = authz.ModeSingle
	platformConfig.Single.ModelPath = "testdata/platform_model.conf"
	platformConfig.Single.PolicyPath = "testdata/platform_policy.csv"
	platformConfig.AutoLoad = false

	platformManager, err := authz.New(platformConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer platformManager.Close()

	// 2. 创建租户管理器（多租户模式）
	tenantConfig := authz.DefaultConfig()
	tenantConfig.Mode = authz.ModeMulti
	tenantConfig.Multi.ModelPath = "testdata/tenant_model.conf"
	tenantConfig.Multi.PolicyPath = "testdata/tenant_policy.csv"
	tenantConfig.AutoLoad = false

	tenantManager, err := authz.New(tenantConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer tenantManager.Close()

	// 3. 平台权限管理
	platformSingle := platformManager.Single()
	_ = platformSingle.AddPolicy("platform_admin", "/platform/tenants", "*")
	_ = platformSingle.AddRoleForUser("admin@platform.com", "platform_admin")

	// 4. 租户权限管理
	tenantMulti := tenantManager.Multi()
	_ = tenantMulti.AddPolicy("tenant_001", "admin", "/api/v1/users", "*")
	_ = tenantMulti.AddRoleForUser("tenant_001", "alice@company-a.com", "admin")

	// 5. 权限检查
	// 平台权限检查
	allowed, _ := platformSingle.CheckPermission(
		"admin@platform.com",
		"/platform/tenants",
		"POST",
	)
	fmt.Printf("平台管理员可以创建租户: %v\n", allowed)

	// 租户权限检查
	allowed, _ = tenantMulti.CheckPermission(
		"tenant_001",
		"alice@company-a.com",
		"/api/v1/users",
		"POST",
	)
	fmt.Printf("租户管理员可以创建用户: %v\n", allowed)

	fmt.Println("注意: 双 Manager 模式需要维护两套配置，不推荐使用")
}

