package openapi

type Manager struct {
	registry *Registry
	analyzer *Analyzer
	builder  *Builder
	opts     Options
}

func New(opts ...Option) *Manager {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	analyzeOpts := AnalyzeOptions{
		NameResolver:        cfg.NameResolver,
		FieldNamer:          cfg.FieldNamer,
		DescriptionProvider: cfg.DescriptionProvider,
		ConstraintParser:    cfg.ConstraintParser,
	}

	return &Manager{
		registry: NewRegistry(),
		analyzer: NewAnalyzer(analyzeOpts),
		builder:  NewBuilder(cfg),
		opts:     cfg,
	}
}

func (m *Manager) AddOperation(op Operation) error {
	return m.registry.Add(op)
}

func (m *Manager) MustAddOperation(op Operation) *Manager {
	if err := m.AddOperation(op); err != nil {
		panic(err)
	}
	return m
}

func (m *Manager) AddOperations(ops ...Operation) error {
	for _, op := range ops {
		if err := m.AddOperation(op); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) MustAddOperations(ops ...Operation) *Manager {
	if err := m.AddOperations(ops...); err != nil {
		panic(err)
	}
	return m
}

func (m *Manager) Build() (*Document, error) {
	return m.builder.Build(m.registry, m.analyzer)
}

func (m *Manager) BuildWithInfo(title, version string, opts ...Option) (*Document, error) {
	clone := m.CloneWithOptions(opts...)
	if title != "" {
		clone.opts.Title = title
	}
	if version != "" {
		clone.opts.Version = version
	}
	clone.builder = NewBuilder(clone.opts)
	return clone.Build()
}

func (m *Manager) MustBuild() *Document {
	doc, err := m.Build()
	if err != nil {
		panic(err)
	}
	return doc
}

func (m *Manager) CloneWithOptions(opts ...Option) *Manager {
	cfg := m.opts
	if cfg.Servers != nil {
		cfg.Servers = append([]Server(nil), cfg.Servers...)
	}
	if cfg.SecuritySchemes != nil {
		cloned := make(map[string]*SecurityScheme, len(cfg.SecuritySchemes))
		for name, scheme := range cfg.SecuritySchemes {
			if scheme == nil {
				continue
			}
			cp := *scheme
			cloned[name] = &cp
		}
		cfg.SecuritySchemes = cloned
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	clone := &Manager{
		registry: NewRegistry(),
		analyzer: NewAnalyzer(AnalyzeOptions{
			NameResolver:        cfg.NameResolver,
			FieldNamer:          cfg.FieldNamer,
			DescriptionProvider: cfg.DescriptionProvider,
			ConstraintParser:    cfg.ConstraintParser,
		}),
		builder: NewBuilder(cfg),
		opts:    cfg,
	}

	for _, op := range m.registry.List() {
		if err := clone.registry.Add(op); err != nil {
			panic(err)
		}
	}
	return clone
}
