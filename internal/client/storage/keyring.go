package storage

import (
	"fmt"

	"github.com/99designs/keyring"
)

const serviceName = "gophkeeper"

// TokenStore сохраняет токены в системном keyring.
type TokenStore struct {
	ring keyring.Keyring
}

func NewTokenStore() (*TokenStore, error) {
	r, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}
	return &TokenStore{ring: r}, nil
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

// SaveEncryptionSalt сохраняет salt для шифрования.
func (s *TokenStore) SaveEncryptionSalt(salt []byte) error {
	return s.ring.Set(keyring.Item{
		Key:  "encryption_salt",
		Data: salt,
	})
}

// GetEncryptionSalt загружает salt для шифрования.
func (s *TokenStore) GetEncryptionSalt() ([]byte, error) {
	item, err := s.ring.Get("encryption_salt")
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	return item.Data, nil
}
