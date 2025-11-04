package authz

import (
	"fmt"

	"github.com/casbin/casbin/v2"
)

// MultiEnforcer 多租户权限控制器
type MultiEnforcer struct {
	enforcer *casbin.Enforcer
	config   *Config
	logger   Logger
}

// NewMultiEnforcer 创建多租户权限控制器
func NewMultiEnforcer(enforcer *casbin.Enforcer, config *Config, logger Logger) *MultiEnforcer {
	if logger == nil {
		logger = &DefaultLogger{}
	}

	return &MultiEnforcer{
		enforcer: enforcer,
		config:   config,
		logger:   logger,
	}
}

// CheckPermission 检查租户用户权限
func (e *MultiEnforcer) CheckPermission(tenantID, userID, resource, action string) (bool, error) {
	if tenantID == "" {
		return false, ErrMissingTenantID
	}
	if userID == "" {
		return false, ErrMissingUserID
	}
	if resource == "" || action == "" {
		return false, ErrInvalidParameters
	}

	allowed, err := e.enforcer.Enforce(tenantID, userID, resource, action)
	if err != nil {
		e.logger.Error("权限检查失败",
			"tenant", tenantID, "user", userID, "resource", resource, "action", action, "error", err)
		return false, WrapError(err, "enforce failed")
	}

	if e.config.EnableLog {
		e.logger.Debug("权限检查",
			"tenant", tenantID, "user", userID, "resource", resource, "action", action, "allowed", allowed)
	}

	return allowed, nil
}

// AddPolicy 添加租户策略
func (e *MultiEnforcer) AddPolicy(tenantID, subject, object, action string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}
	if subject == "" || object == "" || action == "" {
		return ErrInvalidParameters
	}

	added, err := e.enforcer.AddPolicy(tenantID, subject, object, action)
	if err != nil {
		e.logger.Error("添加策略失败",
			"tenant", tenantID, "subject", subject, "object", object, "action", action, "error", err)
		return WrapError(err, "add policy failed")
	}

	if !added {
		e.logger.Warn("策略已存在",
			"tenant", tenantID, "subject", subject, "object", object, "action", action)
	} else if e.config.EnableLog {
		e.logger.Info("添加策略成功",
			"tenant", tenantID, "subject", subject, "object", object, "action", action)
	}

	return nil
}

// RemovePolicy 移除租户策略
func (e *MultiEnforcer) RemovePolicy(tenantID, subject, object, action string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}
	if subject == "" || object == "" || action == "" {
		return ErrInvalidParameters
	}

	removed, err := e.enforcer.RemovePolicy(tenantID, subject, object, action)
	if err != nil {
		e.logger.Error("移除策略失败",
			"tenant", tenantID, "subject", subject, "object", object, "action", action, "error", err)
		return WrapError(err, "remove policy failed")
	}

	if !removed {
		e.logger.Warn("策略不存在",
			"tenant", tenantID, "subject", subject, "object", object, "action", action)
	} else if e.config.EnableLog {
		e.logger.Info("移除策略成功",
			"tenant", tenantID, "subject", subject, "object", object, "action", action)
	}

	return nil
}

// AddRoleForUser 为租户用户分配角色
func (e *MultiEnforcer) AddRoleForUser(tenantID, userID, role string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}
	if userID == "" || role == "" {
		return ErrInvalidParameters
	}

	added, err := e.enforcer.AddRoleForUser(userID, role, tenantID)
	if err != nil {
		e.logger.Error("分配角色失败",
			"tenant", tenantID, "user", userID, "role", role, "error", err)
		return WrapError(err, "add role failed")
	}

	if !added {
		e.logger.Warn("角色已存在",
			"tenant", tenantID, "user", userID, "role", role)
	} else if e.config.EnableLog {
		e.logger.Info("分配角色成功",
			"tenant", tenantID, "user", userID, "role", role)
	}

	return nil
}

// RemoveRoleForUser 移除租户用户角色
func (e *MultiEnforcer) RemoveRoleForUser(tenantID, userID, role string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}
	if userID == "" || role == "" {
		return ErrInvalidParameters
	}

	removed, err := e.enforcer.DeleteRoleForUser(userID, role, tenantID)
	if err != nil {
		e.logger.Error("移除角色失败",
			"tenant", tenantID, "user", userID, "role", role, "error", err)
		return WrapError(err, "remove role failed")
	}

	if !removed {
		e.logger.Warn("角色不存在",
			"tenant", tenantID, "user", userID, "role", role)
	} else if e.config.EnableLog {
		e.logger.Info("移除角色成功",
			"tenant", tenantID, "user", userID, "role", role)
	}

	return nil
}

// GetRolesForUser 获取用户在指定租户的所有角色
func (e *MultiEnforcer) GetRolesForUser(tenantID, userID string) ([]string, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	roles := e.enforcer.GetRolesForUserInDomain(userID, tenantID)

	if e.config.EnableLog {
		e.logger.Debug("获取用户角色",
			"tenant", tenantID, "user", userID, "roles", roles)
	}

	return roles, nil
}

// GetUsersForRole 获取拥有指定角色的所有用户（指定租户）
func (e *MultiEnforcer) GetUsersForRole(tenantID, role string) ([]string, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if role == "" {
		return nil, ErrInvalidParameters
	}

	users := e.enforcer.GetUsersForRoleInDomain(role, tenantID)

	if e.config.EnableLog {
		e.logger.Debug("获取角色用户",
			"tenant", tenantID, "role", role, "users", users)
	}

	return users, nil
}

