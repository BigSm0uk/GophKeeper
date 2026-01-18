package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

const (
	EnvDevelopment = "development"
)

// ClientConfig описывает конфигурацию клиентского приложения.
type ClientConfig struct {
	Env            string        `mapstructure:"env"`
	ServerAddress  string        `mapstructure:"server_address"`
	Insecure       bool          `mapstructure:"insecure"`
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	Logger         LoggerConfig  `mapstructure:"logger"`
	ConfigPath     string        `mapstructure:"-"`
}

type LoggerConfig struct {
	Level string `mapstructure:"level"`
}

// DefaultConfig возвращает конфиг по умолчанию.
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Env:            EnvDevelopment,
		ServerAddress:  "localhost:50051",
		Insecure:       true,
		RequestTimeout: 15 * time.Second,
		Logger: LoggerConfig{
			Level: "info",
		},
	}
}

// configSearchPath возвращает путь по умолчанию: $XDG_CONFIG_HOME/gophkeeper/client.yaml.
func configSearchPath() string {
	return filepath.Join(xdg.ConfigHome, "gophkeeper", "client.yaml")
}

// Load читает конфиг. Если путь пустой и файл по умолчанию не найден, возвращает значения по умолчанию.
func Load(path string) (*ClientConfig, error) {
	cfg := DefaultConfig()

	target := path
	if target == "" {
		target = configSearchPath()
	}

	v := viper.New()
	v.SetConfigFile(target)
	v.SetConfigType("yaml")

	// Не обязательно наличие файла: если нет, используем default.
	if err := v.ReadInConfig(); err == nil {
		if err := v.Unmarshal(&cfg); err != nil {
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}
		cfg.ConfigPath = target
		return &cfg, nil
	}

	// Файл не найден — работаем с дефолтом.
	cfg.ConfigPath = target
	return &cfg, nil
}
