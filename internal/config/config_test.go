package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.JWTExpiration != time.Hour {
		t.Errorf("JWTExpiration = %v, want 1h", cfg.Auth.JWTExpiration)
	}
	if cfg.Google.Enabled() {
		t.Error("Google should be disabled without credentials")
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	tests := map[string]struct {
		env     map[string]string
		wantErr string
	}{
		"bad integer": {
			env:     map[string]string{"DATABASE_MAX_CONNS": "ten"},
			wantErr: "DATABASE_MAX_CONNS",
		},
		"default secret in production": {
			env:     map[string]string{"APP_ENV": "production", "AUTH_JWT_SECRET": ""},
			wantErr: "overridden in production",
		},
		"short secret in production": {
			env:     map[string]string{"APP_ENV": "production", "AUTH_JWT_SECRET": "short"},
			wantErr: "at least 32 bytes",
		},
		"partial google config": {
			env:     map[string]string{"GOOGLE_OAUTH_CLIENT_ID": "id"},
			wantErr: "must be set together",
		},
		"unknown env": {
			env:     map[string]string{"APP_ENV": "staging"},
			wantErr: "APP_ENV",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for _, k := range []string{"APP_ENV", "AUTH_JWT_SECRET", "DATABASE_MAX_CONNS", "GOOGLE_OAUTH_CLIENT_ID", "GOOGLE_OAUTH_CLIENT_SECRET", "GOOGLE_OAUTH_REDIRECT_URL"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
