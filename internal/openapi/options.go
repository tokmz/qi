package openapi

type Options struct {
	Title       string
	Description string
	Version     string
	Servers     []Server

	NameResolver        NameResolver
	FieldNamer          FieldNamer
	DescriptionProvider DescriptionProvider
	ConstraintParser    ConstraintParser
	SecuritySchemes     map[string]*SecurityScheme
}

type Option func(*Options)

func defaultOptions() Options {
	return Options{
		Title:               "OpenAPI",
		Version:             "1.0.0",
		NameResolver:        DefaultNameResolver{},
		FieldNamer:          KeepCaseFieldNamer{},
		DescriptionProvider: NopDescriptionProvider{},
		ConstraintParser:    DefaultConstraintParser{},
	}
}

func WithTitle(v string) Option {
	return func(o *Options) {
		o.Title = v
	}
}

func WithDescription(v string) Option {
	return func(o *Options) {
		o.Description = v
	}
}

func WithVersion(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

func WithServers(v ...Server) Option {
	return func(o *Options) {
		o.Servers = append([]Server(nil), v...)
	}
}

func WithNameResolver(v NameResolver) Option {
	return func(o *Options) {
		if v != nil {
			o.NameResolver = v
		}
	}
}

func WithFieldNamer(v FieldNamer) Option {
	return func(o *Options) {
		if v != nil {
			o.FieldNamer = v
		}
	}
}

// WithRecommendedDefaults 应用推荐的默认策略。
// 当前主要收敛字段命名策略：
// - body/response: 保持 Go 字段名回退
// - query/path/cookie: snake_case
// - header: kebab-case
func WithRecommendedDefaults() Option {
	return func(o *Options) {
		o.FieldNamer = RecommendedFieldNamer{}
	}
}

func WithDescriptionProvider(v DescriptionProvider) Option {
	return func(o *Options) {
		if v != nil {
			o.DescriptionProvider = v
		}
	}
}

func WithConstraintParser(v ConstraintParser) Option {
	return func(o *Options) {
		if v != nil {
			o.ConstraintParser = v
		}
	}
}

func WithSecurityScheme(name string, scheme *SecurityScheme) Option {
	return func(o *Options) {
		if name == "" || scheme == nil {
			return
		}
		if o.SecuritySchemes == nil {
			o.SecuritySchemes = make(map[string]*SecurityScheme)
		}
		cp := *scheme
		o.SecuritySchemes[name] = &cp
	}
}
