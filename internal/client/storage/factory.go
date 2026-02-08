package storage

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"path/filepath"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
	"github.com/BigSm0uk/GophKeeper/pkg/util"
)

// StorageManager manages the complete storage stack with encryption.
type StorageManager struct {
	DB          *LocalDB
	Encrypted   *EncryptedStorage
	TokenStore  *TokenStore
	FileManager *FileManager
	username    string
	initialized bool
}

// SaltProvider abstracts server-side salt retrieval for multi-device support.
// When the server implements GetEncryptionSalt/SaveEncryptionSalt,
// the API client should satisfy this interface.
type SaltProvider interface {
	// GetEncryptionSalt fetches encryption salt from the server. Returns nil, nil if not found.
	GetEncryptionSalt(username string) ([]byte, error)
	// SaveEncryptionSalt sends encryption salt to the server for storage.
	SaveEncryptionSalt(username string, salt []byte) error
}

// InitializeStorage creates a new storage manager for a user.
// If the user already exists, it verifies the master password.
// If the user is new, it creates a new salt and saves the password hash.
// saltProvider may be nil if server is unavailable (offline mode).
func InitializeStorage(dbPath, username, masterPassword string, saltProvider SaltProvider) (*StorageManager, error) {
	db, err := NewLocalDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local database: %w", err)
	}

	tokenStore, err := NewTokenStore()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open token store: %w", err)
	}

	salt, isNewUser, err := resolveSalt(username, tokenStore, saltProvider)
	if err != nil {
		db.Close()
		return nil, err
	}

	if isNewUser {
		// Save Argon2id hash of master password for future verification
		passwordHash, err := util.HashPassword(masterPassword)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to hash master password: %w", err)
		}
		if err := tokenStore.SaveMasterPasswordHash(username, []byte(passwordHash)); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to save master password hash: %w", err)
		}
	} else {
		// Verify master password for existing user
		if err := verifyMasterPassword(username, masterPassword, tokenStore); err != nil {
			db.Close()
			return nil, err
		}
	}

	encryptor, err := crypto.NewEncryptor(masterPassword, salt)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	encryptedStorage := NewEncryptedStorage(db, encryptor)

	filesDir := filepath.Join(filepath.Dir(dbPath), "files")
	fileManager, err := NewFileManager(filesDir, encryptor)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create file manager: %w", err)
	}

	return &StorageManager{
		DB:          db,
		Encrypted:   encryptedStorage,
		TokenStore:  tokenStore,
		FileManager: fileManager,
		username:    username,
		initialized: true,
	}, nil
}

// resolveSalt determines the encryption salt using the following priority:
// 1. Server (if saltProvider is available) — enables multi-device
// 2. Local keyring — offline fallback
// 3. Generate new salt — first-time setup
func resolveSalt(username string, tokenStore *TokenStore, saltProvider SaltProvider) ([]byte, bool, error) {
	// 1. Try server first (multi-device support)
	if saltProvider != nil {
		serverSalt, err := saltProvider.GetEncryptionSalt(username)
		if err == nil && len(serverSalt) > 0 {
			// Cache in local keyring for offline use
			_ = tokenStore.SaveEncryptionSalt(username, serverSalt)
			return serverSalt, false, nil
		}
	}

	// 2. Try local keyring
	localSalt, err := tokenStore.GetEncryptionSalt(username)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get encryption salt: %w", err)
	}
	if len(localSalt) > 0 {
		// Sync to server if possible
		if saltProvider != nil {
			_ = saltProvider.SaveEncryptionSalt(username, localSalt)
		}
		return localSalt, false, nil
	}

	// 3. New user: generate salt
	newSalt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Save locally
	if err := tokenStore.SaveEncryptionSalt(username, newSalt); err != nil {
		return nil, false, fmt.Errorf("failed to save encryption salt: %w", err)
	}

	// Save to server if available
	if saltProvider != nil {
		_ = saltProvider.SaveEncryptionSalt(username, newSalt)
	}

	return newSalt, true, nil
}

