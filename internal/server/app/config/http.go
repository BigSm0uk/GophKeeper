package config

import "time"

type HTTPConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// NewDefaultHTTPConfig returns default HTTP server configuration
func NewDefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
}
