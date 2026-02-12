package orm

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	gormTracerName = "qi.gorm"
)

// TracingPlugin GORM 链路追踪插件
type TracingPlugin struct {
	tracerName     string
	enableSQLTrace bool // 是否记录完整 SQL（默认 false，避免敏感数据泄露）
}

// TracingOption 追踪插件选项
type TracingOption func(*TracingPlugin)

// WithSQLTrace 启用 SQL 语句追踪（注意：可能泄露敏感数据）
func WithSQLTrace(enable bool) TracingOption {
	return func(p *TracingPlugin) {
		p.enableSQLTrace = enable
	}
}

// NewTracingPlugin 创建 GORM 追踪插件
func NewTracingPlugin(opts ...TracingOption) *TracingPlugin {
	p := &TracingPlugin{
		tracerName:     gormTracerName,
		enableSQLTrace: false, // 默认不记录 SQL
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Name 插件名称
func (p *TracingPlugin) Name() string {
	return "otelgorm"
}

// Initialize 初始化插件
func (p *TracingPlugin) Initialize(db *gorm.DB) error {
	if err := p.registerCallbacks(db); err != nil {
		return fmt.Errorf("failed to register callbacks: %w", err)
	}
	return nil
}

// registerCallbacks 注册 GORM 回调
func (p *TracingPlugin) registerCallbacks(db *gorm.DB) error {
	// Create
	if err := db.Callback().Create().Before("gorm:create").Register("otelgorm:before_create", p.before("gorm.Create")); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register("otelgorm:after_create", p.after()); err != nil {
		return err
	}

	// Query
	if err := db.Callback().Query().Before("gorm:query").Register("otelgorm:before_query", p.before("gorm.Query")); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("otelgorm:after_query", p.after()); err != nil {
		return err
	}

	// Update
	if err := db.Callback().Update().Before("gorm:update").Register("otelgorm:before_update", p.before("gorm.Update")); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("otelgorm:after_update", p.after()); err != nil {
		return err
	}

	// Delete
	if err := db.Callback().Delete().Before("gorm:delete").Register("otelgorm:before_delete", p.before("gorm.Delete")); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("otelgorm:after_delete", p.after()); err != nil {
		return err
	}

	// Row
	if err := db.Callback().Row().Before("gorm:row").Register("otelgorm:before_row", p.before("gorm.Row")); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register("otelgorm:after_row", p.after()); err != nil {
		return err
	}

	// Raw
	if err := db.Callback().Raw().Before("gorm:raw").Register("otelgorm:before_raw", p.before("gorm.Raw")); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("otelgorm:after_raw", p.after()); err != nil {
		return err
	}

	return nil
}

// before 创建 before 回调
func (p *TracingPlugin) before(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		ctx := db.Statement.Context
		if ctx == nil {
			ctx = context.Background()
		}

		// 每次回调时获取 tracer，避免 Provider 后初始化导致使用 noop
		tracer := otel.Tracer(p.tracerName)
		ctx, _ = tracer.Start(ctx, operation,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("db.system", "gorm"),
				attribute.String("db.operation", operation),
			),
		)

		db.Statement.Context = ctx
	}
}

// after 创建 after 回调
func (p *TracingPlugin) after() func(*gorm.DB) {
	return func(db *gorm.DB) {
		span := trace.SpanFromContext(db.Statement.Context)
		if !span.IsRecording() {
			return
		}
		defer span.End()

		// 记录 SQL 语句（仅在启用时）
		if p.enableSQLTrace && db.Statement.SQL.String() != "" {
			span.SetAttributes(attribute.String("db.statement", db.Statement.SQL.String()))
		}

		// 记录表名
		if db.Statement.Table != "" {
			span.SetAttributes(attribute.String("db.table", db.Statement.Table))
		}

		// 记录影响行数
		if db.Statement.RowsAffected >= 0 {
			span.SetAttributes(attribute.Int64("db.rows_affected", db.Statement.RowsAffected))
		}

		// 记录错误
		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			span.RecordError(db.Error)
			span.SetStatus(codes.Error, db.Error.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
