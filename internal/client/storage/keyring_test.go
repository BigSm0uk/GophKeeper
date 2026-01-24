package storage_test

import (
	"testing"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
)

func TestTokenStore_SaveAndGetAccessToken(t *testing.T) {
	ts, err := storage.NewTokenStore()
	if err != nil {
		t.Skipf("keyring not available: %v", err)
	}

	username := "testuser"
	token := "test-access-token"

	// Сохраняем
	if err := ts.SaveAccessToken(username, token); err != nil {
		t.Fatalf("failed to save access token: %v", err)
	}

	// Получаем
	retrieved, err := ts.GetAccessToken(username)
	if err != nil {
		t.Fatalf("failed to get access token: %v", err)
	}

	if retrieved != token {
		t.Errorf("expected token %s, got %s", token, retrieved)
	}

	// Cleanup
	ts.DeleteTokens(username)
}

func TestTokenStore_SaveAndGetRefreshToken(t *testing.T) {
	ts, err := storage.NewTokenStore()
	if err != nil {
		t.Skipf("keyring not available: %v", err)
	}

	username := "testuser"
	token := "test-refresh-token"

	// Сохраняем
	if err := ts.SaveRefreshToken(username, token); err != nil {
		t.Fatalf("failed to save refresh token: %v", err)
	}

	// Получаем
	retrieved, err := ts.GetRefreshToken(username)
	if err != nil {
		t.Fatalf("failed to get refresh token: %v", err)
	}

	if retrieved != token {
		t.Errorf("expected token %s, got %s", token, retrieved)
	}

	// Cleanup
	ts.DeleteTokens(username)
}

func TestTokenStore_DeleteTokens(t *testing.T) {
	ts, err := storage.NewTokenStore()
	if err != nil {
		t.Skipf("keyring not available: %v", err)
	}

	username := "testuser"

	// Сохраняем токены
	if err := ts.SaveAccessToken(username, "access"); err != nil {
		t.Fatalf("failed to save access token: %v", err)
	}
	if err := ts.SaveRefreshToken(username, "refresh"); err != nil {
		t.Fatalf("failed to save refresh token: %v", err)
	}

	// Удаляем
	ts.DeleteTokens(username)

	// Проверяем что их нет
	_, err = ts.GetAccessToken(username)
	if err == nil {
		t.Error("expected error when getting deleted access token")
	}

	_, err = ts.GetRefreshToken(username)
	if err == nil {
		t.Error("expected error when getting deleted refresh token")
	}
}

func TestTokenStore_EncryptionSalt(t *testing.T) {
	ts, err := storage.NewTokenStore()
	if err != nil {
		t.Skipf("keyring not available: %v", err)
	}

	salt := []byte("test-salt-123456789012345678901234")

	// Сохраняем
	if err := ts.SaveEncryptionSalt(salt); err != nil {
		t.Fatalf("failed to save encryption salt: %v", err)
	}

	// Получаем
	retrieved, err := ts.GetEncryptionSalt()
	if err != nil {
		t.Fatalf("failed to get encryption salt: %v", err)
	}

	if string(retrieved) != string(salt) {
		t.Errorf("expected salt %s, got %s", salt, retrieved)
	}

	// Cleanup - не удаляем, так как нет отдельного метода для удаления соли
}
