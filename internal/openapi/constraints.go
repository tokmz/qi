package openapi

import (
	"reflect"
	"strconv"
	"strings"
)

type ConstraintSet struct {
	Required         bool
	Enum             []string
	MinLength        *int
	MaxLength        *int
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum bool
	ExclusiveMaximum bool
	MinItems         *int
	MaxItems         *int
	Pattern          string
	Format           string

	Raw map[string]string
}

type ConstraintParser interface {
	Parse(field reflect.StructField) (ConstraintSet, error)
}

type DefaultConstraintParser struct{}

func (DefaultConstraintParser) Parse(field reflect.StructField) (ConstraintSet, error) {
	var out ConstraintSet
	for _, raw := range []string{field.Tag.Get("binding"), field.Tag.Get("validate")} {
		parseConstraintTokens(&out, field.Type, raw)
	}
	return out, nil
}

func parseConstraintTokens(out *ConstraintSet, typ reflect.Type, tag string) {
	for _, token := range splitCSV(tag) {
		token = strings.TrimSpace(token)
		if token == "" || token == "omitempty" {
			continue
		}

		key, value, hasValue := strings.Cut(token, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "required":
			out.Required = true
		case "oneof":
			if hasValue {
				out.Enum = append(out.Enum[:0], strings.Fields(value)...)
			}
		case "min":
			applyMin(out, typ, value)
		case "max":
			applyMax(out, typ, value)
		case "len":
			applyLen(out, typ, value)
		case "gt":
			applyGT(out, typ, value)
		case "gte":
			applyGTE(out, typ, value)
		case "lt":
			applyLT(out, typ, value)
		case "lte":
			applyLTE(out, typ, value)
		case "email":
			out.Format = "email"
		case "url":
			out.Format = "uri"
		case "uuid":
			out.Format = "uuid"
		case "datetime":
			out.Format = "date-time"
		default:
			if out.Raw == nil {
				out.Raw = make(map[string]string)
			}
			if hasValue {
				out.Raw[key] = value
			} else {
				out.Raw[key] = ""
			}
		}
	}
}

func applyLen(out *ConstraintSet, typ reflect.Type, value string) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return
	}
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		out.MinLength = intPtr(n)
		out.MaxLength = intPtr(n)
	case reflect.Array, reflect.Slice:
		out.MinItems = intPtr(n)
		out.MaxItems = intPtr(n)
	}
}

func applyMin(out *ConstraintSet, typ reflect.Type, value string) {
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		if n, err := strconv.Atoi(value); err == nil {
			out.MinLength = intPtr(n)
		}
	case reflect.Array, reflect.Slice:
		if n, err := strconv.Atoi(value); err == nil {
			out.MinItems = intPtr(n)
		}
	default:
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			out.Minimum = floatPtr(n)
		}
	}
}

func applyMax(out *ConstraintSet, typ reflect.Type, value string) {
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		if n, err := strconv.Atoi(value); err == nil {
			out.MaxLength = intPtr(n)
		}
	case reflect.Array, reflect.Slice:
		if n, err := strconv.Atoi(value); err == nil {
			out.MaxItems = intPtr(n)
		}
	default:
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			out.Maximum = floatPtr(n)
		}
	}
}

func applyGT(out *ConstraintSet, typ reflect.Type, value string) {
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		if n, err := strconv.Atoi(value); err == nil {
			out.MinLength = intPtr(n + 1)
		}
	case reflect.Array, reflect.Slice:
		if n, err := strconv.Atoi(value); err == nil {
			out.MinItems = intPtr(n + 1)
		}
	default:
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			out.Minimum = floatPtr(n)
			out.ExclusiveMinimum = true
		}
	}
}

func applyGTE(out *ConstraintSet, typ reflect.Type, value string) {
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		if n, err := strconv.Atoi(value); err == nil {
			out.MinLength = intPtr(n)
		}
	case reflect.Array, reflect.Slice:
		if n, err := strconv.Atoi(value); err == nil {
			out.MinItems = intPtr(n)
		}
	default:
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			out.Minimum = floatPtr(n)
			out.ExclusiveMinimum = false
		}
	}
}

func applyLT(out *ConstraintSet, typ reflect.Type, value string) {
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		if n, err := strconv.Atoi(value); err == nil {
			out.MaxLength = intPtr(n - 1)
		}
	case reflect.Array, reflect.Slice:
		if n, err := strconv.Atoi(value); err == nil {
			out.MaxItems = intPtr(n - 1)
		}
	default:
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			out.Maximum = floatPtr(n)
			out.ExclusiveMaximum = true
		}
	}
}

func applyLTE(out *ConstraintSet, typ reflect.Type, value string) {
	base := indirectType(typ)
	switch base.Kind() {
	case reflect.String:
		if n, err := strconv.Atoi(value); err == nil {
			out.MaxLength = intPtr(n)
		}
	case reflect.Array, reflect.Slice:
		if n, err := strconv.Atoi(value); err == nil {
			out.MaxItems = intPtr(n)
		}
	default:
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			out.Maximum = floatPtr(n)
			out.ExclusiveMaximum = false
		}
	}
}

func intPtr(v int) *int {
	return &v
}

func floatPtr(v float64) *float64 {
	return &v
}
