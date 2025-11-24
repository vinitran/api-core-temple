package config

import (
	"log"
	"otp-core/internal/db"
	"strconv"

	"github.com/spf13/viper"
)

const (
	FlagAddress       = "addr"
	FlagMigrateAction = "action"
	FlagUpAction      = "up"
	FlagDownAction    = "down"
	FlagContainer     = "container"
)

type Config struct {
	Database db.DatabaseConfig `mapstructure:"Database"`
	Redis    db.RedisConfig    `mapstructure:"Redis"`
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