// HasRoleForUser 检查用户在指定租户是否拥有角色
func (e *MultiEnforcer) HasRoleForUser(tenantID, userID, role string) (bool, error) {
	if tenantID == "" {
		return false, ErrMissingTenantID
	}
	if userID == "" || role == "" {
		return false, ErrInvalidParameters
	}

	has, err := e.enforcer.HasRoleForUser(userID, role, tenantID)
	if err != nil {
		e.logger.Error("检查用户角色失败",
			"tenant", tenantID, "user", userID, "role", role, "error", err)
		return false, WrapError(err, "check role failed")
	}

	return has, nil
}

// GetPermissionsForUser 获取用户在指定租户的所有权限
func (e *MultiEnforcer) GetPermissionsForUser(tenantID, userID string) ([][]string, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	permissions := e.enforcer.GetPermissionsForUserInDomain(userID, tenantID)

	if e.config.EnableLog {
		e.logger.Debug("获取用户权限",
			"tenant", tenantID, "user", userID, "count", len(permissions))
	}

	return permissions, nil
}

// DeleteUser 删除租户用户及其所有角色
func (e *MultiEnforcer) DeleteUser(tenantID, userID string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}
	if userID == "" {
		return ErrMissingUserID
	}

	// 删除用户在指定租户的所有角色
	deleted, err := e.enforcer.DeleteRolesForUser(userID, tenantID)
	if err != nil {
		e.logger.Error("删除用户失败",
			"tenant", tenantID, "user", userID, "error", err)
		return WrapError(err, "delete user failed")
	}

	if !deleted {
		e.logger.Warn("用户不存在",
			"tenant", tenantID, "user", userID)
	} else if e.config.EnableLog {
		e.logger.Info("删除用户成功",
			"tenant", tenantID, "user", userID)
	}

	return nil
}

// DeleteRole 删除租户角色及其所有用户关联
func (e *MultiEnforcer) DeleteRole(tenantID, role string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}
	if role == "" {
		return ErrInvalidParameters
	}

	// 获取所有拥有该角色的用户
	users, err := e.GetUsersForRole(tenantID, role)
	if err != nil {
		return err
	}

	// 删除所有用户的该角色
	for _, user := range users {
		if err := e.RemoveRoleForUser(tenantID, user, role); err != nil {
			e.logger.Error("删除用户角色失败",
				"tenant", tenantID, "user", user, "role", role, "error", err)
		}
	}

	// 删除角色的所有策略
	_, err = e.enforcer.RemoveFilteredPolicy(0, tenantID, role)
	if err != nil {
		e.logger.Error("删除角色策略失败",
			"tenant", tenantID, "role", role, "error", err)
		return WrapError(err, "delete role policies failed")
	}

	if e.config.EnableLog {
		e.logger.Info("删除角色成功",
			"tenant", tenantID, "role", role)
	}

	return nil
}

// DeleteTenant 删除租户及其所有数据
func (e *MultiEnforcer) DeleteTenant(tenantID string) error {
	if tenantID == "" {
		return ErrMissingTenantID
	}

	// 删除租户的所有策略
	_, err := e.enforcer.RemoveFilteredPolicy(0, tenantID)
	if err != nil {
		e.logger.Error("删除租户策略失败",
			"tenant", tenantID, "error", err)
		return WrapError(err, "delete tenant policies failed")
	}

	// 删除租户的所有角色关系
	_, err = e.enforcer.RemoveFilteredGroupingPolicy(2, tenantID)
	if err != nil {
		e.logger.Error("删除租户角色失败",
			"tenant", tenantID, "error", err)
		return WrapError(err, "delete tenant roles failed")
	}

	if e.config.EnableLog {
		e.logger.Info("删除租户成功",
			"tenant", tenantID)
	}

	return nil
}

// GetPoliciesForTenant 获取租户的所有策略
func (e *MultiEnforcer) GetPoliciesForTenant(tenantID string) ([][]string, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}

	policies, err := e.enforcer.GetFilteredPolicy(0, tenantID)
	if err != nil {
		return nil, err
	}
	return policies, nil
}

// GetRolesForTenant 获取租户的所有角色关系
func (e *MultiEnforcer) GetRolesForTenant(tenantID string) ([][]string, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}

	roles, err := e.enforcer.GetFilteredGroupingPolicy(2, tenantID)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// LoadPolicy 重新加载策略
func (e *MultiEnforcer) LoadPolicy() error {
	if err := e.enforcer.LoadPolicy(); err != nil {
		e.logger.Error("加载策略失败", "error", err)
		return WrapError(err, "load policy failed")
	}

	if e.config.EnableLog {
		e.logger.Info("加载策略成功")
	}

	return nil
}

// SavePolicy 保存策略到存储
func (e *MultiEnforcer) SavePolicy() error {
	if err := e.enforcer.SavePolicy(); err != nil {
		e.logger.Error("保存策略失败", "error", err)
		return WrapError(err, "save policy failed")
	}

	if e.config.EnableLog {
		e.logger.Info("保存策略成功")
	}

	return nil
}

// GetPolicy 获取所有策略规则
func (e *MultiEnforcer) GetPolicy() ([][]string, error) {
	return e.enforcer.GetPolicy()
}

// GetGroupingPolicy 获取所有角色继承规则
func (e *MultiEnforcer) GetGroupingPolicy() ([][]string, error) {
	return e.enforcer.GetGroupingPolicy()
}

// String 返回控制器的字符串表示
func (e *MultiEnforcer) String() string {
	policies, _ := e.GetPolicy()
	roles, _ := e.GetGroupingPolicy()
	return fmt.Sprintf("MultiEnforcer(policies=%d, roles=%d)",
		len(policies),
		len(roles))
}

