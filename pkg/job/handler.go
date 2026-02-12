package job

import (
	"context"
)

// Handler 任务处理器接口
type Handler interface {
	// Execute 执行任务
	Execute(ctx context.Context, payload string) (string, error)
}

// HandlerFunc 函数式处理器
type HandlerFunc func(ctx context.Context, payload string) (string, error)

// Execute 实现 Handler 接口
func (f HandlerFunc) Execute(ctx context.Context, payload string) (string, error) {
	return f(ctx, payload)
}
