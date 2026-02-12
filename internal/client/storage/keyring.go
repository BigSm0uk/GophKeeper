package storage

import (
	"fmt"

	"github.com/99designs/keyring"
)

const serviceName = "gophkeeper"

// TokenStore сохраняет токены в системном keyring.
type TokenStore struct {
	ring      keyring.Keyring
	saltCache map[string][]byte // кэш для соли
	hashCache map[string][]byte // кэш для хешей паролей
}

// NewTokenStore creates a TokenStore backed by the OS keyring.
// On macOS, KeychainTrustApplication adds the binary to the Keychain ACL so
// subsequent accesses do not trigger a system password dialog.
func NewTokenStore() (*TokenStore, error) {
	r, err := keyring.Open(keyring.Config{
		ServiceName:                    serviceName,
		KeychainTrustApplication:       true,
		KeychainAccessibleWhenUnlocked: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}
	return &TokenStore{
		ring:      r,
		saltCache: make(map[string][]byte),
		hashCache: make(map[string][]byte),
	}, nil
}

func (s *TokenStore) SaveAccessToken(username, token string) error {
	return s.ring.Set(keyring.Item{
		Key:  fmt.Sprintf("%s:access_token", username),
		Data: []byte(token),
	})
}

func (s *TokenStore) SaveRefreshToken(username, token string) error {
	return s.ring.Set(keyring.Item{
		Key:  fmt.Sprintf("%s:refresh_token", username),
		Data: []byte(token),
	})
}

func (s *TokenStore) GetAccessToken(username string) (string, error) {
	item, err := s.ring.Get(fmt.Sprintf("%s:access_token", username))
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (s *TokenStore) GetRefreshToken(username string) (string, error) {
	item, err := s.ring.Get(fmt.Sprintf("%s:refresh_token", username))
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (s *TokenStore) DeleteTokens(username string) {
	_ = s.ring.Remove(fmt.Sprintf("%s:access_token", username))
	_ = s.ring.Remove(fmt.Sprintf("%s:refresh_token", username))
	_ = s.ring.Remove("current_username")
}

// SaveCurrentUsername сохраняет текущий username после успешной авторизации
func (s *TokenStore) SaveCurrentUsername(username string) error {
	return s.ring.Set(keyring.Item{
		Key:  "current_username",
		Data: []byte(username),
	})
}

// GetCurrentUsername получает текущий username
func (s *TokenStore) GetCurrentUsername() (string, error) {
	item, err := s.ring.Get("current_username")
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return "", nil // Ключ не найден - это нормально, пользователь не авторизован
		}
		return "", err
	}
	return string(item.Data), nil
}

// SaveEncryptionSalt сохраняет salt для шифрования для конкретного пользователя.
func (s *TokenStore) SaveEncryptionSalt(username string, salt []byte) error {
	err := s.ring.Set(keyring.Item{
		Key:  fmt.Sprintf("%s:encryption_salt", username),
		Data: salt,
	})
	if err == nil {
		// Обновляем кэш после успешного сохранения
		s.saltCache[username] = salt
	}
	return err
}

// GetEncryptionSalt загружает salt для шифрования для конкретного пользователя.
func (s *TokenStore) GetEncryptionSalt(username string) ([]byte, error) {
	// Проверяем кэш
	if salt, exists := s.saltCache[username]; exists {
		return salt, nil
	}

	// Загружаем из keyring
	item, err := s.ring.Get(fmt.Sprintf("%s:encryption_salt", username))
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	// Сохраняем в кэш
	s.saltCache[username] = item.Data
	return item.Data, nil
}

// SaveMasterPasswordHash сохраняет хеш мастер-пароля для верификации.
func (s *TokenStore) SaveMasterPasswordHash(username string, hash []byte) error {
	err := s.ring.Set(keyring.Item{
		Key:  fmt.Sprintf("%s:master_password_hash", username),
		Data: hash,
	})
	if err == nil {
		// Обновляем кэш после успешного сохранения
		s.hashCache[username] = hash
	}
	return err
}

// GetMasterPasswordHash загружает хеш мастер-пароля.
func (s *TokenStore) GetMasterPasswordHash(username string) ([]byte, error) {
	// Проверяем кэш
	if hash, exists := s.hashCache[username]; exists {
		return hash, nil
	}

	// Загружаем из keyring
	item, err := s.ring.Get(fmt.Sprintf("%s:master_password_hash", username))
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	// Сохраняем в кэш
	s.hashCache[username] = item.Data
	return item.Data, nil
}

// ClearUserData удаляет все данные пользователя из keyring.
func (s *TokenStore) ClearUserData(username string) {
	_ = s.ring.Remove(fmt.Sprintf("%s:access_token", username))
	_ = s.ring.Remove(fmt.Sprintf("%s:refresh_token", username))
	_ = s.ring.Remove(fmt.Sprintf("%s:encryption_salt", username))
	_ = s.ring.Remove(fmt.Sprintf("%s:master_password_hash", username))
	_ = s.ring.Remove("current_username")

	// Очищаем кэши
	delete(s.saltCache, username)
	delete(s.hashCache, username)
}
