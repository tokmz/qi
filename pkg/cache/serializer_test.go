package cache

import (
	"bytes"
	"testing"
)

func TestJSONSerializer_RoundTrip(t *testing.T) {
	s := JSONSerializer{}

	tests := []struct {
		name  string
		value any
		dest  any
	}{
		{"string", "hello", new(string)},
		{"int", 42, new(int)},
		{"bool", true, new(bool)},
		{"struct", struct{ Name string }{"qi"}, &struct{ Name string }{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := s.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if len(b) == 0 {
				t.Fatal("marshaled bytes should not be empty")
			}
			if err := s.Unmarshal(b, tc.dest); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
		})
	}
}

func TestJSONSerializer_UnmarshalError(t *testing.T) {
	s := JSONSerializer{}
	var v int
	if err := s.Unmarshal([]byte("invalid{"), &v); err == nil {
		t.Fatal("expected unmarshal error for invalid JSON")
	}
}

func TestGOBSerializer_RoundTrip(t *testing.T) {
	s := GOBSerializer{}
	type point struct{ X, Y int }

	tests := []struct {
		name  string
		value any
		dest  any
	}{
		{"string", "hello", new(string)},
		{"int", 42, new(int)},
		{"struct", point{1, 2}, &point{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := s.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if len(b) == 0 {
				t.Fatal("marshaled bytes should not be empty")
			}
			if err := s.Unmarshal(b, tc.dest); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
		})
	}
}

func TestGOBSerializer_UnmarshalError(t *testing.T) {
	s := GOBSerializer{}
	var v int
	if err := s.Unmarshal([]byte("not gob"), &v); err == nil {
		t.Fatal("expected unmarshal error for invalid GOB")
	}
}

func TestSerializers_ProduceDifferentBytes(t *testing.T) {
	j := JSONSerializer{}
	g := GOBSerializer{}

	jb, _ := j.Marshal("test")
	gb, _ := g.Marshal("test")

	if bytes.Equal(jb, gb) {
		t.Fatal("JSON and GOB should produce different byte representations")
	}
}
