# Array

切片工具包，提供常用的切片操作函数。

## 安装

```bash
go get qi/pkg/array
```

## 包含类型

- `Slice[T any]` - 泛型切片工具

## 基础操作

| 函数 | 说明 |
|------|------|
| `SliceContains[T comparable](slice []T, item T) bool` | 检查切片是否包含指定元素 |
| `SliceRemove[T comparable](slice []T, item T) []T` | 移除切片中的指定元素 |
| `SliceRemoveAtIndex[T any](slice []T, index int) []T` | 移除指定索引的元素 |
| `SliceUnique[T comparable](slice []T) []T` | 去重 |
| `SliceReverse[T any](slice []T) []T` | 反转切片 |
| `SliceShuffle[T any](slice []T) []T` | 打乱切片 |
| `SliceChunk[T any](slice []T, size int) [][]T` | 将切片分块 |
| `SliceSplit(s string, separator string) []string` | 字符串分割为切片 |
| `SliceJoin[T any](slice []T, separator string, toString func(T) string) string` | 连接切片元素为字符串 |

## 集合操作

| 函数 | 说明 |
|------|------|
| `SliceIntersect[T comparable](slice1, slice2 []T) []T` | 获取两个切片的交集 |
| `SliceUnion[T comparable](slice1, slice2 []T) []T` | 获取两个切片的并集 |
| `SliceDifference[T comparable](slice1, slice2 []T) []T` | 获取 slice1 中有但 slice2 中没有的元素 |

## 函数式操作

| 函数 | 说明 |
|------|------|
| `SliceFilter[T any](slice []T, predicate func(T) bool) []T` | 过滤切片 |
| `SliceMap[T, R any](slice []T, mapper func(T) R) []R` | 映射切片 |
| `SliceReduce[T, R any](slice []T, initial R, reducer func(R, T) R) R` | 规约切片 |
| `SliceFind[T any](slice []T, predicate func(T) bool) (T, bool)` | 查找满足条件的第一个元素 |
| `SliceFindIndex[T any](slice []T, predicate func(T) bool) int` | 查找满足条件的第一个元素的索引 |
| `SliceCount[T any](slice []T, predicate func(T) bool) int` | 统计满足条件的元素数量 |
| `SliceEvery[T any](slice []T, predicate func(T) bool) bool` | 检查是否所有元素都满足条件 |
| `SliceSome[T any](slice []T, predicate func(T) bool) bool` | 检查是否有至少一个元素满足条件 |

## 判断

| 函数 | 说明 |
|------|------|
| `SliceEqual[T comparable](slice1, slice2 []T) bool` | 检查两个切片是否相等 |
| `SliceIsEmpty[T any](slice []T) bool` | 检查切片是否为空 |
| `SliceIsNotEmpty[T any](slice []T) bool` | 检查切片是否非空 |

## 获取元素

| 函数 | 说明 |
|------|------|
| `SliceFirst[T any](slice []T) (T, bool)` | 获取第一个元素 |
| `SliceLast[T any](slice []T) (T, bool)` | 获取最后一个元素 |
| `SliceFind[T any](slice []T, predicate func(T) bool) (T, bool)` | 查找满足条件的第一个元素 |
| `SliceFindIndex[T any](slice []T, predicate func(T) bool) int` | 查找满足条件的第一个元素的索引 |

## 统计计算

| 函数 | 说明 |
|------|------|
| `SliceMin[T int\|int8\|int16\|int32\|int64\|uint\|uint8\|uint16\|uint32\|uint64\|float32\|float64](slice []T) (T, bool)` | 获取最小值 |
| `SliceMax[T int\|int8\|int16\|int32\|int64\|uint\|uint8\|uint16\|uint32\|uint64\|float32\|float64](slice []T) (T, bool)` | 获取最大值 |
| `SliceSum[T int\|int8\|int16\|int32\|int64\|uint\|uint8\|uint16\|uint32\|uint64\|float32\|float64](slice []T) T` | 求和 |
| `SliceAverage[T int\|int8\|int16\|int32\|int64\|uint\|uint8\|uint16\|uint32\|uint64\|float32\|float64](slice []T) (T, bool)` | 求平均值 |

## 使用示例

```go
package main

import (
	"fmt"

	"qi/pkg/array"
)

func main() {
	// 基础操作
	nums := []int{1, 2, 3, 4, 5}
	fmt.Println(array.SliceContains(nums, 3))  // true
	fmt.Println(array.SliceRemove(nums, 3))    // [1 2 4 5]
	fmt.Println(array.SliceUnique([]int{1, 2, 2, 3})) // [1 2 3]
	fmt.Println(array.SliceReverse(nums))       // [5 4 3 2 1]

	// 集合操作
	a := []int{1, 2, 3}
	b := []int{2, 3, 4}
	fmt.Println(array.SliceIntersect(a, b))   // [2 3]
	fmt.Println(array.SliceUnion(a, b))      // [1 2 3 4]
	fmt.Println(array.SliceDifference(a, b)) // [1]

	// 函数式操作
	evens := array.SliceFilter(nums, func(n int) bool {
		return n%2 == 0
	})
	doubles := array.SliceMap(nums, func(n int) int {
		return n * 2
	})
	sum := array.SliceReduce(nums, 0, func(acc, n int) int {
		return acc + n
	})
	fmt.Println(evens)   // [2 4]
	fmt.Println(doubles) // [2 4 6 8 10]
	fmt.Println(sum)     // 15

	// 统计计算
	fmt.Println(array.SliceMin(nums))   // 1, true
	fmt.Println(array.SliceMax(nums))   // 5, true
	fmt.Println(array.SliceSum(nums))   // 15
	fmt.Println(array.SliceAverage(nums)) // 3, true

	// 分块
	chunks := array.SliceChunk([]int{1, 2, 3, 4, 5}, 2)
	fmt.Println(chunks) // [[1 2] [3 4] [5]]
}
```
