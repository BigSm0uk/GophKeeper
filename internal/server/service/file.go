package service

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// FileService provides secure file serving capabilities for static content and documentation
type FileService struct {
	logger  *zap.Logger
	baseDir string
}

// FileResponse represents a file with its content and metadata
type FileResponse struct {
	Content     []byte
	ContentType string
	Size        int64
}

// NewFileService creates a new file service with the specified base directory
func NewFileService(baseDir string, logger *zap.Logger) (*FileService, error) {
	// Ensure base directory is absolute
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absBaseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("base directory does not exist: %s", absBaseDir)
	}

	logger.Info("File service initialized",
		zap.String("base_dir", absBaseDir),
	)

	return &FileService{
		logger:  logger,
		baseDir: absBaseDir,
	}, nil
}

// GetFile reads a file from the base directory with path traversal protection
func (s *FileService) GetFile(relativePath string) (*FileResponse, error) {
	cleanPath := filepath.Clean(relativePath)

	cleanPath = strings.TrimPrefix(cleanPath, "/")

	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		s.logger.Warn("Path traversal attempt detected",
			zap.String("requested_path", relativePath),
			zap.String("resolved_path", fullPath),
		)
		return nil, fmt.Errorf("invalid path: access denied")
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", relativePath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", relativePath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentType := s.detectContentType(fullPath, content)

	s.logger.Debug("File served successfully",
		zap.String("path", relativePath),
		zap.String("content_type", contentType),
		zap.Int64("size", fileInfo.Size()),
	)

	return &FileResponse{
		Content:     content,
		ContentType: contentType,
		Size:        fileInfo.Size(),
	}, nil
}

// FileExists checks if a file exists in the base directory
func (s *FileService) FileExists(relativePath string) bool {
	cleanPath := filepath.Clean(relativePath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		return false
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return false
	}

	return !fileInfo.IsDir()
}

// detectContentType determines the MIME type of a file
func (s *FileService) detectContentType(filePath string, content []byte) string {
	ext := filepath.Ext(filePath)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	switch ext {
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".md":
		return "text/markdown"
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	}

	return "application/octet-stream"
}

// ListFiles returns a list of files in a directory (non-recursive)
func (s *FileService) ListFiles(relativePath string) ([]string, error) {
	cleanPath := filepath.Clean(relativePath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		return nil, fmt.Errorf("invalid path: access denied")
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// SaveFile saves file content to the specified path with path traversal protection
func (s *FileService) SaveFile(relativePath string, content []byte) error {
	cleanPath := filepath.Clean(relativePath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		s.logger.Warn("Path traversal attempt detected",
			zap.String("requested_path", relativePath),
			zap.String("resolved_path", fullPath),
		)
		return fmt.Errorf("invalid path: access denied")
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.logger.Error("Failed to create directory", zap.Error(err), zap.String("dir", dir))
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		s.logger.Error("Failed to write file", zap.Error(err), zap.String("path", fullPath))
		return fmt.Errorf("failed to write file: %w", err)
	}

	s.logger.Debug("File saved successfully",
		zap.String("path", relativePath),
		zap.Int("size", len(content)),
	)

	return nil
}

// DeleteFile deletes a file with path traversal protection
func (s *FileService) DeleteFile(relativePath string) error {
	cleanPath := filepath.Clean(relativePath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		s.logger.Warn("Path traversal attempt detected",
			zap.String("requested_path", relativePath),
			zap.String("resolved_path", fullPath),
		)
		return fmt.Errorf("invalid path: access denied")
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", relativePath)
		}
		s.logger.Error("Failed to delete file", zap.Error(err), zap.String("path", fullPath))
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Debug("File deleted successfully", zap.String("path", relativePath))
	return nil
}

// OpenFileForReading opens a file for reading with path traversal protection
func (s *FileService) OpenFileForReading(relativePath string) (*os.File, int64, error) {
	cleanPath := filepath.Clean(relativePath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		s.logger.Warn("Path traversal attempt detected",
			zap.String("requested_path", relativePath),
			zap.String("resolved_path", fullPath),
		)
		return nil, 0, fmt.Errorf("invalid path: access denied")
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("file not found: %s", relativePath)
		}
		return nil, 0, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, 0, fmt.Errorf("path is a directory, not a file: %s", relativePath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		s.logger.Error("Failed to open file", zap.Error(err), zap.String("path", fullPath))
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}

	return file, fileInfo.Size(), nil
}

// CreateFileForWriting creates a file for writing with path traversal protection
// Returns the file handle and a cleanup function
func (s *FileService) CreateFileForWriting(relativePath string) (*os.File, func(), error) {
	cleanPath := filepath.Clean(relativePath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.baseDir) {
		s.logger.Warn("Path traversal attempt detected",
			zap.String("requested_path", relativePath),
			zap.String("resolved_path", fullPath),
		)
		return nil, nil, fmt.Errorf("invalid path: access denied")
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.logger.Error("Failed to create directory", zap.Error(err), zap.String("dir", dir))
		return nil, nil, fmt.Errorf("failed to create directory: %w", err)
	}

	tempPath := fullPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		s.logger.Error("Failed to create file", zap.Error(err), zap.String("path", tempPath))
		return nil, nil, fmt.Errorf("failed to create file: %w", err)
	}

	cleanup := func() {
		file.Close()
		os.Remove(tempPath)
	}

	return file, cleanup, nil
}

// RenameFile renames a file atomically (used for committing temp files)
func (s *FileService) RenameFile(oldRelativePath, newRelativePath string) error {
	oldCleanPath := filepath.Clean(oldRelativePath)
	oldCleanPath = strings.TrimPrefix(oldCleanPath, "/")
	oldFullPath := filepath.Join(s.baseDir, oldCleanPath)

	newCleanPath := filepath.Clean(newRelativePath)
	newCleanPath = strings.TrimPrefix(newCleanPath, "/")
	newFullPath := filepath.Join(s.baseDir, newCleanPath)

	if !strings.HasPrefix(oldFullPath, s.baseDir) || !strings.HasPrefix(newFullPath, s.baseDir) {
		s.logger.Warn("Path traversal attempt detected in rename")
		return fmt.Errorf("invalid path: access denied")
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		s.logger.Error("Failed to rename file",
			zap.Error(err),
			zap.String("old", oldFullPath),
			zap.String("new", newFullPath),
		)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	s.logger.Debug("File renamed successfully",
		zap.String("old", oldRelativePath),
		zap.String("new", newRelativePath),
	)

	return nil
}
