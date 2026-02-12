package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStreamEncryptor(t *testing.T) *StreamEncryptor {
	t.Helper()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	enc, err := NewEncryptor("test-master-password-123", salt)
	require.NoError(t, err)
	se, err := NewStreamEncryptorFromEncryptor(enc)
	require.NoError(t, err)
	return se
}

func TestStreamEncryptor_RoundTrip_SmallFile(t *testing.T) {
	se := setupStreamEncryptor(t)
	tmpDir := t.TempDir()

	// Create a small test file
	original := []byte("Hello, World! This is a small test file.")
	srcPath := filepath.Join(tmpDir, "original.txt")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	// Encrypt
	encPath := filepath.Join(tmpDir, "encrypted.enc")
	encSize, checksum, err := se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)
	assert.Greater(t, encSize, int64(0))
	assert.NotEmpty(t, checksum)

	// Verify encrypted file is different from original
	encData, err := os.ReadFile(encPath)
	require.NoError(t, err)
	assert.NotEqual(t, original, encData)
	assert.Equal(t, encSize, int64(len(encData)))

	// Decrypt
	decPath := filepath.Join(tmpDir, "decrypted.txt")
	err = se.DecryptFile(encPath, decPath)
	require.NoError(t, err)

	// Verify round-trip
	decrypted, err := os.ReadFile(decPath)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestStreamEncryptor_RoundTrip_MultiChunkFile(t *testing.T) {
	se := setupStreamEncryptor(t)
	// Use a small chunk size for testing
	se.chunkSize = 1024 // 1KB chunks
	tmpDir := t.TempDir()

	// Create a file that spans multiple chunks (5KB)
	original := make([]byte, 5*1024)
	_, err := io.ReadFull(rand.Reader, original)
	require.NoError(t, err)

	srcPath := filepath.Join(tmpDir, "multi_chunk.bin")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	// Encrypt
	encPath := filepath.Join(tmpDir, "encrypted.enc")
	encSize, checksum, err := se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)
	assert.Greater(t, encSize, int64(len(original)))
	assert.NotEmpty(t, checksum)

	// Verify predicted size matches
	expectedSize := se.EncryptedFileSize(int64(len(original)))
	assert.Equal(t, expectedSize, encSize)

	// Decrypt
	decPath := filepath.Join(tmpDir, "decrypted.bin")
	err = se.DecryptFile(encPath, decPath)
	require.NoError(t, err)

	decrypted, err := os.ReadFile(decPath)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestStreamEncryptor_RoundTrip_ExactChunkBoundary(t *testing.T) {
	se := setupStreamEncryptor(t)
	se.chunkSize = 256
	tmpDir := t.TempDir()

	// File size exactly equals chunk size
	original := make([]byte, 256)
	_, err := io.ReadFull(rand.Reader, original)
	require.NoError(t, err)

	srcPath := filepath.Join(tmpDir, "exact.bin")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	encPath := filepath.Join(tmpDir, "encrypted.enc")
	_, _, err = se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)

	decPath := filepath.Join(tmpDir, "decrypted.bin")
	err = se.DecryptFile(encPath, decPath)
	require.NoError(t, err)

	decrypted, err := os.ReadFile(decPath)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestStreamEncryptor_RoundTrip_EmptyFile(t *testing.T) {
	se := setupStreamEncryptor(t)
	tmpDir := t.TempDir()

	// Empty file
	srcPath := filepath.Join(tmpDir, "empty.txt")
	require.NoError(t, os.WriteFile(srcPath, []byte{}, 0o600))

	encPath := filepath.Join(tmpDir, "encrypted.enc")
	encSize, _, err := se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)
	assert.Equal(t, int64(StreamHeaderSize), encSize)

	decPath := filepath.Join(tmpDir, "decrypted.txt")
	err = se.DecryptFile(encPath, decPath)
	require.NoError(t, err)

	decrypted, err := os.ReadFile(decPath)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestStreamEncryptor_RoundTrip_LargeFile(t *testing.T) {
	se := setupStreamEncryptor(t)
	se.chunkSize = 64 * 1024 // 64KB chunks for faster test
	tmpDir := t.TempDir()

	// 1MB file with multiple chunks
	size := 1024 * 1024
	original := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, original)
	require.NoError(t, err)

	srcPath := filepath.Join(tmpDir, "large.bin")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	encPath := filepath.Join(tmpDir, "encrypted.enc")
	encSize, checksum, err := se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)
	assert.Greater(t, encSize, int64(size))
	assert.NotEmpty(t, checksum)

	// Verify checksum matches actual file
	encData, err := os.ReadFile(encPath)
	require.NoError(t, err)
	h := sha256.Sum256(encData)
	actualChecksum := hex.EncodeToString(h[:])
	assert.Equal(t, actualChecksum, checksum)

	decPath := filepath.Join(tmpDir, "decrypted.bin")
	err = se.DecryptFile(encPath, decPath)
	require.NoError(t, err)

	decrypted, err := os.ReadFile(decPath)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestStreamEncryptor_RoundTrip_100MB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 100MB test in short mode")
	}

	se := setupStreamEncryptor(t)
	// Default chunk size (1MB) — real production scenario
	tmpDir := t.TempDir()

	const fileSize = 100 * 1024 * 1024 // 100MB

	// Generate 100MB random file
	srcPath := filepath.Join(tmpDir, "large_100mb.bin")
	f, err := os.Create(srcPath)
	require.NoError(t, err)

	// Write in 1MB blocks to avoid allocating 100MB in memory at once
	block := make([]byte, 1024*1024)
	written := int64(0)
	for written < fileSize {
		_, err := io.ReadFull(rand.Reader, block)
		require.NoError(t, err)
		n, err := f.Write(block)
		require.NoError(t, err)
		written += int64(n)
	}
	f.Close()

	// Compute source SHA256 for verification
	srcChecksum, err := ComputeFileChecksum(srcPath)
	require.NoError(t, err)

	// Encrypt
	encPath := filepath.Join(tmpDir, "encrypted_100mb.enc")
	t.Log("Encrypting 100MB file...")
	encSize, encChecksum, err := se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)
	assert.Greater(t, encSize, int64(fileSize), "encrypted file should be larger than original")
	assert.NotEmpty(t, encChecksum)
	t.Logf("Encrypted size: %d bytes (overhead: %.2f%%)", encSize, float64(encSize-fileSize)/float64(fileSize)*100)

	// Verify predicted size matches
	expectedSize := se.EncryptedFileSize(fileSize)
	assert.Equal(t, expectedSize, encSize, "predicted encrypted size should match actual")

	// Verify encrypted checksum matches recomputation
	recomputedChecksum, err := ComputeFileChecksum(encPath)
	require.NoError(t, err)
	assert.Equal(t, encChecksum, recomputedChecksum, "encrypted checksum should be reproducible")

	// Verify original size from header
	origSize, err := se.GetOriginalSize(encPath)
	require.NoError(t, err)
	assert.Equal(t, int64(fileSize), origSize, "header should store correct original size")

	// Decrypt
	decPath := filepath.Join(tmpDir, "decrypted_100mb.bin")
	t.Log("Decrypting 100MB file...")
	err = se.DecryptFile(encPath, decPath)
	require.NoError(t, err)

	// Verify decrypted file matches original via checksum (avoid loading 100MB into memory)
	decChecksum, err := ComputeFileChecksum(decPath)
	require.NoError(t, err)
	assert.Equal(t, srcChecksum, decChecksum, "decrypted file checksum must match original")

	// Verify decrypted file size
	decInfo, err := os.Stat(decPath)
	require.NoError(t, err)
	assert.Equal(t, int64(fileSize), decInfo.Size(), "decrypted file size must match original")

	t.Log("100MB round-trip test passed successfully")
}

