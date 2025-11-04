# Qi - 通用项目管理系统

## 1. 项目概述

Qi 是一个基于 Go 语言开发的通用项目管理系统，旨在提供高性能、可扩展的项目协作和任务管理解决方案。系统采用现代化的技术栈，注重性能优化、可观测性和高可用性。

### 1.1 核心特性

- **高性能缓存**: 基于 Redis v9 的分布式缓存，使用 singleflight 防止缓存击穿
- **灵活配置**: 使用 Viper 实现多环境配置管理
- **完善的日志系统**: Zap 高性能日志 + Lumberjack 日志分割归档
- **可观测性**: 基于 OpenTelemetry 的分布式链路追踪
- **权限控制**: 基于 Casbin 的 RBAC 权限管理，支持动态权限配置和角色继承
- **定时任务**: 基于 Cron v3 的定时任务调度，支持标准 cron 表达式和秒级任务
- **数据持久化**: GORM ORM 框架，支持多种数据库
- **RESTful API**: 统一的 API 设计规范和响应格式

## 2. 技术架构

### 2.1 技术栈

| 技术组件 | 版本 | 用途 |
|---------|------|------|
| Go | 1.21+ | 开发语言 |
| Gin | v1.9+ | Web 框架 |
| GORM | v1.25+ | ORM 框架 |
| Redis | v9 | 缓存中间件 |
| Viper | v1.18+ | 配置管理 |
| Zap | v1.26+ | 日志管理 |
| Lumberjack | v2 | 日志分割 |
| OpenTelemetry | latest | 链路追踪 |
| Casbin | v2.82+ | RBAC 权限控制 |
| Cron | v3.0+ | 定时任务调度 |
| golang.org/x/sync/singleflight | - | 防缓存击穿 |
| MySQL/PostgreSQL | 8.0+/13+ | 数据库 |
| JWT-Go | v5+ | 身份认证 |
| Validator | v10+ | 参数验证 |

### 2.2 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端层                              │
│                 (Web/Mobile/Desktop)                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway 层                          │
│           (认证/鉴权/限流/统一响应/OpenTelemetry)            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                       业务服务层                             │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐ │
│  │  用户服务   │  项目服务   │  任务服务   │  团队服务   │ │
│  └─────────────┴─────────────┴─────────────┴─────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                       核心组件层                             │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐ │
│  │  缓存管理   │  日志管理   │  配置管理   │  链路追踪   │ │
│  │ (Redis)     │  (Zap)      │  (Viper)    │ (OTel)      │ │
│  └─────────────┴─────────────┴─────────────┴─────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                       数据存储层                             │
│         ┌──────────────────┬──────────────────┐            │
│         │   MySQL/PgSQL    │      Redis       │            │
│         │   (主数据库)      │     (缓存)       │            │
│         └──────────────────┴──────────────────┘            │
└─────────────────────────────────────────────────────────────┘
```


## 3. 功能模块设计

### 3.1 用户管理模块

**功能点**:
- 用户注册/登录（支持邮箱、手机号）
- JWT 身份认证与授权
- 用户信息管理（个人资料、头像、密码修改）
- 基于 Casbin 的 RBAC 权限管理
- 用户状态管理（启用/禁用）
- 动态权限配置和权限继承

**核心接口**:
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/logout` - 用户登出
- `GET /api/v1/users/:id` - 获取用户信息
- `PUT /api/v1/users/:id` - 更新用户信息
- `DELETE /api/v1/users/:id` - 删除用户（软删除）

### 3.2 项目管理模块

**功能点**:
- 项目创建/编辑/删除
- 项目成员管理
- 项目状态跟踪（进行中/已完成/已归档）
- 项目权限控制
- 项目统计分析

**核心接口**:
- `POST /api/v1/projects` - 创建项目
- `GET /api/v1/projects` - 获取项目列表
- `GET /api/v1/projects/:id` - 获取项目详情
- `PUT /api/v1/projects/:id` - 更新项目
- `DELETE /api/v1/projects/:id` - 删除项目
- `POST /api/v1/projects/:id/members` - 添加项目成员
- `DELETE /api/v1/projects/:id/members/:userId` - 移除项目成员

### 3.3 任务管理模块

**功能点**:
- 任务创建/分配/更新
- 任务状态流转（待办/进行中/已完成/已关闭）
- 任务优先级管理
- 任务标签和分类
- 任务评论和附件
- 任务时间追踪