// verifyMasterPassword checks the master password against the stored Argon2id hash.
// Also supports legacy SHA-256 hashes for backward compatibility.
func verifyMasterPassword(username, masterPassword string, tokenStore *TokenStore) error {
	storedHash, err := tokenStore.GetMasterPasswordHash(username)
	if err != nil {
		return fmt.Errorf("failed to get master password hash: %w", err)
	}
	if storedHash == nil {
		// No hash stored — skip verification (first time on this device via server salt)
		// Save hash now for future verification
		passwordHash, err := util.HashPassword(masterPassword)
		if err != nil {
			return fmt.Errorf("failed to hash master password: %w", err)
		}
		return tokenStore.SaveMasterPasswordHash(username, []byte(passwordHash))
	}

	hashStr := string(storedHash)

	// Try Argon2id format first
	if util.IsValidHashFormat(hashStr) {
		valid, err := util.VerifyPassword(masterPassword, hashStr)
		if err != nil {
			return fmt.Errorf("failed to verify master password: %w", err)
		}
		if !valid {
			return fmt.Errorf("invalid master password")
		}
		return nil
	}

	// Legacy: raw 32-byte SHA-256 hash — migrate to Argon2id
	if len(storedHash) == 32 {
		if !legacySHA256Verify(masterPassword, storedHash) {
			return fmt.Errorf("invalid master password")
		}
		// Migrate to Argon2id
		newHash, err := util.HashPassword(masterPassword)
		if err == nil {
			_ = tokenStore.SaveMasterPasswordHash(username, []byte(newHash))
		}
		return nil
	}

	return fmt.Errorf("invalid master password")
}

// legacySHA256Verify checks password against a raw SHA-256 hash (backward compat).
func legacySHA256Verify(password string, storedHash []byte) bool {
	computed := sha256.Sum256([]byte(password))
	return subtle.ConstantTimeCompare(computed[:], storedHash) == 1
}

// Close closes all storage resources.
func (sm *StorageManager) Close() error {
	if sm.DB != nil {
		return sm.DB.Close()
	}
	return nil
}

// GetUsername returns the current username.
func (sm *StorageManager) GetUsername() string {
	return sm.username
}

// IsInitialized returns true if the storage is properly initialized.
func (sm *StorageManager) IsInitialized() bool {
	return sm.initialized
}

// ChangeMasterPassword changes the master password and re-encrypts all data.
func (sm *StorageManager) ChangeMasterPassword(oldPassword, newPassword string) error {
	// Verify old password
	if err := verifyMasterPassword(sm.username, oldPassword, sm.TokenStore); err != nil {
		return fmt.Errorf("old password verification failed: %w", err)
	}

	oldSalt, err := sm.TokenStore.GetEncryptionSalt(sm.username)
	if err != nil {
		return fmt.Errorf("failed to get encryption salt: %w", err)
	}

	oldEncryptor, err := crypto.NewEncryptor(oldPassword, oldSalt)
	if err != nil {
		return fmt.Errorf("failed to create old encryptor: %w", err)
	}

	newSalt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate new salt: %w", err)
	}

	newEncryptor, err := crypto.NewEncryptor(newPassword, newSalt)
	if err != nil {
		return fmt.Errorf("failed to create new encryptor: %w", err)
	}

	// Re-encrypt credentials
	if err := sm.reEncryptCredentials(oldEncryptor, newEncryptor); err != nil {
		return err
	}

	// Re-encrypt cards
	if err := sm.reEncryptCards(oldEncryptor, newEncryptor); err != nil {
		return err
	}

	// Re-encrypt texts
	if err := sm.reEncryptTexts(oldEncryptor, newEncryptor); err != nil {
		return err
	}

	// Re-encrypt binary metadata
	if err := sm.reEncryptBinaries(oldEncryptor, newEncryptor); err != nil {
		return err
	}

	// Save new salt and password hash
	if err := sm.TokenStore.SaveEncryptionSalt(sm.username, newSalt); err != nil {
		return fmt.Errorf("failed to save new encryption salt: %w", err)
	}

	newPasswordHash, err := util.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}
	if err := sm.TokenStore.SaveMasterPasswordHash(sm.username, []byte(newPasswordHash)); err != nil {
		return fmt.Errorf("failed to save new master password hash: %w", err)
	}

	sm.Encrypted = NewEncryptedStorage(sm.DB, newEncryptor)

	return nil
}

