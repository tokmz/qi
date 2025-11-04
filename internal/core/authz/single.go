package authz

import (
	"fmt"

	"github.com/casbin/casbin/v2"
)

// SingleEnforcer 单租户权限控制器
type SingleEnforcer struct {
	enforcer *casbin.Enforcer
	config   *Config
	logger   Logger
}

// NewSingleEnforcer 创建单租户权限控制器
func NewSingleEnforcer(enforcer *casbin.Enforcer, config *Config, logger Logger) *SingleEnforcer {
	if logger == nil {
		logger = &DefaultLogger{}
	}

	return &SingleEnforcer{
		enforcer: enforcer,
		config:   config,
		logger:   logger,
	}
}

// CheckPermission 检查用户权限
func (e *SingleEnforcer) CheckPermission(userID, resource, action string) (bool, error) {
	if userID == "" {
		return false, ErrMissingUserID
	}
	if resource == "" || action == "" {
		return false, ErrInvalidParameters
	}

	allowed, err := e.enforcer.Enforce(userID, resource, action)
	if err != nil {
		e.logger.Error("权限检查失败", "user", userID, "resource", resource, "action", action, "error", err)
		return false, WrapError(err, "enforce failed")
	}

	if e.config.EnableLog {
		e.logger.Debug("权限检查", "user", userID, "resource", resource, "action", action, "allowed", allowed)
	}

	return allowed, nil
}

// AddPolicy 添加策略
func (e *SingleEnforcer) AddPolicy(subject, object, action string) error {
	if subject == "" || object == "" || action == "" {
		return ErrInvalidParameters
	}

	added, err := e.enforcer.AddPolicy(subject, object, action)
	if err != nil {
		e.logger.Error("添加策略失败", "subject", subject, "object", object, "action", action, "error", err)
		return WrapError(err, "add policy failed")
	}

	if !added {
		e.logger.Warn("策略已存在", "subject", subject, "object", object, "action", action)
	} else if e.config.EnableLog {
		e.logger.Info("添加策略成功", "subject", subject, "object", object, "action", action)
	}

	return nil
}

// RemovePolicy 移除策略
func (e *SingleEnforcer) RemovePolicy(subject, object, action string) error {
	if subject == "" || object == "" || action == "" {
		return ErrInvalidParameters
	}

	removed, err := e.enforcer.RemovePolicy(subject, object, action)
	if err != nil {
		e.logger.Error("移除策略失败", "subject", subject, "object", object, "action", action, "error", err)
		return WrapError(err, "remove policy failed")
	}

	if !removed {
		e.logger.Warn("策略不存在", "subject", subject, "object", object, "action", action)
	} else if e.config.EnableLog {
		e.logger.Info("移除策略成功", "subject", subject, "object", object, "action", action)
	}

	return nil
}

// AddRoleForUser 为用户分配角色
func (e *SingleEnforcer) AddRoleForUser(userID, role string) error {
	if userID == "" || role == "" {
		return ErrInvalidParameters
	}

	added, err := e.enforcer.AddRoleForUser(userID, role)
	if err != nil {
		e.logger.Error("分配角色失败", "user", userID, "role", role, "error", err)
		return WrapError(err, "add role failed")
	}

	if !added {
		e.logger.Warn("角色已存在", "user", userID, "role", role)
	} else if e.config.EnableLog {
		e.logger.Info("分配角色成功", "user", userID, "role", role)
	}

	return nil
}

// RemoveRoleForUser 移除用户角色
func (e *SingleEnforcer) RemoveRoleForUser(userID, role string) error {
	if userID == "" || role == "" {
		return ErrInvalidParameters
	}

	removed, err := e.enforcer.DeleteRoleForUser(userID, role)
	if err != nil {
		e.logger.Error("移除角色失败", "user", userID, "role", role, "error", err)
		return WrapError(err, "remove role failed")
	}

	if !removed {
		e.logger.Warn("角色不存在", "user", userID, "role", role)
	} else if e.config.EnableLog {
		e.logger.Info("移除角色成功", "user", userID, "role", role)
	}

	return nil
}

// GetRolesForUser 获取用户的所有角色
func (e *SingleEnforcer) GetRolesForUser(userID string) ([]string, error) {
	if userID == "" {
		return nil, ErrMissingUserID
	}

	roles, err := e.enforcer.GetRolesForUser(userID)
	if err != nil {
		e.logger.Error("获取用户角色失败", "user", userID, "error", err)
		return nil, WrapError(err, "get roles failed")
	}

	if e.config.EnableLog {
		e.logger.Debug("获取用户角色", "user", userID, "roles", roles)
	}

	return roles, nil
}