**核心接口**:
- `POST /api/v1/tasks` - 创建任务
- `GET /api/v1/tasks` - 获取任务列表（支持筛选）
- `GET /api/v1/tasks/:id` - 获取任务详情
- `PUT /api/v1/tasks/:id` - 更新任务
- `DELETE /api/v1/tasks/:id` - 删除任务
- `POST /api/v1/tasks/:id/comments` - 添加评论
- `PUT /api/v1/tasks/:id/status` - 更新任务状态

### 3.4 团队管理模块

**功能点**:
- 团队创建/管理
- 团队成员管理
- 团队权限配置
- 团队工作量统计

**核心接口**:
- `POST /api/v1/teams` - 创建团队
- `GET /api/v1/teams` - 获取团队列表
- `GET /api/v1/teams/:id` - 获取团队详情
- `PUT /api/v1/teams/:id` - 更新团队
- `DELETE /api/v1/teams/:id` - 删除团队

## 4. 核心组件设计

### 4.1 缓存管理（Redis + Singleflight）

**缓存策略**:
- **Cache-Aside**: 应用层负责缓存维护
- **过期策略**: 设置合理的 TTL，防止缓存雪崩
- **预热机制**: 系统启动时预加载热点数据
- **更新策略**: 数据变更时主动失效缓存

**防缓存击穿**:
```go
// 使用 singleflight 防止缓存击穿
var sf singleflight.Group

func GetUser(id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)
    
    // 先查缓存
    if val, err := redis.Get(key); err == nil {
        return val, nil
    }
    
    // singleflight 保证同一时间只有一个请求查询数据库
    v, err, _ := sf.Do(key, func() (interface{}, error) {
        user, err := db.GetUser(id)
        if err != nil {
            return nil, err
        }
        // 设置缓存
        redis.Set(key, user, 5*time.Minute)
        return user, nil
    })
    
    return v.(*User), err
}
```

### 4.2 日志管理（Zap + Lumberjack）

**日志级别**:
- **Debug**: 开发调试信息
- **Info**: 一般信息（请求日志、业务流程）
- **Warn**: 警告信息（非致命错误）
- **Error**: 错误信息（需要关注）
- **Fatal**: 致命错误（程序退出）

**日志分割配置**:
```yaml
logger:
  level: info
  filename: logs/qi.log
  max_size: 100        # 单个日志文件最大 100MB
  max_backups: 30      # 保留 30 个备份
  max_age: 7           # 保留 7 天
  compress: true       # 压缩旧日志
```

### 4.3 配置管理（Viper）

**配置文件结构**:
```yaml
server:
  port: 8080
  mode: debug         # debug/release
  read_timeout: 60s
  write_timeout: 60s

database:
  driver: mysql
  host: localhost
  port: 3306
  username: root
  password: password
  database: qi
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600s

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10

jwt:
  secret: your-secret-key
  expire: 7200          # 2小时

tracer:
  enabled: true
  endpoint: localhost:4318
  service_name: qi-service
```

### 4.4 链路追踪（OpenTelemetry）

**追踪范围**:
- HTTP 请求链路
- 数据库查询操作
- 缓存操作
- 外部服务调用

**Span 命名规范**:
- HTTP: `HTTP {METHOD} {PATH}`
- DB: `DB {OPERATION} {TABLE}`
- Cache: `Cache {OPERATION} {KEY}`

### 4.5 定时任务（Cron）

**设计目标**:
- 支持标准 cron 表达式
- 支持秒级精度任务
- 任务执行监控和日志
- 优雅的任务管理

**Cron 表达式格式**:
```
标准格式（5个字段）:
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日 (1 - 31)
│ │ │ ┌───────────── 月 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6) (0表示星期日)
│ │ │ │ │
* * * * *

可选秒字段格式（6个字段）:
┌───────────── 秒 (0 - 59)
│ ┌───────────── 分钟 (0 - 59)
│ │ ┌───────────── 小时 (0 - 23)
│ │ │ ┌───────────── 日 (1 - 31)
│ │ │ │ ┌───────────── 月 (1 - 12)
│ │ │ │ │ ┌───────────── 星期 (0 - 6)
│ │ │ │ │ │
* * * * * *
```

