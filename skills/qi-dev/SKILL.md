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

适用于 POST / PUT / PATCH（body 方法），handler 签名固定为 `func(*qi.Context, *Req) (*Resp, error)`。
也支持仅含路径参数的 GET/DELETE 场景（自动识别绑定方式）。

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

### 1.3 有请求参数、无响应体 — `qi.BindE[Req]`

适用于 DELETE / 无需返回数据的操作，handler 签名为 `func(*qi.Context, *Req) error`。
支持路径参数（`uri` tag）和 query 参数（`form` tag），以及 body 绑定（POST/PUT）。

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

`Bind`/`BindE` 的自动绑定决策逻辑（按顺序）：

1. **body 方法**（POST/PUT/PATCH）→ 调用 `c.Bind(req)`，gin 自动按 Content-Type 绑定
2. **非 body 方法**（GET/DELETE 等）+ 结构体含 `form` tag → 调用 `c.BindQuery(req)` 绑定 query 参数
3. **非 body 方法** + 无 `form` tag → 跳过 query 绑定
4. **结构体含 `uri` tag** → 调用 `c.BindURI(req)` 绑定路径参数（与上述步骤独立，可同时生效）

> 关键：仅有 `uri` tag 的结构体在 GET/DELETE 中不会触发 BindQuery 校验，`binding:"required"` 只在 BindURI 阶段校验。

```go
type GetUserReq struct {
    ID string `uri:"id" binding:"required" desc:"用户ID" example:"1"`
}

func getUser(c *qi.Context, req *GetUserReq) (*User, error) {
    // req.ID 已从 /users/:id 路径绑定，不会触发 BindQuery 校验
    return &User{ID: req.ID, Name: "Alice"}, nil
}

v1.API().
    GET("/users/:id", qi.Bind(getUser)).
    Summary("用户详情").Tags("用户").Done()
```

> **混合 uri + form 场景**（不推荐，建议拆分为独立结构体）：
>
> 结构体同时含 `uri` 和 `form` tag 时，绑定顺序为 BindQuery → BindURI。
> **BindQuery 会校验所有字段的 `binding` tag**（不仅限于 form 字段），此时 uri 字段尚未绑定值为空，若加了 `binding:"required"` 会导致校验失败，BindURI 根本不会执行。
>
> ```go
> // ❌ 错误：uri 字段加了 binding:"required"，BindQuery 阶段校验失败
> type BadReq struct {
>     ID   string `uri:"id" binding:"required" desc:"用户ID"`
>     Page int    `form:"page"                 desc:"页码"`
> }
>
> // ✅ 方案一：uri 字段不加 binding:"required"（混合场景可工作，但 uri 缺少校验）
> type SearchReq struct {
>     ID   string `uri:"id"    desc:"用户ID"`
>     Page int    `form:"page" desc:"页码" default:"1"`
> }
> // GET /users/:id?page=2 → ID 从路径绑定，Page 从 query 绑定
>
> // ✅ 方案二（推荐）：拆分为两个独立结构体，使用 RouteBuilder 的显式绑定
> type SearchPathReq struct {
>     ID string `uri:"id" binding:"required" desc:"用户ID"`
> }
> type SearchQueryReq struct {
>     Page int `form:"page" desc:"页码" default:"1"`
> }
> ```
>
> **最佳实践：一个请求结构体统一使用一种参数来源**（uri / form / json），避免混合。

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
| `c.FailWithCode(code, status, msg)` | 自定义 code/status，参数顺序：(code, status, msg) |
| `c.JSON(status, obj)` | 原始 JSON（不走统一 Response 封装） |

所有响应（除 `c.JSON`）自动走 `qi.Response` 封装：`{code, message, data, trace_id}`。
`trace_id` 在启用 tracing 中间件时自动填充，也可 `c.Set("trace_id", "xxx")` 手动设置。
panic 恢复同样返回统一 JSON 格式：`{code:1000, message:"server error", data:null}`。

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
    WithMessagef("用户 %s 不存在", id)
