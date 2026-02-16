package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	password := "test-master-password-123"
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	enc, err := NewEncryptor(password, salt)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"Simple text", "Hello, World!"},
		{"With special chars", "Пароль123!@#$%^&*()"},
		{"Long text", "This is a very long text that should still be encrypted and decrypted correctly without any issues"},
		{"Empty string", ""},
		{"Unicode", "🔐🔑🛡️"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Empty plaintext should result in empty ciphertext
			if tc.plaintext == "" && ciphertext != "" {
				t.Errorf("Expected empty ciphertext for empty plaintext, got: %s", ciphertext)
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify
			if decrypted != tc.plaintext {
				t.Errorf("Decryption mismatch:\nExpected: %s\nGot: %s", tc.plaintext, decrypted)
			}
		})
	}
}

func TestEncryptionUniqueness(t *testing.T) {
	password := "test-password"
	salt, _ := GenerateSalt()
	enc, _ := NewEncryptor(password, salt)

	plaintext := "test message"

	// Encrypt the same plaintext multiple times
	ciphertext1, _ := enc.Encrypt(plaintext)
	ciphertext2, _ := enc.Encrypt(plaintext)

	// Ciphertexts should be different due to unique nonces
	if ciphertext1 == ciphertext2 {
		t.Error("Expected different ciphertexts for the same plaintext (unique nonces)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := enc.Decrypt(ciphertext1)
	decrypted2, _ := enc.Decrypt(ciphertext2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to the original plaintext")
	}
}

func TestDifferentPasswordsDifferentKeys(t *testing.T) {
	salt, _ := GenerateSalt()

	enc1, _ := NewEncryptor("password1", salt)
	enc2, _ := NewEncryptor("password2", salt)

	plaintext := "secret data"

	// Encrypt with first password
	ciphertext, _ := enc1.Encrypt(plaintext)

	// Try to decrypt with second password
	_, err := enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Expected decryption to fail with wrong password")
	}
}

func TestGenerateSaltUniqueness(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt1: %v", err)
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt2: %v", err)
	}

	if len(salt1) != SaltLen {
		t.Errorf("Expected salt length %d, got %d", SaltLen, len(salt1))
	}

	if string(salt1) == string(salt2) {
		t.Error("Expected different salts")
	}
}

func TestInvalidSaltLength(t *testing.T) {
	password := "test"
	shortSalt := []byte("short")

	_, err := NewEncryptor(password, shortSalt)
	if err == nil {
		t.Error("Expected error for invalid salt length")
	}
}

func TestEncryptIfNotEmpty(t *testing.T) {
	password := "test-password"
	salt, _ := GenerateSalt()
	enc, _ := NewEncryptor(password, salt)

	// Test with non-empty string
	result, err := enc.EncryptIfNotEmpty("test")
	if err != nil {
		t.Errorf("EncryptIfNotEmpty failed: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result for non-empty input")
	}

	// Test with empty string
	result, err = enc.EncryptIfNotEmpty("")
	if err != nil {
		t.Errorf("EncryptIfNotEmpty failed: %v", err)
	}
	if result != "" {
		t.Error("Expected empty result for empty input")
	}
}

func TestDecryptIfNotEmpty(t *testing.T) {
	password := "test-password"
	salt, _ := GenerateSalt()
	enc, _ := NewEncryptor(password, salt)

	// Encrypt something first
	ciphertext, _ := enc.Encrypt("test")

	// Test with non-empty string
	result, err := enc.DecryptIfNotEmpty(ciphertext)
	if err != nil {
		t.Errorf("DecryptIfNotEmpty failed: %v", err)
	}
	if result != "test" {
		t.Errorf("Expected 'test', got '%s'", result)
	}

	// Test with empty string
	result, err = enc.DecryptIfNotEmpty("")
	if err != nil {
		t.Errorf("DecryptIfNotEmpty failed: %v", err)
	}
	if result != "" {
		t.Error("Expected empty result for empty input")
	}
}
