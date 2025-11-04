# SaaS 系统权限管理指南

## 问题场景

在 SaaS 系统中，通常需要两层权限控制：

```
┌─────────────────────────────────────────────┐
│          平台总后台 (Platform)               │
│                                              │
│  • 平台管理员管理所有租户                     │
│  • 系统配置和运营数据                        │
│  • 使用标准 RBAC                             │
│                                              │
│  角色: super_admin, platform_operator       │
│  资源: /platform/*                          │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│         租户后台 (Tenant Backend)            │
│                                              │
│  租户A: company-a.com                        │
│    • 租户管理员管理自己的用户                 │
│    • 租户成员访问自己租户的资源               │
│                                              │
│  租户B: company-b.com                        │
│    • 完全独立，数据隔离                       │
│                                              │
│  角色: tenant_admin, member, viewer          │
│  资源: /api/v1/*                            │
└─────────────────────────────────────────────┘
```

## 解决方案对比

### 方案一：统一多租户模式（✅ 推荐）

**核心思想**：将平台管理作为特殊租户（`platform`），所有权限统一用多租户模式管理。

#### 优点
- ✅ **统一管理**：只需一个 Manager，一套配置
- ✅ **简单维护**：不需要区分平台/租户的逻辑
- ✅ **扩展性好**：未来添加新权限层级容易
- ✅ **性能好**：只需维护一个 Casbin Enforcer

#### 缺点
- ⚠️ 需要约定平台租户ID（如 `platform`）
- ⚠️ 平台权限和租户权限在同一张表

#### 数据结构
```
Casbin Policy:
┌──────────────┬──────────────┬────────────────────┬────────┐
│  tenant_id   │    role      │     resource       │ action │
├──────────────┼──────────────┼────────────────────┼────────┤
│  platform    │ super_admin  │ /platform/tenants  │   *    │ ← 平台权限
│  platform    │ super_admin  │ /platform/users    │   *    │
│  tenant_001  │ tenant_admin │ /api/v1/users      │   *    │ ← 租户权限
│  tenant_001  │ member       │ /api/v1/tasks      │   *    │
│  tenant_002  │ tenant_admin │ /api/v1/users      │   *    │
└──────────────┴──────────────┴────────────────────┴────────┘

Casbin Grouping (Role Assignment):
┌──────────────┬─────────────────────┬──────────────┐
│  tenant_id   │      user_id        │     role     │
├──────────────┼─────────────────────┼──────────────┤
│  platform    │ admin@platform.com  │ super_admin  │ ← 平台管理员
│  tenant_001  │ alice@company-a.com │ tenant_admin │ ← 租户管理员
│  tenant_001  │ bob@company-a.com   │ member       │ ← 租户成员
└──────────────┴─────────────────────┴──────────────┘
```

### 方案二：双 Manager 模式（不推荐）

**核心思想**：创建两个独立的 Manager，一个单租户（平台），一个多租户（租户）。

#### 优点
- ✅ 平台和租户权限完全分离
- ✅ 可以使用不同的存储

#### 缺点
- ❌ **维护复杂**：需要管理两个 Manager 实例
- ❌ **配置繁琐**：两套配置文件、两套策略
- ❌ **性能开销**：两个 Casbin Enforcer，内存占用翻倍
- ❌ **代码复杂**：需要在代码中判断使用哪个 Manager

## 推荐实现：统一多租户模式

### 1. 初始化 SaaS 管理器

```go
package main

import (
    "qi/internal/core/authz"
)

func main() {
    // 创建配置
    config := authz.DefaultConfig()
    config.Mode = authz.ModeMulti
    config.Multi.ModelPath = "configs/casbin/model_tenant.conf"
    
    // 使用数据库存储（推荐）
    config.Adapter.Type = authz.AdapterTypeGorm
    config.Adapter.DBType = "mysql"
    config.Adapter.DSN = "user:pass@tcp(127.0.0.1:3306)/casbin"

    // 创建 SaaS 管理器
    saasManager, err := authz.NewSaaSManager(config)
    if err != nil {
        panic(err)
    }
    defer saasManager.Close()

    // 初始化平台角色（系统启动时执行一次）
    if err := saasManager.InitializePlatformRoles(); err != nil {
        panic(err)
    }
}
```

### 2. 创建平台管理员

```go
// 创建平台超级管理员
err := saasManager.AssignPlatformRole("admin@platform.com", "super_admin")

// 创建平台运营人员
err = saasManager.AssignPlatformRole("operator@platform.com", "platform_operator")
```

### 3. 创建租户和租户管理员

