package openapi

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type AnalyzeOptions struct {
	NameResolver        NameResolver
	FieldNamer          FieldNamer
	DescriptionProvider DescriptionProvider
	ConstraintParser    ConstraintParser
}

type Analyzer struct {
	cache *Cache
	opts  AnalyzeOptions

	mu         sync.RWMutex
	components map[componentKey]*SchemaNode
	building   map[componentBuildKey]string
}

type componentKey struct {
	Name string
	Mode AnalyzeMode
}

type componentBuildKey struct {
	Type reflect.Type
	Mode AnalyzeMode
}

func NewAnalyzer(opts AnalyzeOptions) *Analyzer {
	if opts.NameResolver == nil {
		opts.NameResolver = DefaultNameResolver{}
	}
	if opts.FieldNamer == nil {
		opts.FieldNamer = KeepCaseFieldNamer{}
	}
	if opts.DescriptionProvider == nil {
		opts.DescriptionProvider = NopDescriptionProvider{}
	}
	if opts.ConstraintParser == nil {
		opts.ConstraintParser = DefaultConstraintParser{}
	}

	return &Analyzer{
		cache:      NewCache(),
		opts:       opts,
		components: make(map[componentKey]*SchemaNode),
		building:   make(map[componentBuildKey]string),
	}
}

func (a *Analyzer) AnalyzeBody(v any) (*SchemaNode, error) {
	return a.analyzeRoot(typeOf(v), AnalyzeModeBody)
}

func (a *Analyzer) AnalyzeResponse(v any) (*SchemaNode, error) {
	return a.analyzeRoot(typeOf(v), AnalyzeModeResponse)
}

func (a *Analyzer) AnalyzeParameters(v any, in ParamIn) ([]ParameterSpec, error) {
	t := typeOf(v)
	if t == nil {
		return nil, nil
	}

	mode := modeFromParamIn(in)
	base := indirectType(t)
	if base.Kind() != reflect.Struct || isTimeType(base) {
		return nil, fmt.Errorf("openapi: parameters must be a struct, got %s", base.String())
	}

	key := CacheKey{Type: base, Mode: mode}
	if params, ok := a.cache.GetParams(key); ok {
		return params, nil
	}

	params, err := a.buildParameters(base, base, mode, in)
	if err != nil {
		return nil, err
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})
	a.cache.SetParams(key, params)
	return params, nil
}

func (a *Analyzer) Components() map[string]*SchemaNode {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make(map[string]*SchemaNode, len(a.components))
	for _, v := range a.components {
		if v == nil || v.Name == "" {
			continue
		}
		out[v.Name] = cloneNode(v)
	}
	return out
}

func (a *Analyzer) analyzeRoot(t reflect.Type, mode AnalyzeMode) (*SchemaNode, error) {
	if t == nil {
		return nil, nil
	}
	return a.analyzeSchema(t, mode)
}

func (a *Analyzer) analyzeSchema(t reflect.Type, mode AnalyzeMode) (*SchemaNode, error) {
	if t == nil {
		return nil, nil
	}

	key := CacheKey{Type: t, Mode: mode}
	if node, ok := a.cache.GetSchema(key); ok {
		return cloneNode(node), nil
	}

	node, err := a.analyzeSchemaUncached(t, mode)
	if err != nil {
		return nil, err
	}
	a.cache.SetSchema(key, node)
	return cloneNode(node), nil
}

