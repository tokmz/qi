package errors

import (
	stderrors "errors"
	"net/http"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	err := New(1001, "failed")

	if err.Code != 1001 {
		t.Fatalf("unexpected code: got %d", err.Code)
	}
	if err.Message != "failed" {
		t.Fatalf("unexpected message: got %q", err.Message)
	}
	if err.Status() != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d", err.Status())
	}
	if err.Error() != "failed" {
		t.Fatalf("unexpected error string: got %q", err.Error())
	}
}

func TestWithMethodsReturnClones(t *testing.T) {
	base := New(1001, "base")
	wrapped := base.WithErr(stderrors.New("db down"))
	updated := base.WithMessage("updated").WithStatus(http.StatusBadRequest)

	if base.err != nil {
		t.Fatal("base error mutated")
	}
	if base.Message != "base" {
		t.Fatalf("base message mutated: got %q", base.Message)
	}
	if base.Status() != http.StatusInternalServerError {
		t.Fatalf("base status mutated: got %d", base.Status())
	}

	if wrapped == base {
		t.Fatal("WithErr returned original instance")
	}
	if wrapped.Unwrap() == nil {
		t.Fatal("wrapped error missing cause")
	}
	if updated.Message != "updated" {
		t.Fatalf("unexpected updated message: got %q", updated.Message)
	}
	if updated.Status() != http.StatusBadRequest {
		t.Fatalf("unexpected updated status: got %d", updated.Status())
	}
}

func TestWithMessagefFormatsMessage(t *testing.T) {
	err := New(1001, "base").WithMessagef("user %d not found", 42)

	if err.Message != "user 42 not found" {
		t.Fatalf("unexpected message: got %q", err.Message)
	}
}

func TestWrapAndUnwrap(t *testing.T) {
	cause := stderrors.New("disk full")
	err := WrapWithStatus(cause, 2001, http.StatusBadGateway, "write failed")

	if !stderrors.Is(err, cause) {
		t.Fatal("wrapped error should match original cause")
	}
	if err.Unwrap() != cause {
		t.Fatal("unexpected unwrap result")
	}
	if err.Error() != "write failed: disk full" {
		t.Fatalf("unexpected error string: got %q", err.Error())
	}
	if GetCode(err) != 2001 {
		t.Fatalf("unexpected code: got %d", GetCode(err))
	}
	if GetStatus(err) != http.StatusBadGateway {
		t.Fatalf("unexpected status: got %d", GetStatus(err))
	}
}

func TestWrapNilFallsBackToNew(t *testing.T) {
	err := Wrap(nil, 3001, "plain")

	if err.Unwrap() != nil {
		t.Fatal("expected nil cause")
	}
	if err.Error() != "plain" {
		t.Fatalf("unexpected error string: got %q", err.Error())
	}
	if err.Status() != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d", err.Status())
	}
}

func TestWrapWithStatusNilFallsBackToNewWithStatus(t *testing.T) {
	err := WrapWithStatus(nil, 3002, http.StatusConflict, "conflict")

	if err.Unwrap() != nil {
		t.Fatal("expected nil cause")
	}
	if err.Status() != http.StatusConflict {
		t.Fatalf("unexpected status: got %d", err.Status())
	}
}

func TestStandardLibraryHelpers(t *testing.T) {
	target := New(4001, "target")
	err := target.WithErr(stderrors.New("inner"))

	if !Is(err, target) {
		t.Fatal("expected Is helper to match same code")
	}

	asErr, ok := As(err)
	if !ok {
		t.Fatal("expected As helper to succeed")
	}
	if asErr.Code != target.Code {
		t.Fatalf("unexpected code from As: got %d", asErr.Code)
	}
}

func TestGettersForForeignError(t *testing.T) {
	err := stderrors.New("foreign")

	if GetCode(err) != -1 {
		t.Fatalf("unexpected code: got %d", GetCode(err))
	}
	if GetStatus(err) != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d", GetStatus(err))
	}
	if IsCode(err, 1) {
		t.Fatal("foreign error should not match code")
	}
}

func TestNilReceiverSafety(t *testing.T) {
	var err *Error

	if err.Error() != "<nil>" {
		t.Fatalf("unexpected nil error string: got %q", err.Error())
	}
	if err.Status() != http.StatusInternalServerError {
		t.Fatalf("unexpected nil status: got %d", err.Status())
	}
	if err.Unwrap() != nil {
		t.Fatal("expected nil unwrap")
	}

	cloned := err.WithMessage("safe").WithStatus(http.StatusGone)
	if cloned.Message != "safe" {
		t.Fatalf("unexpected cloned message: got %q", cloned.Message)
	}
	if cloned.Status() != http.StatusGone {
		t.Fatalf("unexpected cloned status: got %d", cloned.Status())
	}
}
