package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tokmz/qi"
	"github.com/tokmz/qi/middleware"
	"github.com/tokmz/qi/pkg/errors"
	"github.com/tokmz/qi/pkg/logger"
	"github.com/tokmz/qi/pkg/orm"
	"github.com/tokmz/qi/pkg/tracing"

	"gorm.io/gorm"
)

// User 示例模型
type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// UserResp 用户响应
type UserResp struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// GetUserReq 获取用户请求
type GetUserReq struct {
	ID uint `uri:"id" binding:"required"`
}

var (
	db  *gorm.DB
	log logger.Logger
)

func main() {
	// 1. 初始化链路追踪
	tp, err := tracing.NewTracerProvider(&tracing.Config{
		ServiceName:      "qi-tracing-example",
		ServiceVersion:   "1.0.0",
		Environment:      "development",
		ExporterType:     "stdout", // 使用 stdout 导出器便于演示
		SamplingRate:     1.0,      // 100% 采样
		SamplingType:     "always",
		Enabled:          true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create tracer provider: %v", err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracing.Shutdown(ctx); err != nil {
			fmt.Printf("failed to shutdown tracer provider: %v\n", err)
		}
	}()

	fmt.Printf("TracerProvider initialized: %v\n", tp)

	// 2. 初始化日志
	log, err = logger.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer log.Sync()

	// 3. 初始化数据库（自动启用追踪）
	db, err = orm.New(&orm.Config{
		Type: orm.SQLite,
		DSN:  "file::memory:?cache=shared", // 使用内存数据库
	})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	// 注册 GORM 追踪插件（默认不记录 SQL，避免敏感数据泄露）
	// 开发环境可以启用 SQL 追踪：orm.NewTracingPlugin(orm.WithSQLTrace(true))
	if err := db.Use(orm.NewTracingPlugin()); err != nil {
		panic(fmt.Sprintf("failed to register gorm plugin: %v", err))
	}

	// 自动迁移
	if err := db.AutoMigrate(&User{}); err != nil {
		panic(fmt.Sprintf("failed to migrate database: %v", err))
	}

	// 4. 创建 Qi Engine
	engine := qi.New()

	// 5. 注册全局中间件
	engine.Use(middleware.Tracing(&middleware.TracingConfig{
		Filter: func(c *qi.Context) bool {
			// 过滤健康检查
			return c.Request().URL.Path != "/health"
		},
	}))

	// 6. 注册路由
	r := engine.Router()

	// 健康检查（不追踪）
	r.GET("/health", func(c *qi.Context) {
		c.Success(map[string]string{"status": "ok"})
	})

	// 创建用户（使用泛型路由）
	qi.Handle[CreateUserReq, UserResp](r.POST, "/users", createUserHandler)

	// 获取用户（使用泛型路由）
	qi.Handle[GetUserReq, UserResp](r.GET, "/users/:id", getUserHandler)

	// 列表用户（手动追踪）
	r.GET("/users", listUsersHandler)

	// 7. 启动服务
	fmt.Println("Server starting on :8080")
	fmt.Println("Try:")
	fmt.Println("  curl -X POST http://localhost:8080/users -H 'Content-Type: application/json' -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
	fmt.Println("  curl http://localhost:8080/users/1")
	fmt.Println("  curl http://localhost:8080/users")

	if err := engine.Run(":8080"); err != nil {
		panic(err)
	}
}

// createUserHandler 创建用户处理器
func createUserHandler(c *qi.Context, req *CreateUserReq) (*UserResp, error) {
	ctx := c.RequestContext()

	// 日志自动关联 TraceID 和 SpanID
	log.InfoContext(ctx, "Creating user")

	// 手动创建业务 Span
	ctx, span := tracing.StartSpan(ctx, "validate-user")
	if err := validateUser(req); err != nil {
		tracing.RecordError(span, err)
		span.End()
		return nil, err
	}
	span.End()

	// 数据库操作自动追踪
	user := &User{
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		log.ErrorContext(ctx, "Failed to create user")
		return nil, errors.ErrServer.WithError(err)
	}

	log.InfoContext(ctx, "User created successfully")

	return &UserResp{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// getUserHandler 获取用户处理器
func getUserHandler(c *qi.Context, req *GetUserReq) (*UserResp, error) {
	ctx := c.RequestContext()

	log.InfoContext(ctx, "Fetching user")

	var user User
	if err := db.WithContext(ctx).First(&user, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound.WithMessage("用户不存在")
		}
		return nil, errors.ErrServer.WithError(err)
	}

	return &UserResp{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// listUsersHandler 列表用户处理器（演示手动追踪）
func listUsersHandler(c *qi.Context) {
	ctx := c.RequestContext()

	// 手动创建 Span
	ctx, span := tracing.StartSpan(ctx, "list-users")
	defer span.End()

	log.InfoContext(ctx, "Listing users")

	var users []User
	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		tracing.RecordError(span, err)
		c.RespondError(errors.ErrServer.WithError(err))
		return
	}

	// 添加 Span 属性
	tracing.SetAttributes(span, map[string]any{
		"user.count": len(users),
	})

	// 添加 Span 事件
	tracing.AddEvent(span, "users.fetched", map[string]any{
		"count": len(users),
	})

	resp := make([]UserResp, len(users))
	for i, user := range users {
		resp[i] = UserResp{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		}
	}

	c.Success(resp)
}

// validateUser 验证用户（业务逻辑示例）
func validateUser(req *CreateUserReq) error {
	if len(req.Name) < 2 {
		return errors.ErrBadRequest.WithMessage("用户名至少 2 个字符")
	}
	return nil
}