func TestStreamEncryptor_CorruptedChunk(t *testing.T) {
	se := setupStreamEncryptor(t)
	se.chunkSize = 256
	tmpDir := t.TempDir()

	original := make([]byte, 512)
	_, err := io.ReadFull(rand.Reader, original)
	require.NoError(t, err)

	srcPath := filepath.Join(tmpDir, "original.bin")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	encPath := filepath.Join(tmpDir, "encrypted.enc")
	_, _, err = se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)

	// Corrupt a byte in the middle of the encrypted file (inside first chunk)
	encData, err := os.ReadFile(encPath)
	require.NoError(t, err)
	corruptIdx := StreamHeaderSize + gcmNonceSize + 10
	if corruptIdx < len(encData) {
		encData[corruptIdx] ^= 0xFF
	}
	require.NoError(t, os.WriteFile(encPath, encData, 0o600))

	// Decryption should fail due to GCM tag mismatch
	decPath := filepath.Join(tmpDir, "decrypted.bin")
	err = se.DecryptFile(encPath, decPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt chunk")
}

func TestStreamEncryptor_WrongKey(t *testing.T) {
	salt, err := GenerateSalt()
	require.NoError(t, err)

	enc1, err := NewEncryptor("password1", salt)
	require.NoError(t, err)
	se1, err := NewStreamEncryptorFromEncryptor(enc1)
	require.NoError(t, err)

	enc2, err := NewEncryptor("password2", salt)
	require.NoError(t, err)
	se2, err := NewStreamEncryptorFromEncryptor(enc2)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	original := []byte("secret data that must be protected")
	srcPath := filepath.Join(tmpDir, "original.txt")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	// Encrypt with key1
	encPath := filepath.Join(tmpDir, "encrypted.enc")
	_, _, err = se1.EncryptFile(srcPath, encPath)
	require.NoError(t, err)

	// Decrypt with key2 should fail
	decPath := filepath.Join(tmpDir, "decrypted.txt")
	err = se2.DecryptFile(encPath, decPath)
	assert.Error(t, err)
}

