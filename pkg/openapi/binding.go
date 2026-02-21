package openapi

import (
	"strconv"
	"strings"
)

// BindingConstraints 从 binding tag 解析出的约束
type BindingConstraints struct {
	Required  bool
	Minimum   *float64
	Maximum   *float64
	MinLength *int
	MaxLength *int
	Format    string   // email, uri, uuid
	Enum      []any    // oneof 值列表
}

// ParseBindingTag 解析 binding tag 提取 OpenAPI 约束
//
//	binding:"required,min=1,max=100"
//	binding:"email"
//	binding:"oneof=active inactive"
func ParseBindingTag(tag string) BindingConstraints {
	var c BindingConstraints
	if tag == "" || tag == "-" {
		return c
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch {
		case part == "required":
			c.Required = true

		case strings.HasPrefix(part, "min="):
			if v, err := strconv.ParseFloat(part[4:], 64); err == nil {
				c.Minimum = &v
			}

		case strings.HasPrefix(part, "max="):
			if v, err := strconv.ParseFloat(part[4:], 64); err == nil {
				c.Maximum = &v
			}

		case strings.HasPrefix(part, "len="):
			if v, err := strconv.Atoi(part[4:]); err == nil {
				c.MinLength = &v
				c.MaxLength = &v
			}

		case part == "email":
			c.Format = "email"

		case part == "url":
			c.Format = "uri"

		case part == "uuid":
			c.Format = "uuid"

		case strings.HasPrefix(part, "oneof="):
			values := strings.Fields(part[6:])
			for _, v := range values {
				c.Enum = append(c.Enum, v)
			}
		}
	}
	return c
}

// ApplyToSchema 将约束应用到 Schema 上
func (c *BindingConstraints) ApplyToSchema(s *Schema) {
	if c.Minimum != nil {
		s.Minimum = c.Minimum
	}
	if c.Maximum != nil {
		s.Maximum = c.Maximum
	}
	if c.MinLength != nil {
		s.MinLength = c.MinLength
	}
	if c.MaxLength != nil {
		s.MaxLength = c.MaxLength
	}
	if c.Format != "" && s.Format == "" {
		s.Format = c.Format
	}
	if len(c.Enum) > 0 {
		s.Enum = c.Enum
	}
}

// FieldLocation 字段在 OpenAPI 中的位置
type FieldLocation int

const (
	LocBody   FieldLocation = iota // requestBody
	LocQuery                       // in: query
	LocPath                        // in: path
	LocHeader                      // in: header
)

// InferFieldLocation 根据 struct tag 和 HTTP 方法推断字段位置
//
// uri tag → path (所有方法)
// header tag → header (所有方法)
// GET/DELETE: 默认 → query
// POST/PUT/PATCH: 默认 → body
func InferFieldLocation(method string, hasURI, hasHeader bool) FieldLocation {
	if hasHeader {
		return LocHeader
	}
	if hasURI {
		return LocPath
	}

	switch method {
	case "GET", "DELETE":
		return LocQuery
	default:
		return LocBody
	}
}
