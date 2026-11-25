// Package config loads and validates application configuration from the
// environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"api-core/internal/db"
)

const (
	FlagAddress       = "addr"
	FlagMigrateAction = "action"
	FlagUpAction      = "up"
	FlagDownAction    = "down"
	FlagSkipMigrate   = "skip-migrate"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	defaultJWTSecret         = "dev-secret-change-me"
	minProductionSecretBytes = 32
)

type Config struct {
	Env      string
	Database db.DatabaseConfig
	Redis    db.RedisConfig
	HTTP     HTTPConfig
	Auth     AuthConfig
	Google   GoogleConfig
}

type HTTPConfig struct {
	// AllowedOrigins for CORS. "*" allows any origin without credentials.
	AllowedOrigins []string
}

type AuthConfig struct {
	JWTSecret     string
	JWTIssuer     string
	JWTAudience   string
	JWTExpiration time.Duration
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Enabled reports whether Google sign-in is fully configured.
func (g GoogleConfig) Enabled() bool {
	return g.ClientID != "" && g.ClientSecret != "" && g.RedirectURL != ""
}

func (g GoogleConfig) partiallySet() bool {
	return !g.Enabled() && (g.ClientID != "" || g.ClientSecret != "" || g.RedirectURL != "")
}

// IsProduction reports whether APP_ENV is production.
func (c *Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// Load reads configuration from environment variables and validates it.
// Malformed values are reported instead of silently falling back to defaults.
func Load() (*Config, error) {
	e := &envReader{}

	cfg := &Config{
		Env: e.String("APP_ENV", EnvDevelopment),
		Database: db.DatabaseConfig{
			Host:     e.String("DATABASE_HOST", "localhost"),
			Port:     e.String("DATABASE_PORT", "5432"),
			User:     e.String("DATABASE_USER", "postgres"),
			Password: e.String("DATABASE_PASSWORD", ""),
			Name:     e.String("DATABASE_NAME", "postgres"),
			SSLMode:  e.String("DATABASE_SSLMODE", "disable"),
			MaxConns: e.Int("DATABASE_MAX_CONNS", 0),
		},
		Redis: db.RedisConfig{
			Host:     e.String("REDIS_HOST", "localhost"),
			Port:     e.String("REDIS_PORT", "6379"),
			Password: e.String("REDIS_PASSWORD", ""),
			DB:       e.Int("REDIS_DB", 0),
		},
		HTTP: HTTPConfig{
			AllowedOrigins: e.StringSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
		Auth: AuthConfig{
			JWTSecret:     e.String("AUTH_JWT_SECRET", defaultJWTSecret),
			JWTIssuer:     e.String("AUTH_JWT_ISSUER", "api-core"),
			JWTAudience:   e.String("AUTH_JWT_AUDIENCE", "api-core"),
			JWTExpiration: time.Duration(e.Int("AUTH_JWT_EXP_MINUTES", 60)) * time.Minute,
		},
		Google: GoogleConfig{
			ClientID:     e.String("GOOGLE_OAUTH_CLIENT_ID", ""),
			ClientSecret: e.String("GOOGLE_OAUTH_CLIENT_SECRET", ""),
			RedirectURL:  e.String("GOOGLE_OAUTH_REDIRECT_URL", ""),
			Scopes:       e.StringSlice("GOOGLE_OAUTH_SCOPES", nil),
		},
	}

	if err := errors.Join(append(e.errs, cfg.validate()...)...); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() []error {
	var errs []error

	if c.Env != EnvDevelopment && c.Env != EnvProduction {
		errs = append(errs, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, c.Env))
	}

	switch {
	case c.Auth.JWTSecret == "":
		errs = append(errs, errors.New("AUTH_JWT_SECRET is required"))
	case c.IsProduction() && c.Auth.JWTSecret == defaultJWTSecret:
		errs = append(errs, errors.New("AUTH_JWT_SECRET must be overridden in production"))
	case c.IsProduction() && len(c.Auth.JWTSecret) < minProductionSecretBytes:
		errs = append(errs, fmt.Errorf("AUTH_JWT_SECRET must be at least %d bytes in production", minProductionSecretBytes))
	case c.Auth.JWTSecret == defaultJWTSecret:
		slog.Warn("using default JWT secret, override AUTH_JWT_SECRET outside local development")
	}

	if c.Auth.JWTExpiration <= 0 {
		errs = append(errs, errors.New("AUTH_JWT_EXP_MINUTES must be positive"))
	}

	if c.Google.partiallySet() {
		errs = append(errs, errors.New("GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET and GOOGLE_OAUTH_REDIRECT_URL must be set together"))
	}

	if c.IsProduction() && len(c.HTTP.AllowedOrigins) == 1 && c.HTTP.AllowedOrigins[0] == "*" {
		slog.Warn("CORS allows any origin; set CORS_ALLOWED_ORIGINS in production")
	}

	return errs
}

// envReader reads typed environment variables and collects parse errors.
type envReader struct {
	errs []error
}

func (e *envReader) String(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func (e *envReader) Int(key string, fallback int) int {
	value := e.String(key, "")
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s: invalid integer %q", key, value))
		return fallback
	}

	return n
}

func (e *envReader) StringSlice(key string, fallback []string) []string {
	value := e.String(key, "")
	if value == "" {
		return fallback
	}

	var result []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return fallback
	}

	return result
}
