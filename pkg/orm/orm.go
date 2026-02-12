package orm

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

// New 创建 GORM 数据库实例
func New(cfg *Config) (*gorm.DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.DSN == "" {
		return nil, fmt.Errorf("DSN is required")
	}

	// 创建 GORM 配置
	gormConfig := &gorm.Config{
		SkipDefaultTransaction: cfg.SkipDefaultTransaction,
		PrepareStmt:            cfg.PrepareStmt,
		DisableAutomaticPing:   cfg.DisableAutomaticPing,
		DryRun:                 cfg.DryRun,
		Logger:                 newLogger(cfg),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   cfg.TablePrefix,
			SingularTable: cfg.SingularTable,
		},
	}

	// 根据数据库类型选择驱动
	dialector, err := getDialector(cfg.Type, cfg.DSN)
	if err != nil {
		return nil, err
	}

	// 打开数据库连接
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 配置读写分离
	if cfg.ReadWriteSplit != nil {
		if err := setupReadWriteSplit(db, cfg); err != nil {
			return nil, fmt.Errorf("failed to setup read-write split: %w", err)
		}
	}

	return db, nil
}

// getDialector 根据数据库类型返回对应的 Dialector
func getDialector(dbType DBType, dsn string) (gorm.Dialector, error) {
	switch dbType {
	case MySQL:
		return mysql.Open(dsn), nil
	case PostgreSQL:
		return postgres.Open(dsn), nil
	case SQLite:
		return sqlite.Open(dsn), nil
	case SQLServer:
		return sqlserver.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// newLogger 创建 GORM 日志记录器
func newLogger(cfg *Config) logger.Interface {
	return logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             cfg.SlowThreshold,
			LogLevel:                  logger.LogLevel(cfg.LogLevel),
			IgnoreRecordNotFoundError: true,
			Colorful:                  cfg.Colorful,
		},
	)
}

// setupReadWriteSplit 配置读写分离
func setupReadWriteSplit(db *gorm.DB, cfg *Config) error {
	rwCfg := cfg.ReadWriteSplit
	if len(rwCfg.Sources) == 0 {
		return fmt.Errorf("read-write split enabled but no sources provided")
	}

	// 构建从库 Dialector 列表
	replicas := make([]gorm.Dialector, 0, len(rwCfg.Sources))
	for _, dsn := range rwCfg.Sources {
		dialector, err := getDialector(cfg.Type, dsn)
		if err != nil {
			return err
		}
		replicas = append(replicas, dialector)
	}

	// 构建 DBResolver 配置
	resolverConfig := dbresolver.Config{
		Replicas: replicas,
		Policy:   getLoadBalancePolicy(rwCfg.Policy),
	}

	// 注册 DBResolver 插件
	resolver := dbresolver.Register(resolverConfig)
	if err := db.Use(resolver); err != nil {
		return err
	}

	// 配置从库连接池（如果提供）
	if rwCfg.MaxIdleConns != nil {
		resolver.SetMaxIdleConns(*rwCfg.MaxIdleConns)
	}
	if rwCfg.MaxOpenConns != nil {
		resolver.SetMaxOpenConns(*rwCfg.MaxOpenConns)
	}
	if rwCfg.ConnMaxLifetime != nil {
		resolver.SetConnMaxLifetime(*rwCfg.ConnMaxLifetime)
	}
	if rwCfg.ConnMaxIdleTime != nil {
		resolver.SetConnMaxIdleTime(*rwCfg.ConnMaxIdleTime)
	}

	return nil
}

// getLoadBalancePolicy 获取负载均衡策略
func getLoadBalancePolicy(policy string) dbresolver.Policy {
	switch policy {
	case "random":
		return dbresolver.RandomPolicy{}
	case "round_robin":
		return dbresolver.RoundRobinPolicy()
	default:
		return dbresolver.RandomPolicy{} // 默认随机策略
	}
}

