package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tokmz/qi"
	"github.com/tokmz/qi/pkg/errors"
	"github.com/tokmz/qi/pkg/openapi"
)

func main() {
	// 创建 Engine（带默认中间件：Logger + Recovery）
	// 启用 OpenAPI 文档自动生成 + Swagger UI
	engine := qi.Default(
		qi.WithOpenAPI(&openapi.Config{
			Title:       "Qi Example API",
			Version:     "1.0.0",
			Description: "Qi 框架示例 API 文档",
			Path:        "/openapi.json",
			SwaggerUI:   "/docs",
			SecuritySchemes: map[string]openapi.SecurityScheme{
				"BearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		}),
	)
	r := engine.Router()

	// 注册全局中间件
	engine.Use(traceMiddleware)

	// ============ 基础路由示例 ============
	r.GET("/ping", func(c *qi.Context) {
		c.Success("pong")
	})
	r.DocRoute("GET", "/ping", openapi.Doc(
		openapi.Summary("健康检查"),
		openapi.Tags("System"),
	))

	// 手动绑定参数（绑定失败时自动响应错误）
	r.POST("/manual", func(c *qi.Context) {
		var req CreateUserReq
		if err := c.BindJSON(&req); err != nil {
			return // 绑定失败已自动响应错误
		}
		c.Success(&UserResp{ID: 1, Name: req.Name, Email: req.Email})
	})
	r.DocRoute("POST", "/manual", openapi.Doc(
		openapi.Summary("手动绑定示例"),
		openapi.Tags("System"),
		openapi.RequestType(CreateUserReq{}),
		openapi.ResponseType(UserResp{}),
	))

	// ============ 泛型路由 + OpenAPI 文档 ============

	// 有请求有响应
	qi.POST[CreateUserReq, UserResp](r, "/user", createUserHandler,
		openapi.Doc(openapi.Summary("创建用户"), openapi.Tags("Users")),
	)

	// 有请求无响应（删除操作）
	qi.DELETE0[DeleteUserReq](r, "/user/:id", deleteUserHandler,
		openapi.Doc(openapi.Summary("删除用户"), openapi.Tags("Users")),
	)

	// 无请求有响应（查询操作）
	qi.GETOnly[InfoResp](r, "/info", infoHandler,
		openapi.Doc(openapi.Summary("系统信息"), openapi.Tags("System")),
	)

	// GET 请求自动绑定 Query 参数
	qi.GET[ListUserReq, ListUserResp](r, "/users", listUsersHandler,
		openapi.Doc(openapi.Summary("用户列表"), openapi.Tags("Users")),
	)

	// ============ 路由组示例 ============

	// API v1 路由组（带认证 + Tag）
	v1 := r.Group("/api/v1")
	v1.SetTag("V1", "V1 版本接口")
	v1.SetSecurity("BearerAuth")
	v1.Use(authMiddleware) // 路由组级别中间件

	// 登录接口：NoSecurity 覆盖组级认证
	qi.POST[LoginReq, TokenResp](v1, "/login", loginHandler,
		openapi.Doc(openapi.Summary("用户登录"), openapi.NoSecurity()),
	)
	qi.GETOnly[UserResp](v1, "/profile", profileHandler,
		openapi.Doc(openapi.Summary("获取个人信息")),
	)
	qi.PUT[UpdateProfileReq, UserResp](v1, "/profile", updateProfileHandler,
		openapi.Doc(openapi.Summary("更新个人信息")),
	)

	// API v2 路由组（嵌套路由组）
	v2 := r.Group("/api/v2")
	v2.SetTag("V2", "V2 版本接口")
	v2.SetSecurity("BearerAuth")
	v2.Use(authMiddleware)

	// v2 下的用户管理子组
	userGroup := v2.Group("/users")
	qi.POST[CreateUserReq, UserResp](userGroup, "", createUserHandler,
		openapi.Doc(openapi.Summary("创建用户")),
	)
	qi.GETOnly[ListUserResp](userGroup, "", listUsersHandlerV2,
		openapi.Doc(openapi.Summary("用户列表")),
	)
	qi.GETOnly[UserResp](userGroup, "/:id", getUserHandler,
		openapi.Doc(openapi.Summary("获取用户详情")),
	)
	qi.PUT[UpdateUserReq, UserResp](userGroup, "/:id", updateUserHandler,
		openapi.Doc(openapi.Summary("更新用户")),
	)
	qi.DELETE0[DeleteUserReq](userGroup, "/:id", deleteUserHandler,
		openapi.Doc(openapi.Summary("删除用户")),
	)

	// ============ 错误处理示例 ============
	r.GET("/error/bad-request", func(c *qi.Context) {
		c.RespondError(errors.ErrBadRequest.WithMessage("参数错误"))
	})

	r.GET("/error/unauthorized", func(c *qi.Context) {
		c.RespondError(errors.ErrUnauthorized)
	})

	r.GET("/error/custom", func(c *qi.Context) {
		c.RespondError(errors.New(2001, 403, "自定义错误", nil))
	})

	// ============ 分页响应示例 ============
	r.GET("/page", func(c *qi.Context) {
		users := []UserResp{
			{ID: 1, Name: "Alice", Email: "alice@example.com"},
			{ID: 2, Name: "Bob", Email: "bob@example.com"},
		}
		c.Page(users, 100)
	})

	// ============ 上下文辅助方法示例 ============
	r.GET("/context", func(c *qi.Context) {
		c.Success(map[string]any{
			"trace_id": qi.GetContextTraceID(c),
			"uid":      qi.GetContextUid(c),
			"language": qi.GetContextLanguage(c),
		})
	})

	// ============ 静态文件服务示例 ============
	// r.Static("/static", "./public")
	// r.StaticFile("/favicon.ico", "./public/favicon.ico")

	// 启动服务器
	// GET /openapi.json → OpenAPI 3.0 spec
	// GET /docs          → Swagger UI
	log.Println("Server starting on :8080")
	if err := engine.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// ============ 数据结构定义 ============

type CreateUserReq struct {
	Name  string `json:"name" binding:"required" desc:"用户名"`
	Email string `json:"email" binding:"required,email" desc:"邮箱"`
}

type UserResp struct {
	ID    int64  `json:"id" desc:"用户ID"`
	Name  string `json:"name" desc:"用户名"`
	Email string `json:"email" desc:"邮箱"`
}

type DeleteUserReq struct {
	ID int64 `uri:"id" binding:"required,min=1" desc:"用户ID"`
}

type InfoResp struct {
	Version   string `json:"version" desc:"版本号"`
	BuildTime string `json:"build_time" desc:"构建时间"`
}

type ListUserReq struct {
	Page int `form:"page" binding:"required,min=1" desc:"页码"`
	Size int `form:"size" binding:"required,min=1,max=100" desc:"每页数量"`
}

type ListUserResp struct {
	List  []UserResp `json:"list" desc:"用户列表"`
	Total int64      `json:"total" desc:"总数"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required" desc:"用户名"`
	Password string `json:"password" binding:"required,min=6" desc:"密码"`
}

type TokenResp struct {
	Token     string `json:"token" desc:"访问令牌"`
	ExpiresAt int64  `json:"expires_at" desc:"过期时间戳"`
}

type UpdateProfileReq struct {
	Name  string `json:"name" binding:"required" desc:"用户名"`
	Email string `json:"email" binding:"required,email" desc:"邮箱"`
}

type UpdateUserReq struct {
	ID    int64  `uri:"id" binding:"required,min=1" desc:"用户ID"`
	Name  string `json:"name" binding:"required" desc:"用户名"`
	Email string `json:"email" binding:"required,email" desc:"邮箱"`
}

// ============ 中间件实现 ============

// traceMiddleware 链路追踪中间件
func traceMiddleware(c *qi.Context) {
	// 生成或从请求头获取 TraceID
	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	qi.SetContextTraceID(c, traceID)

	// 设置响应头
	c.Header("X-Trace-ID", traceID)

	c.Next()
}

// authMiddleware 认证中间件
func authMiddleware(c *qi.Context) {
	// 从请求头获取 token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.RespondError(errors.ErrUnauthorized.WithMessage("缺少认证令牌"))
		c.Abort()
		return
	}

	// 验证 token（这里简化处理）
	if token != "Bearer valid-token" {
		c.RespondError(errors.ErrUnauthorized.WithMessage("无效的认证令牌"))
		c.Abort()
		return
	}

	// 设置用户信息到上下文
	qi.SetContextUid(c, 12345)
	qi.SetContextLanguage(c, "zh-CN")

	c.Next()
}

// ============ 业务处理函数 ============

func createUserHandler(_ *qi.Context, req *CreateUserReq) (*UserResp, error) {
	// 业务逻辑：创建用户
	if req.Name == "admin" {
		return nil, errors.New(2001, 403, "禁止使用保留用户名", nil)
	}

	return &UserResp{
		ID:    time.Now().Unix(),
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func deleteUserHandler(_ *qi.Context, req *DeleteUserReq) error {
	// 业务逻辑：删除用户
	if req.ID == 1 {
		return errors.New(2002, 403, "禁止删除管理员账户", nil)
	}

	log.Printf("User %d deleted", req.ID)
	return nil
}

func infoHandler(_ *qi.Context) (*InfoResp, error) {
	return &InfoResp{
		Version:   "1.0.0",
		BuildTime: time.Now().Format(time.RFC3339),
	}, nil
}

func listUsersHandler(_ *qi.Context, req *ListUserReq) (*ListUserResp, error) {
	// 模拟分页查询
	users := []UserResp{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
	}

	// 简单分页逻辑
	start := (req.Page - 1) * req.Size
	end := start + req.Size
	if start >= len(users) {
		return &ListUserResp{List: []UserResp{}, Total: int64(len(users))}, nil
	}
	if end > len(users) {
		end = len(users)
	}

	return &ListUserResp{
		List:  users[start:end],
		Total: int64(len(users)),
	}, nil
}

func listUsersHandlerV2(_ *qi.Context) (*ListUserResp, error) {
	// V2 版本：返回更多用户
	users := []UserResp{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
		{ID: 4, Name: "David", Email: "david@example.com"},
		{ID: 5, Name: "Eve", Email: "eve@example.com"},
	}

	return &ListUserResp{
		List:  users,
		Total: int64(len(users)),
	}, nil
}

func loginHandler(_ *qi.Context, req *LoginReq) (*TokenResp, error) {
	// 验证用户名密码
	if req.Username != "admin" || req.Password != "123456" {
		return nil, errors.ErrUnauthorized.WithMessage("用户名或密码错误")
	}

	// 生成 token（这里简化处理）
	return &TokenResp{
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

func profileHandler(c *qi.Context) (*UserResp, error) {
	// 从上下文获取用户 ID
	uid := qi.GetContextUid(c)

	return &UserResp{
		ID:    uid,
		Name:  "Admin User",
		Email: "admin@example.com",
	}, nil
}

func updateProfileHandler(c *qi.Context, req *UpdateProfileReq) (*UserResp, error) {
	// 从上下文获取用户 ID
	uid := qi.GetContextUid(c)

	return &UserResp{
		ID:    uid,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func getUserHandler(c *qi.Context) (*UserResp, error) {
	// 从 URI 获取用户 ID
	var req DeleteUserReq
	if err := c.BindURI(&req); err != nil {
		return nil, err
	}

	return &UserResp{
		ID:    req.ID,
		Name:  "User " + fmt.Sprint(req.ID),
		Email: fmt.Sprintf("user%d@example.com", req.ID),
	}, nil
}

func updateUserHandler(_ *qi.Context, req *UpdateUserReq) (*UserResp, error) {
	return &UserResp{
		ID:    req.ID,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}
