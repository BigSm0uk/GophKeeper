package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"go.uber.org/zap"
)

func TestClient_New(t *testing.T) {
	// Этот тест проверяет создание клиента
	// В реальной среде требуется запущенный сервер, поэтому тест минимальный
	logger := zap.NewNop()
	client, err := api.New("localhost:50051", true, 10*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("client is nil")
	}
}

func TestClient_SetAccessToken(t *testing.T) {
	logger := zap.NewNop()
	client, err := api.New("localhost:50051", true, 10*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	token := "test-token"
	client.SetAccessToken(token)

	// Нет способа напрямую проверить токен, но тест проверяет что метод не паникует
}

func TestClient_WithAuth(t *testing.T) {
	// Интеграционный тест - требует запущенного сервера
	// Здесь тестируем только что методы не паникуют при вызове без сервера
	logger := zap.NewNop()
	client, err := api.New("localhost:50051", true, 1*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Эти вызовы должны завершиться с ошибкой (нет сервера), но не паниковать
	_, err = client.Register(ctx, "testuser", "password", "test@example.com")
	if err == nil {
		t.Log("expected error when connecting to non-existent server (this is ok in test)")
	}
}

func TestClient_PasswordToken(t *testing.T) {
	logger := zap.NewNop()
	client, err := api.New("localhost:50051", true, 1*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Попытка получить токен без сервера
	_, err = client.PasswordToken(ctx, "testuser", "password", "client-id", "", "")
	if err == nil {
		t.Log("expected error when connecting to non-existent server (this is ok in test)")
	}
}

func TestClient_RefreshToken(t *testing.T) {
	logger := zap.NewNop()
	client, err := api.New("localhost:50051", true, 1*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Попытка обновить токен без сервера
	_, err = client.RefreshToken(ctx, "refresh-token", "client-id", "")
	if err == nil {
		t.Log("expected error when connecting to non-existent server (this is ok in test)")
	}
}
