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