**常用任务示例**:
```go
// 初始化 Cron 调度器
c := cron.New(
    cron.WithSeconds(),              // 启用秒字段
    cron.WithLogger(logger),          // 设置日志
    cron.WithChain(                   // 添加中间件
        cron.Recover(logger),         // panic 恢复
        cron.SkipIfStillRunning(logger), // 跳过仍在运行的任务
    ),
)

// 每分钟执行
c.AddFunc("0 * * * * *", func() {
    log.Println("执行任务：每分钟")
})

// 每天凌晨2点执行
c.AddFunc("0 0 2 * * *", func() {
    log.Println("执行任务：清理过期数据")
})

// 每周一早上9点执行
c.AddFunc("0 0 9 * * 1", func() {
    log.Println("执行任务：生成周报")
})

// 每30分钟执行
c.AddFunc("0 */30 * * * *", func() {
    log.Println("执行任务：更新统计数据")
})

// 启动调度器
c.Start()
defer c.Stop()
```

**预定义任务**:
- 清理过期 Token：每天凌晨执行
- 更新统计数据：每30分钟执行
- 数据备份：每天凌晨2点执行
- 发送通知提醒：根据业务需求配置

### 4.6 权限管理（Casbin）

**设计目标**:
- 灵活的 RBAC 权限模型
- 支持角色继承
- 动态权限配置
- 细粒度权限控制

**Casbin 模型配置** (`configs/casbin/model.conf`):
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

**权限策略示例** (`configs/casbin/policy.csv`):
```csv
p, admin, /api/v1/users, *
p, admin, /api/v1/projects, *
p, admin, /api/v1/tasks, *
p, user, /api/v1/users/:id, GET
p, user, /api/v1/projects, GET
p, user, /api/v1/tasks, GET
p, user, /api/v1/tasks, POST

g, alice, admin
g, bob, user
```

**使用示例**:
```go
// 权限检查
func CheckPermission(enforcer *casbin.Enforcer, user, resource, action string) bool {
    ok, err := enforcer.Enforce(user, resource, action)
    if err != nil {
        return false
    }
    return ok
}

// 权限中间件
func CasbinMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.GetString("username")
        path := c.Request.URL.Path
        method := c.Request.Method
        
        if ok := CheckPermission(enforcer, user, path, method); !ok {
            c.JSON(403, gin.H{"error": "权限不足"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// 动态添加权限
func AddPermission(enforcer *casbin.Enforcer, role, resource, action string) error {
    _, err := enforcer.AddPolicy(role, resource, action)
    return err
}

// 为用户分配角色
func AssignRole(enforcer *casbin.Enforcer, userID, role string) error {
    _, err := enforcer.AddRoleForUser(userID, role)
    return err
}

// 删除用户角色
func RemoveRole(enforcer *casbin.Enforcer, userID, role string) error {
    _, err := enforcer.DeleteRoleForUser(userID, role)
    return err
}

// 获取用户的所有角色
func GetUserRoles(enforcer *casbin.Enforcer, userID string) ([]string, error) {
    return enforcer.GetRolesForUser(userID)
}

// 检查用户是否拥有角色
func HasRole(enforcer *casbin.Enforcer, userID, role string) (bool, error) {
    return enforcer.HasRoleForUser(userID, role)
}
```

**配置示例** (`configs/config.yaml`):
```yaml
casbin:
  # 模型配置文件路径
  model_path: configs/casbin/model.conf
  
  # 策略文件路径
  policy_path: configs/casbin/policy.csv
  
  # 数据库存储（推荐用于生产环境）
  adapter:
    type: gorm  # gorm/file
    db_type: mysql
    dsn: "user:password@tcp(localhost:3306)/casbin?charset=utf8mb4"
    table_name: casbin_rule
  
  # 是否自动加载策略
  auto_load: true
  
  # 策略更新间隔（秒）
  auto_load_interval: 60
```

## 5. API 设计规范