func TestStreamEncryptor_StreamInterface(t *testing.T) {
	se := setupStreamEncryptor(t)
	se.chunkSize = 128

	original := []byte("streaming test data that spans multiple chunks to verify stream interface works correctly end to end")

	// Encrypt via stream
	var encBuf bytes.Buffer
	encSize, _, err := se.EncryptStream(bytes.NewReader(original), &encBuf, int64(len(original)))
	require.NoError(t, err)
	assert.Equal(t, int64(encBuf.Len()), encSize)

	// Decrypt via stream
	var decBuf bytes.Buffer
	err = se.DecryptStream(&encBuf, &decBuf)
	require.NoError(t, err)
	assert.Equal(t, original, decBuf.Bytes())
}

func TestStreamEncryptor_EncryptedFileSize(t *testing.T) {
	se := setupStreamEncryptor(t)
	se.chunkSize = 1024

	tests := []struct {
		name         string
		originalSize int64
	}{
		{"empty", 0},
		{"small", 100},
		{"exact chunk", 1024},
		{"one and half chunks", 1536},
		{"two chunks", 2048},
		{"large", 10240},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicted := se.EncryptedFileSize(tt.originalSize)

			// Create file and encrypt to verify
			tmpDir := t.TempDir()
			data := make([]byte, tt.originalSize)
			if tt.originalSize > 0 {
				_, _ = io.ReadFull(rand.Reader, data)
			}

			srcPath := filepath.Join(tmpDir, "src.bin")
			require.NoError(t, os.WriteFile(srcPath, data, 0o600))

			encPath := filepath.Join(tmpDir, "enc.bin")
			actualSize, _, err := se.EncryptFile(srcPath, encPath)
			require.NoError(t, err)
			assert.Equal(t, predicted, actualSize, "predicted size should match actual encrypted size")
		})
	}
}

func TestStreamEncryptor_GetOriginalSize(t *testing.T) {
	se := setupStreamEncryptor(t)
	tmpDir := t.TempDir()

	original := make([]byte, 12345)
	_, err := io.ReadFull(rand.Reader, original)
	require.NoError(t, err)

	srcPath := filepath.Join(tmpDir, "original.bin")
	require.NoError(t, os.WriteFile(srcPath, original, 0o600))

	encPath := filepath.Join(tmpDir, "encrypted.enc")
	_, _, err = se.EncryptFile(srcPath, encPath)
	require.NoError(t, err)

	origSize, err := se.GetOriginalSize(encPath)
	require.NoError(t, err)
	assert.Equal(t, int64(12345), origSize)
}

func TestStreamEncryptor_InvalidHeader(t *testing.T) {
	se := setupStreamEncryptor(t)
	tmpDir := t.TempDir()

	// File with wrong magic
	badFile := filepath.Join(tmpDir, "bad.enc")
	require.NoError(t, os.WriteFile(badFile, []byte("XXXX\x01"+string(make([]byte, 20))), 0o600))

	decPath := filepath.Join(tmpDir, "dec.bin")
	err := se.DecryptFile(badFile, decPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid magic")
}

func TestNewStreamEncryptor_InvalidKeyLength(t *testing.T) {
	_, err := NewStreamEncryptor([]byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be")
}

func TestNewStreamEncryptorFromEncryptor_Nil(t *testing.T) {
	_, err := NewStreamEncryptorFromEncryptor(nil)
	assert.Error(t, err)
}

func TestCounterNonce(t *testing.T) {
	nonce0 := counterNonce(0, 12)
	nonce1 := counterNonce(1, 12)
	nonce2 := counterNonce(2, 12)

	// All nonces should be different
	assert.NotEqual(t, nonce0, nonce1)
	assert.NotEqual(t, nonce1, nonce2)
	assert.NotEqual(t, nonce0, nonce2)

	// Nonce 0 should be all zeros
	assert.Equal(t, make([]byte, 12), nonce0)

	// Nonce 1 should have 1 in the last byte
	expected := make([]byte, 12)
	expected[11] = 1
	assert.Equal(t, expected, nonce1)
}
