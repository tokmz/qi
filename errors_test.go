package qi

import (
	"net/http"
	"testing"

	"github.com/tokmz/qi/pkg/errors"
)

func TestPredefinedErrors_Codes(t *testing.T) {
	cases := []struct {
		name   string
		err    *errors.Error
		code   int
		status int
	}{
		{"ErrServer", ErrServer, 1000, http.StatusInternalServerError},
		{"ErrBadRequest", ErrBadRequest, 1001, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, 1002, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, 1003, http.StatusForbidden},
		{"ErrNotFound", ErrNotFound, 1004, http.StatusNotFound},
		{"ErrConflict", ErrConflict, 1005, http.StatusConflict},
		{"ErrTooManyRequests", ErrTooManyRequests, 1006, http.StatusTooManyRequests},
		// 1100-1103 必须是 400 状态码
		{"ErrInvalidParams", ErrInvalidParams, 1100, http.StatusBadRequest},
		{"ErrMissingParams", ErrMissingParams, 1101, http.StatusBadRequest},
		{"ErrInvalidFormat", ErrInvalidFormat, 1102, http.StatusBadRequest},
		{"ErrOutOfRange", ErrOutOfRange, 1103, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.code {
				t.Errorf("Code = %d, want %d", tc.err.Code, tc.code)
			}
			if tc.err.Status() != tc.status {
				t.Errorf("Status() = %d, want %d", tc.err.Status(), tc.status)
			}
		})
	}
}

func TestPredefinedErrors_Immutable(t *testing.T) {
	original := ErrNotFound.Code
	_ = ErrNotFound.WithMessage("changed")
	if ErrNotFound.Code != original {
		t.Error("WithMessage should not modify the sentinel error")
	}
	_ = ErrNotFound.WithStatus(http.StatusBadRequest)
	if ErrNotFound.Status() != http.StatusNotFound {
		t.Error("WithStatus should not modify the sentinel error")
	}
}
