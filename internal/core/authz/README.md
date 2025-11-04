# Authz - 权限管理包

基于 Casbin 封装的权限管理包，支持单体系统的标准 RBAC 和 SaaS 系统的多租户权限控制。

## 特性

### 核心特性

- ✅ **双模式支持**: 单租户模式（标准 RBAC）和多租户模式（SaaS 系统）
- ✅ **灵活配置**: 支持文件和数据库两种适配器
- ✅ **自动加载**: 支持策略自动重新加载
- ✅ **Gin 集成**: 提供开箱即用的 Gin 中间件
- ✅ **角色管理**: 完整的角色分配、查询和删除功能
- ✅ **策略管理**: 动态添加、移除和查询权限策略
- ✅ **租户隔离**: 多租户模式下的完全数据隔离
- ✅ **高性能**: 基于 Casbin 的高性能权限检查
- ✅ **类型安全**: 完整的类型定义和错误处理

### 技术栈

- [Casbin v2](https://github.com/casbin/casbin) - 权限管理框架
- [Casbin GORM Adapter v3](https://github.com/casbin/gorm-adapter) - 数据库适配器
- [Gin](https://github.com/gin-gonic/gin) - Web 框架集成

## 快速开始

### 1. 单租户模式

适用于企业内部系统、单体应用。

```go
package main

import (
    "log"
    "qi/internal/core/authz"
)

func main() {
    // 1. 创建配置
    config := &authz.Config{
        Mode: authz.ModeSingle,
        Single: authz.SingleConfig{
            ModelPath:  "configs/casbin/model.conf",
            PolicyPath: "configs/casbin/policy.csv",
        },
        Adapter: authz.AdapterConfig{
            Type: authz.AdapterTypeFile,
        },
        AutoLoad:         true,
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

    // 4. 添加策略
    single.AddPolicy("admin", "/api/v1/users", "*")
    single.AddPolicy("user", "/api/v1/users", "GET")

    // 5. 分配角色
    single.AddRoleForUser("alice", "admin")
    single.AddRoleForUser("bob", "user")

    // 6. 检查权限
    allowed, _ := single.CheckPermission("alice", "/api/v1/users", "POST")
    log.Printf("Alice can POST users: %v", allowed)
}
```

### 2. 多租户模式

适用于 SaaS 应用、云服务平台。

```go
package main

import (
    "log"
    "qi/internal/core/authz"
)

func main() {
    // 1. 创建配置
    config := &authz.Config{
        Mode: authz.ModeMulti,
        Multi: authz.MultiConfig{
            ModelPath:  "configs/casbin/model_tenant.conf",
            PolicyPath: "configs/casbin/policy_tenant.csv",
        },
        Adapter: authz.AdapterConfig{
            Type: authz.AdapterTypeFile,
        },
        AutoLoad:         true,
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

    // 4. 为租户添加策略
    multi.AddPolicy("tenant_001", "admin", "/api/v1/projects", "*")
    multi.AddPolicy("tenant_001", "member", "/api/v1/projects", "GET")

    // 5. 为租户用户分配角色
    multi.AddRoleForUser("tenant_001", "alice", "admin")
    multi.AddRoleForUser("tenant_001", "bob", "member")

    // 6. 检查租户用户权限
    allowed, _ := multi.CheckPermission("tenant_001", "alice", "/api/v1/projects", "POST")
    log.Printf("Tenant1-Alice can POST projects: %v", allowed)
}
```

### 3. 使用数据库适配器

```go
config := &authz.Config{
    Mode: authz.ModeSingle,
    Single: authz.SingleConfig{
        ModelPath: "configs/casbin/model.conf",
    },
    Adapter: authz.AdapterConfig{
        Type:      authz.AdapterTypeGorm,
        DBType:    "mysql",
        DSN:       "user:password@tcp(127.0.0.1:3306)/casbin?charset=utf8mb4&parseTime=True&loc=Local",
        TableName: "casbin_rule",
    },
    AutoLoad:         true,
    AutoLoadInterval: 60,
    EnableLog:        true,
}

manager, err := authz.New(config, nil)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()
```

## Gin 中间件

### 基础用法

```go
package main

import (
    "github.com/gin-gonic/gin"
    "qi/internal/core/authz"
)

func main() {
    // 1. 初始化全局管理器
    config := authz.DefaultConfig()
    authz.InitGlobal(config, nil)

    // 2. 创建 Gin 应用
    r := gin.Default()

    // 3. 使用权限中间件
    r.Use(authz.GlobalMiddleware())

    // 4. 定义路由
    r.GET("/api/v1/users", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "users list"})
    })

    r.Run(":8080")
}
```

### 自定义中间件配置

```go
// 创建自定义配置
middlewareConfig := authz.DefaultMiddlewareConfig()

// 跳过某些路径
middlewareConfig.Skipper = authz.CombineSkippers(
    authz.SkipPaths("/health", "/login"),
    authz.SkipPrefixes("/public"),
)

// 自定义用户提取器
middlewareConfig.UserExtractor = func(c *gin.Context) string {
    if userID, exists := c.Get("user_id"); exists {
        return userID.(string)
    }
    return c.GetHeader("X-User-ID")
}

// 自定义租户提取器（多租户模式）
middlewareConfig.TenantExtractor = func(c *gin.Context) string {
    if tenantID, exists := c.Get("tenant_id"); exists {
        return tenantID.(string)
    }
    return c.GetHeader("X-Tenant-ID")
}

// 使用自定义配置
r.Use(authz.GlobalMiddleware(middlewareConfig))
```

### 角色中间件

```go
// 要求 admin 角色
admin := r.Group("/api/v1/admin")
admin.Use(authz.RequireRole("admin"))
{
    admin.GET("/dashboard", adminDashboard)
    admin.GET("/users", adminUsers)
}

// 要求任意一个角色
moderator := r.Group("/api/v1/moderate")
moderator.Use(authz.RequireAnyRole("admin", "moderator"))
{
    moderator.POST("/review", reviewContent)
}

// 要求所有角色
special := r.Group("/api/v1/special")
special.Use(authz.RequireAllRoles("admin", "auditor"))
{
    special.GET("/sensitive", sensitiveData)
}
```

## 配置文件

### 单租户模型 (model.conf)

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

### 多租户模型 (model_tenant.conf)

```ini
[request_definition]
r = tenant, sub, obj, act

[policy_definition]
p = tenant, sub, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.tenant) && r.tenant == p.tenant && r.obj == p.obj && r.act == p.act
```

### 配置示例 (config.yaml)

```yaml
casbin:
  # 模式：single（单租户）或 multi（多租户）
  mode: single
  
  # 单租户模式配置
  single:
    model_path: configs/casbin/model.conf
    policy_path: configs/casbin/policy.csv
  
  # 多租户模式配置
  multi:
    model_path: configs/casbin/model_tenant.conf
    policy_path: configs/casbin/policy_tenant.csv
    
  # 适配器配置
  adapter:
    type: file  # file/gorm
    # GORM 适配器配置（使用数据库时）
    # db_type: mysql
    # dsn: "user:password@tcp(127.0.0.1:3306)/casbin?charset=utf8mb4"
    # table_name: casbin_rule
    
  # 是否自动加载策略
  auto_load: true
  
  # 策略更新间隔（秒）
  auto_load_interval: 60
  
  # 是否启用日志
  enable_log: true
```

## API 文档

### 单租户模式 API

#### SingleEnforcer

| 方法 | 说明 |
|------|------|
| `CheckPermission(userID, resource, action)` | 检查用户权限 |
| `AddPolicy(subject, object, action)` | 添加策略 |
| `RemovePolicy(subject, object, action)` | 移除策略 |
| `AddRoleForUser(userID, role)` | 为用户分配角色 |
| `RemoveRoleForUser(userID, role)` | 移除用户角色 |
| `GetRolesForUser(userID)` | 获取用户的所有角色 |
| `GetUsersForRole(role)` | 获取拥有指定角色的所有用户 |
| `HasRoleForUser(userID, role)` | 检查用户是否拥有角色 |
| `GetPermissionsForUser(userID)` | 获取用户的所有权限 |
| `DeleteUser(userID)` | 删除用户及其所有角色 |
| `DeleteRole(role)` | 删除角色及其所有用户关联 |
| `LoadPolicy()` | 重新加载策略 |
| `SavePolicy()` | 保存策略到存储 |

### 多租户模式 API

#### MultiEnforcer

| 方法 | 说明 |
|------|------|
| `CheckPermission(tenantID, userID, resource, action)` | 检查租户用户权限 |
| `AddPolicy(tenantID, subject, object, action)` | 添加租户策略 |
| `RemovePolicy(tenantID, subject, object, action)` | 移除租户策略 |
| `AddRoleForUser(tenantID, userID, role)` | 为租户用户分配角色 |
| `RemoveRoleForUser(tenantID, userID, role)` | 移除租户用户角色 |
| `GetRolesForUser(tenantID, userID)` | 获取用户在指定租户的所有角色 |
| `GetUsersForRole(tenantID, role)` | 获取拥有指定角色的所有用户 |
| `HasRoleForUser(tenantID, userID, role)` | 检查用户是否在租户中拥有角色 |
| `GetPermissionsForUser(tenantID, userID)` | 获取用户在租户的所有权限 |
| `DeleteUser(tenantID, userID)` | 删除租户用户及其所有角色 |
| `DeleteRole(tenantID, role)` | 删除租户角色及其所有用户关联 |
| `DeleteTenant(tenantID)` | 删除租户及其所有数据 |
| `GetPoliciesForTenant(tenantID)` | 获取租户的所有策略 |
| `GetRolesForTenant(tenantID)` | 获取租户的所有角色关系 |

### 全局快捷函数

```go
// 单租户模式快捷函数
authz.CheckPermission(userID, resource, action)
authz.AddRoleForUser(userID, role)

// 多租户模式快捷函数
authz.CheckTenantPermission(tenantID, userID, resource, action)
authz.AddTenantRoleForUser(tenantID, userID, role)
```

## 目录结构

```
internal/core/authz/
├── config.go           # 配置定义
├── errors.go           # 错误定义
├── types.go            # 类型定义
├── enforcer.go         # 统一的权限管理器
├── single.go           # 单租户模式实现
├── multi.go            # 多租户模式实现
├── middleware.go       # Gin 中间件
├── example_test.go     # 示例代码
└── README.md           # 文档
```

## 代码质量

- ✅ 完整的中文注释
- ✅ 详细的错误处理
- ✅ 类型安全
- ✅ 并发安全（使用 sync.RWMutex）
- ✅ 资源管理（优雅关闭）
- ✅ 日志支持
- ✅ 示例代码
- ✅ 文档完善

## 最佳实践

### 1. 权限设计

```go
// 使用通配符
single.AddPolicy("admin", "/api/v1/*", "*")

// 具体路径
single.AddPolicy("user", "/api/v1/users/:id", "GET")

// HTTP 方法映射
// GET    -> 读取
// POST   -> 创建
// PUT    -> 更新
// DELETE -> 删除
// *      -> 所有操作
```

### 2. 角色继承

```go
// 定义角色层级
single.AddPolicy("viewer", "/api/v1/projects", "GET")
single.AddPolicy("editor", "/api/v1/projects", "PUT")
single.AddPolicy("admin", "/api/v1/projects", "*")

// admin 继承 editor 的权限（如果需要）
// 在 model.conf 中配置角色继承规则
```

### 3. 多租户隔离

```go
// 租户1的数据
multi.AddPolicy("tenant_001", "admin", "/api/v1/projects", "*")
multi.AddRoleForUser("tenant_001", "alice", "admin")

// 租户2的数据
multi.AddPolicy("tenant_002", "admin", "/api/v1/projects", "*")
multi.AddRoleForUser("tenant_002", "bob", "admin")

// alice 不能访问 tenant_002 的数据
allowed, _ := multi.CheckPermission("tenant_002", "alice", "/api/v1/projects", "GET")
// allowed == false
```

### 4. 数据库存储

生产环境推荐使用数据库适配器：

```go
config := &authz.Config{
    Mode: authz.ModeSingle,
    Single: authz.SingleConfig{
        ModelPath: "configs/casbin/model.conf",
    },
    Adapter: authz.AdapterConfig{
        Type:      authz.AdapterTypeGorm,
        DBType:    "mysql",
        DSN:       "user:password@tcp(127.0.0.1:3306)/casbin",
        TableName: "casbin_rule",
    },
    AutoLoad:         true,  // 启用自动加载
    AutoLoadInterval: 60,    // 每60秒重新加载
}
```

### 5. 性能优化

```go
// 1. 启用自动加载避免频繁重启
config.AutoLoad = true
config.AutoLoadInterval = 60

// 2. 使用数据库适配器时添加索引
// CREATE INDEX idx_ptype ON casbin_rule(ptype);
// CREATE INDEX idx_v0 ON casbin_rule(v0);

// 3. 批量操作
for _, policy := range policies {
    single.AddPolicy(policy[0], policy[1], policy[2])
}
manager.SavePolicy()  // 一次性保存

// 4. 使用角色而不是直接分配权限
// 推荐：single.AddRoleForUser("alice", "admin")
// 避免：直接为 alice 添加大量 policy
```

## 常见问题

### Q: 如何在单租户和多租户之间切换？

A: 修改配置中的 `mode` 字段，并使用对应的模型文件：

```go
// 单租户
config.Mode = authz.ModeSingle

// 多租户
config.Mode = authz.ModeMulti
```

### Q: 如何自定义日志？

A: 实现 `Logger` 接口并传入：

```go
type MyLogger struct {
    // your logger
}

func (l *MyLogger) Debug(msg string, fields ...interface{}) { ... }
func (l *MyLogger) Info(msg string, fields ...interface{}) { ... }
func (l *MyLogger) Warn(msg string, fields ...interface{}) { ... }
func (l *MyLogger) Error(msg string, fields ...interface{}) { ... }

manager, err := authz.New(config, &MyLogger{})
```

### Q: 如何处理动态路径参数？

A: 在策略中使用路径模式匹配，或在中间件中自定义资源提取器：

```go
middlewareConfig.ResourceExtractor = func(c *gin.Context) string {
    // 将 /api/v1/users/123 转换为 /api/v1/users/:id
    path := c.Request.URL.Path
    // 自定义路径转换逻辑
    return convertPath(path)
}
```

### Q: 如何测试权限？

A: 使用示例代码中的测试模式：

```go
config.AutoLoad = false  // 禁用自动加载
config.Adapter.Type = authz.AdapterTypeFile  // 使用文件适配器

manager, _ := authz.New(config, nil)
// 添加测试数据
// 执行测试
```

## 许可证

MIT License

