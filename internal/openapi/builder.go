package openapi

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	operations []Operation
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Add(op Operation) error {
	if strings.TrimSpace(op.Method) == "" {
		return fmt.Errorf("openapi: operation method is required")
	}
	if strings.TrimSpace(op.Path) == "" {
		return fmt.Errorf("openapi: operation path is required")
	}

	op.Method = strings.ToUpper(strings.TrimSpace(op.Method))
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.operations {
		if existing.Method == op.Method && existing.Path == op.Path {
			return fmt.Errorf("openapi: duplicate operation %s %s", op.Method, op.Path)
		}
		if op.OperationID != "" && existing.OperationID == op.OperationID {
			return fmt.Errorf("openapi: duplicate operationId %s", op.OperationID)
		}
	}
	r.operations = append(r.operations, op)
	return nil
}

func (r *Registry) List() []Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Operation, len(r.operations))
	copy(out, r.operations)
	return out
}

type Builder struct {
	opts Options
}

func NewBuilder(opts Options) *Builder {
	return &Builder{opts: opts}
}

func (b *Builder) Build(reg *Registry, analyzer *Analyzer) (*Document, error) {
	doc := &Document{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       b.opts.Title,
			Description: b.opts.Description,
			Version:     b.opts.Version,
		},
		Servers: append([]Server(nil), b.opts.Servers...),
		Paths:   make(map[string]*PathItem),
	}

	tagSet := make(map[string]struct{})
	for _, op := range reg.List() {
		pathItem := doc.Paths[op.Path]
		if pathItem == nil {
			pathItem = &PathItem{}
			doc.Paths[op.Path] = pathItem
		}

		opObj, err := b.buildOperation(op, analyzer)
		if err != nil {
			return nil, err
		}

		switch op.Method {
		case http.MethodGet:
			pathItem.Get = opObj
		case http.MethodPost:
			pathItem.Post = opObj
		case http.MethodPut:
			pathItem.Put = opObj
		case http.MethodDelete:
			pathItem.Delete = opObj
		case http.MethodPatch:
			pathItem.Patch = opObj
		case http.MethodHead:
			pathItem.Head = opObj
		case http.MethodOptions:
			pathItem.Options = opObj
		default:
			return nil, fmt.Errorf("openapi: unsupported method %s", op.Method)
		}

		for _, tag := range op.Tags {
			if tag != "" {
				tagSet[tag] = struct{}{}
			}
		}
	}

	for name, node := range analyzer.Components() {
		schema := b.buildSchema(node)
		if schema == nil {
			continue
		}
		if doc.Components.Schemas == nil {
			doc.Components.Schemas = make(map[string]*Schema)
		}
		doc.Components.Schemas[name] = schema
	}

	if len(b.opts.SecuritySchemes) > 0 {
		if doc.Components.SecuritySchemes == nil {
			doc.Components.SecuritySchemes = make(map[string]*SecurityScheme, len(b.opts.SecuritySchemes))
		}
		for name, scheme := range b.opts.SecuritySchemes {
			if scheme == nil {
				continue
			}
			cp := *scheme
			doc.Components.SecuritySchemes[name] = &cp
		}
	}

	tags := mapKeys(tagSet)
	sort.Strings(tags)
	for _, tag := range tags {
		doc.Tags = append(doc.Tags, Tag{Name: tag})
	}

	return doc, nil
}

func (b *Builder) buildOperation(op Operation, analyzer *Analyzer) (*OperationObject, error) {
	out := &OperationObject{
		OperationID: op.OperationID,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        append([]string(nil), op.Tags...),
		Deprecated:  op.Deprecated,
		Responses:   make(map[string]*APIResponse),
	}

	if req := op.Request; req != nil {
		for _, part := range []struct {
			value any
			in    ParamIn
		}{
			{value: req.PathParams, in: ParamInPath},
			{value: req.QueryParams, in: ParamInQuery},
			{value: req.Headers, in: ParamInHeader},
			{value: req.Cookies, in: ParamInCookie},
		} {
			params, err := analyzer.AnalyzeParameters(part.value, part.in)
			if err != nil {
				return nil, err
			}
			for _, param := range params {
				out.Parameters = append(out.Parameters, b.buildParameter(param))
			}
		}

		if req.Body != nil {
			schemaNode, err := analyzer.AnalyzeBody(req.Body)
			if err != nil {
				return nil, err
			}
			contentType := req.BodyContentType
			if contentType == "" {
				contentType = "application/json"
			}
			out.RequestBody = &RequestBody{
				Required: req.BodyRequired,
				Content: map[string]*MediaType{
					contentType: {
						Schema: b.buildSchema(schemaNode),
					},
				},
			}
		}
	}

	if len(op.Responses) == 0 {
		out.Responses["200"] = &APIResponse{Description: "OK"}
	} else {
		for _, resp := range op.Responses {
			key := strconv.Itoa(resp.Status)
			if resp.Status <= 0 {
				key = "default"
			}
			item := &APIResponse{
				Description: resp.Description,
			}
			if item.Description == "" {
				item.Description = http.StatusText(resp.Status)
				if item.Description == "" {
					item.Description = "Response"
				}
			}
			if resp.Body != nil {
				schemaNode, err := analyzer.AnalyzeResponse(resp.Body)
				if err != nil {
					return nil, err
				}
				contentType := resp.ContentType
				if contentType == "" {
					contentType = "application/json"
				}
				item.Content = map[string]*MediaType{
					contentType: {
						Schema: b.buildSchema(schemaNode),
					},
				}
			}
			out.Responses[key] = item
		}
	}

	for _, sec := range op.Security {
		item := make(map[string][]string, len(sec))
		for k, v := range sec {
			item[k] = append([]string(nil), v...)
		}
		out.Security = append(out.Security, item)
	}

	sort.Slice(out.Parameters, func(i, j int) bool {
		if out.Parameters[i].In == out.Parameters[j].In {
			return out.Parameters[i].Name < out.Parameters[j].Name
		}
		return out.Parameters[i].In < out.Parameters[j].In
	})
	return out, nil
}

