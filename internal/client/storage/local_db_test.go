package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
)

func setupTestDB(t *testing.T) (*storage.LocalDB, string) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.NewLocalDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	return db, dbPath
}

func TestLocalDB_SaveAndGetCredential(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	cred := &storage.LocalCredential{
		ID:         "test-id",
		Name:       "Test Credential",
		Login:      "testuser",
		Password:   "encrypted-password",
		URL:        strPtr("https://example.com"),
		Metadata:   strPtr(`{"key": "value"}`),
		SyncStatus: storage.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Сохраняем
	err := db.SaveCredential(cred)
	if err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}

	// Получаем
	retrieved, err := db.GetCredential("test-id")
	if err != nil {
		t.Fatalf("failed to get credential: %v", err)
	}

	if retrieved == nil {
		t.Fatal("retrieved credential is nil")
	}

	if retrieved.ID != cred.ID {
		t.Errorf("expected GetID %s, got %s", cred.ID, retrieved.ID)
	}
	if retrieved.Name != cred.Name {
		t.Errorf("expected Name %s, got %s", cred.Name, retrieved.Name)
	}
	if retrieved.Login != cred.Login {
		t.Errorf("expected Login %s, got %s", cred.Login, retrieved.Login)
	}
	if retrieved.Password != cred.Password {
		t.Errorf("expected Password %s, got %s", cred.Password, retrieved.Password)
	}
}

func TestLocalDB_ListCredentials(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()

	// Сохраняем несколько credentials
	creds := []*storage.LocalCredential{
		{
			ID:         "id-1",
			Name:       "Cred 1",
			Login:      "user1",
			Password:   "pass1",
			SyncStatus: storage.StatusPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "id-2",
			Name:       "Cred 2",
			Login:      "user2",
			Password:   "pass2",
			SyncStatus: storage.StatusSynced,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	for _, cred := range creds {
		if err := db.SaveCredential(cred); err != nil {
			t.Fatalf("failed to save credential: %v", err)
		}
	}

	// Получаем список
	list, err := db.ListCredentials()
	if err != nil {
		t.Fatalf("failed to list credentials: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(list))
	}
}

func TestLocalDB_UpdateCredentialSyncStatus(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	cred := &storage.LocalCredential{
		ID:         "test-id",
		Name:       "Test Credential",
		Login:      "testuser",
		Password:   "encrypted-password",
		SyncStatus: storage.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Сохраняем
	if err := db.SaveCredential(cred); err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}

	// Обновляем статус
	if err := db.UpdateCredentialSyncStatus("test-id", storage.StatusSynced); err != nil {
		t.Fatalf("failed to update sync status: %v", err)
	}

	// Проверяем
	retrieved, err := db.GetCredential("test-id")
	if err != nil {
		t.Fatalf("failed to get credential: %v", err)
	}

	if retrieved.SyncStatus != storage.StatusSynced {
		t.Errorf("expected status %s, got %s", storage.StatusSynced, retrieved.SyncStatus)
	}

	if retrieved.SyncedAt == nil {
		t.Error("expected SyncedAt to be set")
	}
}

func TestLocalDB_GetPendingCredentials(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()

	// Сохраняем credentials с разными статусами
	creds := []*storage.LocalCredential{
		{
			ID:         "pending-1",
			Name:       "Pending 1",
			Login:      "user1",
			Password:   "pass1",
			SyncStatus: storage.StatusPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "synced-1",
			Name:       "Synced 1",
			Login:      "user2",
			Password:   "pass2",
			SyncStatus: storage.StatusSynced,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "updated-1",
			Name:       "Updated 1",
			Login:      "user3",
			Password:   "pass3",
			SyncStatus: storage.StatusUpdated,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	for _, cred := range creds {
		if err := db.SaveCredential(cred); err != nil {
			t.Fatalf("failed to save credential: %v", err)
		}
	}

	// Получаем pending
	pending, err := db.GetPendingCredentials()
	if err != nil {
		t.Fatalf("failed to get pending credentials: %v", err)
	}

	// Должно быть 2 (pending и updated)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending credentials, got %d", len(pending))
	}
}

func TestLocalDB_DeleteCredential(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	cred := &storage.LocalCredential{
		ID:         "test-id",
		Name:       "Test Credential",
		Login:      "testuser",
		Password:   "encrypted-password",
		SyncStatus: storage.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Сохраняем
	if err := db.SaveCredential(cred); err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}

	// Удаляем
	if err := db.DeleteCredential("test-id"); err != nil {
		t.Fatalf("failed to delete credential: %v", err)
	}

	// Проверяем что его нет
	retrieved, err := db.GetCredential("test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved != nil {
		t.Error("expected credential to be deleted")
	}
}

func TestLocalDB_SaveCard(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	card := &storage.LocalCard{
		ID:             "card-1",
		Name:           "Test Card",
		CardNumber:     "1234567812345678",
		CardholderName: "John Doe",
		ExpiryDate:     "12/25",
		CVV:            "123",
		SyncStatus:     storage.StatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Сохраняем
	if err := db.SaveCard(card); err != nil {
		t.Fatalf("failed to save card: %v", err)
	}

	// Получаем
	retrieved, err := db.GetCard("card-1")
	if err != nil {
		t.Fatalf("failed to get card: %v", err)
	}

	if retrieved == nil {
		t.Fatal("retrieved card is nil")
	}

	if retrieved.CardNumber != card.CardNumber {
		t.Errorf("expected CardNumber %s, got %s", card.CardNumber, retrieved.CardNumber)
	}
}

func TestLocalDB_SaveText(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	text := &storage.LocalText{
		ID:         "text-1",
		Name:       "Test Note",
		Content:    "This is a test note",
		SyncStatus: storage.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Сохраняем
	if err := db.SaveText(text); err != nil {
		t.Fatalf("failed to save text: %v", err)
	}

	// Получаем
	retrieved, err := db.GetText("text-1")
	if err != nil {
		t.Fatalf("failed to get text: %v", err)
	}

	if retrieved == nil {
		t.Fatal("retrieved text is nil")
	}

	if retrieved.Content != text.Content {
		t.Errorf("expected Content %s, got %s", text.Content, retrieved.Content)
	}
}

func strPtr(s string) *string {
	return &s
}
