package config

import (
	"api-core/internal/db"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	FlagAddress       = "addr"
	FlagMigrateAction = "action"
	FlagUpAction      = "up"
	FlagDownAction    = "down"
	FlagContainer     = "container"
)

const defaultJWTSecret = "dev-secret-change-me"

type Config struct {
	Database db.DatabaseConfig `mapstructure:"Database"`
	Redis    db.RedisConfig    `mapstructure:"Redis"`
	Auth     AuthConfig
	Google   GoogleConfig
}

type AuthConfig struct {
	JWTSecret     string
	JWTIssuer     string
	JWTExpiration time.Duration
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Set up viper to read from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("")

	// Set default values
	setDefaults()

	// Load Database config
	cfg.Database = db.DatabaseConfig{
		Host:     getEnvString("DATABASE_HOST", "postgres"),
		Port:     getEnvString("DATABASE_PORT", "5432"),
		User:     getEnvString("DATABASE_USER", "postgres"),
		Password: getEnvString("DATABASE_PASSWORD", ""),
		Name:     getEnvString("DATABASE_NAME", "postgres"),
		MaxConns: getEnvInt("DATABASE_MAX_CONNS", 0),
	}

	// Load Redis config
	cfg.Redis = db.RedisConfig{
		Host:     getEnvString("REDIS_HOST", "redis"),
		Port:     getEnvString("REDIS_PORT", "6379"),
		Password: getEnvString("REDIS_PASSWORD", ""),
		Name:     getEnvString("REDIS_NAME", "redis"),
	}

	// Auth config
	jwtExpMinutes := getEnvInt("AUTH_JWT_EXP_MINUTES", 60)
	cfg.Auth = AuthConfig{
		JWTSecret:     getEnvString("AUTH_JWT_SECRET", defaultJWTSecret),
		JWTIssuer:     getEnvString("AUTH_JWT_ISSUER", "api-core"),
		JWTExpiration: time.Duration(jwtExpMinutes) * time.Minute,
	}
	if cfg.Auth.JWTSecret == defaultJWTSecret {
		log.Println("Warning: using default JWT secret, override AUTH_JWT_SECRET in production")
	}

	// Google OAuth config
	cfg.Google = GoogleConfig{
		ClientID:     getEnvString("GOOGLE_OAUTH_CLIENT_ID", ""),
		ClientSecret: getEnvString("GOOGLE_OAUTH_CLIENT_SECRET", ""),
		RedirectURL:  getEnvString("GOOGLE_OAUTH_REDIRECT_URL", ""),
		Scopes:       getEnvStringSlice("GOOGLE_OAUTH_SCOPES", DefaultGoogleScopes()),
	}

	log.Println("Configuration loaded from environment variables")
	log.Printf("Database: %s@%s:%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	log.Printf("Redis: %s:%s", cfg.Redis.Host, cfg.Redis.Port)

	return cfg, nil
}

// setDefaults sets default values for viper
func setDefaults() {
	// Database defaults
	viper.SetDefault("DATABASE_HOST", "postgres")
	viper.SetDefault("DATABASE_PORT", "5432")
	viper.SetDefault("DATABASE_USER", "postgres")
	viper.SetDefault("DATABASE_PASSWORD", "")
	viper.SetDefault("DATABASE_NAME", "postgres")
	viper.SetDefault("DATABASE_MAX_CONNS", 0)

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "redis")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_NAME", "redis")

	// Auth defaults
	viper.SetDefault("AUTH_JWT_ISSUER", "api-core")
	viper.SetDefault("AUTH_JWT_SECRET", defaultJWTSecret)
	viper.SetDefault("AUTH_JWT_EXP_MINUTES", 60)

	// Google OAuth defaults
	viper.SetDefault("GOOGLE_OAUTH_CLIENT_ID", "")
	viper.SetDefault("GOOGLE_OAUTH_CLIENT_SECRET", "")
	viper.SetDefault("GOOGLE_OAUTH_REDIRECT_URL", "")
	viper.SetDefault("GOOGLE_OAUTH_SCOPES", strings.Join(DefaultGoogleScopes(), ","))
}

// getEnvString gets environment variable as string with fallback
func getEnvString(key, defaultValue string) string {
	value := viper.GetString(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt gets environment variable as int with fallback
func getEnvInt(key string, defaultValue int) int {
	value := viper.GetString(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
		return defaultValue
	}
	return intValue
}

func getEnvStringSlice(key string, fallback []string) []string {
	value := viper.GetString(key)
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := []string{}
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func DefaultGoogleScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}
}
