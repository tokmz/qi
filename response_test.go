package qi

import (
	"testing"
)

func TestNewResponse(t *testing.T) {
	r := NewResponse(0, "success", map[string]string{"key": "value"})
	if r.Code != 0 {
		t.Errorf("Code = %d, want 0", r.Code)
	}
	if r.Message != "success" {
		t.Errorf("Message = %s, want success", r.Message)
	}
	if r.Data == nil {
		t.Error("Data should not be nil")
	}
	if r.TraceID != "" {
		t.Errorf("TraceID = %q, want empty", r.TraceID)
	}
}

func TestNewResponse_NilData(t *testing.T) {
	r := NewResponse(1000, "error", nil)
	if r.Data != nil {
		t.Errorf("Data = %v, want nil", r.Data)
	}
}

func TestResponse_SetTraceID(t *testing.T) {
	r := NewResponse(0, "ok", nil)
	r.SetTraceID("abc123")
	if r.TraceID != "abc123" {
		t.Errorf("TraceID = %q, want abc123", r.TraceID)
	}
}

func TestResponse_SetTraceID_Empty(t *testing.T) {
	r := NewResponse(0, "ok", "data")
	r.SetTraceID("")
	// empty trace_id 应被 omitempty 忽略
	if r.TraceID != "" {
		t.Errorf("TraceID = %q, want empty", r.TraceID)
	}
}
