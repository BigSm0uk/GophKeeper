package logger_test

import (
	"testing"

	"github.com/BigSm0uk/GophKeeper/internal/client/app/logger"
)

func TestNewZapLogger_Development(t *testing.T) {
	log, err := logger.NewZapLogger("debug", true)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	if log == nil {
		t.Fatal("logger is nil")
	}

	// Тестируем что логгер работает без паники
	log.Debug("test debug message")
	log.Info("test info message")
	log.Warn("test warn message")
}

func TestNewZapLogger_Production(t *testing.T) {
	log, err := logger.NewZapLogger("info", false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	if log == nil {
		t.Fatal("logger is nil")
	}

	// Тестируем что логгер работает без паники
	log.Info("test info message in production")
	log.Warn("test warn message in production")
	log.Error("test error message in production")
}

func TestNewZapLogger_AllLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		log, err := logger.NewZapLogger(level, false)
		if err != nil {
			t.Errorf("failed to create logger with level %s: %v", level, err)
			continue
		}

		if log == nil {
			t.Errorf("logger is nil for level %s", level)
		}
	}
}
