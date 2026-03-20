package openapi

type SecurityRequirement map[string][]string

type Operation struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool

	Request   *Request
	Responses []Response
	Security  []SecurityRequirement
}

type Request struct {
	PathParams  any
	QueryParams any
	Headers     any
	Cookies     any
	Body        any

	BodyRequired    bool
	BodyContentType string
}

type Response struct {
	Status      int
	Description string
	Body        any
	ContentType string
}
