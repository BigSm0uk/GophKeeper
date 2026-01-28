package storage

import (
	"crypto/sha256"
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
)

// StorageManager manages the complete storage stack with encryption.
type StorageManager struct {
	DB          *LocalDB
	Encrypted   *EncryptedStorage
	TokenStore  *TokenStore
	username    string
	initialized bool
}

// InitializeStorage creates a new storage manager for a user.
// If the user already exists, it verifies the master password.
// If the user is new, it creates a new salt and saves the password hash.
func InitializeStorage(dbPath, username, masterPassword string) (*StorageManager, error) {
	// Open local database
	db, err := NewLocalDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local database: %w", err)
	}

	// Open keyring
	tokenStore, err := NewTokenStore()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open token store: %w", err)
	}

	// Check if user already has encryption salt
	existingSalt, err := tokenStore.GetEncryptionSalt(username)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get encryption salt: %w", err)
	}

	var salt []byte
	isNewUser := existingSalt == nil

	if isNewUser {
		// Generate new salt for new user
		salt, err = crypto.GenerateSalt()
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}

		// Save salt
		if err := tokenStore.SaveEncryptionSalt(username, salt); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to save encryption salt: %w", err)
		}

		// Save master password hash for future verification
		passwordHash := sha256.Sum256([]byte(masterPassword))
		if err := tokenStore.SaveMasterPasswordHash(username, passwordHash[:]); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to save master password hash: %w", err)
		}
	} else {
		// Verify master password for existing user
		salt = existingSalt
		storedHash, err := tokenStore.GetMasterPasswordHash(username)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to get master password hash: %w", err)
		}

		if storedHash != nil {
			passwordHash := sha256.Sum256([]byte(masterPassword))
			// Compare hashes
			if !bytesEqual(passwordHash[:], storedHash) {
				db.Close()
				return nil, fmt.Errorf("invalid master password")
			}
		}
	}

	// Create encryptor with derived key
	encryptor, err := crypto.NewEncryptor(masterPassword, salt)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Create encrypted storage wrapper
	encryptedStorage := NewEncryptedStorage(db, encryptor)

	return &StorageManager{
		DB:          db,
		Encrypted:   encryptedStorage,
		TokenStore:  tokenStore,
		username:    username,
		initialized: true,
	}, nil
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

// bytesEqual compares two byte slices in constant time.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// ChangeMasterPassword changes the master password and re-encrypts all data.
// WARNING: This is a destructive operation that requires re-encryption of all local data.
func (sm *StorageManager) ChangeMasterPassword(oldPassword, newPassword string) error {
	// Verify old password
	storedHash, err := sm.TokenStore.GetMasterPasswordHash(sm.username)
	if err != nil {
		return fmt.Errorf("failed to get master password hash: %w", err)
	}

	oldPasswordHash := sha256.Sum256([]byte(oldPassword))
	if !bytesEqual(oldPasswordHash[:], storedHash) {
		return fmt.Errorf("invalid old password")
	}

	// Get existing salt
	oldSalt, err := sm.TokenStore.GetEncryptionSalt(sm.username)
	if err != nil {
		return fmt.Errorf("failed to get encryption salt: %w", err)
	}

	// Create old encryptor
	oldEncryptor, err := crypto.NewEncryptor(oldPassword, oldSalt)
	if err != nil {
		return fmt.Errorf("failed to create old encryptor: %w", err)
	}

	// Generate new salt
	newSalt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate new salt: %w", err)
	}

	// Create new encryptor
	newEncryptor, err := crypto.NewEncryptor(newPassword, newSalt)
	if err != nil {
		return fmt.Errorf("failed to create new encryptor: %w", err)
	}

	// Re-encrypt all credentials
	credentials, err := sm.DB.ListCredentials()
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	for _, cred := range credentials {
		// Decrypt with old key
		decryptedLogin, err := oldEncryptor.Decrypt(cred.Login)
		if err != nil {
			return fmt.Errorf("failed to decrypt credential %s: %w", cred.ID, err)
		}

		decryptedPassword, err := oldEncryptor.Decrypt(cred.Password)
		if err != nil {
			return fmt.Errorf("failed to decrypt credential %s: %w", cred.ID, err)
		}

		// Encrypt with new key
		cred.Login, err = newEncryptor.Encrypt(decryptedLogin)
		if err != nil {
			return fmt.Errorf("failed to encrypt credential %s: %w", cred.ID, err)
		}

		cred.Password, err = newEncryptor.Encrypt(decryptedPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt credential %s: %w", cred.ID, err)
		}

		// Handle optional fields
		if cred.URL != nil {
			decrypted, err := oldEncryptor.Decrypt(*cred.URL)
			if err != nil {
				return fmt.Errorf("failed to decrypt credential url %s: %w", cred.ID, err)
			}
			encrypted, err := newEncryptor.Encrypt(decrypted)
			if err != nil {
				return fmt.Errorf("failed to encrypt credential url %s: %w", cred.ID, err)
			}
			cred.URL = &encrypted
		}

		if cred.Metadata != nil {
			decrypted, err := oldEncryptor.Decrypt(*cred.Metadata)
			if err != nil {
				return fmt.Errorf("failed to decrypt credential metadata %s: %w", cred.ID, err)
			}
			encrypted, err := newEncryptor.Encrypt(decrypted)
			if err != nil {
				return fmt.Errorf("failed to encrypt credential metadata %s: %w", cred.ID, err)
			}
			cred.Metadata = &encrypted
		}

		// Save re-encrypted credential
		if err := sm.DB.SaveCredential(cred); err != nil {
			return fmt.Errorf("failed to save re-encrypted credential %s: %w", cred.ID, err)
		}
	}

	// TODO: Re-encrypt cards, texts, and binary metadata similarly

	// Save new salt and password hash
	if err := sm.TokenStore.SaveEncryptionSalt(sm.username, newSalt); err != nil {
		return fmt.Errorf("failed to save new encryption salt: %w", err)
	}

	newPasswordHash := sha256.Sum256([]byte(newPassword))
	if err := sm.TokenStore.SaveMasterPasswordHash(sm.username, newPasswordHash[:]); err != nil {
		return fmt.Errorf("failed to save new master password hash: %w", err)
	}

	// Update encryptor in encrypted storage
	sm.Encrypted = NewEncryptedStorage(sm.DB, newEncryptor)

	return nil
}

// ResetUserData removes all local data for the user.
// Use with caution: this will delete all local cached data.
func (sm *StorageManager) ResetUserData() error {
	// Clear keyring data
	sm.TokenStore.ClearUserData(sm.username)

	// Close database
	if err := sm.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}
