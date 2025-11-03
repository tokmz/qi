package tracing_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"qi/internal/core/tracing"
	"qi/internal/middleware"
)

// Example_basic 基础使用示例
func Example_basic() {
	// 1. 创建配置
	cfg := &tracing.Config{
		Enabled:        true,
		ServiceName:    "example-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Sampler: tracing.SamplerConfig{
			Type:  "always_on",
			Ratio: 1.0,
		},
		Exporter: tracing.ExporterConfig{
			Type: "stdout",
			Stdout: tracing.StdoutConfig{
				PrettyPrint: true,
			},
		},
	}

	// 2. 初始化全局 Tracer
	if err := tracing.InitGlobal(cfg); err != nil {
		log.Fatal(err)
	}

	// 3. 使用 Tracer
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "example-operation")
	defer tracing.EndSpan(span)

	// 4. 添加属性
	tracing.SetAttributes(ctx,
		tracing.UserIDKey.String("user-123"),
		attribute.String("custom-key", "custom-value"),
	)

	// 5. 执行业务逻辑
	doWork(ctx)

	// 6. 关闭 Tracer
	tracer := tracing.GetGlobal()
	_ = tracer.Shutdown(context.Background())
}

func doWork(ctx context.Context) {
	// 创建子 span
	_, span := tracing.StartSpan(ctx, "do-work")
	defer tracing.EndSpan(span)

	// 模拟工作
	time.Sleep(100 * time.Millisecond)

	fmt.Println("Work completed")
}

// Example_ginMiddleware Gin 中间件使用示例
func Example_ginMiddleware() {
	// 初始化 Tracer
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	// 创建 Gin 应用
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 使用链路追踪中间件
	r.Use(middleware.TracingMiddleware("my-api"))

	// 定义路由
	r.GET("/users/:id", getUserHandler)
	r.POST("/users", createUserHandler)

	// 启动服务器
	go r.Run(":8080")

	// 模拟请求
	time.Sleep(100 * time.Millisecond)
	resp, err := http.Get("http://localhost:8080/users/123")
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %d\n", resp.StatusCode)
}

func getUserHandler(c *gin.Context) {
	// 从 context 获取 trace ID
	traceID := middleware.GetTraceIDFromGin(c)
	log.Printf("Processing request with trace ID: %s", traceID)

	// 创建子 span
	ctx := c.Request.Context()
	ctx, span := tracing.StartSpan(ctx, "getUserFromDB",
		tracing.WithSpanKind(trace.SpanKindClient),
	)
	defer tracing.EndSpan(span)

	// 模拟数据库查询
	userID := c.Param("id")
	tracing.SetAttributes(ctx, tracing.UserIDKey.String(userID))

	c.JSON(200, gin.H{
		"id":   userID,
		"name": "Alice",
	})
}

func createUserHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// 使用 SpanWrapper 简化代码
	err := tracing.SpanWrapper(ctx, "createUser", func(ctx context.Context) error {
		// 业务逻辑
		return saveUserToDB(ctx)
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"message": "User created"})
}

func saveUserToDB(ctx context.Context) error {
	_, span := tracing.StartSpan(ctx, "db.insert.users",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			tracing.DBAttributes("mysql", "mydb", "INSERT", "users", "INSERT INTO users (name) VALUES (?)")...,
		),
	)
	defer tracing.EndSpan(span)

	// 模拟数据库操作
	time.Sleep(50 * time.Millisecond)

	return nil
}

// Example_errorHandling 错误处理示例
func Example_errorHandling() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "error-example")
	defer tracing.EndSpan(span)

	// 调用可能失败的操作
	if err := operationThatMightFail(ctx); err != nil {
		// 记录错误到 span
		tracing.RecordError(ctx, err)
		tracing.SetSpanStatus(ctx, codes.Error, "Operation failed")
		log.Printf("Error: %v", err)
	} else {
		tracing.SetSpanStatus(ctx, codes.Ok, "Success")
	}
}

func operationThatMightFail(ctx context.Context) error {
	_, span := tracing.StartSpan(ctx, "risky-operation")
	defer tracing.EndSpan(span)

	// 模拟错误
	return errors.New("something went wrong")
}

// Example_httpClient HTTP 客户端调用示例
func Example_httpClient() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()

	// 发送 HTTP 请求
	if err := callRemoteAPI(ctx, "https://api.example.com/data"); err != nil {
		log.Printf("Error: %v", err)
	}
}

