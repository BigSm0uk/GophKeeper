package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters
const (
	Argon2Time    = 1         // Number of iterations
	Argon2Memory  = 64 * 1024 // 64 MB memory
	Argon2Threads = 4         // Number of threads
	Argon2KeyLen  = 32        // Length of the hash
	SaltLen       = 32        // Length of the salt
)

// HashPassword hashes a password using Argon2id and returns a string
// in the format: argon2id$base64salt$base64hash
func HashPassword(password string) (string, error) {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	// Format: argon2id$salt$hash
	return fmt.Sprintf("argon2id$%s$%s", saltB64, hashB64), nil
}

// VerifyPassword verifies a password against a hashed password
func VerifyPassword(password, hashedPassword string) (bool, error) {
	parts := strings.Split(hashedPassword, "$")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid hash format")
	}

	if parts[0] != "argon2id" {
		return false, fmt.Errorf("unsupported hash algorithm: %s", parts[0])
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	if len(hash) != len(expectedHash) {
		return false, nil
	}

	result := true
	for i := range hash {
		if hash[i] != expectedHash[i] {
			result = false
		}
	}

	return result, nil
}

// IsValidHashFormat checks if the hashed password has a valid format
func IsValidHashFormat(hashedPassword string) bool {
	parts := strings.Split(hashedPassword, "$")
	if len(parts) != 3 {
		return false
	}

	if parts[0] != "argon2id" {
		return false
	}

	_, saltErr := base64.RawStdEncoding.DecodeString(parts[1])
	_, hashErr := base64.RawStdEncoding.DecodeString(parts[2])

	return saltErr == nil && hashErr == nil
}
