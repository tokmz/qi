package openapi

import "reflect"

type AnalyzeMode string

const (
	AnalyzeModeBody     AnalyzeMode = "body"
	AnalyzeModeResponse AnalyzeMode = "response"
	AnalyzeModePath     AnalyzeMode = "path"
	AnalyzeModeQuery    AnalyzeMode = "query"
	AnalyzeModeHeader   AnalyzeMode = "header"
	AnalyzeModeCookie   AnalyzeMode = "cookie"
)

type ParamIn string

const (
	ParamInPath   ParamIn = "path"
	ParamInQuery  ParamIn = "query"
	ParamInHeader ParamIn = "header"
	ParamInCookie ParamIn = "cookie"
)

type SchemaNode struct {
	Ref         string
	Name        string
	GoType      reflect.Type
	Mode        AnalyzeMode
	Type        string
	Format      string
	Description string
	Nullable    bool
	Deprecated  bool

	Properties           map[string]*SchemaNode
	Required             []string
	Items                *SchemaNode
	AdditionalProperties *SchemaNode

	Enum    []any
	Default any
	Example any

	Constraints ConstraintSet
}

type ParameterSpec struct {
	Name        string
	In          ParamIn
	Description string
	Required    bool
	Deprecated  bool

	Schema  *SchemaNode
	Default any
	Example any
	Style   string
	Explode *bool
}

type NameResolver interface {
	SchemaName(t reflect.Type) string
}

type FieldNamer interface {
	FieldName(field reflect.StructField, mode AnalyzeMode) string
}

type DescriptionProvider interface {
	TypeDescription(t reflect.Type) (string, bool)
	FieldDescription(owner reflect.Type, field reflect.StructField) (string, bool)
}

type KeepCaseFieldNamer struct{}

func (KeepCaseFieldNamer) FieldName(field reflect.StructField, _ AnalyzeMode) string {
	return field.Name
}

type NopDescriptionProvider struct{}

func (NopDescriptionProvider) TypeDescription(reflect.Type) (string, bool) {
	return "", false
}

func (NopDescriptionProvider) FieldDescription(reflect.Type, reflect.StructField) (string, bool) {
	return "", false
}