```go
// 创建租户（会自动初始化租户的默认角色）
err := saasManager.CreateTenant("tenant_001")

// 为租户分配管理员
err = saasManager.AssignTenantRole("tenant_001", "alice@company-a.com", "tenant_admin")

// 为租户分配普通成员
err = saasManager.AssignTenantRole("tenant_001", "bob@company-a.com", "member")

// 为租户分配访客
err = saasManager.AssignTenantRole("tenant_001", "guest@company-a.com", "viewer")
```

### 4. 权限检查

```go
// 检查平台管理员权限
allowed, err := saasManager.CheckPlatformPermission(
    "admin@platform.com",
    "/platform/tenants",
    "POST",
)
// allowed == true

// 检查租户管理员权限
allowed, err = saasManager.CheckTenantPermission(
    "tenant_001",
    "alice@company-a.com",
    "/api/v1/users",
    "POST",
)
// allowed == true

// 租户隔离：Alice 不能访问 tenant_002
allowed, err = saasManager.CheckTenantPermission(
    "tenant_002",
    "alice@company-a.com",
    "/api/v1/projects",
    "GET",
)
// allowed == false
```

### 5. Gin 中间件集成

```go
package main

import (
    "github.com/gin-gonic/gin"
    "qi/internal/core/authz"
    "qi/internal/middleware"
)

func main() {
    // 创建 SaaS 管理器
    config := authz.DefaultConfig()
    saasManager, _ := authz.NewSaaSManager(config)
    
    r := gin.Default()
    
    // 配置中间件
    middlewareConfig := middleware.DefaultAuthzMiddlewareConfig()
    middlewareConfig.Skipper = middleware.AuthzCombineSkippers(
        middleware.AuthzSkipPaths("/login", "/register"),
        middleware.AuthzSkipPrefixes("/public"),
    )
    
    // 自动识别平台/租户请求
    r.Use(middleware.SaaSAuthzMiddleware(saasManager, middlewareConfig))
    
    // ===== 平台后台路由 =====
    platform := r.Group("/platform")
    {
        // 只有平台管理员可以访问
        platform.Use(middleware.SaaSRequirePlatformAdmin(saasManager))
        
        platform.GET("/tenants", listTenants)           // 查看所有租户
        platform.POST("/tenants", createTenant)         // 创建租户
        platform.DELETE("/tenants/:id", deleteTenant)   // 删除租户
        platform.GET("/analytics", getAnalytics)        // 查看分析数据
    }
    
    // ===== 租户后台路由 =====
    api := r.Group("/api/v1")
    {
        // 需要租户上下文
        api.GET("/projects", listProjects)         // 成员可访问
        api.POST("/projects", createProject)       // 成员可访问
        
        // 只有租户管理员可以访问
        admin := api.Group("/admin")
        admin.Use(middleware.SaaSRequireTenantRole(saasManager, "tenant_admin"))
        {
            admin.GET("/users", listTenantUsers)
            admin.POST("/users", createTenantUser)
            admin.GET("/settings", getTenantSettings)
            admin.PUT("/settings", updateTenantSettings)
        }
    }
    
    r.Run(":8080")
}
```

### 6. 中间件自动识别

`SaaSAuthzMiddleware` 会自动根据路径识别：
- **平台请求**：路径以 `/platform/` 开头 → 检查平台权限
- **租户请求**：其他路径 → 检查租户权限（需要 tenant_id）

```go
// 平台请求示例
GET /platform/tenants
  → 检查: CheckPlatformPermission(userID, "/platform/tenants", "GET")
  → 使用平台租户: "platform"

// 租户请求示例
GET /api/v1/projects
  → 检查: CheckTenantPermission(tenantID, userID, "/api/v1/projects", "GET")
  → 使用请求中的 tenant_id
```

## 数据库设计

### 用户和租户关系表

```sql
-- 用户表（跨租户）
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    is_platform_admin BOOLEAN DEFAULT FALSE,  -- 是否是平台管理员
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 租户表
CREATE TABLE tenants (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status ENUM('active', 'suspended', 'deleted') DEFAULT 'active',
    plan VARCHAR(20) DEFAULT 'free',
    max_users INT DEFAULT 10,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 租户成员关系表
CREATE TABLE tenant_users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    tenant_id VARCHAR(50) NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(50) NOT NULL,  -- tenant_admin/member/viewer
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE KEY uk_tenant_user (tenant_id, user_id)
);

-- Casbin 策略表（自动创建）
CREATE TABLE casbin_rule (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    ptype VARCHAR(100),
    v0 VARCHAR(100),  -- tenant_id
    v1 VARCHAR(100),  -- role/user
    v2 VARCHAR(100),  -- resource
    v3 VARCHAR(100),  -- action
    v4 VARCHAR(100),
    v5 VARCHAR(100)
);
```

## JWT Token 设计

### 平台管理员 Token

