package openapi

type Document struct {
	OpenAPI    string               `json:"openapi" yaml:"openapi"`
	Info       Info                 `json:"info" yaml:"info"`
	Servers    []Server             `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths" yaml:"paths"`
	Components Components           `json:"components,omitempty" yaml:"components,omitempty"`
	Tags       []Tag                `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type PathItem struct {
	Get     *OperationObject `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *OperationObject `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *OperationObject `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *OperationObject `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch   *OperationObject `json:"patch,omitempty" yaml:"patch,omitempty"`
	Head    *OperationObject `json:"head,omitempty" yaml:"head,omitempty"`
	Options *OperationObject `json:"options,omitempty" yaml:"options,omitempty"`
}

type OperationObject struct {
	OperationID string                  `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Summary     string                  `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string                `json:"tags,omitempty" yaml:"tags,omitempty"`
	Deprecated  bool                    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Parameters  []*Parameter            `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]*APIResponse `json:"responses" yaml:"responses"`
	Security    []map[string][]string   `json:"security,omitempty" yaml:"security,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated  bool    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     any     `json:"example,omitempty" yaml:"example,omitempty"`
	Style       string  `json:"style,omitempty" yaml:"style,omitempty"`
	Explode     *bool   `json:"explode,omitempty" yaml:"explode,omitempty"`
}

type RequestBody struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                  `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]*MediaType `json:"content" yaml:"content"`
}

type APIResponse struct {
	Description string                `json:"description" yaml:"description"`
	Content     map[string]*MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type MediaType struct {
	Schema  *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example any     `json:"example,omitempty" yaml:"example,omitempty"`
}

type Schema struct {
	Ref                  string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	Type                 string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string             `json:"format,omitempty" yaml:"format,omitempty"`
	Description          string             `json:"description,omitempty" yaml:"description,omitempty"`
	Nullable             bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required             []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Enum                 []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default              any                `json:"default,omitempty" yaml:"default,omitempty"`
	Example              any                `json:"example,omitempty" yaml:"example,omitempty"`
	MinLength            *int               `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMinimum     bool               `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     bool               `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	MinItems             *int               `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	Pattern              string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	XConstraints         map[string]string  `json:"x-constraints,omitempty" yaml:"x-constraints,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type,omitempty" yaml:"type,omitempty"`
	Description  string `json:"description,omitempty" yaml:"description,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
	In           string `json:"in,omitempty" yaml:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
}
