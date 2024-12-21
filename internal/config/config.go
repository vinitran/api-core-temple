package config

import (
	"bytes"
	"log"
	"otp-core/internal/db"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
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

// Default parses the default configuration values.
func Default() (*Config, error) {
	var cfg Config
	viper.SetConfigType("toml")

	err := viper.ReadConfig(bytes.NewBuffer([]byte(DefaultValues)))
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Load loads the configuration
func Load(configFilePath string) (*Config, error) {
	log.Println("config", configFilePath)
	cfg, err := Default()
	if err != nil {
		return nil, err
	}
	if configFilePath != "" {
		dirName, fileName := filepath.Split(configFilePath)
		log.Println("config path: ", configFilePath)

		fileExtension := strings.TrimPrefix(filepath.Ext(fileName), ".")
		fileNameWithoutExtension := strings.TrimSuffix(fileName, "."+fileExtension)

		viper.AddConfigPath(dirName)
		viper.SetConfigName(fileNameWithoutExtension)
		viper.SetConfigType(fileExtension)
	}

	err = viper.ReadInConfig()
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if ok {
			log.Println("config file not found")
		} else {
			log.Println("error reading config file: ", err)
			return nil, err
		}
	}

	decodeHooks := []viper.DecoderConfigOption{
		// this allows arrays to be decoded from env var separated by ",", example: MY_VAR="value1,value2,value3"
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc(), mapstructure.StringToSliceHookFunc(","))),
	}

	err = viper.Unmarshal(&cfg, decodeHooks...)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
