package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	// ScryptN is the CPU/memory cost parameter for Scrypt
	ScryptN = 32768
	// ScryptR is the block size parameter for Scrypt
	ScryptR = 8
	// ScryptP is the parallelization parameter for Scrypt
	ScryptP = 1
	// KeyLen is the length of the derived key (256 bits for AES-256)
	KeyLen = 32
	// SaltLen is the length of the salt
	SaltLen = 32
)

// Encryptor provides client-side encryption/decryption using AES-256-GCM.
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new Encryptor from a master password.
// The key is derived using Scrypt with a random salt.
func NewEncryptor(masterPassword string, salt []byte) (*Encryptor, error) {
	if len(salt) != SaltLen {
		return nil, fmt.Errorf("salt must be %d bytes", SaltLen)
	}

	key, err := scrypt.Key([]byte(masterPassword), salt, ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	return &Encryptor{key: key}, nil
}

// GenerateSalt generates a random salt for key derivation.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext.
// Format: nonce + ciphertext
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a unique nonce for each encryption
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for safe storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM.
func (e *Encryptor) Decrypt(ciphertextBase64 string) (string, error) {
	if ciphertextBase64 == "" {
		return "", nil
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce from the beginning
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// EncryptIfNotEmpty encrypts plaintext if it's not empty, otherwise returns empty string.
func (e *Encryptor) EncryptIfNotEmpty(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	return e.Encrypt(plaintext)
}

// DecryptIfNotEmpty decrypts ciphertext if it's not empty, otherwise returns empty string.
func (e *Encryptor) DecryptIfNotEmpty(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return e.Decrypt(ciphertext)
}
