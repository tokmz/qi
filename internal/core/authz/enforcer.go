package authz

import (
	"context"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

// Manager 权限管理器
type Manager struct {
	config   *Config
	enforcer *casbin.Enforcer
	single   *SingleEnforcer
	multi    *MultiEnforcer
	logger   Logger
	mu       sync.RWMutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

var (
	globalManager     *Manager
	globalManagerOnce sync.Once
	globalManagerMu   sync.RWMutex
)

// New 创建权限管理器
func New(config *Config, logger Logger) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = &DefaultLogger{}
	}

	m := &Manager{
		config: config,
		logger: logger,
		stopCh: make(chan struct{}),
	}

	// 创建 enforcer
	if err := m.initEnforcer(); err != nil {
		return nil, err
	}

	// 根据模式创建对应的控制器
	if config.Mode == ModeSingle {
		m.single = NewSingleEnforcer(m.enforcer, config, logger)
	} else {
		m.multi = NewMultiEnforcer(m.enforcer, config, logger)
	}

	// 启动自动加载策略
	if config.AutoLoad {
		m.startAutoLoad()
	}

	return m, nil
}

// initEnforcer 初始化 Casbin enforcer
func (m *Manager) initEnforcer() error {
	// 加载模型
	modelPath := m.config.GetModelPath()
	casbinModel, err := model.NewModelFromFile(modelPath)
	if err != nil {
		m.logger.Error("加载模型文件失败", "path", modelPath, "error", err)
		return WrapError(err, "load model failed")
	}

	// 创建适配器
	adapter, err := m.createAdapter()
	if err != nil {
		return err
	}

	// 创建 enforcer
	m.enforcer, err = casbin.NewEnforcer(casbinModel, adapter)
	if err != nil {
		m.logger.Error("创建 Enforcer 失败", "error", err)
		return WrapError(err, "create enforcer failed")
	}

	// 启用日志
	if !m.config.EnableLog {
		m.enforcer.EnableLog(false)
	}

	// 加载策略
	if err := m.enforcer.LoadPolicy(); err != nil {
		m.logger.Error("加载策略失败", "error", err)
		return WrapError(err, "load policy failed")
	}

	m.logger.Info("权限管理器初始化成功", "mode", m.config.Mode, "model", modelPath)
	return nil
}

// createAdapter 创建适配器
func (m *Manager) createAdapter() (persist.Adapter, error) {
	switch m.config.Adapter.Type {
	case AdapterTypeFile:
		policyPath := m.config.GetPolicyPath()
		adapter := fileadapter.NewAdapter(policyPath)
		m.logger.Debug("使用文件适配器", "path", policyPath)
		return adapter, nil

	case AdapterTypeGorm:
		adapter, err := gormadapter.NewAdapter(
			m.config.Adapter.DBType,
			m.config.Adapter.DSN,
			m.config.GetTableName(),
		)
		if err != nil {
			m.logger.Error("创建 GORM 适配器失败",
				"dbType", m.config.Adapter.DBType,
				"table", m.config.GetTableName(),
				"error", err)
			return nil, WrapError(err, "create gorm adapter failed")
		}
		m.logger.Debug("使用 GORM 适配器",
			"dbType", m.config.Adapter.DBType,
			"table", m.config.GetTableName())
		return adapter, nil

	default:
		return nil, ErrInvalidAdapterType
	}
}

// startAutoLoad 启动自动加载策略
func (m *Manager) startAutoLoad() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(m.config.GetAutoLoadDuration())
		defer ticker.Stop()

		m.logger.Info("启动自动加载策略", "interval", m.config.GetAutoLoadDuration())

		for {
			select {
			case <-ticker.C:
				if err := m.enforcer.LoadPolicy(); err != nil {
					m.logger.Error("自动加载策略失败", "error", err)
				} else if m.config.EnableLog {
					m.logger.Debug("自动加载策略成功")
				}
			case <-m.stopCh:
				m.logger.Info("停止自动加载策略")
				return
			}
		}
	}()
}

// InitGlobal 初始化全局权限管理器
func InitGlobal(config *Config, logger Logger) error {
	var err error
	globalManagerOnce.Do(func() {
		globalManagerMu.Lock()
		defer globalManagerMu.Unlock()

		globalManager, err = New(config, logger)
	})
	return err
}

