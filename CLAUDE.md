# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Qi** is a lightweight Go web framework built on top of Gin, providing enhanced developer experience through unified JSON responses, automatic parameter binding, and generic routing with type safety.

**Key Stats:**
- ~985 lines of core code
- Go 1.25+ (requires generics support)
- Single dependency: Gin v1.11.0
- Module path: `github.com/tokmz/qi`

## Build & Development Commands

```bash
# Build the project
go build

# Run the example server
go run example/main.go

# Build specific packages
go build ./pkg/errors/

# Vet code
go vet ./...

# Format code
go fmt ./...

# Update dependencies
go mod tidy
```

**Note:** Test files exist for core i18n integration (`i18n_test.go`) and `pkg/i18n/` package. No Makefile present - use standard Go toolchain.

## Architecture Overview

### Request Flow

```
HTTP Request
    ↓
Engine (http.Server wrapper)
    ↓
Gin Engine (underlying router)
    ↓
Middleware Chain (global → group → route-specific)
    ↓
Wrapper Layer (gin.Context → qi.Context)
    ↓
Handler (with automatic binding & response)
    ↓
Response (unified JSON format with TraceID)
```

### Core Components

**Engine** (`engine.go`)
- Wraps `gin.Engine` and `http.Server`
- Manages lifecycle with graceful shutdown
- Monitors SIGINT/SIGTERM signals automatically
- Configuration via Options pattern
- Stores `i18n.Translator` when i18n is enabled via `WithI18n()`

**Context** (`context.go`, `i18n.go`)
- Wraps `gin.Context` to provide enhanced API
- **Critical:** All `Bind*()` methods automatically respond with 400 errors on failure
- Developers only need to check `err != nil` and `return` - no manual error response needed
- Unified response methods: `Success()`, `Fail()`, `RespondError()`, `Page()`
- Automatic TraceID injection into all responses
- `T()`/`Tn()` convenience methods for i18n translation (defined in `i18n.go`)

**Router** (`router.go`)
- Wraps `gin.RouterGroup`
- Standard HTTP methods support variadic middleware: `r.GET(path, handler, mw1, mw2...)`
- **Generic routing functions** for automatic binding + response handling:
  - `Handle[Req, Resp]()` - Request + Response
  - `Handle0[Req]()` - Request only (no response data)
  - `HandleOnly[Resp]()` - Response only (no request data)

**Auto-Binding Strategy** (`autoBind` function in router.go)
- **GET/DELETE**: `ShouldBindUri()` + `ShouldBindQuery()` (URI first to avoid validation blocking)
- **POST/PUT/PATCH**: `ShouldBind()` (auto-detects Content-Type) + `ShouldBindUri()`
- **Others**: `ShouldBind()` (fallback)
- **Note**: URI binding errors are ignored for GET/DELETE (routes may not have URI params)

### Middleware Chain

**Execution Order:**
1. Global middleware (via `engine.Use()`)
2. Route group middleware (via `group.Use()`)
3. Route-specific middleware (passed as variadic args)
4. Handler function

**Wrapper Pattern** (`wrapper.go`):
- `HandlerFunc` type: `func(*Context)` (not `func(*gin.Context)`)
- `wrap()` converts `qi.HandlerFunc` → `gin.HandlerFunc`
- Creates new `qi.Context` wrapper for each request
- Middleware must call `c.Next()` to continue chain

### Error Handling System (`pkg/errors/`)

**Custom Error Type:**
```go
type Error struct {
    Code     int    // Business error code (1000-9999)
    HttpCode int    // HTTP status code (200, 400, 401, etc.)
    Message  string // Error message
    Err      error  // Wrapped original error
}
```

**Predefined Errors:**
- `ErrServer` - 1000, HTTP 500
- `ErrBadRequest` - 1001, HTTP 400
- `ErrUnauthorized` - 1002, HTTP 401
- `ErrForbidden` - 1003, HTTP 403
- `ErrNotFound` - 1004, HTTP 404

**Error Flow:**
1. Handler returns `error`
2. `RespondError()` checks if it's `*errors.Error`
3. Maps to appropriate HTTP status code
4. Returns JSON with business code + message

