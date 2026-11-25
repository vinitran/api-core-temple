package errorx

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestWrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		kind       Kind
		wantKind   Kind
		wantStatus int
	}{
		{"no rows through fmt wrap", fmt.Errorf("query: %w", sql.ErrNoRows), Database, NotExist, http.StatusNotFound},
		{"unique violation through fmt wrap", fmt.Errorf("insert: %w", &pgconn.PgError{Code: pgUniqueViolation}), Database, Exist, http.StatusConflict},
		{"other pg error", &pgconn.PgError{Code: "42P01"}, Service, Database, http.StatusInternalServerError},
		{"plain error uses given kind", errors.New("boom"), Invalid, Invalid, http.StatusBadRequest},
		{"existing kind is preserved", New(Authn, "nope"), Service, Authn, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Wrap(tt.err, tt.kind)
			var target *Error
			if !errors.As(err, &target) {
				t.Fatalf("Wrap() = %T, want *Error", err)
			}
			if target.Kind() != tt.wantKind {
				t.Errorf("kind = %v, want %v", target.Kind(), tt.wantKind)
			}
			if target.Status() != tt.wantStatus {
				t.Errorf("status = %d, want %d", target.Status(), tt.wantStatus)
			}
		})
	}
}

func TestWrapNil(t *testing.T) {
	t.Parallel()

	if err := Wrap(nil, Service); err != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", err)
	}
	if err := WrapMsg(nil, Service, "x"); err != nil {
		t.Fatalf("WrapMsg(nil) = %v, want nil", err)
	}
}

func TestMaskErrorMessage(t *testing.T) {
	t.Parallel()

	if got := MaskErrorMessage(WrapMsg(errors.New("dial tcp: secret host"), Service, "secret")); got != maskedMessage {
		t.Errorf("service error message = %q, want masked", got)
	}
	if got := MaskErrorMessage(New(Invalid, "missing code")); got != "missing code" {
		t.Errorf("client error message = %q, want original", got)
	}
	if got := MaskErrorMessage(New(Unavailable, "google sign-in is not configured")); got != "google sign-in is not configured" {
		t.Errorf("unavailable message = %q, want original", got)
	}
}