func (a *Analyzer) analyzeSchemaUncached(t reflect.Type, mode AnalyzeMode) (*SchemaNode, error) {
	if t.Kind() == reflect.Ptr {
		node, err := a.analyzeSchema(t.Elem(), mode)
		if err != nil {
			return nil, err
		}
		return wrapRefNode(cloneNode(node), func(n *SchemaNode) {
			n.Nullable = true
		}), nil
	}

	if isTimeType(t) {
		return &SchemaNode{Type: "string", Format: "date-time", GoType: t}, nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return &SchemaNode{Type: "boolean", GoType: t}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &SchemaNode{Type: "integer", Format: "int32", GoType: t}, nil
	case reflect.Int64:
		return &SchemaNode{Type: "integer", Format: "int64", GoType: t}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &SchemaNode{Type: "integer", Format: "int32", GoType: t}, nil
	case reflect.Uint64:
		return &SchemaNode{Type: "integer", Format: "int64", GoType: t}, nil
	case reflect.Float32:
		return &SchemaNode{Type: "number", Format: "float", GoType: t}, nil
	case reflect.Float64:
		return &SchemaNode{Type: "number", Format: "double", GoType: t}, nil
	case reflect.String:
		return &SchemaNode{Type: "string", GoType: t}, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return &SchemaNode{Type: "string", Format: "byte", GoType: t}, nil
		}
		items, err := a.analyzeSchema(t.Elem(), mode)
		if err != nil {
			return nil, err
		}
		return &SchemaNode{Type: "array", Items: items, GoType: t}, nil
	case reflect.Array:
		items, err := a.analyzeSchema(t.Elem(), mode)
		if err != nil {
			return nil, err
		}
		return &SchemaNode{Type: "array", Items: items, GoType: t}, nil
	case reflect.Map:
		value, err := a.analyzeSchema(t.Elem(), mode)
		if err != nil {
			return nil, err
		}
		return &SchemaNode{Type: "object", AdditionalProperties: value, GoType: t}, nil
	case reflect.Interface:
		return &SchemaNode{Type: "object", GoType: t}, nil
	case reflect.Struct:
		if shouldUseComponent(t) {
			return a.analyzeNamedStruct(t, mode)
		}
		return a.buildInlineStructSchema(t, mode)
	default:
		return &SchemaNode{Type: "string", GoType: t}, nil
	}
}

func (a *Analyzer) analyzeNamedStruct(t reflect.Type, mode AnalyzeMode) (*SchemaNode, error) {
	name := schemaNameForMode(a.opts.NameResolver.SchemaName(t), mode)
	compKey := componentKey{Name: name, Mode: mode}
	buildKey := componentBuildKey{Type: t, Mode: mode}

	a.mu.RLock()
	existing, ok := a.components[compKey]
	buildingName, building := a.building[buildKey]
	a.mu.RUnlock()

	if ok && existing != nil {
		return &SchemaNode{Ref: "#/components/schemas/" + name, Name: name, GoType: t, Mode: mode}, nil
	}
	if building && buildingName == name {
		return &SchemaNode{Ref: "#/components/schemas/" + name, Name: name, GoType: t, Mode: mode}, nil
	}

	a.mu.Lock()
	if _, exists := a.components[compKey]; !exists {
		a.components[compKey] = &SchemaNode{Name: name, GoType: t, Mode: mode, Type: "object"}
	}
	a.building[buildKey] = name
	a.mu.Unlock()

	node, err := a.buildInlineStructSchema(t, mode)
	if err != nil {
		return nil, err
	}
	node.Name = name
	node.GoType = t
	node.Mode = mode
	if desc, ok := a.opts.DescriptionProvider.TypeDescription(t); ok && node.Description == "" {
		node.Description = desc
	}

	a.mu.Lock()
	a.components[compKey] = node
	delete(a.building, buildKey)
	a.mu.Unlock()

	return &SchemaNode{Ref: "#/components/schemas/" + name, Name: name, GoType: t, Mode: mode}, nil
}

func (a *Analyzer) buildInlineStructSchema(t reflect.Type, mode AnalyzeMode) (*SchemaNode, error) {
	properties, required, err := a.buildStructProperties(t, t, mode)
	if err != nil {
		return nil, err
	}

	node := &SchemaNode{
		Type:       "object",
		GoType:     t,
		Mode:       mode,
		Properties: properties,
		Required:   required,
	}
	if desc, ok := a.opts.DescriptionProvider.TypeDescription(t); ok {
		node.Description = desc
	}
	return node, nil
}

func (a *Analyzer) buildStructProperties(owner, current reflect.Type, mode AnalyzeMode) (map[string]*SchemaNode, []string, error) {
	props := make(map[string]*SchemaNode)
	requiredSet := make(map[string]struct{})

	for i := 0; i < current.NumField(); i++ {
		field := current.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if field.Anonymous && isEmbeddable(field.Type) {
			embeddedProps, embeddedRequired, err := a.buildStructProperties(owner, indirectType(field.Type), mode)
			if err != nil {
				return nil, nil, err
			}
			for name, schema := range embeddedProps {
				props[name] = schema
			}
			for _, name := range embeddedRequired {
				requiredSet[name] = struct{}{}
			}
			continue
		}

		tagInfo, err := ParseFieldTags(field, mode, a.opts.FieldNamer)
		if err != nil {
			return nil, nil, err
		}
		if tagInfo.Ignore {
			continue
		}

		schema, err := a.analyzeSchema(field.Type, mode)
		if err != nil {
			return nil, nil, err
		}
		constraints, err := a.opts.ConstraintParser.Parse(field)
		if err != nil {
			return nil, nil, err
		}

		name := tagInfo.Name
		if name == "" {
			name = field.Name
		}
		schema = applyTagAndConstraintMetadata(schema, owner, field, tagInfo, constraints, a.opts.DescriptionProvider)
		props[name] = schema

		required := constraints.Required
		if tagInfo.RequiredOverride != nil {
			required = *tagInfo.RequiredOverride
		}
		if required {
			requiredSet[name] = struct{}{}
		}
	}

	required := mapKeys(requiredSet)
	sort.Strings(required)
	return props, required, nil
}

