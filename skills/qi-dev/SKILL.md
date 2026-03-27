---
name: qi-dev
description: >-
  qi HTTP 框架开发助手。当用户在 qi 框架项目中执行以下操作时使用：
  新建接口/路由、定义业务错误码、编写中间件、生成 OpenAPI 文档注解、
  编写测试、使用 pkg/* 子包（cache/database/logger/config）。
  关键词：新建接口、添加路由、定义错误、写中间件、qi框架。
---

# qi 框架开发助手

本 Skill 帮助 AI 根据 qi 框架约定生成符合规范的代码。

## 项目基本信息

- 模块路径：`github.com/tokmz/qi`
- Go 1.25+，注释和文档使用**中文**
- 测试仅用标准 `testing` 包，禁止 testify
- 错误使用 `pkg/errors` 不可变克隆链，禁止修改哨兵错误

---

## 一、新建接口（核心模式）

### 1.1 有请求体 — `qi.Bind[Req, Resp]`

适用于 POST / PUT / PATCH，handler 签名固定为 `func(*qi.Context, *Req) (*Resp, error)`。

```go
type CreateUserReq struct {
    Name  string `json:"name"  binding:"required,min=2,max=64" desc:"用户名，2-64个字符" example:"Alice"`
    Email string `json:"email" binding:"required,email"         desc:"邮箱地址"           example:"alice@example.com"`
}

type CreateUserResp struct {
    ID    string `json:"id"    desc:"用户ID"`
    Name  string `json:"name"  desc:"用户名"`
    Email string `json:"email" desc:"邮箱地址"`
}

func createUser(c *qi.Context, req *CreateUserReq) (*CreateUserResp, error) {
    if req.Email == "exists@example.com" {
        return nil, ErrUserExists
    }
    return &CreateUserResp{ID: "1", Name: req.Name, Email: req.Email}, nil
}

// 注册
auth.API().
    POST("/users", qi.Bind(createUser)).
    Summary("创建用户").
    Tags("用户").
    Done()
```

### 1.2 无请求体 — `qi.BindR[Resp]`

适用于 GET，handler 签名为 `func(*qi.Context) (*Resp, error)`。

```go
type ListUsersResp struct {
    Total int64  `json:"total" desc:"总条数"`
    List  []User `json:"list"  desc:"用户列表"`
}

func listUsers(c *qi.Context) (*ListUsersResp, error) {
    users := []User{{ID: "1", Name: "Alice"}}
    return &ListUsersResp{Total: 1, List: users}, nil
}

v1.API().
    GET("/users", qi.BindR(listUsers)).
    Summary("用户列表").
    Tags("用户").
    Done()
```

### 1.3 有请求体、无响应体 — `qi.BindE[Req]`

适用于 DELETE / 无需返回数据的操作，handler 签名为 `func(*qi.Context, *Req) error`。

```go
type DeleteUserReq struct {
    ID string `uri:"id" binding:"required" desc:"用户ID" example:"u-123"`
}

func deleteUser(c *qi.Context, req *DeleteUserReq) error {
    return service.Delete(req.ID)
}

auth.API().
    DELETE("/users/:id", qi.BindE(deleteUser)).
    Summary("删除用户").
    Tags("用户").
    Done()
```

### 1.4 无请求体、无响应体 — `qi.BindRE`

handler 签名为 `func(*qi.Context) error`。

```go
func clearCache(c *qi.Context) error {
    return cache.Flush(c.Request().Context())
}

auth.API().
    POST("/cache/flush", qi.BindRE(clearCache)).
    Summary("清空缓存").
    Tags("运维").
    Done()
```

### 1.5 含路径参数 — URI 绑定

请求结构体含 `uri` tag 时，`Bind`/`BindE` 自动调用 `BindURI`。

```go
type GetUserReq struct {
    ID string `uri:"id" binding:"required"`
}

func getUser(c *qi.Context, req *GetUserReq) (*User, error) {
    // req.ID 已从 /users/:id 路径绑定
    return &User{ID: req.ID, Name: "Alice"}, nil
}

v1.API().
    GET("/users/:id", qi.Bind(getUser)).
    Summary("用户详情").Tags("用户").Done()
```

### 1.6 普通 HandlerFunc（无 OpenAPI 类型推导）

```go
app.DELETE("/users/:id", func(c *qi.Context) {
    id := c.Param("id")
    if id == "" {
        c.Fail(qi.ErrBadRequest)
        return
    }
    c.OK(nil, "删除成功")
})
```

---

## 二、响应方法速查

| 方法 | 说明 |
|------|------|
| `c.OK(data, ...msg)` | 成功，code=0，status=200 |
| `c.Page(total, list)` | 分页响应，data={total,list} |
| `c.Fail(err)` | 失败，从 `errors.Error` 提取 code/status |
| `c.JSON(status, obj)` | 原始 JSON |

使用 `Bind`/`BindR`/`BindE`/`BindRE` 时，框架自动处理响应，handler 只需 `return resp, err` 或 `return err`。

### handler 组合一览

| 函数 | 请求体 | 响应体 | handler 签名 |
|------|--------|--------|-------------|
| `Bind[Req, Resp]` | ✓ | ✓ | `func(*Context, *Req) (*Resp, error)` |
| `BindR[Resp]` | ✗ | ✓ | `func(*Context) (*Resp, error)` |
| `BindE[Req]` | ✓ | ✗ | `func(*Context, *Req) error` |
| `BindRE` | ✗ | ✗ | `func(*Context) error` |

---

## 三、结构体字段 Tag 规范

所有请求/响应结构体字段必须添加描述 tag，框架通过反射读取并生成 OpenAPI 文档。

### 支持的 Tag 一览

| Tag | 作用 | 示例 |
|-----|------|------|
| `desc` | 字段描述（推荐，简写） | `desc:"用户名"` |
| `description` | 字段描述（同 desc，完整写法） | `description:"用户名"` |
| `example` | 示例值（字符串/数字直接写） | `example:"Alice"` / `example:"18"` |
| `default` | 默认值 | `default:"1"` |
| `enum` | 枚举值，逗号分隔（独立 tag） | `enum:"male,female,other"` |
| `format` | 格式约束（OpenAPI format） | `format:"date-time"` |
| `openapi` | 复合标注，`-` 表示忽略该字段 | `openapi:"-"` |
| `binding` | 校验规则，**同时**自动映射 OpenAPI 约束 | `binding:"required,min=2,max=64"` |

### binding tag 自动映射 OpenAPI 约束（无需重复定义）

框架同时解析 `binding` 和 `validate` tag，自动转换为 OpenAPI schema 约束，**不需要额外 tag 重复声明**：

| binding 写法 | 字段类型 | 映射到 OpenAPI |
|---|---|---|
| `min=N` / `max=N` | string | `minLength` / `maxLength` |
| `min=N` / `max=N` | int/float | `minimum` / `maximum` |
| `min=N` / `max=N` | []T / array | `minItems` / `maxItems` |
| `gte=N` / `lte=N` | 数字 | `minimum` / `maximum`（含边界） |
| `gt=N` / `lt=N` | 数字 | `exclusiveMinimum` / `exclusiveMaximum` |
| `oneof=a b c` | 任意 | `enum: [a, b, c]`（**空格**分隔，非逗号） |
| `email` | string | `format: email` |
| `url` | string | `format: uri` |
| `uuid` | string | `format: uuid` |
| `datetime` | string | `format: date-time` |

> 注意：`oneof=` 用**空格**分隔枚举值；独立 `enum` tag 用**逗号**分隔，两者效果相同，选一即可。

### 完整示例

```go
// 请求结构体：min/max/oneof 直接写在 binding 里，框架自动转 OpenAPI 约束
type CreateOrderReq struct {
    UserID    string `json:"user_id"   uri:"user_id"  binding:"required"                    desc:"用户ID"     example:"u-123"`
    ProductID string `json:"product_id"               binding:"required"                    desc:"商品ID"     example:"p-456"`
    Quantity  int    `json:"quantity"                 binding:"required,min=1,max=99"       desc:"购买数量"   example:"2"     default:"1"`
    Remark    string `json:"remark"                   binding:"omitempty,max=200"           desc:"备注（可选）"`
    Channel   string `json:"channel"                  binding:"required,oneof=web app mini" desc:"来源渠道"   example:"web"`
}

// 响应结构体：同样加 desc
type CreateOrderResp struct {
    OrderID   string `json:"order_id"   desc:"订单ID"           example:"o-789"`
    Status    string `json:"status"     desc:"订单状态"         enum:"pending,paid,cancelled"`
    CreatedAt string `json:"created_at" desc:"创建时间(RFC3339)"  format:"date-time"`
    Amount    int64  `json:"amount"     desc:"订单金额（分）"    example:"9900"`
}

// 需要在 OpenAPI 文档中隐藏的内部字段
type InternalResp struct {
    Data   string `json:"data"  desc:"业务数据"`
    RawSQL string `json:"-"     openapi:"-"`  // 不对外暴露
}
```

### 枚举类型写法

```go
type UpdateStatusReq struct {
    Status string `json:"status" binding:"required" desc:"目标状态" enum:"active,inactive,banned" example:"active"`
    Gender int    `json:"gender"                    desc:"性别"    enum:"0,1,2"               example:"1" default:"0"`
}
```

---

## 四、业务错误定义

```go
// 业务包内 errors.go
var (
    // 从 2000+ 开始，避免与框架 1000-1103 冲突
    ErrUserNotFound  = errors.NewWithStatus(2001, http.StatusNotFound, "用户不存在")
    ErrUserExists    = errors.NewWithStatus(2002, http.StatusConflict, "用户已存在")
    ErrPasswordWrong = errors.NewWithStatus(2003, http.StatusUnauthorized, "密码错误")
)

// 克隆链附加上下文（不修改哨兵）
return nil, ErrUserNotFound.
    WithErr(sql.ErrNoRows).
    WithMessage("用户 " + id + " 不存在")
```

框架预定义错误码见 `references/错误码.md`。

---

## 四、路由分组与中间件

```go
// 路由分组
v1 := app.Group("/api/v1")
auth := v1.Group("", authMiddleware())

// 编写中间件
func authMiddleware() qi.HandlerFunc {
    return func(c *qi.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.Fail(qi.ErrUnauthorized)
            c.Abort() // 必须 Abort，否则后续 handler 继续执行
            return
        }
        c.Set("uid", "user-123")
        c.Next()
    }
}
```

### 路由元信息（RouteMeta）

通过 `RouteBuilder` 注册的路由，元信息在运行时可被中间件查询。
key 格式为 `"METHOD:/full/path"`，由框架内部管理，无需手动构造。

```go
// 注册时声明元信息
auth.API().
    DELETE("/users/:id", qi.BindE(deleteUser)).
    Summary("删除用户").
    Tags("用户").
    OperationID("deleteUser").
    Done()

// 操作日志中间件示例
func OperationLogMiddleware(e *qi.Engine) qi.HandlerFunc {
    return func(c *qi.Context) {
        c.Next()
        meta := e.RouteMeta(c.Request().Method, c.FullPath())
        // meta 始终非 nil：RouteBuilder 路由有完整信息；直接注册的路由 Summary 降级为 handlerName
        log.Printf("uid=%s op=%s tags=%v", c.GetString("uid"), meta.Summary, meta.Tags)
    }
}

app.Use(OperationLogMiddleware(app))
```

**RouteMeta 字段：**

| 字段 | 类型 | 来源 |
|------|------|------|
| `Summary` | string | `.Summary("...")` 或降级为 handlerName |
| `Description` | string | `.Description("...")` |
| `Tags` | []string | `.Tags("...")` |
| `OperationID` | string | `.OperationID("...")` |
| `Deprecated` | bool | `.Deprecated()` |

---

## 五、Engine 初始化

```go
app := qi.New(
    qi.WithAddr(":8080"),
    qi.WithMode("debug"), // debug / release / test
    qi.WithLogger(&qi.LoggerConfig{
        SkipPaths: []string{"/ping"},
    }),
    qi.WithTracing(&qi.TracingConfig{
        ServiceName: "my-service",
        Exporter:    qi.TracingExporterOTLPHTTP,
        Endpoint:    "http://127.0.0.1:4318", // https:// → TLS
        SampleRate:  1.0,
        SkipPaths:   []string{"/ping"},
    }),
    qi.WithOpenAPI(&qi.OpenAPIConfig{
        Title:     "My API",
        Version:   "1.0.0",
        SwaggerUI: "/docs",
    }),
)

if err := app.Run(); err != nil {
    log.Fatal(err)
}
```

---

## 六、测试规范

```go
func TestCreateUser(t *testing.T) {
    app := qi.New(qi.WithMode("test"))
    app.POST("/users", qi.Bind(createUser))

    body := `{"name":"Alice","email":"alice@example.com"}`
    req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    app.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("期望 200，实际 %d，body: %s", w.Code, w.Body.String())
    }
}
```

---

## 七、pkg/* 子包速查

详见 `references/API模式.md`。

| 子包 | 初始化 | 关键注意 |
|------|--------|----------|
| `pkg/logger` | `logger.New(*Config)` | 传给 database 实现统一日志 |
| `pkg/database` | `database.New(*Config)` | `ZapLogger` 字段接入日志；`TracingEnabled` 开启 OTel |
| `pkg/cache` | `cache.New(*Config)` | `Flush` 需要 `KeyPrefix`；三种驱动 |
| `pkg/config` | `config.New(*Config)` | viper 封装，支持热重载 |

---

## 八、禁忌清单

1. 不要修改哨兵错误本身，用 `.WithErr()` / `.WithMessage()` 克隆
2. GET 接口用 `BindR`，有请求体才用 `Bind`
3. 中间件拦截请求后必须调用 `c.Abort()`
4. `RouteBuilder` 必须以 `.Done()` 结尾，否则路由不注册
5. 业务错误码从 2000+ 开始，框架占用 1000–1103
6. 测试不用 testify，仅用标准 `testing` 包
7. `internal/*` 不依赖根包，避免循环依赖