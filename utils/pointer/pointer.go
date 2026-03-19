package pointer

import (
	"strconv"
)

// ===== 基础转换 =====

// Of 返回值的指针
func Of[T any](v T) *T {
	return &v
}

// Get 获取指针的值，如果指针为 nil 返回零值
func Get[T any](p *T) T {
	if p == nil {
		return *new(T)
	}
	return *p
}

// GetOrDefault 获取指针的值，如果指针为 nil 返回默认值
func GetOrDefault[T any](p *T, defaultValue T) T {
	if p == nil {
		return defaultValue
	}
	return *p
}

// ===== 判断 =====

// IsNil 检查指针是否为 nil
func IsNil[T any](p *T) bool {
	return p == nil
}

// IsNotNil 检查指针是否不为 nil
func IsNotNil[T any](p *T) bool {
	return p != nil
}

// ===== 条件取值 =====

// Coalesce 返回第一个非 nil 指针的值
func Coalesce[T any](pointers ...*T) *T {
	for _, p := range pointers {
		if p != nil {
			return p
		}
	}
	return nil
}

// ===== 转换 =====

// ToString 将指针转为字符串
func ToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ToInt 将指针转为 int
func ToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// ToInt64 将指针转为 int64
func ToInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// ToFloat64 将指针转为 float64
func ToFloat64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// ToBool 将指针转为 bool
func ToBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

// ===== 字符串转换 =====

// ToStringPtr 将字符串转为 *string
func ToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ToIntPtr 将字符串转为 *int
func ToIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &val
}

// ToInt64Ptr 将字符串转为 *int64
func ToInt64Ptr(s string) *int64 {
	if s == "" {
		return nil
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &val
}

// ToFloat64Ptr 将字符串转为 *float64
func ToFloat64Ptr(s string) *float64 {
	if s == "" {
		return nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &val
}

// ToBoolPtr 将字符串转为 *bool
func ToBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	val, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return &val
}

// ===== 切片转换 =====

// ToSlicePtr 将切片转为指针切片
func ToSlicePtr[T any](slice []T) []*T {
	if slice == nil {
		return nil
	}
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}

// ToSlice 从指针切片获取值切片
func ToSlice[T any](ptrs []*T) []T {
	if ptrs == nil {
		return nil
	}
	result := make([]T, len(ptrs))
	for i, p := range ptrs {
		if p != nil {
			result[i] = *p
		}
	}
	return result
}

// FilterNotNil 过滤掉 nil 指针
func FilterNotNil[T any](ptrs []*T) []*T {
	if ptrs == nil {
		return nil
	}
	result := make([]*T, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			result = append(result, p)
		}
	}
	return result
}

// MapPtr 映射指针切片
func MapPtr[T, R any](slice []T, mapper func(T) *R) []*R {
	if slice == nil {
		return nil
	}
	result := make([]*R, len(slice))
	for i, v := range slice {
		result[i] = mapper(v)
	}
	return result
}

// ===== Map 操作 =====

// ToMapPtr 将切片转为 map（值为指针）
func ToMapPtr[K comparable, V any](slice []V, keyFunc func(V) K) map[K]*V {
	if slice == nil {
		return nil
	}
	result := make(map[K]*V, len(slice))
	for i := range slice {
		k := keyFunc(slice[i])
		result[k] = &slice[i]
	}
	return result
}

// ===== 指针比较 =====

// Equal 比较两个指针是否指向同一个值（仅 nil 检查）
func Equal[T any](p1, p2 *T) bool {
	return p1 == p2
}