func (a *Analyzer) buildParameters(owner, current reflect.Type, mode AnalyzeMode, in ParamIn) ([]ParameterSpec, error) {
	var params []ParameterSpec

	for i := 0; i < current.NumField(); i++ {
		field := current.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if field.Anonymous && isEmbeddable(field.Type) {
			embedded, err := a.buildParameters(owner, indirectType(field.Type), mode, in)
			if err != nil {
				return nil, err
			}
			params = append(params, embedded...)
			continue
		}

		tagInfo, err := ParseFieldTags(field, mode, a.opts.FieldNamer)
		if err != nil {
			return nil, err
		}
		if tagInfo.Ignore {
			continue
		}

		schema, err := a.analyzeSchema(field.Type, mode)
		if err != nil {
			return nil, err
		}
		constraints, err := a.opts.ConstraintParser.Parse(field)
		if err != nil {
			return nil, err
		}
		schema = applyTagAndConstraintMetadata(schema, owner, field, tagInfo, constraints, a.opts.DescriptionProvider)

		required := constraints.Required
		if tagInfo.RequiredOverride != nil {
			required = *tagInfo.RequiredOverride
		}
		if in == ParamInPath {
			required = true
		}

		name := tagInfo.Name
		if name == "" {
			name = field.Name
		}

		params = append(params, ParameterSpec{
			Name:        name,
			In:          in,
			Description: schema.Description,
			Required:    required,
			Deprecated:  schema.Deprecated,
			Schema:      schema,
			Default:     schema.Default,
			Example:     schema.Example,
			Style:       tagInfo.Style,
			Explode:     tagInfo.Explode,
		})
	}

	return params, nil
}

func applyTagAndConstraintMetadata(schema *SchemaNode, owner reflect.Type, field reflect.StructField, tagInfo TagInfo, constraints ConstraintSet, descProvider DescriptionProvider) *SchemaNode {
	out := wrapRefNode(cloneNode(schema), nil)
	out.Constraints = mergeConstraints(out.Constraints, constraints)

	if out.Description == "" {
		out.Description = tagInfo.Description
	}
	if out.Description == "" {
		if desc, ok := descProvider.FieldDescription(owner, field); ok {
			out.Description = desc
		}
	}
	if len(tagInfo.Enum) > 0 {
		out.Enum = append([]any(nil), tagInfo.Enum...)
	}
	if tagInfo.Default != nil {
		out.Default = tagInfo.Default
	}
	if tagInfo.Example != nil {
		out.Example = tagInfo.Example
	}
	if tagInfo.Format != "" {
		out.Format = tagInfo.Format
	}
	if tagInfo.Deprecated {
		out.Deprecated = true
	}
	if tagInfo.NullableOverride != nil {
		out.Nullable = *tagInfo.NullableOverride
	}
	if len(out.Enum) == 0 && len(out.Constraints.Enum) > 0 {
		for _, item := range out.Constraints.Enum {
			out.Enum = append(out.Enum, parseLiteral(item, indirectType(field.Type)))
		}
	}
	if out.Format == "" && out.Constraints.Format != "" {
		out.Format = out.Constraints.Format
	}
	return out
}

type DefaultNameResolver struct{}

func (DefaultNameResolver) SchemaName(t reflect.Type) string {
	base := indirectType(t)
	if base.Name() == "" {
		return sanitizeName(base.String())
	}
	if base.PkgPath() == "" {
		return sanitizeName(base.Name())
	}
	return sanitizeName(base.PkgPath() + "." + base.Name())
}

func sanitizeName(v string) string {
	replacer := strings.NewReplacer("/", ".", "-", "_", "*", "", "[", "", "]", "", " ", "_")
	return replacer.Replace(v)
}

func schemaNameForMode(base string, mode AnalyzeMode) string {
	switch mode {
	case AnalyzeModeQuery:
		return base + ".Query"
	case AnalyzeModePath:
		return base + ".Path"
	case AnalyzeModeHeader:
		return base + ".Header"
	case AnalyzeModeCookie:
		return base + ".Cookie"
	case AnalyzeModeResponse:
		return base + ".Response"
	default:
		return base + ".Body"
	}
}

