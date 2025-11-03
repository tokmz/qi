package scheduler

import "errors"

var (
	// ErrCronDisabled 定时任务未启用
	ErrCronDisabled = errors.New("cron scheduler is disabled")

	// ErrCronNotStarted 定时任务调度器未启动
	ErrCronNotStarted = errors.New("cron scheduler not started")

	// ErrCronAlreadyStarted 定时任务调度器已启动
	ErrCronAlreadyStarted = errors.New("cron scheduler already started")

	// ErrInvalidTimezone 无效的时区
	ErrInvalidTimezone = errors.New("invalid timezone")

	// ErrInvalidJobName 无效的任务名称
	ErrInvalidJobName = errors.New("invalid job name")

	// ErrInvalidCronSpec 无效的 Cron 表达式
	ErrInvalidCronSpec = errors.New("invalid cron spec")

	// ErrDuplicateJobName 重复的任务名称
	ErrDuplicateJobName = errors.New("duplicate job name")

	// ErrJobNotFound 任务不存在
	ErrJobNotFound = errors.New("job not found")

	// ErrJobAlreadyExists 任务已存在
	ErrJobAlreadyExists = errors.New("job already exists")

	// ErrJobNotRegistered 任务未注册
	ErrJobNotRegistered = errors.New("job handler not registered")

	// ErrJobTimeout 任务执行超时
	ErrJobTimeout = errors.New("job execution timeout")

	// ErrJobPanic 任务执行 panic
	ErrJobPanic = errors.New("job execution panic")
)

