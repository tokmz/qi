package cache

import "errors"

var (
	// ErrNotFound key 不存在
	ErrNotFound = errors.New("cache: key not found")
	// ErrNilValue 写入值为 nil
	ErrNilValue = errors.New("cache: nil value")
	// ErrNotSupported 当前驱动不支持该操作
	ErrNotSupported = errors.New("cache: operation not supported")
	// ErrLockNotHeld 解锁时 token 不匹配（锁已过期或被其他持有者占用）
	ErrLockNotHeld = errors.New("cache: lock not held")
)