// GetUsersForRole 获取拥有指定角色的所有用户
func (e *SingleEnforcer) GetUsersForRole(role string) ([]string, error) {
	if role == "" {
		return nil, ErrInvalidParameters
	}

	users, err := e.enforcer.GetUsersForRole(role)
	if err != nil {
		e.logger.Error("获取角色用户失败", "role", role, "error", err)
		return nil, WrapError(err, "get users failed")
	}

	if e.config.EnableLog {
		e.logger.Debug("获取角色用户", "role", role, "users", users)
	}

	return users, nil
}

// HasRoleForUser 检查用户是否拥有角色
func (e *SingleEnforcer) HasRoleForUser(userID, role string) (bool, error) {
	if userID == "" || role == "" {
		return false, ErrInvalidParameters
	}

	has, err := e.enforcer.HasRoleForUser(userID, role)
	if err != nil {
		e.logger.Error("检查用户角色失败", "user", userID, "role", role, "error", err)
		return false, WrapError(err, "check role failed")
	}

	return has, nil
}

// GetPermissionsForUser 获取用户的所有权限
func (e *SingleEnforcer) GetPermissionsForUser(userID string) ([][]string, error) {
	if userID == "" {
		return nil, ErrMissingUserID
	}

	permissions, err := e.enforcer.GetPermissionsForUser(userID)
	if err != nil {
		e.logger.Error("获取用户权限失败", "user", userID, "error", err)
		return nil, WrapError(err, "get permissions failed")
	}

	if e.config.EnableLog {
		e.logger.Debug("获取用户权限", "user", userID, "count", len(permissions))
	}

	return permissions, nil
}

// DeleteUser 删除用户及其所有角色
func (e *SingleEnforcer) DeleteUser(userID string) error {
	if userID == "" {
		return ErrMissingUserID
	}

	deleted, err := e.enforcer.DeleteUser(userID)
	if err != nil {
		e.logger.Error("删除用户失败", "user", userID, "error", err)
		return WrapError(err, "delete user failed")
	}

	if !deleted {
		e.logger.Warn("用户不存在", "user", userID)
	} else if e.config.EnableLog {
		e.logger.Info("删除用户成功", "user", userID)
	}

	return nil
}

// DeleteRole 删除角色及其所有用户关联
func (e *SingleEnforcer) DeleteRole(role string) error {
	if role == "" {
		return ErrInvalidParameters
	}

	deleted, err := e.enforcer.DeleteRole(role)
	if err != nil {
		e.logger.Error("删除角色失败", "role", role, "error", err)
		return WrapError(err, "delete role failed")
	}

	if !deleted {
		e.logger.Warn("角色不存在", "role", role)
	} else if e.config.EnableLog {
		e.logger.Info("删除角色成功", "role", role)
	}

	return nil
}

// LoadPolicy 重新加载策略
func (e *SingleEnforcer) LoadPolicy() error {
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
func (e *SingleEnforcer) SavePolicy() error {
	if err := e.enforcer.SavePolicy(); err != nil {
		e.logger.Error("保存策略失败", "error", err)
		return WrapError(err, "save policy failed")
	}

	if e.config.EnableLog {
		e.logger.Info("保存策略成功")
	}

	return nil
}

// GetAllSubjects 获取所有主体（用户/角色）
func (e *SingleEnforcer) GetAllSubjects() ([]string, error) {
	return e.enforcer.GetAllSubjects()
}

// GetAllObjects 获取所有资源
func (e *SingleEnforcer) GetAllObjects() ([]string, error) {
	return e.enforcer.GetAllObjects()
}

// GetAllActions 获取所有操作
func (e *SingleEnforcer) GetAllActions() ([]string, error) {
	return e.enforcer.GetAllActions()
}

// GetAllRoles 获取所有角色
func (e *SingleEnforcer) GetAllRoles() ([]string, error) {
	return e.enforcer.GetAllRoles()
}

// GetPolicy 获取所有策略规则
func (e *SingleEnforcer) GetPolicy() ([][]string, error) {
	return e.enforcer.GetPolicy()
}

// GetGroupingPolicy 获取所有角色继承规则
func (e *SingleEnforcer) GetGroupingPolicy() ([][]string, error) {
	return e.enforcer.GetGroupingPolicy()
}

// String 返回控制器的字符串表示
func (e *SingleEnforcer) String() string {
	subjects, _ := e.GetAllSubjects()
	objects, _ := e.GetAllObjects()
	actions, _ := e.GetAllActions()
	return fmt.Sprintf("SingleEnforcer(subjects=%d, objects=%d, actions=%d)",
		len(subjects),
		len(objects),
		len(actions))
}

