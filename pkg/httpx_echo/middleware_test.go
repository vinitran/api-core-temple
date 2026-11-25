package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"api-core/pkg/auth"
	"api-core/pkg/jwtx"
)

func TestBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		header string
		want   string
		ok     bool
	}{
		{"Bearer abc", "abc", true},
		{"bearer  abc ", "abc", true},
		{"Bearer", "", false},
		{"Bearer ", "", false},
		{"Basic abc", "", false},
		{"xBearer abc", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		got, ok := bearerToken(tt.header)
		if got != tt.want || ok != tt.ok {
			t.Errorf("bearerToken(%q) = (%q, %v), want (%q, %v)", tt.header, got, ok, tt.want, tt.ok)
		}
	}
}

type fakeVerifier struct{}

func (fakeVerifier) Verify(token string) (*jwtx.Claims, error) {
	if token != "good" {
		return nil, errors.New("bad token")
	}

	claims := &jwtx.Claims{Email: "a@example.com"}
	claims.Subject = "7"
	return claims, nil
}

func TestAuthn(t *testing.T) {
	t.Parallel()

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		sub, err := auth.ResolveValidSubject(c.Request().Context())
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, sub)
	}, Authn(fakeVerifier{}))

	tests := map[string]struct {
		header string
		status int
		body   string
	}{
		"valid token":   {"Bearer good", http.StatusOK, "7"},
		"invalid token": {"Bearer bad", http.StatusUnauthorized, ""},
		"missing":       {"", http.StatusUnauthorized, ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set(echo.HeaderAuthorization, tt.header)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.status, rec.Body.String())
			}
			if tt.body != "" && rec.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.body)
			}
		})
	}
}
