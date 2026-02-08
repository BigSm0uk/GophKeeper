package models

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Binary represents a binary file entry in the domain layer.
type Binary struct {
	ID          string
	UserID      string
	Name        string
	Filename    string
	Size        int64
	ContentType string
	Metadata    *string
	StoragePath string // internal path where file is stored
	Checksum    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func (b *Binary) GetID() string {
	return b.ID
}

func (b *Binary) SetID(id string) {
	b.ID = id
}

func (b *Binary) GetUserID() string {
	return b.UserID
}

// NewBinary creates a new binary entry with validation.
func NewBinary(userID, name, filename string, size int64, contentType string, metadata *string) (*Binary, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if name == "" {
		return nil, ErrInvalidName
	}
	if filename == "" {
		return nil, ErrInvalidFilename
	}
	if size <= 0 {
		return nil, ErrInvalidFileSize
	}
	if contentType == "" {
		return nil, ErrInvalidContentType
	}

	now := time.Now()
	binary := &Binary{
		UserID:      userID,
		Name:        name,
		Filename:    filename,
		Size:        size,
		ContentType: contentType,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return binary, nil
}

// UpdateMetadata updates the metadata and sets the updated timestamp.
func (b *Binary) UpdateMetadata(metadata *string) {
	b.Metadata = metadata
	b.UpdatedAt = time.Now()
}

// UpdateName updates the name and sets the updated timestamp.
func (b *Binary) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	b.Name = name
	b.UpdatedAt = time.Now()
	return nil
}

// IsOwnedBy checks if the binary is owned by the specified user.
func (b *Binary) IsOwnedBy(userID string) bool {
	return b.UserID == userID
}

// GetFileExtension returns the file extension from filename.
func (b *Binary) GetFileExtension() string {
	return filepath.Ext(b.Filename)
}

// GetFileSizeInMB returns the file size in megabytes.
func (b *Binary) GetFileSizeInMB() float64 {
	return float64(b.Size) / (1024 * 1024)
}

// IsValidFileSize checks if the file size is within acceptable limits.
func (b *Binary) IsValidFileSize(maxSize int64) bool {
	return b.Size > 0 && b.Size <= maxSize
}

// GenerateStoragePath generates a storage path for the binary file.
// This should create a unique path to avoid collisions.
func (b *Binary) GenerateStoragePath(baseDir string) {
	// Create path like: baseDir/userID/year/month/filename_uuid.ext
	year, month, _ := b.CreatedAt.Date()
	ext := b.GetFileExtension()

	// Remove extension from filename for path generation
	nameWithoutExt := strings.TrimSuffix(b.Filename, ext)

	path := filepath.Join(
		baseDir,
		b.UserID,
		fmt.Sprintf("%d", year),
		fmt.Sprintf("%02d", month),
		fmt.Sprintf("%s_%s%s", nameWithoutExt, b.ID, ext),
	)

	b.StoragePath = path
}

// GetMimeTypeCategory returns a category based on the MIME type.
func (b *Binary) GetMimeTypeCategory() string {
	contentType := strings.ToLower(b.ContentType)

	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	case strings.HasPrefix(contentType, "application/pdf"):
		return "document"
	case strings.HasPrefix(contentType, "application/"):
		return "application"
	default:
		return "other"
	}
}

// IsImage checks if the binary is an image file.
func (b *Binary) IsImage() bool {
	return b.GetMimeTypeCategory() == "image"
}

// IsVideo checks if the binary is a video file.
func (b *Binary) IsVideo() bool {
	return b.GetMimeTypeCategory() == "video"
}

// IsAudio checks if the binary is an audio file.
func (b *Binary) IsAudio() bool {
	return b.GetMimeTypeCategory() == "audio"
}

// IsDocument checks if the binary is a document file.
func (b *Binary) IsDocument() bool {
	return b.GetMimeTypeCategory() == "document"
}