func (sm *StorageManager) reEncryptCredentials(oldEnc, newEnc *crypto.Encryptor) error {
	credentials, err := sm.DB.ListCredentials()
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	for _, cred := range credentials {
		cred.Login, err = reEncryptField(oldEnc, newEnc, cred.Login)
		if err != nil {
			return fmt.Errorf("re-encrypt credential %s login: %w", cred.ID, err)
		}
		cred.Password, err = reEncryptField(oldEnc, newEnc, cred.Password)
		if err != nil {
			return fmt.Errorf("re-encrypt credential %s password: %w", cred.ID, err)
		}
		cred.URL, err = reEncryptOptionalField(oldEnc, newEnc, cred.URL)
		if err != nil {
			return fmt.Errorf("re-encrypt credential %s url: %w", cred.ID, err)
		}
		cred.Metadata, err = reEncryptOptionalField(oldEnc, newEnc, cred.Metadata)
		if err != nil {
			return fmt.Errorf("re-encrypt credential %s metadata: %w", cred.ID, err)
		}
		if err := sm.DB.SaveCredential(cred); err != nil {
			return fmt.Errorf("save re-encrypted credential %s: %w", cred.ID, err)
		}
	}
	return nil
}

func (sm *StorageManager) reEncryptCards(oldEnc, newEnc *crypto.Encryptor) error {
	cards, err := sm.DB.ListCards()
	if err != nil {
		return fmt.Errorf("failed to list cards: %w", err)
	}

	for _, card := range cards {
		card.CardNumber, err = reEncryptField(oldEnc, newEnc, card.CardNumber)
		if err != nil {
			return fmt.Errorf("re-encrypt card %s number: %w", card.ID, err)
		}
		card.ExpiryDate, err = reEncryptField(oldEnc, newEnc, card.ExpiryDate)
		if err != nil {
			return fmt.Errorf("re-encrypt card %s expiry: %w", card.ID, err)
		}
		card.CVV, err = reEncryptField(oldEnc, newEnc, card.CVV)
		if err != nil {
			return fmt.Errorf("re-encrypt card %s cvv: %w", card.ID, err)
		}
		card.Metadata, err = reEncryptOptionalField(oldEnc, newEnc, card.Metadata)
		if err != nil {
			return fmt.Errorf("re-encrypt card %s metadata: %w", card.ID, err)
		}
		if err := sm.DB.SaveCard(card); err != nil {
			return fmt.Errorf("save re-encrypted card %s: %w", card.ID, err)
		}
	}
	return nil
}

func (sm *StorageManager) reEncryptTexts(oldEnc, newEnc *crypto.Encryptor) error {
	texts, err := sm.DB.ListTexts()
	if err != nil {
		return fmt.Errorf("failed to list texts: %w", err)
	}

	for _, text := range texts {
		text.Content, err = reEncryptField(oldEnc, newEnc, text.Content)
		if err != nil {
			return fmt.Errorf("re-encrypt text %s content: %w", text.ID, err)
		}
		text.Metadata, err = reEncryptOptionalField(oldEnc, newEnc, text.Metadata)
		if err != nil {
			return fmt.Errorf("re-encrypt text %s metadata: %w", text.ID, err)
		}
		if err := sm.DB.SaveText(text); err != nil {
			return fmt.Errorf("save re-encrypted text %s: %w", text.ID, err)
		}
	}
	return nil
}

func (sm *StorageManager) reEncryptBinaries(oldEnc, newEnc *crypto.Encryptor) error {
	binaries, err := sm.DB.ListBinaries()
	if err != nil {
		return fmt.Errorf("failed to list binaries: %w", err)
	}

	for _, binary := range binaries {
		binary.Metadata, err = reEncryptOptionalField(oldEnc, newEnc, binary.Metadata)
		if err != nil {
			return fmt.Errorf("re-encrypt binary %s metadata: %w", binary.ID, err)
		}
		if err := sm.DB.SaveBinary(binary); err != nil {
			return fmt.Errorf("save re-encrypted binary %s: %w", binary.ID, err)
		}
	}
	return nil
}

// reEncryptField decrypts a value with oldEnc and re-encrypts with newEnc.
func reEncryptField(oldEnc, newEnc *crypto.Encryptor, ciphertext string) (string, error) {
	plaintext, err := oldEnc.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return newEnc.Encrypt(plaintext)
}

// reEncryptOptionalField re-encrypts an optional (nullable) field.
func reEncryptOptionalField(oldEnc, newEnc *crypto.Encryptor, field *string) (*string, error) {
	if field == nil {
		return nil, nil
	}
	reEncrypted, err := reEncryptField(oldEnc, newEnc, *field)
	if err != nil {
		return nil, err
	}
	return &reEncrypted, nil
}

// ResetUserData removes all local data for the user.
func (sm *StorageManager) ResetUserData() error {
	sm.TokenStore.ClearUserData(sm.username)

	if err := sm.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}
