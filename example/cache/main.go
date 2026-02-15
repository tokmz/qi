package main

import (
	"fmt"
	"log"
	"time"

	"qi"
	"qi/pkg/cache"
)

// User 用户模型
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// 模拟数据库
var userDB = map[int64]*User{
	1: {ID: 1, Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
	2: {ID: 2, Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
	3: {ID: 3, Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
}

func main() {
	// 创建缓存实例（使用内存缓存）
	c, err := cache.NewWithOptions(
		cache.WithMemory(&cache.MemoryConfig{
			DefaultExpiration: 10 * time.Minute,
			CleanupInterval:   5 * time.Minute,
		}),
		cache.WithKeyPrefix("myapp:"),
		cache.WithDefaultTTL(10*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer c.Close()

	// 创建 Qi Engine
	engine := qi.Default()
	r := engine.Router()

	// 示例 1: 基础缓存使用
	r.GET("/user/:id", func(ctx *qi.Context) {
		id := ctx.Param("id")
		key := "user:" + id

		// 尝试从缓存获取
		var user User
		err := c.Get(ctx.RequestContext(), key, &user)
		if err == nil {
			ctx.Success(map[string]any{
				"user":   user,
				"cached": true,
			})
			return
		}

		// 缓存未命中，从数据库查询
		userID := int64(0)
		fmt.Sscanf(id, "%d", &userID)
		dbUser, exists := userDB[userID]
		if !exists {
			ctx.Fail(404, "User not found")
			return
		}

		// 写入缓存
		_ = c.Set(ctx.RequestContext(), key, dbUser, 10*time.Minute)

		ctx.Success(map[string]any{
			"user":   dbUser,
			"cached": false,
		})
	})

	// 示例 2: Remember 模式（自动缓存）
	r.GET("/user/:id/remember", func(ctx *qi.Context) {
		id := ctx.Param("id")
		key := "user:" + id

		// Remember 模式：自动处理缓存逻辑
		user, err := cache.Remember(ctx.RequestContext(), c, key, 10*time.Minute, func() (*User, error) {
			userID := int64(0)
			fmt.Sscanf(id, "%d", &userID)
			dbUser, exists := userDB[userID]
			if !exists {
				return nil, fmt.Errorf("user not found")
			}
			return dbUser, nil
		})

		if err != nil {
			ctx.Fail(404, err.Error())
			return
		}

		ctx.Success(user)
	})

	// 示例 3: 泛型 API
	r.GET("/user/:id/typed", func(ctx *qi.Context) {
		id := ctx.Param("id")
		key := "user:" + id

		// 使用泛型 API（类型安全）
		user, err := cache.GetTyped[User](ctx.RequestContext(), c, key)
		if err == nil {
			ctx.Success(user)
			return
		}

		// 缓存未命中
		userID := int64(0)
		fmt.Sscanf(id, "%d", &userID)
		dbUser, exists := userDB[userID]
		if !exists {
			ctx.Fail(404, "User not found")
			return
		}

		// 使用泛型 Set
		_ = cache.SetTyped(ctx.RequestContext(), c, key, *dbUser, 10*time.Minute)
		ctx.Success(dbUser)
	})

	// 示例 4: 批量操作
	r.GET("/users/batch", func(ctx *qi.Context) {
		// 批量设置
		items := map[string]any{
			"user:1": userDB[1],
			"user:2": userDB[2],
			"user:3": userDB[3],
		}
		_ = c.MSet(ctx.RequestContext(), items, 10*time.Minute)

		// 批量获取
		var users []User
		keys := []string{"user:1", "user:2", "user:3"}
		err := c.MGet(ctx.RequestContext(), keys, &users)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		ctx.Success(users)
	})

	// 示例 5: 计数器（原子操作）
	r.POST("/page/:id/view", func(ctx *qi.Context) {
		id := ctx.Param("id")
		key := "page_views:" + id

		// 增加浏览次数
		count, err := c.Incr(ctx.RequestContext(), key)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		ctx.Success(map[string]any{
			"page_id": id,
			"views":   count,
		})
	})

	// 示例 6: TTL 管理
	r.GET("/cache/:key/ttl", func(ctx *qi.Context) {
		key := ctx.Param("key")

		ttl, err := c.TTL(ctx.RequestContext(), key)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		ctx.Success(map[string]any{
			"key": key,
			"ttl": ttl.String(),
		})
	})

	// 示例 7: 删除缓存
	r.DELETE("/cache/:key", func(ctx *qi.Context) {
		key := ctx.Param("key")

		err := c.Delete(ctx.RequestContext(), key)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		ctx.Success(map[string]any{
			"message": "Cache deleted successfully",
			"key":     key,
		})
	})

	// 示例 8: 检查缓存是否存在
	r.GET("/cache/:key/exists", func(ctx *qi.Context) {
		key := ctx.Param("key")

		exists, err := c.Exists(ctx.RequestContext(), key)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		ctx.Success(map[string]any{
			"key":    key,
			"exists": exists,
		})
	})

	// 示例 9: 健康检查
	r.GET("/cache/health", func(ctx *qi.Context) {
		err := c.Ping(ctx.RequestContext())
		if err != nil {
			ctx.Fail(500, "Cache is unhealthy")
			return
		}

		ctx.Success(map[string]any{
			"status": "healthy",
		})
	})

	// 启动服务器
	fmt.Println("Server starting on :8080")
	fmt.Println("Example endpoints:")
	fmt.Println("  GET    /user/:id              - Basic cache usage")
	fmt.Println("  GET    /user/:id/remember     - Remember pattern")
	fmt.Println("  GET    /user/:id/typed        - Generic API")
	fmt.Println("  GET    /users/batch           - Batch operations")
	fmt.Println("  POST   /page/:id/view         - Counter (atomic)")
	fmt.Println("  GET    /cache/:key/ttl        - Get TTL")
	fmt.Println("  DELETE /cache/:key            - Delete cache")
	fmt.Println("  GET    /cache/:key/exists     - Check existence")
	fmt.Println("  GET    /cache/health          - Health check")

	if err := engine.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
