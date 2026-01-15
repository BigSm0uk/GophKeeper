package config

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	EnvDevelopment = "development"
)

type ServerConfig struct {
	Env    string       `mapstructure:"env"`
	Logger LoggerConfig `mapstructure:"logger"`
}

func NewDefaultServerConfig() ServerConfig {
	return ServerConfig{
		Env: EnvDevelopment,
		Logger: LoggerConfig{
			Level: "debug",
		},
	}
}

func ReadConfig() (*ServerConfig, error) {
	configPath := pflag.StringP("config", "c", "", "Path to config file")
	pflag.Parse()

	if *configPath == "" {
		return nil, fmt.Errorf("config file path is required (use --config flag)")
	}
	viper.SetConfigFile(*configPath)

	// Читаем конфиг
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := NewDefaultServerConfig()
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
