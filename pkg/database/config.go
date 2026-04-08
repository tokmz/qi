package database

import (
	"time"

	"go.uber.org/zap"
)

// DBType 数据库类型
type DBType string

const (
	MySQL       DBType = "mysql"
	Postgres DBType = "postgres"
	SQLite      DBType = "sqlite"
	SQLServer   DBType = "sqlserver"
)

type Config struct {
	// 数据库类型
	Type DBType // 数据库类型: mysql, postgres, sqlite, sqlserver

	// 数据库连接配置
	DSN string // 数据源名称 (Data Source Name)

	// 连接池配置
	MaxIdleConns    int           // 最大空闲连接数
	MaxOpenConns    int           // 最大打开连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期
	ConnMaxIdleTime time.Duration // 连接最大空闲时间

	// GORM 配置
	SkipDefaultTransaction bool // 跳过默认事务
	PrepareStmt            bool // 预编译语句
	DisableAutomaticPing   bool // 禁用自动 Ping

	// 日志配置
	LogLevel      int           // 日志级别 (1:Silent 2:Error 3:Warn 4:Info)
	SlowThreshold time.Duration // 慢查询阈值
	Colorful      bool          // 是否彩色输出

	// 命名策略
	TablePrefix   string // 表名前缀
	SingularTable bool   // 使用单数表名

	// 其他配置
	DryRun bool // 空跑模式（生成 SQL 但不执行）

	// 读写分离配置
	ReadWriteSplit *ReadWriteSplitConfig // 读写分离配置（可选）

	// 链路追踪配置
	TracingEnabled bool // 是否启用 OpenTelemetry 链路追踪
	EnableSQLTrace bool // 追踪时是否记录完整 SQL（注意：可能泄露敏感数据）

	// 日志
	ZapLogger *zap.Logger // 传入外部 zap 实例；为 nil 时使用 GORM 默认 logger
}

// ReadWriteSplitConfig 读写分离配置
type ReadWriteSplitConfig struct {
	// 从库配置（读节点）
	Replicas []string // 从库 DSN 列表（只读，对应 dbresolver.Replicas）

	// 负载均衡策略
	Policy string // 负载均衡策略: random(随机), round_robin(轮询)

	// 从库连接池配置（可选，不设置则使用主库配置）
	MaxIdleConns    *int           // 从库最大空闲连接数
	MaxOpenConns    *int           // 从库最大打开连接数
	ConnMaxLifetime *time.Duration // 从库连接最大生命周期
	ConnMaxIdleTime *time.Duration // 从库连接最大空闲时间
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type:                   MySQL,
		MaxIdleConns:           10,
		MaxOpenConns:           100,
		ConnMaxLifetime:        time.Hour,
		ConnMaxIdleTime:        10 * time.Minute,
		SkipDefaultTransaction: false,
		PrepareStmt:            true,
		DisableAutomaticPing:   false,
		LogLevel:               3, // Warn
		SlowThreshold:          200 * time.Millisecond,
		Colorful:               false,
		SingularTable:          false,
		DryRun:                 false,
	}
}
