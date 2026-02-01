package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
)

// FileManager управляет хранением зашифрованных файлов на диске.
type FileManager struct {
	baseDir   string
	encryptor *crypto.Encryptor
}

// NewFileManager creates a new file manager.
func NewFileManager(baseDir string, encryptor *crypto.Encryptor) (*FileManager, error) {
	// Создаем базовую директорию если не существует
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &FileManager{
		baseDir:   baseDir,
		encryptor: encryptor,
	}, nil
}

// SaveFile сохраняет файл в зашифрованном виде.
// Возвращает путь к сохраненному файлу и checksum оригинального файла.
func (fm *FileManager) SaveFile(id, sourcePath string) (destPath, checksum string, size int64, err error) {
	// Читаем исходный файл
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read source file: %w", err)
	}

	size = int64(len(data))

	// Вычисляем checksum оригинального файла
	hash := sha256.Sum256(data)
	checksum = hex.EncodeToString(hash[:])

	// Шифруем содержимое
	encryptedData, err := fm.encryptor.EncryptBytes(data)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Сохраняем зашифрованный файл
	destPath = filepath.Join(fm.baseDir, id+".enc")
	if err := os.WriteFile(destPath, encryptedData, 0600); err != nil {
		return "", "", 0, fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return destPath, checksum, size, nil
}

// GetFile читает и расшифровывает файл.
func (fm *FileManager) GetFile(filePath string) ([]byte, error) {
	// Читаем зашифрованный файл
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Расшифровываем
	decryptedData, err := fm.encryptor.DecryptBytes(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt file: %w", err)
	}

	return decryptedData, nil
}

// ExportFile расшифровывает файл и сохраняет в указанное место.
func (fm *FileManager) ExportFile(encryptedFilePath, destPath string) error {
	// Расшифровываем
	decryptedData, err := fm.GetFile(encryptedFilePath)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Сохраняем расшифрованный файл
	if err := os.WriteFile(destPath, decryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}

// DeleteFile удаляет зашифрованный файл с диска.
func (fm *FileManager) DeleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// VerifyChecksum проверяет контрольную сумму расшифрованного файла.
func (fm *FileManager) VerifyChecksum(filePath, expectedChecksum string) (bool, error) {
	// Расшифровываем файл
	data, err := fm.GetFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to get file: %w", err)
	}

	// Вычисляем checksum
	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	return actualChecksum == expectedChecksum, nil
}

// GetFileSize возвращает размер зашифрованного файла.
func (fm *FileManager) GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// CopyFile копирует файл с шифрованием из источника в хранилище.
func (fm *FileManager) CopyFile(id, sourcePath string) (destPath, checksum string, size int64, err error) {
	// Открываем исходный файл
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Создаем временный буфер для чтения и вычисления checksum
	hasher := sha256.New()
	var data []byte
	
	// Читаем файл и вычисляем checksum одновременно
	data, err = io.ReadAll(io.TeeReader(srcFile, hasher))
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read source file: %w", err)
	}

	size = int64(len(data))
	checksum = hex.EncodeToString(hasher.Sum(nil))

	// Шифруем
	encryptedData, err := fm.encryptor.EncryptBytes(data)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Сохраняем
	destPath = filepath.Join(fm.baseDir, id+".enc")
	if err := os.WriteFile(destPath, encryptedData, 0600); err != nil {
		return "", "", 0, fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return destPath, checksum, size, nil
}

// GetBaseDir returns the base directory for file storage.
func (fm *FileManager) GetBaseDir() string {
	return fm.baseDir
}