### HTTP Client (`pkg/request/`)

Chainable HTTP client built on `net/http.Client`, with retry, interceptors, and OpenTelemetry tracing.

**Core Types:**
- `Client` — wraps `http.Client`, created via `New(opts...)` or `NewWithConfig(cfg)`
- `Request` — chainable builder: `SetHeader`, `SetQuery`, `SetBody`, `SetBearerToken`, `SetTimeout`, `SetRetry`, etc.
- `Response` — wraps status code, headers, body bytes, duration. Methods: `IsSuccess()`, `IsError()`, `Unmarshal()`, `String()`
- `Interceptor` — interface with `BeforeRequest` / `AfterResponse` hooks
- `Logger` — minimal interface (`InfoContext` / `ErrorContext` with `keysAndValues ...any`), nil by default

**Generic Response Parsing (package-level functions):**
```go
user, err := request.Do[User](client.Post("/users").SetBody(&req))
items, err := request.DoList[Item](client.Get("/items"))
```

**Retry:**
- Exponential backoff with ±25% jitter
- Default condition: network error or 5xx
- `SetBody` JSON body supports replay across retries; `SetRawBody` does not
- Per-request override via `SetRetry()`

**Interceptors:**
- `NewLoggingInterceptor(log)` — logs request/response via `Logger` interface
- `NewAuthInterceptor(tokenFunc)` — dynamic Bearer token injection

**Tracing:**
- `WithTracing(true)` enables OTel client spans + W3C header propagation
- Uses `tracingTransport` at RoundTripper layer

**Error Codes (4000 series):**
- `ErrRequestFailed` (4001) / `ErrTimeout` (4002) / `ErrMarshal` (4003) / `ErrUnmarshal` (4004) / `ErrMaxRetry` (4005) / `ErrInvalidURL` (4006)

**Key Design Decisions:**
- `bodyBytes []byte` caching for retry replay (not `io.Reader`)
- `SetBody` marshal errors deferred to `Do()` via `Request.err`
- `mergeHeaders` returns new map — never mutates `Request.headers`
- `RetryConfig` value-copied before `normalize()` — never mutates caller's config
- `Do[T]` / `DoList[T]` check HTTP status before unmarshal; error body truncated to 512 bytes
- Zero dependency on `zap` / `pkg/logger` — own `Logger` interface in `logger.go`



**Standard Response:**
```json
{
  "code": 200,
  "data": {...},
  "message": "success",
  "trace_id": "trace-1234567890"
}
```

**Pagination Response:**
```json
{
  "code": 200,
  "data": {
    "list": [...],
    "total": 100
  },
  "message": "success"
}
```

## Key Architectural Decisions

1. **Wrapper Pattern Over Inheritance**
   - Wraps Gin types instead of embedding
   - Provides controlled API surface
   - Allows seamless Gin upgrades

2. **Generic Routing for Type Safety**
   - Leverages Go 1.18+ generics
   - Eliminates manual type assertions
   - Automatic binding + response handling
   - Reduces boilerplate by ~70%

3. **Auto-Response on Binding Errors**
   - All `Bind*()` methods automatically respond with 400 errors
   - Developers just check `err != nil` and `return`
   - Simplifies error handling in handlers

4. **Separation of Business Code and HTTP Status**
   - Business error codes (1000-9999) separate from HTTP status codes
   - Single error can map to appropriate HTTP status
   - Frontend gets consistent error format

5. **Graceful Shutdown Built-in**
   - No external libraries needed
   - Signal handling (SIGINT/SIGTERM) automatic
   - Lifecycle callbacks for cleanup
   - Timeout-based forced shutdown

6. **TraceID Auto-Injection**
   - Middleware sets TraceID in context
   - Automatically added to all responses
   - No manual tracking needed

7. **Built-in i18n Integration**
   - Enabled via `WithI18n()` option — nil means disabled
   - Engine auto-initializes translator and registers language detection middleware
   - Language detection priority: `Query(lang)` > `X-Language` header > `Accept-Language` header > default
   - `Context.T()`/`Tn()` for direct translation in handlers
   - **Auto-translation in error responses**: `RespondError()` automatically translates i18n keys in error messages
   - Falls back to returning key when i18n is not enabled

