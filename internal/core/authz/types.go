package authz

// Logger 日志接口
type Logger interface {
	// Debug 调试日志
	Debug(msg string, fields ...interface{})
	// Info 信息日志
	Info(msg string, fields ...interface{})
	// Warn 警告日志
	Warn(msg string, fields ...interface{})
	// Error 错误日志
	Error(msg string, fields ...interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct{}

// Debug 调试日志
func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	// 可以使用标准库 log 或其他日志库
}

// Info 信息日志
func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	// 可以使用标准库 log 或其他日志库
}

// Warn 警告日志
func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	// 可以使用标准库 log 或其他日志库
}

// Error 错误日志
func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	// 可以使用标准库 log 或其他日志库
}

// Permission 权限定义
type Permission struct {
	// 资源路径
	Resource string `json:"resource"`
	// 操作（GET/POST/PUT/DELETE/*）
	Action string `json:"action"`
}

// Role 角色定义
type Role struct {
	// 角色名称
	Name string `json:"name"`
	// 角色描述
	Description string `json:"description,omitempty"`
	// 权限列表
	Permissions []Permission `json:"permissions,omitempty"`
}

// UserRole 用户角色关系
type UserRole struct {
	// 用户ID
	UserID string `json:"user_id"`
	// 角色名称
	Role string `json:"role"`
	// 租户ID（多租户模式）
	TenantID string `json:"tenant_id,omitempty"`
}

// Policy 策略定义
type Policy struct {
	// 主体（用户/角色）
	Subject string `json:"subject"`
	// 资源
	Object string `json:"object"`
	// 操作
	Action string `json:"action"`
	// 租户ID（多租户模式）
	TenantID string `json:"tenant_id,omitempty"`
}

// EnforceRequest 权限检查请求
type EnforceRequest struct {
	// 用户ID
	UserID string `json:"user_id"`
	// 资源路径
	Resource string `json:"resource"`
	// 操作
	Action string `json:"action"`
	// 租户ID（多租户模式）
	TenantID string `json:"tenant_id,omitempty"`
}

// EnforceResult 权限检查结果
type EnforceResult struct {
	// 是否允许
	Allowed bool `json:"allowed"`
	// 原因（拒绝时）
	Reason string `json:"reason,omitempty"`
}