func callRemoteAPI(ctx context.Context, url string) error {
	// 创建 span
	ctx, span := tracing.StartSpan(ctx, "http.client.call",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			tracing.HTTPMethodKey.String("GET"),
			tracing.HTTPURLKey.String(url),
		),
	)
	defer tracing.EndSpan(span)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		tracing.RecordError(ctx, err)
		return err
	}

	// 注入 trace context 到请求头
	headers := make(map[string]string)
	tracing.InjectHTTPHeaders(ctx, headers)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		tracing.RecordError(ctx, err)
		return err
	}
	defer resp.Body.Close()

	// 记录响应状态
	tracing.SetAttributes(ctx, tracing.HTTPStatusCodeKey.Int(resp.StatusCode))

	if resp.StatusCode >= 400 {
		tracing.SetSpanStatus(ctx, codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	return nil
}

// Example_databaseQuery 数据库查询示例
func Example_databaseQuery() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()

	// 执行查询
	users, _ := queryUsers(ctx, "active")
	fmt.Printf("Found %d users\n", len(users))
}

type User struct {
	ID   int
	Name string
}

func queryUsers(ctx context.Context, status string) ([]User, error) {
	// 创建数据库 span
	ctx, span := tracing.StartSpan(ctx, "db.query.users",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			tracing.DBSystemKey.String("mysql"),
			tracing.DBNameKey.String("mydb"),
			tracing.DBOperationKey.String("SELECT"),
			tracing.DBTableKey.String("users"),
			tracing.DBStatementKey.String("SELECT * FROM users WHERE status = ?"),
			attribute.String("db.status_filter", status),
		),
	)
	defer tracing.EndSpan(span)

	// 模拟数据库查询
	time.Sleep(50 * time.Millisecond)

	// 添加结果信息
	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	tracing.SetAttributes(ctx,
		attribute.Int("db.rows_returned", len(users)),
	)

	return users, nil
}

// Example_cacheOperation 缓存操作示例
func Example_cacheOperation() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()

	// 从缓存获取数据
	value, hit := getFromCache(ctx, "user:123")
	if hit {
		fmt.Printf("Cache hit: %s\n", value)
	} else {
		fmt.Println("Cache miss")
	}
}

func getFromCache(ctx context.Context, key string) (string, bool) {
	// 创建缓存 span
	ctx, span := tracing.StartSpan(ctx, "cache.get",
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			tracing.CacheSystemKey.String("redis"),
			tracing.CacheKeyKey.String(key),
		),
	)
	defer tracing.EndSpan(span)

	// 模拟缓存查询
	hit := false // 假设未命中
	var value string

	// 更新 hit 属性
	tracing.SetAttributes(ctx, tracing.CacheHitKey.Bool(hit))

	if !hit {
		// 如果未命中，从数据库加载
		value = loadFromDB(ctx, key)
		saveToCache(ctx, key, value)
	}

	return value, hit
}

func loadFromDB(ctx context.Context, key string) string {
	_, span := tracing.StartSpan(ctx, "db.load")
	defer tracing.EndSpan(span)

	time.Sleep(50 * time.Millisecond)
	return "user-data"
}

func saveToCache(ctx context.Context, key, value string) {
	_, span := tracing.StartSpan(ctx, "cache.set",
		tracing.WithAttributes(
			tracing.CacheSystemKey.String("redis"),
			tracing.CacheKeyKey.String(key),
		),
	)
	defer tracing.EndSpan(span)

	time.Sleep(10 * time.Millisecond)
}

// Example_asyncOperation 异步操作示例
func Example_asyncOperation() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "async-example")
	defer tracing.EndSpan(span)

	// 使用 AsyncSpanWrapper
	tracing.AsyncSpanWrapper(ctx, "background-task", func(ctx context.Context) {
		// 异步任务逻辑
		time.Sleep(100 * time.Millisecond)
		log.Println("Background task completed")
	})

	// 主流程继续
	log.Println("Main flow continues")

	time.Sleep(200 * time.Millisecond)
}

// Example_customAttributes 自定义属性示例
func Example_customAttributes() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "custom-attributes-example")
	defer tracing.EndSpan(span)

	// 添加各种类型的属性
	tracing.SetAttributes(ctx,
		attribute.String("string.key", "value"),
		attribute.Int("int.key", 123),
		attribute.Int64("int64.key", 456),
		attribute.Float64("float64.key", 78.9),
		attribute.Bool("bool.key", true),
		attribute.StringSlice("string_array.key", []string{"a", "b", "c"}),
	)

	// 添加事件
	tracing.AddEvent(ctx, "custom-event",
		attribute.String("event.type", "user_action"),
		attribute.String("action", "button_click"),
	)
}

// Example_spanWrapper SpanWrapper 使用示例
func Example_spanWrapper() {
	cfg := tracing.DefaultConfig()
	cfg.Exporter.Type = "stdout"
	tracing.InitGlobal(cfg)

	ctx := context.Background()

	// 使用 SpanWrapper 简化代码
	err := tracing.SpanWrapper(ctx, "wrapped-operation", func(ctx context.Context) error {
		// 业务逻辑
		return performWork(ctx)
	}, tracing.WithAttributes(
		attribute.String("operation.type", "business"),
	))

	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func performWork(ctx context.Context) error {
	time.Sleep(100 * time.Millisecond)
	return nil
}
