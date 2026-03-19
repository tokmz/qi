package array

import (
	"math/rand"
	"strings"
	"time"
)

// SliceContains 检查切片是否包含指定元素
func SliceContains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// SliceRemove 移除切片中的指定元素
func SliceRemove[T comparable](slice []T, item T) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if v != item {
			result = append(result, v)
		}
	}
	// 确保返回空切片而不是 nil
	if result == nil {
		return []T{}
	}
	return result
}

// SliceRemoveAtIndex 移除指定索引的元素
func SliceRemoveAtIndex[T any](slice []T, index int) []T {
	if index < 0 || index >= len(slice) {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}

// SliceUnique 去重
func SliceUnique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// SliceIntersect 获取两个切片的交集
func SliceIntersect[T comparable](slice1, slice2 []T) []T {
	set := make(map[T]struct{})
	for _, v := range slice2 {
		set[v] = struct{}{}
	}

	result := make([]T, 0)
	for _, v := range slice1 {
		if _, exists := set[v]; exists {
			result = append(result, v)
			delete(set, v) // 避免重复
		}
	}
	// 确保返回空切片而不是 nil
	if result == nil {
		return []T{}
	}
	return result
}

// SliceUnion 获取两个切片的并集
func SliceUnion[T comparable](slice1, slice2 []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice1)+len(slice2))

	for _, v := range slice1 {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	for _, v := range slice2 {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}

// SliceDifference 获取 slice1 中有但 slice2 中没有的元素
func SliceDifference[T comparable](slice1, slice2 []T) []T {
	set := make(map[T]struct{})
	for _, v := range slice2 {
		set[v] = struct{}{}
	}

	result := make([]T, 0)
	for _, v := range slice1 {
		if _, exists := set[v]; !exists {
			result = append(result, v)
		}
	}
	return result
}

// SliceFilter 过滤切片
func SliceFilter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// SliceMap 映射切片
func SliceMap[T, R any](slice []T, mapper func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = mapper(v)
	}
	return result
}

// SliceReduce 规约切片
func SliceReduce[T, R any](slice []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, v := range slice {
		result = reducer(result, v)
	}
	return result
}

// SliceReverse 反转切片
func SliceReverse[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

// SliceShuffle 打乱切片
// 注意：每次调用都会创建新的随机源，避免全局锁竞争
func SliceShuffle[T any](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)

	// 使用当前时间纳秒作为种子，每次调用创建新的随机源
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(result) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// SliceChunk 将切片分块
func SliceChunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return [][]T{slice}
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// SliceFind 查找满足条件的第一个元素
func SliceFind[T any](slice []T, predicate func(T) bool) (T, bool) {
	for _, v := range slice {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// SliceFindIndex 查找满足条件的第一个元素的索引
func SliceFindIndex[T any](slice []T, predicate func(T) bool) int {
	for i, v := range slice {
		if predicate(v) {
			return i
		}
	}
	return -1
}

// SliceCount 统计满足条件的元素数量
func SliceCount[T any](slice []T, predicate func(T) bool) int {
	count := 0
	for _, v := range slice {
		if predicate(v) {
			count++
		}
	}
	return count
}

// SliceEvery 检查是否所有元素都满足条件
func SliceEvery[T any](slice []T, predicate func(T) bool) bool {
	for _, v := range slice {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// SliceSome 检查是否有至少一个元素满足条件
func SliceSome[T any](slice []T, predicate func(T) bool) bool {
	for _, v := range slice {
		if predicate(v) {
			return true
		}
	}
	return false
}

// SliceEqual 检查两个切片是否相等
func SliceEqual[T comparable](slice1, slice2 []T) bool {
	if len(slice1) != len(slice2) {
		return false
	}
	for i, v := range slice1 {
		if slice2[i] != v {
			return false
		}
	}
	return true
}

// SliceIsEmpty 检查切片是否为空
func SliceIsEmpty[T any](slice []T) bool {
	return len(slice) == 0
}

// SliceIsNotEmpty 检查切片是否非空
func SliceIsNotEmpty[T any](slice []T) bool {
	return len(slice) > 0
}

// SliceFirst 获取第一个元素
func SliceFirst[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[0], true
}

// SliceLast 获取最后一个元素
func SliceLast[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[len(slice)-1], true
}

// SliceMin 获取最小值
func SliceMin[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	minVal := slice[0]
	for _, v := range slice[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal, true
}

// SliceMax 获取最大值
func SliceMax[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	maxVal := slice[0]
	for _, v := range slice[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal, true
}

// SliceSum 求和（适用于数字类型）
func SliceSum[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](slice []T) T {
	var sum T
	for _, v := range slice {
		sum += v
	}
	return sum
}

// SliceAverage 求平均值
func SliceAverage[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	sum := SliceSum(slice)
	return sum / T(len(slice)), true
}

// SliceJoin 连接切片元素为字符串
func SliceJoin[T any](slice []T, separator string, toString func(T) string) string {
	if len(slice) == 0 {
		return ""
	}
	if len(slice) == 1 {
		return toString(slice[0])
	}

	var builder strings.Builder
	builder.WriteString(toString(slice[0]))
	for _, v := range slice[1:] {
		builder.WriteString(separator)
		builder.WriteString(toString(v))
	}
	return builder.String()
}

// SliceSplit 字符串分割为切片
func SliceSplit(s string, separator string) []string {
	if s == "" {
		return []string{}
	}
	if separator == "" {
		return []string{s}
	}

	var result []string
	start := 0
	for i := 0; i <= len(s)-len(separator); i++ {
		if s[i:i+len(separator)] == separator {
			result = append(result, s[start:i])
			start = i + len(separator)
			i += len(separator) - 1
		}
	}
	result = append(result, s[start:])
	return result
}