8. **Automatic Translator Injection**
   - Translator automatically injected into all routes and middlewares
   - No manual middleware registration needed
   - Available in Context via `GetContextTranslator()`
   - Seamless integration with error handling system

## Common Patterns

### Basic Route
```go
r.GET("/ping", func(c *qi.Context) {
    c.Success("pong")
})
```

### Manual Binding (with auto-error-response)
```go
r.POST("/user", func(c *qi.Context) {
    var req CreateUserReq
    if err := c.BindJSON(&req); err != nil {
        return // Auto-responded with 400, just return
    }
    c.Success(&UserResp{ID: 1, Name: req.Name})
})
```

### Generic Route (Auto-binding)
```go
qi.Handle[CreateUserReq, UserResp](r.POST, "/user",
    func(c *qi.Context, req *CreateUserReq) (*UserResp, error) {
        return &UserResp{ID: 1, Name: req.Name}, nil
    })
```

### Route Groups with Middleware
```go
v1 := r.Group("/api/v1")
v1.Use(authMiddleware)
qi.Handle[LoginReq, TokenResp](v1.POST, "/login", loginHandler)
```

### Middleware with Generic Routes
```go
qi.Handle[CreateUserReq, UserResp](r.POST, "/admin/user",
    createUserHandler,
    authMiddleware,
    adminMiddleware)
```

### Error Handling
```go
import "qi/pkg/errors"

// Use predefined errors
return nil, errors.ErrBadRequest.WithMessage("用户名不能为空")

// Custom errors
return nil, errors.New(2001, "禁止访问", 403)
```

### Error Handling with i18n
```go
// Define error messages in locales/zh-CN.json
{
  "error.invalid_username": "用户名格式不正确",
  "error.user_not_found": "用户不存在"
}

// Use i18n key in error message
return nil, errors.ErrBadRequest.WithMessage("error.invalid_username")

// RespondError automatically translates based on request language
// Chinese request: {"code": 1001, "message": "用户名格式不正确"}
// English request: {"code": 1001, "message": "Invalid username format"}
```

### i18n Translation
```go
// Enable i18n via config
engine := qi.New(
    qi.WithI18n(&i18n.Config{
        Dir:             "./locales",
        DefaultLanguage: "zh-CN",
        Languages:       []string{"zh-CN", "en-US"},
    }),
)

// Use in handler — language auto-detected from request
r.GET("/hello", func(c *qi.Context) {
    msg := c.T("hello", "Name", "World")
    c.Success(msg)
})

// Plural form
msg := c.Tn("item_one", "item_other", count)
```

## Important Notes

### Gin Mode is Global State
`gin.SetMode()` is a global operation. Only create one Engine instance per process:

```go
// ✅ Recommended: Single instance
func main() {
    engine := qi.New(qi.WithMode(gin.ReleaseMode))
    setupRoutes(engine)
    engine.Run(":8080")
}

// ❌ Avoid: Multiple engines in same process
func main() {
    engine1 := qi.New(qi.WithMode(gin.ReleaseMode))
    engine2 := qi.New(qi.WithMode(gin.DebugMode))  // May affect engine1
}
```

### Context Helpers
```go
// TraceID
qi.SetContextTraceID(c, "trace-123")
traceID := qi.GetContextTraceID(c)

// User UID
qi.SetContextUid(c, 12345)
uid := qi.GetContextUid(c)

// Language
qi.SetContextLanguage(c, "zh-CN")
lang := qi.GetContextLanguage(c)
```

### Configuration Options
```go
qi.New(
    qi.WithMode(gin.ReleaseMode),
    qi.WithAddr(":8080"),
    qi.WithReadTimeout(10 * time.Second),
    qi.WithWriteTimeout(10 * time.Second),
    qi.WithShutdownTimeout(30 * time.Second),
    qi.WithBeforeShutdown(func() { /* cleanup */ }),
    qi.WithAfterShutdown(func() { /* finalize */ }),
    qi.WithTrustedProxies("127.0.0.1"),
    qi.WithMaxMultipartMemory(32 << 20),
    qi.WithI18n(&i18n.Config{
        Dir:             "./locales",
        DefaultLanguage: "zh-CN",
        Languages:       []string{"zh-CN", "en-US"},
    }),
)
```

