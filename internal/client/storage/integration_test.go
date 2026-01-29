package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) (*storage.StorageManager, func()) {
	t.Helper()

	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "gophkeeper-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	username := "test@example.com"
	masterPassword := "test-master-password-123"

	// Initialize storage
	sm, err := storage.InitializeStorage(dbPath, username, masterPassword)
	require.NoError(t, err)
	require.NotNil(t, sm)

	// Cleanup function
	cleanup := func() {
		if sm != nil {
			sm.Close()
		}
		os.RemoveAll(tmpDir)
	}

	return sm, cleanup
}

func stringPtr(s string) *string {
	return &s
}

func TestStorageManager_Credentials(t *testing.T) {
	sm, cleanup := setupTestStorage(t)
	defer cleanup()

	t.Run("save and retrieve credential", func(t *testing.T) {
		// Create credential
		cred := storage.CreateCredentialWithEncryption(
			"test-cred-1",
			"GitHub Account",
			"user@example.com",
			"super-secret-password",
			stringPtr("https://github.com"),
			stringPtr(`{"note": "work account"}`),
		)

		// Save credential
		err := sm.Encrypted.SaveCredential(cred)
		require.NoError(t, err)

		// Retrieve credential
		retrieved, err := sm.Encrypted.GetCredential("test-cred-1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Verify decrypted data
		assert.Equal(t, "test-cred-1", retrieved.ID)
		assert.Equal(t, "GitHub Account", retrieved.Name)
		assert.Equal(t, "user@example.com", retrieved.Login)
		assert.Equal(t, "super-secret-password", retrieved.Password)
		assert.Equal(t, "https://github.com", *retrieved.URL)
		assert.Equal(t, `{"note": "work account"}`, *retrieved.Metadata)
		assert.Equal(t, storage.StatusPending, retrieved.SyncStatus)
	})

	t.Run("list credentials", func(t *testing.T) {
		// Create multiple credentials
		cred2 := storage.CreateCredentialWithEncryption(
			"test-cred-2",
			"GitLab Account",
			"user@gitlab.com",
			"another-password",
			nil,
			nil,
		)

		err := sm.Encrypted.SaveCredential(cred2)
		require.NoError(t, err)

		// List all credentials
		creds, err := sm.Encrypted.ListCredentials()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(creds), 2)

		// Verify all credentials are decrypted
		for _, cred := range creds {
			assert.NotEmpty(t, cred.Login)
			assert.NotEmpty(t, cred.Password)
		}
	})

	t.Run("update credential", func(t *testing.T) {
		// Get existing credential
		cred, err := sm.Encrypted.GetCredential("test-cred-1")
		require.NoError(t, err)

		// Update password
		cred.Password = "new-password-123"
		cred.SyncStatus = storage.StatusUpdated

		// Save updated credential
		err = sm.Encrypted.SaveCredential(cred)
		require.NoError(t, err)

		// Retrieve and verify
		updated, err := sm.Encrypted.GetCredential("test-cred-1")
		require.NoError(t, err)
		assert.Equal(t, "new-password-123", updated.Password)
		assert.Equal(t, storage.StatusUpdated, updated.SyncStatus)
	})

	t.Run("delete credential", func(t *testing.T) {
		err := sm.Encrypted.DeleteCredential("test-cred-1")
		require.NoError(t, err)

		// Verify deletion
		deleted, err := sm.Encrypted.GetCredential("test-cred-1")
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

func TestStorageManager_Cards(t *testing.T) {
	sm, cleanup := setupTestStorage(t)
	defer cleanup()

	t.Run("save and retrieve card", func(t *testing.T) {
		card := storage.CreateCardWithEncryption(
			"test-card-1",
			"Work Visa",
			"4111111111111111",
			"John Doe",
			"12/25",
			"123",
			stringPtr("Chase Bank"),
			nil,
		)

		err := sm.Encrypted.SaveCard(card)
		require.NoError(t, err)

		retrieved, err := sm.Encrypted.GetCard("test-card-1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, "test-card-1", retrieved.ID)
		assert.Equal(t, "Work Visa", retrieved.Name)
		assert.Equal(t, "4111111111111111", retrieved.CardNumber)
		assert.Equal(t, "John Doe", retrieved.CardholderName)
		assert.Equal(t, "12/25", retrieved.ExpiryDate)
		assert.Equal(t, "123", retrieved.CVV)
		assert.Equal(t, "Chase Bank", *retrieved.BankName)
	})

	t.Run("list cards", func(t *testing.T) {
		cards, err := sm.Encrypted.ListCards()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(cards), 1)

		for _, card := range cards {
			assert.NotEmpty(t, card.CardNumber)
			assert.NotEmpty(t, card.CVV)
		}
	})

	t.Run("delete card", func(t *testing.T) {
		err := sm.Encrypted.DeleteCard("test-card-1")
		require.NoError(t, err)

		deleted, err := sm.Encrypted.GetCard("test-card-1")
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

func TestStorageManager_Texts(t *testing.T) {
	sm, cleanup := setupTestStorage(t)
	defer cleanup()

	t.Run("save and retrieve text", func(t *testing.T) {
		text := storage.CreateTextWithEncryption(
			"test-text-1",
			"Meeting Notes",
			"This is a secret note about the project timeline...",
			stringPtr(`{"tags": ["important", "project"]}`),
		)

		err := sm.Encrypted.SaveText(text)
		require.NoError(t, err)

		retrieved, err := sm.Encrypted.GetText("test-text-1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, "test-text-1", retrieved.ID)
		assert.Equal(t, "Meeting Notes", retrieved.Name)
		assert.Equal(t, "This is a secret note about the project timeline...", retrieved.Content)
		assert.Equal(t, `{"tags": ["important", "project"]}`, *retrieved.Metadata)
	})

	t.Run("list texts", func(t *testing.T) {
		texts, err := sm.Encrypted.ListTexts()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(texts), 1)

		for _, text := range texts {
			assert.NotEmpty(t, text.Content)
		}
	})

	t.Run("delete text", func(t *testing.T) {
		err := sm.Encrypted.DeleteText("test-text-1")
		require.NoError(t, err)

		deleted, err := sm.Encrypted.GetText("test-text-1")
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

func TestStorageManager_Binaries(t *testing.T) {
	sm, cleanup := setupTestStorage(t)
	defer cleanup()

	t.Run("save and retrieve binary metadata", func(t *testing.T) {
		binary := storage.CreateBinaryWithEncryption(
			"test-binary-1",
			"Passport Scan",
			"passport.pdf",
			"/tmp/files/passport.pdf",
			1024000,
			"application/pdf",
			"abc123def456",
			stringPtr(`{"encrypted": true}`),
		)

		err := sm.Encrypted.SaveBinary(binary)
		require.NoError(t, err)

		retrieved, err := sm.Encrypted.GetBinary("test-binary-1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, "test-binary-1", retrieved.ID)
		assert.Equal(t, "Passport Scan", retrieved.Name)
		assert.Equal(t, "passport.pdf", retrieved.Filename)
		assert.Equal(t, "/tmp/files/passport.pdf", retrieved.FilePath)
		assert.Equal(t, int64(1024000), retrieved.Size)
		assert.Equal(t, "application/pdf", retrieved.ContentType)
		assert.Equal(t, "abc123def456", retrieved.Checksum)
		assert.Equal(t, `{"encrypted": true}`, *retrieved.Metadata)
	})

	t.Run("list binaries", func(t *testing.T) {
		binaries, err := sm.Encrypted.ListBinaries()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(binaries), 1)
	})

	t.Run("delete binary", func(t *testing.T) {
		err := sm.Encrypted.DeleteBinary("test-binary-1")
		require.NoError(t, err)

		deleted, err := sm.Encrypted.GetBinary("test-binary-1")
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})
}

func TestStorageManager_SyncStatus(t *testing.T) {
	sm, cleanup := setupTestStorage(t)
	defer cleanup()

	t.Run("track pending credentials", func(t *testing.T) {
		// Create credentials with pending status
		cred1 := storage.CreateCredentialWithEncryption(
			"pending-1", "Test 1", "user1", "pass1", nil, nil,
		)
		cred2 := storage.CreateCredentialWithEncryption(
			"pending-2", "Test 2", "user2", "pass2", nil, nil,
		)

		err := sm.Encrypted.SaveCredential(cred1)
		require.NoError(t, err)
		err = sm.Encrypted.SaveCredential(cred2)
		require.NoError(t, err)

		// Get pending credentials
		pending, err := sm.Encrypted.GetPendingCredentials()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(pending), 2)

		// Mark one as synced
		err = sm.Encrypted.UpdateCredentialSyncStatus("pending-1", storage.StatusSynced)
		require.NoError(t, err)

		// Verify status updated
		synced, err := sm.Encrypted.GetCredential("pending-1")
		require.NoError(t, err)
		assert.Equal(t, storage.StatusSynced, synced.SyncStatus)
		assert.NotNil(t, synced.SyncedAt)

		// Get pending again (should be one less)
		pending2, err := sm.Encrypted.GetPendingCredentials()
		require.NoError(t, err)
		assert.Less(t, len(pending2), len(pending))
	})
}

func TestStorageManager_MasterPassword(t *testing.T) {
	t.Run("invalid master password", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gophkeeper-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		dbPath := filepath.Join(tmpDir, "test.db")
		// Use unique username to avoid keyring conflicts
		username := "test-invalid-pw@example.com"

		// Initialize with password
		sm, err := storage.InitializeStorage(dbPath, username, "correct-password")
		require.NoError(t, err)
		require.NotNil(t, sm)

		// Clean up keyring data on test end
		defer func() {
			sm.TokenStore.ClearUserData(username)
			sm.Close()
		}()

		sm.Close()

		// Try to open with wrong password
		sm2, err := storage.InitializeStorage(dbPath, username, "wrong-password")
		assert.Error(t, err)
		assert.Nil(t, sm2)
		assert.Contains(t, err.Error(), "invalid master password")
	})

	t.Run("correct master password reopens storage", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gophkeeper-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		dbPath := filepath.Join(tmpDir, "test.db")
		// Use unique username to avoid keyring conflicts
		username := "test-correct-pw@example.com"
		password := "correct-password"

		// Initialize and save data
		sm, err := storage.InitializeStorage(dbPath, username, password)
		require.NoError(t, err)

		cred := storage.CreateCredentialWithEncryption(
			"test-1", "Test", "user", "pass", nil, nil,
		)
		err = sm.Encrypted.SaveCredential(cred)
		require.NoError(t, err)
		sm.Close()

		// Reopen with same password
		sm2, err := storage.InitializeStorage(dbPath, username, password)
		require.NoError(t, err)
		require.NotNil(t, sm2)

		// Clean up keyring data on test end
		defer func() {
			sm2.TokenStore.ClearUserData(username)
			sm2.Close()
		}()

		// Verify data is accessible
		retrieved, err := sm2.Encrypted.GetCredential("test-1")
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "user", retrieved.Login)
	})
}

func TestStorageManager_TokenStore(t *testing.T) {
	sm, cleanup := setupTestStorage(t)
	defer cleanup()

	username := sm.GetUsername()

	t.Run("save and retrieve tokens", func(t *testing.T) {
		err := sm.TokenStore.SaveAccessToken(username, "access-token-123")
		require.NoError(t, err)

		err = sm.TokenStore.SaveRefreshToken(username, "refresh-token-456")
		require.NoError(t, err)

		err = sm.TokenStore.SaveCurrentUsername(username)
		require.NoError(t, err)

		// Retrieve tokens
		accessToken, err := sm.TokenStore.GetAccessToken(username)
		require.NoError(t, err)
		assert.Equal(t, "access-token-123", accessToken)

		refreshToken, err := sm.TokenStore.GetRefreshToken(username)
		require.NoError(t, err)
		assert.Equal(t, "refresh-token-456", refreshToken)

		currentUser, err := sm.TokenStore.GetCurrentUsername()
		require.NoError(t, err)
		assert.Equal(t, username, currentUser)
	})

	t.Run("delete tokens", func(t *testing.T) {
		sm.TokenStore.DeleteTokens(username)

		// Verify tokens are deleted
		_, err := sm.TokenStore.GetAccessToken(username)
		assert.Error(t, err)

		_, err = sm.TokenStore.GetRefreshToken(username)
		assert.Error(t, err)
	})
}
