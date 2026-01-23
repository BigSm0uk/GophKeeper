package config

import "time"

type GRPCConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// NewDefaultGRPCConfig returns default gRPC server configuration
func NewDefaultGRPCConfig() GRPCConfig {
	return GRPCConfig{
		Host:         "0.0.0.0",
		Port:         50051,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
}