func (b *Builder) buildParameter(spec ParameterSpec) *Parameter {
	return &Parameter{
		Name:        spec.Name,
		In:          string(spec.In),
		Description: spec.Description,
		Required:    spec.Required,
		Deprecated:  spec.Deprecated,
		Schema:      b.buildSchema(spec.Schema),
		Example:     spec.Example,
		Style:       spec.Style,
		Explode:     spec.Explode,
	}
}

func (b *Builder) buildSchema(node *SchemaNode) *Schema {
	if node == nil {
		return nil
	}

	if node.Ref != "" && schemaNodeHasRefSiblings(node) {
		meta := &Schema{
			Description:      node.Description,
			Nullable:         node.Nullable,
			Deprecated:       node.Deprecated,
			Required:         append([]string(nil), node.Required...),
			Enum:             append([]any(nil), node.Enum...),
			Default:          node.Default,
			Example:          node.Example,
			MinLength:        node.Constraints.MinLength,
			MaxLength:        node.Constraints.MaxLength,
			Minimum:          node.Constraints.Minimum,
			Maximum:          node.Constraints.Maximum,
			ExclusiveMinimum: node.Constraints.ExclusiveMinimum,
			ExclusiveMaximum: node.Constraints.ExclusiveMaximum,
			MinItems:         node.Constraints.MinItems,
			MaxItems:         node.Constraints.MaxItems,
			Pattern:          node.Constraints.Pattern,
			XConstraints:     cloneStringMap(node.Constraints.Raw),
		}
		if node.Items != nil {
			meta.Items = b.buildSchema(node.Items)
		}
		if node.AdditionalProperties != nil {
			meta.AdditionalProperties = b.buildSchema(node.AdditionalProperties)
		}
		if len(node.Properties) > 0 {
			meta.Properties = make(map[string]*Schema, len(node.Properties))
			for name, child := range node.Properties {
				meta.Properties[name] = b.buildSchema(child)
			}
		}
		return &Schema{
			AllOf: []*Schema{
				{Ref: node.Ref},
				meta,
			},
		}
	}

	out := &Schema{
		Ref:              node.Ref,
		Type:             node.Type,
		Format:           node.Format,
		Description:      node.Description,
		Nullable:         node.Nullable,
		Deprecated:       node.Deprecated,
		Required:         append([]string(nil), node.Required...),
		Enum:             append([]any(nil), node.Enum...),
		Default:          node.Default,
		Example:          node.Example,
		MinLength:        node.Constraints.MinLength,
		MaxLength:        node.Constraints.MaxLength,
		Minimum:          node.Constraints.Minimum,
		Maximum:          node.Constraints.Maximum,
		ExclusiveMinimum: node.Constraints.ExclusiveMinimum,
		ExclusiveMaximum: node.Constraints.ExclusiveMaximum,
		MinItems:         node.Constraints.MinItems,
		MaxItems:         node.Constraints.MaxItems,
		Pattern:          node.Constraints.Pattern,
		XConstraints:     cloneStringMap(node.Constraints.Raw),
	}

	if node.Items != nil {
		out.Items = b.buildSchema(node.Items)
	}
	if node.AdditionalProperties != nil {
		out.AdditionalProperties = b.buildSchema(node.AdditionalProperties)
	}
	if len(node.Properties) > 0 {
		out.Properties = make(map[string]*Schema, len(node.Properties))
		for name, child := range node.Properties {
			out.Properties[name] = b.buildSchema(child)
		}
	}
	return out
}

func schemaNodeHasRefSiblings(node *SchemaNode) bool {
	return node.Description != "" ||
		node.Nullable ||
		node.Deprecated ||
		len(node.Required) > 0 ||
		len(node.Enum) > 0 ||
		node.Default != nil ||
		node.Example != nil ||
		node.Constraints.MinLength != nil ||
		node.Constraints.MaxLength != nil ||
		node.Constraints.Minimum != nil ||
		node.Constraints.Maximum != nil ||
		node.Constraints.ExclusiveMinimum ||
		node.Constraints.ExclusiveMaximum ||
		node.Constraints.MinItems != nil ||
		node.Constraints.MaxItems != nil ||
		node.Constraints.Pattern != "" ||
		len(node.Constraints.Raw) > 0 ||
		node.Items != nil ||
		node.AdditionalProperties != nil ||
		len(node.Properties) > 0
}

func cloneStringMap(v map[string]string) map[string]string {
	if len(v) == 0 {
		return nil
	}
	out := make(map[string]string, len(v))
	for k, val := range v {
		out[k] = val
	}
	return out
}
