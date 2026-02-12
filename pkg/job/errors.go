package job

import (
	"errors"
	"fmt"
)

// 错误定义
var (
	// ErrJobNotFound 任务不存在
	ErrJobNotFound = errors.New("job not found")

	// ErrJobAlreadyExists 任务已存在
	ErrJobAlreadyExists = errors.New("job already exists")

	// ErrJobPaused 任务已暂停
	ErrJobPaused = errors.New("job is paused")

	// ErrJobRunning 任务正在运行
	ErrJobRunning = errors.New("job is running")

	// ErrInvalidCronExpression 无效的 Cron 表达式
	ErrInvalidCronExpression = errors.New("invalid cron expression")

	// ErrSchedulerNotStarted 调度器未启动
	ErrSchedulerNotStarted = errors.New("scheduler not started")

	// ErrSchedulerAlreadyStarted 调度器已启动
	ErrSchedulerAlreadyStarted = errors.New("scheduler already started")

	// ErrHandlerNotFound 处理器不存在
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrStorageClosed 存储已关闭
	ErrStorageClosed = errors.New("storage is closed")

	// ErrInvalidJobName 无效的任务名称
	ErrInvalidJobName = errors.New("invalid job name")

	// ErrInvalidPayload 无效的任务参数
	ErrInvalidPayload = errors.New("invalid payload")
)

// Error 任务调度包错误
type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError 创建新错误
func NewError(code int, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 错误码定义
const (
	ErrCodeJobNotFound         = 1001
	ErrCodeJobAlreadyExists    = 1002
	ErrCodeJobPaused           = 1003
	ErrCodeJobRunning          = 1004
	ErrCodeInvalidCron         = 1005
	ErrCodeSchedulerNotStarted = 1006
	ErrCodeSchedulerStarted    = 1007
	ErrCodeHandlerNotFound     = 1008
	ErrCodeStorageClosed       = 1009
	ErrCodeExecutionFailed     = 1010
	ErrCodeInvalidJobName      = 1011
	ErrCodeInvalidPayload      = 1012
)
