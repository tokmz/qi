# Errors Package

统一的错误处理包，提供结构化的错误定义和处理能力。

## 功能特性

- 结构化错误定义，包含错误码、HTTP状态码和错误信息
- 支持错误链，可以包装原始错误
- 提供常用的预定义错误
- 兼容标准库 `errors` 包的 `Is` 和 `As` 方法
- **并发安全**：`WithError` 和 `WithMessage` 方法返回新实例，不修改原对象

## 错误结构

```go
type Error struct {
    Code     int    // 业务错误码
    HttpCode int    // HTTP状态码
    Message  string // 错误信息
    Err      error  // 原始错误（可选）
}
```

## 使用方法

### 创建自定义错误

```go
import "qi/pkg/errors"

// 创建新错误
err := errors.New(2001, 400, "用户名不能为空", nil)

// 包装已有错误
originalErr := someFunction()
err := errors.New(2002, 500, "数据库操作失败", originalErr)
```

### 使用预定义错误

包提供了常用的预定义错误：

```go
// 服务器错误 (1000, 500)
errors.ErrServer

// 客户端请求错误 (1001, 400)
errors.ErrBadRequest

// 未授权 (1002, 401)
errors.ErrUnauthorized

// 禁止访问 (1003, 403)
errors.ErrForbidden

// 资源不存在 (1004, 404)
errors.ErrNotFound
```

### 链式调用

**重要提示**：`WithError` 和 `WithMessage` 方法会返回新的错误实例，不会修改原对象，因此在高并发场景下使用预定义错误是安全的。

```go
// 添加原始错误（返回新实例）
err := errors.ErrServer.WithError(dbErr)

// 修改错误信息（返回新实例）
err := errors.ErrBadRequest.WithMessage("用户ID格式错误")

// 组合使用（每次调用都返回新实例）
err := errors.ErrServer.
    WithError(originalErr).
    WithMessage("处理用户请求失败")

// 克隆错误对象
err := errors.ErrNotFound.Clone()
```

### 并发安全说明

预定义错误（如 `ErrServer`、`ErrBadRequest` 等）是全局共享的，但使用 `WithError` 和 `WithMessage` 方法时会创建新实例，因此不会出现并发问题：

```go
// 并发安全 ✅
// 每个 goroutine 都会得到独立的错误实例
func handler1() error {
    return errors.ErrBadRequest.WithError(someErr)
}

func handler2() error {
    return errors.ErrBadRequest.WithMessage("自定义消息")
}
```

### 错误检查

```go
// 检查错误类型
if errors.Is(err, errors.ErrNotFound) {
    // 处理资源不存在的情况
}

// 转换错误类型
var customErr *errors.Error
if errors.As(err, &customErr) {
    log.Printf("错误码: %d, HTTP状态码: %d", customErr.Code, customErr.HttpCode)
}
```

## 错误码规范

建议的错误码分配规则：

- `1000-1999`: 系统级错误
- `2000-2999`: 业务逻辑错误
- `3000-3999`: 数据验证错误
- `4000-4999`: 第三方服务错误

## 示例

```go
package main

import (
    "fmt"
    "qi/pkg/errors"
)

func GetUser(id string) error {
    if id == "" {
        return errors.New(2001, 400, "用户ID不能为空", nil)
    }

    // 模拟数据库查询失败
    dbErr := fmt.Errorf("connection timeout")
    if dbErr != nil {
        return errors.ErrServer.WithError(dbErr)
    }

    return nil
}

func main() {
    err := GetUser("")
    if err != nil {
        var customErr *errors.Error
        if errors.As(err, &customErr) {
            fmt.Printf("错误码: %d\n", customErr.Code)
            fmt.Printf("HTTP状态码: %d\n", customErr.HttpCode)
            fmt.Printf("错误信息: %s\n", customErr.Message)
        }
    }
}
```

## API 参考

### 函数

- `New(code int, httpCode int, message string, err error) *Error` - 创建新错误
- `Is(err error, target error) bool` - 检查错误类型
- `As(err error, target any) bool` - 转换错误类型

### 方法

- `Error() string` - 实现 error 接口
- `Unwrap() error` - 实现 errors.Unwrap 接口
- `Clone() *Error` - 克隆错误对象（返回新实例）
- `WithError(err error) *Error` - 添加原始错误（返回新实例，不修改原对象）
- `WithMessage(message string) *Error` - 修改错误信息（返回新实例，不修改原对象）
- `Is(target error) bool` - 检查错误类型
- `As(target any) bool` - 转换错误类型