func typeOf(v any) reflect.Type {
	if v == nil {
		return nil
	}
	return reflect.TypeOf(v)
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func shouldUseComponent(t reflect.Type) bool {
	t = indirectType(t)
	return t.Kind() == reflect.Struct && t.Name() != "" && t.PkgPath() != "" && !isTimeType(t)
}

func isEmbeddable(t reflect.Type) bool {
	base := indirectType(t)
	return base.Kind() == reflect.Struct && !isTimeType(base)
}

func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeOf(time.Time{})
}

func modeFromParamIn(in ParamIn) AnalyzeMode {
	switch in {
	case ParamInPath:
		return AnalyzeModePath
	case ParamInHeader:
		return AnalyzeModeHeader
	case ParamInCookie:
		return AnalyzeModeCookie
	default:
		return AnalyzeModeQuery
	}
}

func mergeConstraints(base, extra ConstraintSet) ConstraintSet {
	out := base
	if extra.Required {
		out.Required = true
	}
	if len(extra.Enum) > 0 {
		out.Enum = append([]string(nil), extra.Enum...)
	}
	if extra.MinLength != nil {
		out.MinLength = extra.MinLength
	}
	if extra.MaxLength != nil {
		out.MaxLength = extra.MaxLength
	}
	if extra.Minimum != nil {
		out.Minimum = extra.Minimum
	}
	if extra.Maximum != nil {
		out.Maximum = extra.Maximum
	}
	if extra.ExclusiveMinimum {
		out.ExclusiveMinimum = true
	}
	if extra.ExclusiveMaximum {
		out.ExclusiveMaximum = true
	}
	if extra.MinItems != nil {
		out.MinItems = extra.MinItems
	}
	if extra.MaxItems != nil {
		out.MaxItems = extra.MaxItems
	}
	if extra.Pattern != "" {
		out.Pattern = extra.Pattern
	}
	if extra.Format != "" {
		out.Format = extra.Format
	}
	if len(extra.Raw) > 0 {
		merged := make(map[string]string, len(base.Raw)+len(extra.Raw))
		for k, v := range base.Raw {
			merged[k] = v
		}
		for k, v := range extra.Raw {
			merged[k] = v
		}
		out.Raw = merged
	}
	return out
}

func cloneNode(node *SchemaNode) *SchemaNode {
	if node == nil {
		return nil
	}

	out := *node
	if node.Required != nil {
		out.Required = append([]string(nil), node.Required...)
	}
	if node.Enum != nil {
		out.Enum = append([]any(nil), node.Enum...)
	}
	if node.Properties != nil {
		out.Properties = make(map[string]*SchemaNode, len(node.Properties))
		for k, v := range node.Properties {
			out.Properties[k] = cloneNode(v)
		}
	}
	if node.Items != nil {
		out.Items = cloneNode(node.Items)
	}
	if node.AdditionalProperties != nil {
		out.AdditionalProperties = cloneNode(node.AdditionalProperties)
	}
	if len(node.Constraints.Raw) > 0 {
		out.Constraints.Raw = make(map[string]string, len(node.Constraints.Raw))
		for k, v := range node.Constraints.Raw {
			out.Constraints.Raw[k] = v
		}
	}
	return &out
}

func wrapRefNode(node *SchemaNode, mutate func(*SchemaNode)) *SchemaNode {
	if node == nil || node.Ref == "" {
		if mutate != nil && node != nil {
			mutate(node)
		}
		return node
	}

	wrapped := &SchemaNode{
		Mode:        node.Mode,
		Description: node.Description,
		Nullable:    node.Nullable,
		Deprecated:  node.Deprecated,
		Enum:        append([]any(nil), node.Enum...),
		Default:     node.Default,
		Example:     node.Example,
		Constraints: mergeConstraints(ConstraintSet{}, node.Constraints),
	}
	wrapped.Items = cloneNode(node.Items)
	wrapped.AdditionalProperties = cloneNode(node.AdditionalProperties)
	if node.Properties != nil {
		wrapped.Properties = make(map[string]*SchemaNode, len(node.Properties))
		for k, v := range node.Properties {
			wrapped.Properties[k] = cloneNode(v)
		}
	}
	if node.Required != nil {
		wrapped.Required = append([]string(nil), node.Required...)
	}
	wrapped.Ref = node.Ref
	if mutate != nil {
		mutate(wrapped)
	}
	return wrapped
}

func mapKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