## Quick Reference for Future Claude Instances

1. **This is a Gin wrapper** - All Gin knowledge applies
2. **Use generic routes** - `Handle[Req, Resp]()` for 70% less boilerplate
3. **Binding auto-responds** - Just check `err != nil` and return
4. **Middleware is variadic** - Pass after handler: `r.GET(path, handler, mw1, mw2)`
5. **Errors map to HTTP status** - Use `errors.ErrBadRequest.WithMessage()`
6. **TraceID is automatic** - Set in middleware, auto-injected in responses
7. **Options pattern for config** - `qi.New(qi.WithAddr(":8080"), ...)`
8. **Graceful shutdown built-in** - Just call `engine.Run()`
10. **HTTP client via `pkg/request`** - `request.New(request.WithBaseURL(...))`, chainable `Do[T]()` generics

## Project Structure

```
/Users/aikzy/Desktop/project/qi/
├── README.md              # Comprehensive documentation (Chinese)
├── go.mod                 # Module definition
├── engine.go              # Engine + lifecycle management
├── context.go             # Enhanced context wrapper
├── router.go              # Router + generic handlers
├── response.go            # Response structures
├── config.go              # Configuration + options
├── wrapper.go             # Gin wrapper functions
├── helper.go              # Context helper functions
├── i18n.go                # i18n middleware + Context.T()/Tn() methods
├── i18n_test.go           # i18n integration tests
├── pkg/
│   ├── errors/            # Error handling package
│   │   ├── errors.go      # Error type + utilities
│   │   ├── custom.go      # Predefined errors
│   │   └── README.md      # Error package docs
│   ├── request/           # HTTP client package
│   │   ├── logger.go      # Logger interface (minimal, no zap dep)
│   │   ├── errors.go      # Error definitions (4000 series)
│   │   ├── config.go      # Config + Option functions
│   │   ├── retry.go       # RetryConfig, exponential backoff
│   │   ├── interceptor.go # Interceptor interface + built-in impls
│   │   ├── transport.go   # tracingTransport (OTel propagation)
│   │   ├── multipart.go   # File upload / multipart builder
│   │   ├── response.go    # Response wrapper, Do[T](), DoList[T]()
│   │   ├── request.go     # Request chainable builder
│   │   ├── client.go      # Client core
│   │   ├── request_test.go # Unit tests
│   │   └── README.md      # Package docs
│   └── i18n/              # Internationalization package
│       ├── translator.go  # Translator interface + implementation
│       ├── config.go      # i18n config + options
│       ├── loader.go      # JSON file loader
│       ├── helper.go      # Context language helpers
│       ├── errors.go      # i18n error definitions
│       └── i18n_test.go   # i18n package tests
├── utils/                 # Utility packages
│   ├── array/             # Slice utilities
│   ├── convert/           # Type conversion
│   ├── datetime/          # Time utilities
│   ├── pointer/           # Pointer helpers
│   ├── regexp/            # Regex utilities
│   └── strings/           # String utilities
└── example/
    └── main.go            # Comprehensive example
```

## Recent Development Context

Recent commits show:
- **v1.0.9**: 实现 i18n 错误消息自动翻译 + 代码重构优化（消除代码重复，提升安全性和性能）
- **v1.0.8**: Fixed GET/DELETE URI parameter binding order (URI before Query to prevent validation blocking)
- Added `pkg/request/` HTTP client package (chainable API, generics, retry, interceptors, OTel tracing)
- Integrated i18n into framework core (`WithI18n` option, auto middleware, `Context.T()`/`Tn()`)
- Added `pkg/i18n/` package (Translator, JSONLoader, lazy loading, plural support)
- Enhanced generic routing with middleware support
- Version: v1.0.9

The framework is production-ready for small-to-medium Go web services that want Gin's performance with better developer experience.
