package openapi

import (
	"mime/multipart"
	"reflect"
	"strings"
	"time"
)

// well-known types for special handling
var (
	timeType       = reflect.TypeOf(time.Time{})
	fileHeaderType = reflect.TypeOf(multipart.FileHeader{})
)

// SchemaBuilder 将 reflect.Type 转换为 OpenAPI Schema
type SchemaBuilder struct {
	// Schemas 收集到的命名 schema（用于 components/schemas）
	Schemas map[string]*Schema
}

// NewSchemaBuilder 创建 SchemaBuilder
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		Schemas: make(map[string]*Schema),
	}
}

// Build 将 Go 类型转换为 OpenAPI Schema
func (b *SchemaBuilder) Build(t reflect.Type) *Schema {
	return b.buildSchema(t, make(map[reflect.Type]bool))
}

// buildSchema 递归构建 Schema，visited 用于检测循环引用
func (b *SchemaBuilder) buildSchema(t reflect.Type, visited map[reflect.Type]bool) *Schema {
	// 解引用指针
	nullable := false
	if t.Kind() == reflect.Ptr {
		nullable = true
		t = t.Elem()
	}

	s := b.buildType(t, visited)
	if nullable {
		s.Nullable = true
	}
	return s
}

// buildType 根据类型 Kind 分发构建
func (b *SchemaBuilder) buildType(t reflect.Type, visited map[reflect.Type]bool) *Schema {
	// 特殊类型优先匹配
	switch {
	case t == timeType:
		return &Schema{Type: "string", Format: "date-time"}
	case t == fileHeaderType:
		return &Schema{Type: "string", Format: "binary"}
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}

	case reflect.Bool:
		return &Schema{Type: "boolean"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: "integer", Format: "int32"}

	case reflect.Int64, reflect.Uint64:
		return &Schema{Type: "integer", Format: "int64"}

	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}

	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}

	case reflect.Slice, reflect.Array:
		// []byte → string, format: byte
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{
			Type:  "array",
			Items: b.buildSchema(t.Elem(), visited),
		}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return &Schema{Type: "object"}
		}
		// map[string]any → object without additionalProperties schema
		if t.Elem().Kind() == reflect.Interface {
			return &Schema{Type: "object"}
		}
		return &Schema{
			Type:                 "object",
			AdditionalProperties: b.buildSchema(t.Elem(), visited),
		}

	case reflect.Struct:
		return b.buildStruct(t, visited)

	case reflect.Interface:
		return &Schema{}

	default:
		return &Schema{Type: "string"}
	}
}

// buildStruct 构建 struct 类型的 Schema
func (b *SchemaBuilder) buildStruct(t reflect.Type, visited map[reflect.Type]bool) *Schema {
	// 循环引用检测：返回 $ref
	if visited[t] {
		name := t.Name()
		if name == "" {
			return &Schema{Type: "object"}
		}
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	visited[t] = true
	defer delete(visited, t)

	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	b.collectFields(t, s, visited)

	// 注册到 components（仅命名类型，避免重复写入）
	if t.Name() != "" && len(s.Properties) > 0 {
		if _, exists := b.Schemas[t.Name()]; !exists {
			b.Schemas[t.Name()] = s
		}
	}

	return s
}

// collectFields 递归收集 struct 字段（含嵌入 struct 展开）
func (b *SchemaBuilder) collectFields(t reflect.Type, s *Schema, visited map[reflect.Type]bool) {
	for i := range t.NumField() {
		field := t.Field(i)

		// 嵌入 struct：展开字段到父级（即使类型名未导出也要展开）
		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				b.collectFields(ft, s, visited)
				continue
			}
		}

		// 跳过未导出字段
		if !field.IsExported() {
			continue
		}

		// 确定字段名（json tag 优先）
		name := fieldName(field)
		if name == "-" {
			continue
		}

		// 构建字段 schema
		fieldSchema := b.buildSchema(field.Type, visited)

		// 应用 binding tag 约束
		if bindTag := field.Tag.Get("binding"); bindTag != "" {
			constraints := ParseBindingTag(bindTag)
			constraints.ApplyToSchema(fieldSchema)
			if constraints.Required {
				s.Required = append(s.Required, name)
			}
		}
		if validateTag := field.Tag.Get("validate"); validateTag != "" {
			constraints := ParseBindingTag(validateTag)
			constraints.ApplyToSchema(fieldSchema)
			if constraints.Required {
				if !containsString(s.Required, name) {
					s.Required = append(s.Required, name)
				}
			}
		}

		// 应用 description（来自 comment 或 desc tag）
		if desc := field.Tag.Get("desc"); desc != "" {
			fieldSchema.Description = desc
		}

		// 应用 example tag
		if example := field.Tag.Get("example"); example != "" {
			fieldSchema.Example = example
		}

		s.Properties[name] = fieldSchema
	}
}

// fieldName 从 struct field 提取 JSON 字段名
func fieldName(f reflect.StructField) string {
	// json tag 优先
	if tag := f.Tag.Get("json"); tag != "" {
		name, _, _ := strings.Cut(tag, ",")
		if name != "" {
			return name
		}
	}
	// form tag 次之
	if tag := f.Tag.Get("form"); tag != "" {
		name, _, _ := strings.Cut(tag, ",")
		if name != "" {
			return name
		}
	}
	return f.Name
}

// HasFileUpload 检查类型是否包含文件上传字段（含嵌入 struct 递归检查）
func HasFileUpload(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := range t.NumField() {
		field := t.Field(i)
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft == fileHeaderType {
			return true
		}
		// []*multipart.FileHeader
		if ft.Kind() == reflect.Slice {
			elem := ft.Elem()
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem == fileHeaderType {
				return true
			}
		}
		// 递归检查嵌入 struct
		if field.Anonymous {
			if ft.Kind() == reflect.Struct && HasFileUpload(ft) {
				return true
			}
		}
	}
	return false
}

// containsString 检查 slice 是否包含指定字符串
func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