// GetGlobal 获取全局权限管理器
func GetGlobal() *Manager {
	globalManagerMu.RLock()
	defer globalManagerMu.RUnlock()
	return globalManager
}

// Single 获取单租户控制器
func (m *Manager) Single() *SingleEnforcer {
	if m.config.Mode != ModeSingle {
		m.logger.Warn("当前模式不是单租户模式", "mode", m.config.Mode)
		return nil
	}
	return m.single
}

// Multi 获取多租户控制器
func (m *Manager) Multi() *MultiEnforcer {
	if m.config.Mode != ModeMulti {
		m.logger.Warn("当前模式不是多租户模式", "mode", m.config.Mode)
		return nil
	}
	return m.multi
}

// Enforce 统一的权限检查接口
func (m *Manager) Enforce(ctx context.Context, req *EnforceRequest) (*EnforceResult, error) {
	var allowed bool
	var err error

	if m.config.Mode == ModeSingle {
		allowed, err = m.single.CheckPermission(req.UserID, req.Resource, req.Action)
	} else {
		if req.TenantID == "" {
			return &EnforceResult{
				Allowed: false,
				Reason:  "missing tenant id",
			}, ErrMissingTenantID
		}
		allowed, err = m.multi.CheckPermission(req.TenantID, req.UserID, req.Resource, req.Action)
	}

	if err != nil {
		return &EnforceResult{
			Allowed: false,
			Reason:  err.Error(),
		}, err
	}

	result := &EnforceResult{
		Allowed: allowed,
	}

	if !allowed {
		result.Reason = "permission denied"
	}

	return result, nil
}

// LoadPolicy 重新加载策略
func (m *Manager) LoadPolicy() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.enforcer.LoadPolicy(); err != nil {
		m.logger.Error("加载策略失败", "error", err)
		return WrapError(err, "load policy failed")
	}

	m.logger.Info("加载策略成功")
	return nil
}

// SavePolicy 保存策略
func (m *Manager) SavePolicy() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.enforcer.SavePolicy(); err != nil {
		m.logger.Error("保存策略失败", "error", err)
		return WrapError(err, "save policy failed")
	}

	m.logger.Info("保存策略成功")
	return nil
}

// GetEnforcer 获取底层 Casbin enforcer（高级用法）
func (m *Manager) GetEnforcer() *casbin.Enforcer {
	return m.enforcer
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// GetMode 获取当前模式
func (m *Manager) GetMode() Mode {
	return m.config.Mode
}

// Close 关闭权限管理器
func (m *Manager) Close() error {
	m.logger.Info("关闭权限管理器")

	// 停止自动加载
	if m.config.AutoLoad {
		close(m.stopCh)
		m.wg.Wait()
	}

	// 保存策略
	if err := m.SavePolicy(); err != nil {
		m.logger.Error("保存策略失败", "error", err)
		return err
	}

	m.logger.Info("权限管理器已关闭")
	return nil
}

// Helper functions for quick access

// CheckPermission 快速权限检查（单租户）
func CheckPermission(userID, resource, action string) (bool, error) {
	mgr := GetGlobal()
	if mgr == nil {
		return false, ErrEnforcerNotInitialized
	}
	if mgr.config.Mode != ModeSingle {
		return false, ErrInvalidMode
	}
	return mgr.single.CheckPermission(userID, resource, action)
}

// CheckTenantPermission 快速权限检查（多租户）
func CheckTenantPermission(tenantID, userID, resource, action string) (bool, error) {
	mgr := GetGlobal()
	if mgr == nil {
		return false, ErrEnforcerNotInitialized
	}
	if mgr.config.Mode != ModeMulti {
		return false, ErrInvalidMode
	}
	return mgr.multi.CheckPermission(tenantID, userID, resource, action)
}

// AddRoleForUser 为用户分配角色（单租户）
func AddRoleForUser(userID, role string) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrEnforcerNotInitialized
	}
	if mgr.config.Mode != ModeSingle {
		return ErrInvalidMode
	}
	return mgr.single.AddRoleForUser(userID, role)
}

// AddTenantRoleForUser 为租户用户分配角色（多租户）
func AddTenantRoleForUser(tenantID, userID, role string) error {
	mgr := GetGlobal()
	if mgr == nil {
		return ErrEnforcerNotInitialized
	}
	if mgr.config.Mode != ModeMulti {
		return ErrInvalidMode
	}
	return mgr.multi.AddRoleForUser(tenantID, userID, role)
}

