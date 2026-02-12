# Qi ORM 包

基于 GORM 的数据库 ORM 封装，提供开箱即用的数据库操作能力。

## 特性

- **多数据库支持**：MySQL、PostgreSQL、SQLite、SQLServer
- **连接池管理**：自动配置连接池参数
- **读写分离**：支持主从架构和负载均衡
- **链路追踪**：可选的 OpenTelemetry 集成
- **配置灵活**：支持默认配置和自定义配置

## 快速开始

### 1. 基础使用

```go
import "qi/pkg/orm"

// 使用默认配置
cfg := orm.DefaultConfig()
cfg.DSN = "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"

db, err := orm.New(cfg)
if err != nil {
    panic(err)
}

// 使用数据库
var user User
db.First(&user, 1)
```

### 2. 完整配置

```go
db, err := orm.New(&orm.Config{
    // 数据库类型
    Type: orm.MySQL,
    DSN:  "user:pass@tcp(localhost:3306)/dbname",

    // 连接池配置
    MaxIdleConns:    10,
    MaxOpenConns:    100,
    ConnMaxLifetime: time.Hour,
    ConnMaxIdleTime: 10 * time.Minute,

    // GORM 配置
    SkipDefaultTransaction: false,
    PrepareStmt:            true,
    DisableAutomaticPing:   false,

    // 日志配置
    LogLevel:      3, // 1:Silent 2:Error 3:Warn 4:Info
    SlowThreshold: 200 * time.Millisecond,
    Colorful:      false,

    // 命名策略
    TablePrefix:   "t_",
    SingularTable: false,

    // 其他配置
    DryRun: false,
})
```

## 数据库类型

支持以下数据库类型：

```go
orm.MySQL      // MySQL
orm.PostgreSQL // PostgreSQL
orm.SQLite     // SQLite
orm.SQLServer  // SQL Server
```

### MySQL

```go
db, err := orm.New(&orm.Config{
    Type: orm.MySQL,
    DSN:  "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
})
```

### PostgreSQL

```go
db, err := orm.New(&orm.Config{
    Type: orm.PostgreSQL,
    DSN:  "host=localhost user=postgres password=pass dbname=mydb port=5432 sslmode=disable",
})
```

### SQLite

```go
db, err := orm.New(&orm.Config{
    Type: orm.SQLite,
    DSN:  "test.db",
})
```

### SQL Server

```go
db, err := orm.New(&orm.Config{
    Type: orm.SQLServer,
    DSN:  "sqlserver://user:pass@localhost:1433?database=mydb",
})
```

## 读写分离

支持主从架构的读写分离，自动将读操作路由到从库。

### 基础配置

```go
db, err := orm.New(&orm.Config{
    Type: orm.MySQL,
    DSN:  "user:pass@tcp(master:3306)/db", // 主库

    ReadWriteSplit: &orm.ReadWriteSplitConfig{
        Sources: []string{
            "user:pass@tcp(slave1:3306)/db", // 从库1
            "user:pass@tcp(slave2:3306)/db", // 从库2
        },
        Policy: "round_robin", // 负载均衡策略
    },
})
```

### 负载均衡策略

- `random` - 随机选择从库
- `round_robin` - 轮询选择从库

### 独立连接池配置

可以为从库配置独立的连接池参数：

```go
maxIdle := 5
maxOpen := 50
lifetime := 30 * time.Minute

ReadWriteSplit: &orm.ReadWriteSplitConfig{
    Sources: []string{"user:pass@tcp(slave:3306)/db"},
    Policy:  "random",

    // 从库独立连接池配置
    MaxIdleConns:    &maxIdle,
    MaxOpenConns:    &maxOpen,
    ConnMaxLifetime: &lifetime,
}
```

## 链路追踪

集成 OpenTelemetry 实现数据库操作的自动追踪。

### 启用追踪

```go
import "qi/pkg/orm"

db, err := orm.New(&orm.Config{
    Type: orm.MySQL,
    DSN:  "user:pass@tcp(localhost:3306)/db",
})

// 注册追踪插件（默认不记录 SQL，避免敏感数据泄露）
if err := db.Use(orm.NewTracingPlugin()); err != nil {
    panic(err)
}
```

### 启用 SQL 追踪

**⚠️ 警告：** 完整 SQL 可能包含敏感数据（密码、身份证号等），生产环境请谨慎启用。

```go
// 开发环境：启用 SQL 追踪便于调试
db.Use(orm.NewTracingPlugin(orm.WithSQLTrace(true)))
```

### 追踪信息

插件会自动记录以下信息：

