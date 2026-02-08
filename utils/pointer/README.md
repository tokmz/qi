# Pointer

指针操作工具包，提供指针的创建、转换、判空等常用操作。

## 安装

```bash
go get qi/pkg/pointer
```

## 基础转换

| 函数 | 说明 |
|------|------|
| `Of[T any](v T) *T` | 返回值的指针 |
| `Get[T any](p *T) T` | 获取指针的值，nil 返回零值 |
| `GetOrDefault[T any](p *T, defaultValue T) T` | 获取指针的值，nil 返回默认值 |

## 判断

| 函数 | 说明 |
|------|------|
| `IsNil[T any](p *T) bool` | 检查指针是否为 nil |
| `IsNotNil[T any](p *T) bool` | 检查指针是否不为 nil |

## 条件取值

| 函数 | 说明 |
|------|------|
| `Coalesce[T any](pointers ...*T) *T` | 返回第一个非 nil 指针 |

## 转换

| 函数 | 说明 |
|------|------|
| `ToString(p *string) string` | `*string` 转字符串 |
| `ToInt(p *int) int` | `*int` 转 int |
| `ToInt64(p *int64) int64` | `*int64` 转 int64 |
| `ToFloat64(p *float64) float64` | `*float64` 转 float64 |
| `ToBool(p *bool) bool` | `*bool` 转 bool |

## 字符串转换

| 函数 | 说明 |
|------|------|
| `ToStringPtr(s string) *string` | 字符串转 `*string` |
| `ToIntPtr(s string) *int` | 字符串转 `*int` |
| `ToInt64Ptr(s string) *int64` | 字符串转 `*int64` |
| `ToFloat64Ptr(s string) *float64` | 字符串转 `*float64` |
| `ToBoolPtr(s string) *bool` | 字符串转 `*bool` |

## 切片转换

| 函数 | 说明 |
|------|------|
| `ToSlicePtr[T any](slice []T) []*T` | 切片转为指针切片 |
| `ToSlice[T any](ptrs []*T) []T` | 指针切片转为值切片 |
| `FilterNotNil[T any](ptrs []*T) []*T` | 过滤掉 nil 指针 |
| `MapPtr[T, R any](slice []T, mapper func(T) *R) []*R` | 映射指针切片 |

## Map 操作

| 函数 | 说明 |
|------|------|
| `ToMapPtr[K comparable, V any](slice []V, keyFunc func(V) K) map[K]*V` | 切片转为 map（值为指针） |

## 指针比较

| 函数 | 说明 |
|------|------|
| `Equal[T any](p1, p2 *T) bool` | 比较两个指针是否指向同一地址 |

## 使用示例

```go
package main

import (
	"fmt"

	"qi/pkg/pointer"
)

func main() {
	// 基础转换
	n := 10
	p := pointer.Of(42)
	fmt.Println(pointer.Get(p))        // 42
	fmt.Println(pointer.GetOrDefault(p, 0)) // 42
	fmt.Println(pointer.GetOrDefault[int](nil, 100)) // 100

	// 判断
	fmt.Println(pointer.IsNil(p))   // false
	fmt.Println(pointer.IsNil[int](nil)) // true

	// 条件取值
	var p1, p2, p3 *int
	p2 = pointer.Of(20)
	result := pointer.Coalesce(p1, p2, p3)
	fmt.Println(pointer.Get(result)) // 20

	// 字符串转换
	numStr := "123"
	numPtr := pointer.ToIntPtr(numStr)
	fmt.Println(pointer.ToInt(numPtr)) // 123

	// 切片转换
	slice := []int{1, 2, 3}
	ptrSlice := pointer.ToSlicePtr(slice)
	fmt.Println(pointer.Get(ptrSlice[0])) // 1

	// 过滤 nil
	mixed := []*int{nil, pointer.Of(1), nil, pointer.Of(2)}
	filtered := pointer.FilterNotNil(mixed)
	fmt.Println(len(filtered)) // 2

	// 指针比较
	a := pointer.Of(10)
	b := pointer.Of(10)
	c := pointer.Of(20)
	fmt.Println(pointer.Equal(a, b)) // true
	fmt.Println(pointer.Equal(a, c)) // false
}
```
