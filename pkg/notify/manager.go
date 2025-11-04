package notify

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Manager 通知管理器
// 管理多个通知器，提供统一的发送接口
type Manager struct {
	config    *Config
	notifiers map[MessageType]Notifier
	template  *TemplateManager
	logger    Logger
	stats     map[MessageType]*Stats
	statsMu   sync.RWMutex
	closed    atomic.Bool
}

// NewManager 创建通知管理器
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	m := &Manager{
		config:    config,
		notifiers: make(map[MessageType]Notifier),
		logger:    config.Logger,
		stats:     make(map[MessageType]*Stats),
	}

	// 初始化模板管理器
	if config.Template != nil {
		tm, err := NewTemplateManager(config.Template)
		if err != nil {
			return nil, err
		}
		m.template = tm
	}

	// 初始化邮件通知器
	if config.Email != nil {
		emailNotifier, err := NewEmailNotifier(config.Email, config.Logger)
		if err != nil {
			return nil, err
		}
		m.notifiers[MessageTypeEmail] = emailNotifier
		m.stats[MessageTypeEmail] = &Stats{}
	}

	// 后续可以添加短信、推送等通知器
	// if config.SMS != nil { ... }
	// if config.Push != nil { ... }

	return m, nil
}

// Send 发送通知
func (m *Manager) Send(ctx context.Context, message *Message, opts ...*SendOptions) (*SendResult, error) {
	if m.closed.Load() {
		return nil, ErrNotifierClosed
	}

	// 验证消息
	if err := m.validateMessage(message); err != nil {
		return nil, err
	}

	// 处理模板
	if message.Template != "" && m.template != nil {
		content, err := m.template.Render(message.Template, message.TemplateData)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTemplateRenderFailed, err)
		}
		message.Content = content
		if message.ContentType == "" {
			message.ContentType = ContentTypeHTML
		}
	}

	// 获取发送选项
	opt := DefaultSendOptions()
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	// 获取通知器
	notifier, ok := m.notifiers[message.Type]
	if !ok {
		return nil, ErrNotifierNotFound
	}

	// 异步发送
	if opt.Async {
		go m.sendWithRetry(ctx, notifier, message, opt)
		return &SendResult{
			MessageID: message.ID,
			Success:   true,
			SentAt:    time.Now(),
		}, nil
	}

	// 同步发送
	return m.sendWithRetry(ctx, notifier, message, opt)
}

// SendBatch 批量发送通知
func (m *Manager) SendBatch(ctx context.Context, messages []*Message, opts ...*SendOptions) ([]*SendResult, error) {
	if m.closed.Load() {
		return nil, ErrNotifierClosed
	}

	results := make([]*SendResult, len(messages))
	var wg sync.WaitGroup

	opt := DefaultSendOptions()
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	for i, msg := range messages {
		wg.Add(1)
		go func(index int, message *Message) {
			defer wg.Done()
			result, err := m.Send(ctx, message, opt)
			if err != nil {
				result = &SendResult{
					MessageID: message.ID,
					Success:   false,
					Error:     err,
					SentAt:    time.Now(),
				}
			}
			results[index] = result
		}(i, msg)
	}

	wg.Wait()
	return results, nil
}

// sendWithRetry 发送通知（带重试）
func (m *Manager) sendWithRetry(ctx context.Context, notifier Notifier, message *Message, opts *SendOptions) (*SendResult, error) {
	var lastErr error
	startTime := time.Now()

	// 设置超时
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// 重试逻辑
	for i := 0; i <= opts.Retry; i++ {
		if i > 0 {
			// 增加重试统计
			m.incrementRetry(message.Type)

			// 等待重试间隔
			select {
			case <-ctx.Done():
				return nil, ErrSendTimeout
			case <-time.After(opts.RetryInterval):
			}
		}

		// 发送消息
		err := notifier.Send(ctx, message)
		if err == nil {
			// 发送成功
			result := &SendResult{
				MessageID: message.ID,
				Success:   true,
				Provider:  notifier.Name(),
				SentAt:    time.Now(),
				Cost:      time.Since(startTime),
			}

			// 更新统计
			m.incrementSuccess(message.Type)

			// 回调
			if opts.Callback != nil {
				opts.Callback(result)
			}

			return result, nil
		}

		lastErr = err

		// 记录日志
		if m.logger != nil {
			m.logger.Warn("send notification failed, retry %d/%d: %v", i+1, opts.Retry, err)
		}
	}

	// 重试用尽
	result := &SendResult{
		MessageID: message.ID,
		Success:   false,
		Error:     lastErr,
		Provider:  notifier.Name(),
		SentAt:    time.Now(),
		Cost:      time.Since(startTime),
	}

	// 更新统计
	m.incrementFailure(message.Type, lastErr)

	// 回调
	if opts.Callback != nil {
		opts.Callback(result)
	}

	return result, fmt.Errorf("%w: %v", ErrRetryExhausted, lastErr)
}