- **db.system**: 数据库系统（gorm）
- **db.operation**: 操作类型（gorm.Create/Query/Update/Delete/Row/Raw）
- **db.table**: 表名
- **db.rows_affected**: 影响行数
- **db.statement**: SQL 语句（仅在启用 `WithSQLTrace(true)` 时）
- **error**: 错误信息（如果有）

### 使用示例

```go
// 确保传递 context
ctx := c.RequestContext() // 从 Qi Context 获取

// 数据库操作会自动追踪
db.WithContext(ctx).Where("id = ?", userID).First(&user)
db.WithContext(ctx).Create(&user)
db.WithContext(ctx).Model(&user).Update("name", "new_name")
```

## 配置说明

### Config 结构

```go
type Config struct {
    // 数据库类型
    Type DBType // mysql, postgres, sqlite, sqlserver

    // 数据库连接配置
    DSN string // 数据源名称

    // 连接池配置
    MaxIdleConns    int           // 最大空闲连接数（默认 10）
    MaxOpenConns    int           // 最大打开连接数（默认 100）
    ConnMaxLifetime time.Duration // 连接最大生命周期（默认 1 小时）
    ConnMaxIdleTime time.Duration // 连接最大空闲时间（默认 10 分钟）

    // GORM 配置
    SkipDefaultTransaction bool // 跳过默认事务（默认 false）
    PrepareStmt            bool // 预编译语句（默认 true）
    DisableAutomaticPing   bool // 禁用自动 Ping（默认 false）

    // 日志配置
    LogLevel      int           // 日志级别（默认 3:Warn）
    SlowThreshold time.Duration // 慢查询阈值（默认 200ms）
    Colorful      bool          // 是否彩色输出（默认 false）

    // 命名策略
    TablePrefix   string // 表名前缀
    SingularTable bool   // 使用单数表名（默认 false）

    // 其他配置
    DryRun bool // 空跑模式（默认 false）

    // 读写分离配置
    ReadWriteSplit *ReadWriteSplitConfig // 读写分离配置（可选）
}
```

### 默认配置

```go
cfg := orm.DefaultConfig()
// Type: MySQL
// MaxIdleConns: 10
// MaxOpenConns: 100
// ConnMaxLifetime: 1 hour
// ConnMaxIdleTime: 10 minutes
// PrepareStmt: true
// LogLevel: 3 (Warn)
// SlowThreshold: 200ms
```

## 最佳实践

### 1. 连接池配置

根据应用负载调整连接池参数：

```go
// 低负载应用
MaxIdleConns: 5
MaxOpenConns: 25

// 中等负载应用
MaxIdleConns: 10
MaxOpenConns: 100

// 高负载应用
MaxIdleConns: 20
MaxOpenConns: 200
```

### 2. 慢查询监控

设置合理的慢查询阈值：

```go
SlowThreshold: 200 * time.Millisecond // 生产环境
SlowThreshold: 100 * time.Millisecond // 性能敏感应用
```

### 3. 预编译语句

生产环境建议启用预编译语句：

```go
PrepareStmt: true // 提升性能，防止 SQL 注入
```

### 4. 事务管理

高并发场景可以跳过默认事务：

```go
SkipDefaultTransaction: true // 提升性能
```

### 5. 读写分离

主从延迟敏感的场景，可以强制使用主库：

```go
// 强制使用主库
db.Clauses(dbresolver.Write).First(&user)

// 强制使用从库
db.Clauses(dbresolver.Read).Find(&users)
```

### 6. Context 传递

始终传递 context 以支持超时控制和链路追踪：

```go
// ✅ 正确
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
db.WithContext(ctx).Find(&users)

// ❌ 错误
db.Find(&users) // 缺少 context
```

## 故障排查

### 1. 连接失败

检查 DSN 格式和网络连接：

```bash
# MySQL
mysql -h localhost -u user -p

# PostgreSQL
psql -h localhost -U postgres -d mydb
```

### 2. 连接池耗尽

增加最大连接数或检查连接泄漏：

```go
MaxOpenConns: 200 // 增加最大连接数

// 检查是否正确关闭连接
sqlDB, _ := db.DB()
stats := sqlDB.Stats()
fmt.Printf("Open: %d, InUse: %d, Idle: %d\n",
    stats.OpenConnections, stats.InUse, stats.Idle)
```

### 3. 慢查询

启用详细日志定位问题：

```go
LogLevel: 4 // Info 级别，记录所有 SQL
```

### 4. 读写分离不生效

确保使用 `WithContext` 传递 context：

```go
// ✅ 正确
db.WithContext(ctx).Find(&users) // 会路由到从库

// ❌ 错误
db.Find(&users) // 不会路由到从库
```

## 完整示例

参考 `example/tracing/main.go` 查看完整示例代码。

## 相关文档

- [GORM 官方文档](https://gorm.io/docs/)
- [链路追踪文档](../tracing/README.md)
