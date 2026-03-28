package openapi

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

type TagInfo struct {
	Name        string
	Description string
	Example     any
	Default     any
	Enum        []any
	Format      string
	Deprecated  bool

	RequiredOverride *bool
	NullableOverride *bool

	Style   string
	Explode *bool

	Ignore bool
}

func ParseFieldTags(field reflect.StructField, mode AnalyzeMode, namer FieldNamer) (TagInfo, error) {
	info := TagInfo{}

	sourceTag := tagForMode(mode)
	raw := field.Tag.Get(sourceTag)
	if raw != "" {
		name, ignore := parseNameTag(raw)
		if ignore {
			info.Ignore = true
			return info, nil
		}
		info.Name = name
	}

	// 查询参数模式下，跳过仅有 uri tag 且无 form tag 的字段（避免与路径参数重复）
	if mode == AnalyzeModeQuery && raw == "" {
		if field.Tag.Get("uri") != "" {
			info.Ignore = true
			return info, nil
		}
	}

	if openapiTag := field.Tag.Get("openapi"); openapiTag != "" {
		if openapiTag == "-" {
			info.Ignore = true
			return info, nil
		}
		parseOpenAPITag(&info, openapiTag)
	}

	if info.Description == "" {
		info.Description = strings.TrimSpace(field.Tag.Get("description"))
	}
	if info.Description == "" {
		info.Description = strings.TrimSpace(field.Tag.Get("desc"))
	}

	baseType := indirectType(field.Type)
	if raw := strings.TrimSpace(field.Tag.Get("example")); raw != "" {
		info.Example = parseLiteral(raw, baseType)
	}
	if raw := strings.TrimSpace(field.Tag.Get("default")); raw != "" {
		info.Default = parseLiteral(raw, baseType)
	}
	if raw := strings.TrimSpace(field.Tag.Get("enum")); raw != "" {
		for _, item := range splitCSV(raw) {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			info.Enum = append(info.Enum, parseLiteral(item, baseType))
		}
	}
	if raw := strings.TrimSpace(field.Tag.Get("format")); raw != "" {
		info.Format = raw
	}

	if info.Name == "" {
		if namer == nil {
			namer = KeepCaseFieldNamer{}
		}
		info.Name = namer.FieldName(field, mode)
	}
	return info, nil
}

func parseOpenAPITag(info *TagInfo, tag string) {
	for _, token := range splitCSV(tag) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		key, value, hasValue := strings.Cut(token, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "name":
			if hasValue {
				info.Name = value
			}
		case "required":
			v := true
			if hasValue {
				if parsed, err := strconv.ParseBool(value); err == nil {
					v = parsed
				}
			}
			info.RequiredOverride = &v
		case "nullable":
			v := true
			if hasValue {
				if parsed, err := strconv.ParseBool(value); err == nil {
					v = parsed
				}
			}
			info.NullableOverride = &v
		case "deprecated":
			v := true
			if hasValue {
				if parsed, err := strconv.ParseBool(value); err == nil {
					v = parsed
				}
			}
			info.Deprecated = v
		case "style":
			if hasValue {
				info.Style = value
			}
		case "explode":
			if hasValue {
				if parsed, err := strconv.ParseBool(value); err == nil {
					info.Explode = &parsed
				}
			}
		}
	}
}

func parseNameTag(tag string) (string, bool) {
	if tag == "-" {
		return "", true
	}
	name := strings.TrimSpace(strings.Split(tag, ",")[0])
	return name, false
}

func tagForMode(mode AnalyzeMode) string {
	switch mode {
	case AnalyzeModeQuery:
		return "form"
	case AnalyzeModePath:
		return "uri"
	case AnalyzeModeHeader:
		return "header"
	case AnalyzeModeCookie:
		return "cookie"
	default:
		return "json"
	}
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	return strings.Split(v, ",")
}

func parseLiteral(raw string, typ reflect.Type) any {
	if typ == nil {
		return raw
	}

	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "\"") {
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			return v
		}
	}

	switch indirectType(typ).Kind() {
	case reflect.Bool:
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return v
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
			return v
		}
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
	case reflect.String:
		return raw
	}
	return raw
}

type SnakeCaseFieldNamer struct{}

func (SnakeCaseFieldNamer) FieldName(field reflect.StructField, _ AnalyzeMode) string {
	return toSnakeCase(field.Name)
}

type LowerCamelFieldNamer struct{}

func (LowerCamelFieldNamer) FieldName(field reflect.StructField, _ AnalyzeMode) string {
	if field.Name == "" {
		return ""
	}
	runes := []rune(field.Name)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

type KebabCaseFieldNamer struct{}

func (KebabCaseFieldNamer) FieldName(field reflect.StructField, _ AnalyzeMode) string {
	return strings.ReplaceAll(toSnakeCase(field.Name), "_", "-")
}

type RecommendedFieldNamer struct{}

func (RecommendedFieldNamer) FieldName(field reflect.StructField, mode AnalyzeMode) string {
	switch mode {
	case AnalyzeModeQuery, AnalyzeModePath, AnalyzeModeCookie:
		return SnakeCaseFieldNamer{}.FieldName(field, mode)
	case AnalyzeModeHeader:
		return KebabCaseFieldNamer{}.FieldName(field, mode)
	default:
		return KeepCaseFieldNamer{}.FieldName(field, mode)
	}
}

func toSnakeCase(v string) string {
	if v == "" {
		return ""
	}

	var b strings.Builder
	runes := []rune(v)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