### 5.1 统一响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "trace_id": "xxx"
}
```

### 5.2 错误码设计

```go
const (
    Success         = 200   // 成功
    InvalidParams   = 400   // 参数错误
    Unauthorized    = 401   // 未授权
    Forbidden       = 403   // 禁止访问
    NotFound        = 404   // 资源不存在
    ServerError     = 500   // 服务器错误
    DatabaseError   = 1001  // 数据库错误
    CacheError      = 1002  // 缓存错误
    UserNotFound    = 2001  // 用户不存在
    ProjectNotFound = 3001  // 项目不存在
    TaskNotFound    = 4001  // 任务不存在
)
```

### 5.3 版本控制

所有 API 路径以 `/api/v1` 开头，便于后续版本迭代。

## 6. 性能优化方案

### 6.1 数据库优化
- 合理使用索引，避免全表扫描
- 使用预编译语句，防止 SQL 注入
- 读写分离（主从复制）
- 分表分库（用户量增长后）

### 6.2 缓存优化
- 热点数据缓存（用户信息、项目基础信息）
- 查询结果缓存（列表数据）
- 合理设置缓存过期时间
- 使用 Redis Pipeline 批量操作

### 6.3 并发优化
- 使用 goroutine pool 控制并发数
- 使用 sync.WaitGroup 等待任务完成
- 使用 channel 进行协程间通信
- 合理使用互斥锁保护共享资源

## 7. 安全方案

### 7.1 身份认证
- JWT Token 认证
- Token 刷新机制
- 登录失败限制（防暴力破解）

### 7.2 权限控制
- 基于 Casbin 的 RBAC 权限模型
- 支持角色继承和动态权限配置
- API 级别的权限校验
- 数据级别的权限隔离
- 细粒度资源访问控制

### 7.3 数据安全
- 密码加密存储（bcrypt）
- 敏感数据加密传输（HTTPS）
- SQL 注入防护
- XSS 防护

### 7.4 限流策略
- IP 级别限流
- 用户级别限流
- API 级别限流

## 8. 测试方案

### 8.1 单元测试
- 测试覆盖率目标：>= 80%
- 使用 testify 断言库
- Mock 外部依赖

### 8.2 集成测试
- API 接口测试
- 数据库事务测试
- 缓存一致性测试

### 8.3 压力测试
- 使用 wrk/ab 进行压测
- QPS 目标：>= 5000
- 响应时间：P95 < 100ms

## 9. 开发计划

### Phase 1: 基础架构搭建（2周）
- [x] 项目初始化
- [ ] 核心组件开发
  - [ ] 数据库连接池
  - [ ] Redis 客户端
  - [ ] 日志系统
  - [ ] 配置管理
  - [ ] 链路追踪
  - [ ] 定时任务调度器
- [ ] 中间件开发
  - [ ] 认证中间件
  - [ ] 日志中间件
  - [ ] 跨域中间件
  - [ ] 限流中间件
  - [ ] 恢复中间件
- [ ] 统一响应和错误处理

### Phase 2: 用户模块（1周）
- [ ] 数据库表设计
- [ ] 用户注册/登录
- [ ] JWT 认证
- [ ] 用户信息管理
- [ ] 基于 Casbin 的权限管理

### Phase 3: 项目管理模块（2周）
- [ ] 项目 CRUD
- [ ] 项目成员管理
- [ ] 项目权限控制
- [ ] 项目统计

### Phase 4: 任务管理模块（2周）
- [ ] 任务 CRUD
- [ ] 任务分配与流转
- [ ] 任务评论
- [ ] 任务附件
- [ ] 任务时间追踪

### Phase 5: 团队管理模块（1周）
- [ ] 团队 CRUD
- [ ] 团队成员管理
- [ ] 团队统计

### Phase 6: 测试与优化（1周）
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能测试与优化
- [ ] 安全加固

### Phase 7: 部署上线（1周）
- [ ] Docker 容器化
- [ ] CI/CD 流程
- [ ] 监控告警
- [ ] 文档完善

## 10. 监控与运维

### 10.1 监控指标
- 系统指标：CPU、内存、磁盘、网络
- 应用指标：QPS、响应时间、错误率
- 业务指标：用户数、项目数、任务数

### 10.2 日志收集
- 使用 ELK/Loki 收集日志
- 日志分级存储
- 异常日志告警

### 10.3 链路追踪
- Jaeger/Zipkin 可视化
- 性能瓶颈分析
- 调用链路分析

## 11. 贡献指南

欢迎贡献代码，请遵循以下规范：
- Fork 项目并创建分支
- 遵循代码规范和注释要求
- 编写单元测试
- 提交 Pull Request

## 12. 开源协议

MIT License

## 13. 联系方式

如有问题，请提交 Issue 或联系项目维护者。

