# errors

错误处理包，提供统一的业务错误定义和错误链支持。

## 功能特性

- 业务错误码 + 用户友好消息分离
- 支持 HTTP 状态码关联
- 完整的错误链支持（Go 1.13+）
- 链式调用构建错误
- 标准库 errors 兼容

## 快速开始

```go
import "github.com/abc-inc/qi/pkg/errors"

// 创建业务错误
err := errors.New(404, "资源不存在")

// 带 HTTP 状态码
err := errors.NewWithStatus(400, http.StatusBadRequest, "请求参数错误")

// 包装已有错误
err := errors.Wrap(originalErr, 500, "内部错误")

// 链式调用
err := errors.New(400, "参数错误").
    WithStatus(http.StatusBadRequest).
    WithMessage("用户名不能为空")
```

## 错误类型

```go
type Error struct {
    Code    int    // 业务错误码
    Message string // 用户可见消息
    err     error  // 原始错误
    status  int    // HTTP 状态码
}
```

## 创建错误

| 函数 | 说明 |
|------|------|
| `New(code, message)` | 创建基本错误，默认 status=500 |
| `NewWithStatus(code, status, message)` | 创建带 HTTP 状态码的错误 |
| `Wrap(err, code, message)` | 包装已有错误 |
| `WrapWithStatus(err, code, status, message)` | 包装已有错误（带 status） |

## 链式方法

| 方法 | 说明 |
|------|------|
| `WithErr(err)` | 附加原始错误 |
| `WithStatus(status)` | 设置 HTTP 状态码 |
| `WithMessage(message)` | 覆盖错误消息 |
| `WithMessagef(format, args...)` | 格式化设置消息 |

> 注意：链式方法返回新实例，不会修改原错误

## 访问器

| 方法 | 说明 |
|------|------|
| `Error()` | 实现 error 接口，返回完整错误信息 |
| `Status()` | 获取 HTTP 状态码，未设置默认 500 |
| `Unwrap()` | 实现 errors.Unwrap 接口，获取原始错误 |
| `Is(target)` | 实现错误比较，通过 Code 判断相等 |

## 辅助函数

| 函数 | 说明 |
|------|------|
| `GetCode(err)` | 从 error 提取业务码，非本类型返回 -1 |
| `GetStatus(err)` | 从 error 提取 HTTP 状态码，非本类型返回 500 |
| `IsCode(err, code)` | 判断错误是否匹配指定业务码 |
| `As(err)` | 标准库 errors.As 的快捷方式 |
| `Is(err, target)` | 标准库 errors.Is 的透传 |

## 使用示例

### 基本用法

```go
// 定义业务错误码
const (
    ErrCodeNotFound    = 404
    ErrCodeInvalidArgs = 400
    ErrCodeInternal    = 500
)

// 返回错误
func GetUser(id int) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        return nil, errors.Wrap(err, ErrCodeNotFound, "用户不存在")
    }
    return user, nil
}

// 处理错误
func handleErr(err error) {
    code := errors.GetCode(err)
    status := errors.GetStatus(err)
    // ...
}
```

### 预定义错误

```go
var (
    ErrNotFound    = errors.New(404, "资源不存在")
    ErrUnauthorized = errors.New(401, "未授权").WithStatus(http.StatusUnauthorized)
)

// 复用预定义错误
func findUser(id int) error {
    if id <= 0 {
        return ErrNotFound.WithMessage("无效的用户ID")
    }
    // ...
}
```

### 错误链追踪

```go
err := errors.Wrap(io.EOF, ErrCodeInvalidArgs, "读取配置失败")
fmt.Println(err)                   // 读取配置失败: EOF
fmt.Println(errors.Unwrap(err))    // EOF

// 使用 errors.Is 检查链
if errors.Is(err, io.EOF) {
    // ...
}
```

### 通过 Code 比较

```go
err := errors.New(ErrCodeNotFound, "用户不存在")

// Is 方法通过 Code 判断
if err.Is(ErrNotFound) {  // 假设 ErrNotFound.Code == 404
    // ...
}

// 或使用辅助函数
if errors.IsCode(err, 404) {
    // ...
}
```
