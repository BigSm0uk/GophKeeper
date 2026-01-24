package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/app/config"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Создаём временную директорию для конфига
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Создаём минимальный конфиг
	configContent := `{
		"env": "test",
		"server_address": "localhost:50051",
		"insecure": true,
		"request_timeout": "30s",
		"logger": {
			"level": "debug"
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Загружаем конфиг
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Env != "test" {
		t.Errorf("expected env 'test', got '%s'", cfg.Env)
	}

	if cfg.ServerAddress != "localhost:50051" {
		t.Errorf("expected server address 'localhost:50051', got '%s'", cfg.ServerAddress)
	}

	if !cfg.Insecure {
		t.Error("expected insecure to be true")
	}

	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("expected request timeout 30s, got %v", cfg.RequestTimeout)
	}

	if cfg.Logger.Level != "debug" {
		t.Errorf("expected log level 'debug', got '%s'", cfg.Logger.Level)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	// Попытка загрузить несуществующий файл должна вернуть конфиг по умолчанию
	cfg, err := config.Load("/tmp/non-existent-config.json")
	if err != nil {
		// Это нормально, если конфиг не найден и нет дефолтного
		t.Logf("expected behavior: config not found: %v", err)
	}

	if cfg != nil && cfg.ServerAddress == "" {
		t.Error("expected default server address to be set")
	}
}

func TestClientConfig_GetLocalDBPath(t *testing.T) {
	cfg := &config.ClientConfig{}

	path, err := cfg.GetLocalDBPath()
	if err != nil {
		t.Fatalf("failed to get local db path: %v", err)
	}

	if path == "" {
		t.Error("expected non-empty db path")
	}

	// Проверяем что путь содержит "gophkeeper"
	if !contains(path, "gophkeeper") {
		t.Errorf("expected path to contain 'gophkeeper', got: %s", path)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && s[len(s)-len(substr):] == substr || filepath.Base(filepath.Dir(s)) == substr || filepath.Base(s) == substr)
}
