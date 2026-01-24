package config

import "path/filepath"

type StorageConfig struct {
	// BasePath is the base directory for storing binary files
	BasePath string `mapstructure:"base_path"`
	// MaxFileSize is the maximum allowed file size in bytes (default 100MB)
	MaxFileSize int64 `mapstructure:"max_file_size"`
}

// NewDefaultStorageConfig returns default storage configuration
func NewDefaultStorageConfig() StorageConfig {
	return StorageConfig{
		BasePath:    "./data/files",
		MaxFileSize: 100 * 1024 * 1024, // 100 MB
	}
}

// GetAbsoluteBasePath returns the absolute path to the storage directory
func (c *StorageConfig) GetAbsoluteBasePath() (string, error) {
	return filepath.Abs(c.BasePath)
}
