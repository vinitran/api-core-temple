package db

// DatabaseConfig provide fields to configure the pool
type DatabaseConfig struct {
	// Database name
	Name string `mapstructure:"Name"`

	// Database User name
	User string `mapstructure:"User"`

	// Database Password of the user
	Password string `mapstructure:"Password"`

	// Host address of database
	Host string `mapstructure:"Host"`

	// Port Number of database
	Port string `mapstructure:"Port"`

	// MaxConns is the maximum number of connections in the pool.
	MaxConns int `mapstructure:"MaxConns"`
}

// RedisConfig provide fields to configure the pool
type RedisConfig struct {
	// Database name
	Name string `mapstructure:"Name"`

	// Database Password of the user
	Password string `mapstructure:"Password"`

	// Host address of database
	Host string `mapstructure:"Host"`

	// Port Number of database
	Port string `mapstructure:"Port"`
}
