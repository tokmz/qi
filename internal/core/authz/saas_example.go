package authz

// SaaS 系统双层权限示例
// 方案一：统一使用多租户模式（推荐）

import (
	"fmt"
)

const (
	// PlatformTenantID 平台租户ID（特殊租户，用于平台管理）
	PlatformTenantID = "platform"
)

// SaaSManager SaaS 系统权限管理器
// 统一使用多租户模式，平台管理作为特殊租户
type SaaSManager struct {
	manager *Manager
}

// NewSaaSManager 创建 SaaS 权限管理器
func NewSaaSManager(config *Config) (*SaaSManager, error) {
	// 强制使用多租户模式
	config.Mode = ModeMulti

	manager, err := New(config, nil)
	if err != nil {
		return nil, err
	}

	return &SaaSManager{
		manager: manager,
	}, nil
}

// InitializePlatformRoles 初始化平台角色
func (m *SaaSManager) InitializePlatformRoles() error {
	multi := m.manager.Multi()

	// 平台超级管理员：可以管理所有租户
	platformPolicies := [][]string{
		{PlatformTenantID, "super_admin", "/platform/tenants", "*"},
		{PlatformTenantID, "super_admin", "/platform/users", "*"},
		{PlatformTenantID, "super_admin", "/platform/settings", "*"},
		{PlatformTenantID, "super_admin", "/platform/analytics", "*"},
	}

	// 平台运营人员：只能查看和部分管理
	platformPolicies = append(platformPolicies, [][]string{
		{PlatformTenantID, "platform_operator", "/platform/tenants", "GET"},
		{PlatformTenantID, "platform_operator", "/platform/users", "GET"},
		{PlatformTenantID, "platform_operator", "/platform/analytics", "GET"},
	}...)

	for _, policy := range platformPolicies {
		if err := multi.AddPolicy(policy[0], policy[1], policy[2], policy[3]); err != nil {
			return err
		}
	}

	return nil
}

// InitializeTenantRoles 初始化租户角色模板
func (m *SaaSManager) InitializeTenantRoles(tenantID string) error {
	multi := m.manager.Multi()

	// 租户管理员角色
	tenantPolicies := [][]string{
		{tenantID, "tenant_admin", "/api/v1/users", "*"},
		{tenantID, "tenant_admin", "/api/v1/projects", "*"},
		{tenantID, "tenant_admin", "/api/v1/tasks", "*"},
		{tenantID, "tenant_admin", "/api/v1/settings", "*"},
	}

	// 租户普通成员角色
	tenantPolicies = append(tenantPolicies, [][]string{
		{tenantID, "member", "/api/v1/projects", "GET"},
		{tenantID, "member", "/api/v1/projects", "POST"},
		{tenantID, "member", "/api/v1/tasks", "*"},
	}...)

	// 租户访客角色
	tenantPolicies = append(tenantPolicies, [][]string{
		{tenantID, "viewer", "/api/v1/projects", "GET"},
		{tenantID, "viewer", "/api/v1/tasks", "GET"},
	}...)

	for _, policy := range tenantPolicies {
		if err := multi.AddPolicy(policy[0], policy[1], policy[2], policy[3]); err != nil {
			return err
		}
	}

	return nil
}

// AssignPlatformRole 分配平台角色（给平台管理员）
func (m *SaaSManager) AssignPlatformRole(userID, role string) error {
	return m.manager.Multi().AddRoleForUser(PlatformTenantID, userID, role)
}

// AssignTenantRole 分配租户角色（给租户用户）
func (m *SaaSManager) AssignTenantRole(tenantID, userID, role string) error {
	return m.manager.Multi().AddRoleForUser(tenantID, userID, role)
}

// CheckPlatformPermission 检查平台权限
func (m *SaaSManager) CheckPlatformPermission(userID, resource, action string) (bool, error) {
	return m.manager.Multi().CheckPermission(PlatformTenantID, userID, resource, action)
}

// CheckTenantPermission 检查租户权限
func (m *SaaSManager) CheckTenantPermission(tenantID, userID, resource, action string) (bool, error) {
	return m.manager.Multi().CheckPermission(tenantID, userID, resource, action)
}

// IsPlatformAdmin 检查用户是否是平台管理员
func (m *SaaSManager) IsPlatformAdmin(userID string) (bool, error) {
	return m.manager.Multi().HasRoleForUser(PlatformTenantID, userID, "super_admin")
}

// GetUserTenants 获取用户所属的所有租户
func (m *SaaSManager) GetUserTenants(userID string) ([]string, error) {
	// 这里需要通过业务逻辑查询用户所属的租户
	// 可以从数据库的 tenant_users 表查询
	// 这只是一个示例接口
	return nil, fmt.Errorf("not implemented: query from database")
}

// CreateTenant 创建新租户（平台管理员操作）
func (m *SaaSManager) CreateTenant(tenantID string) error {
	// 初始化租户的默认角色和权限
	return m.InitializeTenantRoles(tenantID)
}

// DeleteTenant 删除租户（平台管理员操作）
func (m *SaaSManager) DeleteTenant(tenantID string) error {
	return m.manager.Multi().DeleteTenant(tenantID)
}

// GetManager 获取底层 Manager
func (m *SaaSManager) GetManager() *Manager {
	return m.manager
}

// Close 关闭管理器
func (m *SaaSManager) Close() error {
	return m.manager.Close()
}