```

框架预定义错误码和错误工具函数见 `references/错误码.md`。

---

## 五、路由分组与中间件

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
        uid, _ := c.Get("uid")
        log.Printf("uid=%v op=%s tags=%v", uid, meta.Summary, meta.Tags)
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

## 六、Engine 初始化

```go
app := qi.New(
    qi.WithAddr(":8080"),
    qi.WithMode("debug"), // debug / release / test
    qi.WithLogger(&qi.LoggerConfig{
        // Output: os.Stdout,   // io.Writer，nil 时默认 os.Stdout
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

## 七、测试规范

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

## 八、pkg/* 子包速查

详见 `references/API模式.md`。

| 子包 | 初始化 | 关键注意 |
|------|--------|----------|
| `pkg/logger` | `logger.New(*Config)` | `Level`/`Format` 是自定义类型非字符串；`Console`/`File`/`Rotate` 控制输出；`defer log.Close()` |
| `pkg/database` | `database.New(*Config)` | 必须指定 `Type`（`MySQL`/`Postgres`/`SQLite`/`SQLServer`）；`ZapLogger` 接入日志；`TracingEnabled` 开启 OTel |
| `pkg/cache` | `cache.New(*Config)` | 三种驱动：`DriverMemory`/`DriverRedis`/`DriverMultiLevel`；`DefaultTTL` 非 `TTL`；`Flush` 需要 `KeyPrefix`；`Get` 需要 `dest` 参数 |
| `pkg/config` | `config.New(...Option)` | 函数式 Option 模式（非 `*Config`）；支持热重载和保护模式 |

---

## 九、OpenAPI 文档生成规则

使用 `Bind`/`BindR`/`BindE`/`BindRE` 注册的路由，框架通过反射自动生成 OpenAPI 3.0 文档。无需手动编写 schema。

### 9.1 请求参数的自动推导

框架根据 HTTP 方法和结构体 tag 自动决定参数位置（`in`）：

| 条件 | 参数位置 | 使用的 tag |
|------|---------|-----------|
| POST/PUT/PATCH | `requestBody` | `json` |
| GET/DELETE + 有 `form` tag 的字段 | `query` | `form` |
| GET/DELETE + 无 `form` tag | 不生成 query 参数 | — |
| 有 `uri` tag 的字段 | `path` | `uri` |

> `uri` 绑定与 body/query 绑定**独立**，一个结构体可以同时包含 `uri` 和 `form`/`json` tag。

### 9.2 路径参数（Path Params）

框架自动从路由路径（如 `/users/:id`）提取参数名，生成基础的 PathParams schema（`string` + `required`）。
如果请求结构体包含 `uri` tag，**用真实结构体覆盖**自动生成的 PathParams，保留 `desc`、`example`、`binding` 等信息。

```go
// 路由: GET /users/:id
type GetUserReq struct {
    ID string `uri:"id" binding:"required" desc:"用户ID" example:"1"`
}
// OpenAPI 生成: path 参数 id (string, required, description="用户ID", example="1")
```

### 9.3 查询参数（Query Params）

非 body 方法的请求结构体中，带 `form` tag 的字段自动生成 query 参数。

**关键规则：仅有 `uri` tag 的字段不会出现在 query 参数中**（框架自动过滤，避免与 path 参数重复）。

> **注意**：虽然 OpenAPI 文档能正确区分 uri 和 form 字段，但运行时绑定中混合 uri + form 存在校验陷阱（见 1.5 节和禁忌清单第 4 条）。建议一个结构体只用一种 tag 类型。

```go
// 路由: GET /users/:id/search
type SearchReq struct {
    ID    string `uri:"id"     desc:"用户ID"`
    Query string `form:"q"     desc:"搜索关键词"`
    Page  int    `form:"page"  desc:"页码" default:"1"`
}
// OpenAPI 生成:
//   path 参数: id (来自 uri tag)
//   query 参数: q, page (来自 form tag)
//   注意: id 不会重复出现在 query 参数中
```

### 9.4 请求体（Request Body）

POST/PUT/PATCH 方法的请求结构体整体作为 `requestBody`，使用 `json` tag 控制字段名。

```go
// 路由: POST /users
type CreateUserReq struct {
    Name  string `json:"name"  binding:"required,min=2" desc:"用户名"`
    Email string `json:"email" binding:"required,email" desc:"邮箱"`
}
// OpenAPI 生成: requestBody (application/json) 包含 name 和 email 字段
```

### 9.5 响应自动推导

`Bind`/`BindR` 的响应类型自动生成 200 响应 schema。`BindE`/`BindRE` 无响应体，生成空响应。

### 9.6 字段隐藏

使用 `openapi:"-"` tag 在文档中隐藏字段（数据库内部字段、敏感信息等）：

```go
type UserResp struct {
    Name   string `json:"name"   desc:"用户名"`
    Salt   string `json:"salt"   openapi:"-"` // 不出现在文档中
}
```

### 9.7 约束自动映射

结构体 `binding` tag 中的校验规则自动映射为 OpenAPI schema 约束（详见第三章 tag 规范）。无需重复声明 `minLength`、`maximum` 等约束。

### 9.8 显式覆盖（高级用法）

通过 RouteBuilder 的链式方法可以显式指定 OpenAPI 文档的请求/响应类型，覆盖自动推导：

```go
auth.API().
    POST("/users", qi.Bind(createUser)).
    Summary("创建用户").
    Tags("用户").
    // 显式覆盖（通常不需要，自动推导已足够）
    // .Body(CreateUserReq{}).
    // .Response(CreateUserResp{}).
    Done()
```

优先级：显式 `.Body()`/`.Query()`/`.Response()` > 自动推导。

---

## 十、禁忌清单

1. 不要修改哨兵错误本身，用 `.WithErr()` / `.WithMessage()` / `.WithMessagef()` 克隆
2. GET 接口用 `BindR`（无请求参数）或 `Bind`（有路径/query 参数），有请求体才用 `Bind`
3. 仅含 `uri` tag 的请求结构体（路径参数）不会触发 `BindQuery` 校验，可安全用于 GET/DELETE
4. **禁止在同一结构体中混合 `uri` + `form` 并在 uri 字段上使用 `binding:"required"`**——BindQuery 会校验所有 binding tag，此时 uri 字段尚未绑定，导致必填校验失败。建议一个结构体统一使用一种 tag（uri / form / json）
5. 中间件拦截请求后必须调用 `c.Abort()`
6. `RouteBuilder` 必须以 `.Done()` 结尾，否则路由不注册
7. 业务错误码从 2000+ 开始，框架占用 1000–1103
8. 测试不用 testify，仅用标准 `testing` 包
9. `internal/*` 不依赖根包，避免循环依赖
10. Cache 方法名是 `Del` 不是 `Delete`；`Get` 需要 `dest` 参数不是返回值
11. `pkg/logger` 的 `Level` 和 `Format` 是自定义类型，不是字符串