// validateMessage 验证消息
func (m *Manager) validateMessage(message *Message) error {
	if message == nil {
		return ErrInvalidMessage
	}

	if len(message.To) == 0 {
		return ErrEmptyRecipient
	}

	if message.Content == "" && message.Template == "" {
		return ErrEmptyContent
	}

	if message.Type == "" {
		message.Type = MessageTypeEmail // 默认邮件
	}

	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	return nil
}

// GetNotifier 获取指定类型的通知器
func (m *Manager) GetNotifier(msgType MessageType) (Notifier, error) {
	notifier, ok := m.notifiers[msgType]
	if !ok {
		return nil, ErrNotifierNotFound
	}
	return notifier, nil
}

// RegisterNotifier 注册通知器
func (m *Manager) RegisterNotifier(msgType MessageType, notifier Notifier) {
	m.notifiers[msgType] = notifier
	m.statsMu.Lock()
	if _, exists := m.stats[msgType]; !exists {
		m.stats[msgType] = &Stats{}
	}
	m.statsMu.Unlock()
}

// GetStats 获取统计信息
func (m *Manager) GetStats(msgType MessageType) *Stats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	stats, ok := m.stats[msgType]
	if !ok {
		return &Stats{}
	}

	// 返回副本
	return &Stats{
		TotalSent:    stats.TotalSent,
		TotalSuccess: stats.TotalSuccess,
		TotalFailed:  stats.TotalFailed,
		TotalRetry:   stats.TotalRetry,
		LastSentAt:   stats.LastSentAt,
		LastError:    stats.LastError,
	}
}

// GetAllStats 获取所有统计信息
func (m *Manager) GetAllStats() map[MessageType]*Stats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	result := make(map[MessageType]*Stats)
	for msgType := range m.stats {
		result[msgType] = m.GetStats(msgType)
	}
	return result
}

// incrementSuccess 增加成功计数
func (m *Manager) incrementSuccess(msgType MessageType) {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	if stats, ok := m.stats[msgType]; ok {
		stats.TotalSent++
		stats.TotalSuccess++
		stats.LastSentAt = time.Now()
	}
}

// incrementFailure 增加失败计数
func (m *Manager) incrementFailure(msgType MessageType, err error) {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	if stats, ok := m.stats[msgType]; ok {
		stats.TotalSent++
		stats.TotalFailed++
		stats.LastSentAt = time.Now()
		stats.LastError = err
	}
}

// incrementRetry 增加重试计数
func (m *Manager) incrementRetry(msgType MessageType) {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	if stats, ok := m.stats[msgType]; ok {
		stats.TotalRetry++
	}
}

// GetTemplateManager 获取模板管理器
func (m *Manager) GetTemplateManager() *TemplateManager {
	return m.template
}

// Close 关闭管理器
func (m *Manager) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil // 已经关闭
	}

	var errs []error

	// 关闭所有通知器
	for _, notifier := range m.notifiers {
		if err := notifier.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close notifiers failed: %v", errs)
	}

	return nil
}

// 全局管理器实例
var (
	globalManager     *Manager
	globalManagerOnce sync.Once
	globalManagerMu   sync.RWMutex
)

// InitGlobal 初始化全局管理器
func InitGlobal(config *Config) error {
	var err error
	globalManagerOnce.Do(func() {
		globalManagerMu.Lock()
		defer globalManagerMu.Unlock()

		globalManager, err = NewManager(config)
	})
	return err
}

// GetGlobal 获取全局管理器
func GetGlobal() *Manager {
	globalManagerMu.RLock()
	defer globalManagerMu.RUnlock()
	return globalManager
}

// Send 使用全局管理器发送通知
func Send(ctx context.Context, message *Message, opts ...*SendOptions) (*SendResult, error) {
	m := GetGlobal()
	if m == nil {
		return nil, fmt.Errorf("global manager not initialized")
	}
	return m.Send(ctx, message, opts...)
}

