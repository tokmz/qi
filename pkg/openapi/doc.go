package openapi

import "reflect"

// DocOption 路由文档元数据
type DocOption struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Security    []string
	NoSecurity  bool         // 显式标记不需要认证（覆盖组级默认）
	ReqType     reflect.Type // 可选：为非泛型路由手动指定请求类型
	RespType    reflect.Type // 可选：为非泛型路由手动指定响应类型
}

// Doc 创建文档选项
func Doc(opts ...func(*DocOption)) *DocOption {
	d := &DocOption{}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Summary 设置摘要
func Summary(s string) func(*DocOption) {
	return func(d *DocOption) { d.Summary = s }
}

// Desc 设置详细描述
func Desc(s string) func(*DocOption) {
	return func(d *DocOption) { d.Description = s }
}

// Tags 设置标签
func Tags(t ...string) func(*DocOption) {
	return func(d *DocOption) { d.Tags = t }
}

// Deprecated 标记为已弃用
func Deprecated() func(*DocOption) {
	return func(d *DocOption) { d.Deprecated = true }
}

// Security 设置安全方案引用
func Security(s ...string) func(*DocOption) {
	return func(d *DocOption) { d.Security = s }
}

// NoSecurity 显式标记不需要认证
func NoSecurity() func(*DocOption) {
	return func(d *DocOption) { d.NoSecurity = true }
}

// RequestType 为非泛型路由手动指定请求类型（传入零值实例）
func RequestType(t any) func(*DocOption) {
	return func(d *DocOption) {
		if t != nil {
			d.ReqType = reflect.TypeOf(t)
			// 如果传入的是指针，取其指向的类型
			if d.ReqType.Kind() == reflect.Ptr {
				d.ReqType = d.ReqType.Elem()
			}
		}
	}
}

// ResponseType 为非泛型路由手动指定响应类型（传入零值实例）
func ResponseType(t any) func(*DocOption) {
	return func(d *DocOption) {
		if t != nil {
			d.RespType = reflect.TypeOf(t)
			if d.RespType.Kind() == reflect.Ptr {
				d.RespType = d.RespType.Elem()
			}
		}
	}
}
