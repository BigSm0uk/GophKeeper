package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
)

// FileManager manages encrypted file storage on disk using streaming encryption.
// Files of any size are supported with constant memory usage.
type FileManager struct {
	baseDir         string
	encryptor       *crypto.Encryptor
	streamEncryptor *crypto.StreamEncryptor
}

// NewFileManager creates a new file manager with streaming encryption support.
func NewFileManager(baseDir string, encryptor *crypto.Encryptor) (*FileManager, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	streamEnc, err := crypto.NewStreamEncryptorFromEncryptor(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream encryptor: %w", err)
	}

	return &FileManager{
		baseDir:         baseDir,
		encryptor:       encryptor,
		streamEncryptor: streamEnc,
	}, nil
}

// SaveFile encrypts a file using streaming and saves it to the local storage.
// Returns the path to the saved encrypted file, SHA256 checksum of the encrypted content,
// and the original plaintext file size. Memory usage is constant regardless of file size.
func (fm *FileManager) SaveFile(id, sourcePath string) (destPath, checksum string, size int64, err error) {
	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to stat source file: %w", err)
	}
	size = srcInfo.Size()

	destPath = filepath.Join(fm.baseDir, id+".enc")

	_, checksum, err = fm.streamEncryptor.EncryptFile(sourcePath, destPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to encrypt file: %w", err)
	}

	return destPath, checksum, size, nil
}

// GetFile reads and decrypts an encrypted file, returning the plaintext content.
// NOTE: This loads the entire decrypted content into memory. For large files,
// use ExportFile instead which streams the decryption to disk.
func (fm *FileManager) GetFile(filePath string) ([]byte, error) {
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "gophkeeper-decrypt-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := fm.streamEncryptor.DecryptFile(filePath, tmpPath); err != nil {
		return nil, fmt.Errorf("failed to decrypt file: %w", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted file: %w", err)
	}

	return data, nil
}

// ExportFile decrypts an encrypted file and saves the plaintext to destPath.
// Uses streaming decryption with constant memory usage.
func (fm *FileManager) ExportFile(encryptedFilePath, destPath string) error {
	if err := fm.streamEncryptor.DecryptFile(encryptedFilePath, destPath); err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}
	return nil
}

// DeleteFile removes an encrypted file from disk.
func (fm *FileManager) DeleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// VerifyChecksum verifies the SHA256 checksum of the encrypted file.
func (fm *FileManager) VerifyChecksum(filePath, expectedChecksum string) (bool, error) {
	actualChecksum, err := crypto.ComputeFileChecksum(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to compute checksum: %w", err)
	}
	return actualChecksum == expectedChecksum, nil
}

// GetFileSize returns the size of the encrypted file on disk.
func (fm *FileManager) GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// GetOriginalSize reads the header of an encrypted file and returns the original plaintext size.
func (fm *FileManager) GetOriginalSize(filePath string) (int64, error) {
	return fm.streamEncryptor.GetOriginalSize(filePath)
}

// CopyFile copies a file with streaming encryption from source into storage.
// Returns the path to the encrypted file, SHA256 checksum of encrypted content,
// and original plaintext file size.
func (fm *FileManager) CopyFile(id, sourcePath string) (destPath, checksum string, size int64, err error) {
	return fm.SaveFile(id, sourcePath)
}

// GetBaseDir returns the base directory for file storage.
func (fm *FileManager) GetBaseDir() string {
	return fm.baseDir
}

// GetEncryptedChecksum computes SHA256 of an existing encrypted file.
func (fm *FileManager) GetEncryptedChecksum(filePath string) (string, error) {
	return crypto.ComputeFileChecksum(filePath)
}
