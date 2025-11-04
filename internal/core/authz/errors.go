package authz

import "errors"

var (
	// ErrInvalidMode 无效的权限模式
	ErrInvalidMode = errors.New("authz: invalid mode, must be 'single' or 'multi'")

	// ErrMissingModelPath 缺少模型文件路径
	ErrMissingModelPath = errors.New("authz: missing model path")

	// ErrMissingPolicyPath 缺少策略文件路径
	ErrMissingPolicyPath = errors.New("authz: missing policy path when using file adapter")

	// ErrInvalidAdapterType 无效的适配器类型
	ErrInvalidAdapterType = errors.New("authz: invalid adapter type, must be 'file' or 'gorm'")

	// ErrMissingDSN 缺少数据库 DSN
	ErrMissingDSN = errors.New("authz: missing DSN when using gorm adapter")

	// ErrMissingDBType 缺少数据库类型
	ErrMissingDBType = errors.New("authz: missing db_type when using gorm adapter")

	// ErrInvalidAutoLoadInterval 无效的自动加载间隔
	ErrInvalidAutoLoadInterval = errors.New("authz: invalid auto_load_interval, must be greater than 0")

	// ErrEnforcerNotInitialized Enforcer 未初始化
	ErrEnforcerNotInitialized = errors.New("authz: enforcer not initialized")

	// ErrLoadModelFailed 加载模型失败
	ErrLoadModelFailed = errors.New("authz: failed to load model")

	// ErrLoadPolicyFailed 加载策略失败
	ErrLoadPolicyFailed = errors.New("authz: failed to load policy")

	// ErrCreateAdapterFailed 创建适配器失败
	ErrCreateAdapterFailed = errors.New("authz: failed to create adapter")

	// ErrEnforceFailed 权限检查失败
	ErrEnforceFailed = errors.New("authz: enforce failed")

	// ErrAddPolicyFailed 添加策略失败
	ErrAddPolicyFailed = errors.New("authz: failed to add policy")

	// ErrRemovePolicyFailed 移除策略失败
	ErrRemovePolicyFailed = errors.New("authz: failed to remove policy")

	// ErrAddRoleFailed 添加角色失败
	ErrAddRoleFailed = errors.New("authz: failed to add role")

	// ErrRemoveRoleFailed 移除角色失败
	ErrRemoveRoleFailed = errors.New("authz: failed to remove role")

	// ErrMissingTenantID 缺少租户ID
	ErrMissingTenantID = errors.New("authz: missing tenant id")

	// ErrMissingUserID 缺少用户ID
	ErrMissingUserID = errors.New("authz: missing user id")

	// ErrPermissionDenied 权限不足
	ErrPermissionDenied = errors.New("authz: permission denied")

	// ErrInvalidParameters 无效的参数
	ErrInvalidParameters = errors.New("authz: invalid parameters")
)

// WrapError 包装错误信息
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return errors.New(message + ": " + err.Error())
}

