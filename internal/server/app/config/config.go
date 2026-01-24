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
	Env     string        `mapstructure:"env"`
	GRPC    GRPCConfig    `mapstructure:"grpc"`
	HTTP    HTTPConfig    `mapstructure:"http"`
	Logger  LoggerConfig  `mapstructure:"logger"`
	DB      DBConfig      `mapstructure:"db"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Storage StorageConfig `mapstructure:"storage"`
}

// NewDefaultServerConfig returns default server configuration
func NewDefaultServerConfig() ServerConfig {
	return ServerConfig{
		Env:     EnvDevelopment,
		GRPC:    NewDefaultGRPCConfig(),
		HTTP:    NewDefaultHTTPConfig(),
		DB:      NewDefaultDBConfig(),
		Logger:  NewDefaultLoggerConfig(),
		JWT:     NewDefaultJWTConfig(),
		Auth:    NewDefaultAuthConfig(),
		Storage: NewDefaultStorageConfig(),
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

func (c *ServerConfig) GRPCAddress() string {
	return fmt.Sprintf("%s:%d", c.GRPC.Host, c.GRPC.Port)
}

func (c *ServerConfig) HTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.HTTP.Host, c.HTTP.Port)
}

func (c *ServerConfig) IsDevelopment() bool {
	return c.Env == EnvDevelopment
}