```json
{
  "user_id": "admin@platform.com",
  "is_platform_admin": true,
  "exp": 1234567890
}
```

### 租户用户 Token

```json
{
  "user_id": "alice@company-a.com",
  "tenant_id": "tenant_001",
  "is_platform_admin": false,
  "exp": 1234567890
}
```

### 跨租户用户 Token

如果一个用户属于多个租户：

```json
{
  "user_id": "consultant@example.com",
  "tenants": ["tenant_001", "tenant_002", "tenant_003"],
  "current_tenant": "tenant_001",  // 当前选择的租户
  "exp": 1234567890
}
```

## 完整示例代码

查看以下文件：
- `internal/core/authz/saas_example.go` - SaaS 管理器实现
- `internal/core/authz/saas_example_test.go` - 完整使用示例
- `internal/middleware/authz_saas.go` - SaaS 中间件

## 常见问题

### Q1: 为什么推荐统一多租户模式？

**A:** 因为本质上平台管理也是一种"租户"，只是它管理的是整个平台。使用统一模式：
- 代码更简单（一个 Manager）
- 维护更容易（一套配置）
- 性能更好（一个 Enforcer）
- 扩展性更好（容易添加新层级）

### Q2: 平台管理员可以访问租户数据吗？

**A:** 默认情况下不能。如果需要，有两种方式：

**方式1**：为平台管理员添加所有租户的权限（不推荐）
```go
// 为平台管理员添加访问某个租户的权限
saasManager.GetManager().Multi().AddRoleForUser(
    "tenant_001", 
    "admin@platform.com", 
    "super_viewer",
)
```

**方式2**：在业务层实现（推荐）
```go
// 在业务逻辑中检查是否是平台管理员
if isPlatformAdmin, _ := saasManager.IsPlatformAdmin(userID); isPlatformAdmin {
    // 平台管理员可以查看任何租户数据
    data = getTenantDataByAdmin(tenantID)
} else {
    // 普通用户只能查看自己租户数据
    data = getTenantData(userID, tenantID)
}
```

### Q3: 用户可以属于多个租户吗？

**A:** 可以！这是多租户模式的优势。

```go
// 用户 consultant@example.com 加入多个租户
saasManager.AssignTenantRole("tenant_001", "consultant@example.com", "member")
saasManager.AssignTenantRole("tenant_002", "consultant@example.com", "member")
saasManager.AssignTenantRole("tenant_003", "consultant@example.com", "viewer")

// 检查权限时指定租户
allowed, _ := saasManager.CheckTenantPermission(
    "tenant_001",  // 在 tenant_001 中检查
    "consultant@example.com",
    "/api/v1/projects",
    "GET",
)
```

### Q4: 如何实现租户间的数据共享？

**A:** 通过业务逻辑实现，不在权限层面处理。

```go
// 在数据库设计时添加共享标记
CREATE TABLE projects (
    id BIGINT PRIMARY KEY,
    tenant_id VARCHAR(50),
    is_shared BOOLEAN DEFAULT FALSE,
    shared_with TEXT,  -- JSON: ["tenant_002", "tenant_003"]
    ...
);

// 在业务逻辑中处理共享
func GetProject(userID, tenantID, projectID string) (*Project, error) {
    project := getProjectFromDB(projectID)
    
    // 检查是否是项目所属租户
    if project.TenantID == tenantID {
        return project, nil
    }
    
    // 检查是否被共享
    if project.IsShared && isInSharedList(project.SharedWith, tenantID) {
        return project, nil
    }
    
    return nil, ErrPermissionDenied
}
```

### Q5: 如何处理租户的套餐限制？

**A:** 在业务层实现，不在权限层面。

```go
// 在创建资源前检查套餐限制
func CreateProject(tenantID string, project *Project) error {
    tenant := getTenantFromDB(tenantID)
    
    // 检查套餐限制
    switch tenant.Plan {
    case "free":
        if getProjectCount(tenantID) >= 5 {
            return errors.New("免费版最多创建 5 个项目")
        }
    case "basic":
        if getProjectCount(tenantID) >= 20 {
            return errors.New("基础版最多创建 20 个项目")
        }
    }
    
    // 权限检查通过后，再检查业务限制
    return saveProject(project)
}
```

## 总结

✅ **推荐方案**：统一使用多租户模式，将平台管理作为特殊租户（`platform`）

✅ **优势**：
- 简单易维护
- 性能好
- 扩展性强
- 代码清晰

✅ **适用场景**：
- SaaS 应用
- 多租户系统
- 需要平台管理后台的系统

📚 **参考代码**：
- `internal/core/authz/saas_example.go`
- `internal/core/authz/saas_example_test.go`
- `internal/middleware/authz_saas.go`

