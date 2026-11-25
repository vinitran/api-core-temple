package db

import (
	"net"
	"net/url"
	"strconv"
)

// DatabaseConfig provides the fields to configure the Postgres pool.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	// SSLMode is the libpq sslmode (disable, require, verify-full, ...).
	SSLMode string
	// MaxConns is the maximum number of connections in the pool; 0 means auto.
	MaxConns int
}

// DSN builds a postgres connection URL. Credentials are URL-escaped, so
// passwords may contain any character.
func (cfg DatabaseConfig) DSN() string {
	return cfg.url().String()
}

// RedactedDSN is DSN with the password masked, safe for logs.
func (cfg DatabaseConfig) RedactedDSN() string {
	return cfg.url().Redacted()
}

func (cfg DatabaseConfig) url() *url.URL {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	return &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     "/" + cfg.Name,
		RawQuery: url.Values{"sslmode": {sslMode}}.Encode(),
	}
}

// RedisConfig provides the fields to configure the Redis client.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// Addr returns host:port.
func (cfg RedisConfig) Addr() string {
	return net.JoinHostPort(cfg.Host, cfg.Port)
}

func (cfg RedisConfig) String() string {
	return cfg.Addr() + "/" + strconv.Itoa(cfg.DB)
}
