package request

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// tracingTransport 在 RoundTripper 层注入 OTel trace headers
type tracingTransport struct {
	base http.RoundTripper
}

func newTracingTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &tracingTransport{base: base}
}

func (t *tracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	return t.base.RoundTrip(req)
}
