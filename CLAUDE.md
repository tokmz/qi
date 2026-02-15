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

**Note:** No test files exist yet. No Makefile present - use standard Go toolchain.

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

**Context** (`context.go`)
- Wraps `gin.Context` to provide enhanced API
- **Critical:** All `Bind*()` methods automatically respond with 400 errors on failure
- Developers only need to check `err != nil` and `return` - no manual error response needed
- Unified response methods: `Success()`, `Fail()`, `RespondError()`, `Page()`
- Automatic TraceID injection into all responses

**Router** (`router.go`)
- Wraps `gin.RouterGroup`
- Standard HTTP methods support variadic middleware: `r.GET(path, handler, mw1, mw2...)`
- **Generic routing functions** for automatic binding + response handling:
  - `Handle[Req, Resp]()` - Request + Response
  - `Handle0[Req]()` - Request only (no response data)
  - `HandleOnly[Resp]()` - Response only (no request data)

**Auto-Binding Strategy** (`autoBind` function in router.go)
- **GET/DELETE**: `ShouldBindQuery()` + `ShouldBindUri()` (URI errors ignored)
- **POST/PUT/PATCH**: `ShouldBind()` (auto-detects Content-Type) + `ShouldBindUri()`
- **Others**: `ShouldBind()` (fallback)

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

### Response Format

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
return nil, errors.New(2001, 403, "禁止访问", nil)
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
├── pkg/
│   └── errors/            # Error handling package
│       ├── errors.go      # Error type + utilities
│       ├── custom.go      # Predefined errors
│       └── README.md      # Error package docs
├── utils/                 # Utility packages
│   ├── array/             # Slice utilities
│   ├── convert/           # Type conversion
│   ├── datetime/          # Time utilities
│   ├── pointer/           # Pointer helpers
│   ├── regexp/            # Regex utilities
│   └── strings/           # String utilities
└── example/
    └── main.go            # Comprehensive example (334 lines)
```

## Recent Development Context

Recent commits show:
- Removed i18n and logger packages (moved out of core)
- Fixed i18n package design issues
- Enhanced generic routing with middleware support

The framework is production-ready for small-to-medium Go web services that want Gin's performance with better developer experience.
