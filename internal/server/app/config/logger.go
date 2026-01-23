package config

type LoggerConfig struct {
	Level string `mapstructure:"level"`
}

// NewDefaultLoggerConfig returns default logger configuration
func NewDefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level: "debug",
	}
}
