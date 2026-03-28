package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/tokmz/qi"
	"github.com/tokmz/qi/pkg/errors"
	"go.uber.org/zap"
)

// ===== 业务错误定义 =====

var (
	ErrUserNotFound = errors.NewWithStatus(2001, http.StatusNotFound, "user not found")
	ErrUserExists   = errors.NewWithStatus(2002, http.StatusConflict, "user already exists")
)

// ===== 数据模型 =====

type User struct {
	ID    string `json:"id"    desc:"用户ID"`
	Name  string `json:"name"  desc:"用户名"`
	Email string `json:"email" desc:"邮箱地址"`
}

type CreateUserReq struct {
	Name  string `json:"name"  binding:"required,min=2,max=64" desc:"用户名，2-64个字符" example:"Alice"`
	Email string `json:"email" binding:"required,email"         desc:"邮箱地址"           example:"alice@example.com"`
}

type ListUsersResp struct {
	Total int64  `json:"total" desc:"总条数"   example:"2"`
	List  []User `json:"list"  desc:"用户列表"`
}

type DeleteUserReq struct {
	ID string `uri:"id" binding:"required" desc:"用户ID" example:"1"`
}

// ===== Handler 实现 =====

func listUsers(c *qi.Context) (*ListUsersResp, error) {
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com"},
		{ID: "2", Name: "Bob", Email: "bob@example.com"},
	}
	return &ListUsersResp{Total: int64(len(users)), List: users}, nil
}

func getUser(c *qi.Context, req *DeleteUserReq) (*User, error) {
	if req.ID != "1" {
		return nil, ErrUserNotFound
	}
	return &User{ID: req.ID, Name: "Alice", Email: "alice@example.com"}, nil
}

func createUser(c *qi.Context, req *CreateUserReq) (*User, error) {
	if req.Email == "exists@example.com" {
		return nil, ErrUserExists
	}
	return &User{ID: "3", Name: req.Name, Email: req.Email}, nil
}

func deleteUser(c *qi.Context, req *DeleteUserReq) error {
	if req.ID == "" {
		return qi.ErrBadRequest
	}
	return nil
}

// ===== 中间件示例 =====

func authMiddleware() qi.HandlerFunc {
	return func(c *qi.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.Fail(qi.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("uid", "user-123")
		c.Next()
	}
}

// ===== 主函数 =====

func main() {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer zapLogger.Sync()

	app := qi.New(
		qi.WithAddr(":8080"),
		qi.WithMode("debug"),

		// 请求日志
		qi.WithLogger(&qi.LoggerConfig{
			SkipPaths: []string{"/ping"},
		}),

		// 链路追踪（OTLP HTTP 导出到 4318）
		qi.WithTracing(&qi.TracingConfig{
			ServiceName: "example-service",
			Exporter:    qi.TracingExporterOTLPHTTP,
			Endpoint:    "http://127.0.0.1:4318",
			SampleRate:  1.0,
			SkipPaths:   []string{"/ping"},
		}),

		// OpenAPI 文档（访问 http://localhost:8080/docs/）
		qi.WithOpenAPI(&qi.OpenAPIConfig{
			Title:       "Example API",
			Version:     "1.0.0",
			Description: "qi 框架示例",
			SwaggerUI:   "/docs",
		}),
	)

	// 健康检查（跳过日志和追踪）
	app.GET("/ping", func(c *qi.Context) { c.OK("pong") })

	v1 := app.Group("/api/v1")

	// 公开接口
	v1.API().
		GET("/users", qi.BindR(listUsers)).
		Summary("获取用户列表").
		Tags("用户").
		Done()

	v1.API().
		GET("/users/:id", qi.Bind(getUser)).
		Summary("获取用户详情").
		Tags("用户").
		Done()

	// 需要鉴权的接口
	auth := v1.Group("", authMiddleware())

	auth.API().
		POST("/users", qi.Bind(createUser)).
		Summary("创建用户").
		Tags("用户").
		Done()

	auth.API().
		DELETE("/users/:id", qi.BindE(deleteUser)).
		Summary("删除用户").
		Tags("用户").
		Done()

	// 分页响应示例
	v1.GET("/items", func(c *qi.Context) {
		items := []map[string]any{
			{"id": 1, "name": "item-1"},
			{"id": 2, "name": "item-2"},
		}
		c.Page(100, items)
	})

	// 自定义错误示例
	v1.GET("/error-demo", func(c *qi.Context) {
		c.Fail(ErrUserNotFound.
			WithErr(sql.ErrNoRows).
			WithMessage("演示自定义错误消息"),
		)
	})

	if err := app.Run(); err != nil {
		zapLogger.Fatal("server exited", zap.Error(err))
	}
}
