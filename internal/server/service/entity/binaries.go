package entity

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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
}

// MapBinaryToResponse converts domain Binary to protobuf Binary message.
func MapBinaryToResponse(binary *Binary) (*pb.Binary, error) {
	if binary == nil {
		return nil, fmt.Errorf("binary is nil")
	}

	response := &pb.Binary{
		Id:          binary.ID,
		Name:        binary.Name,
		Filename:    binary.Filename,
		Size:        binary.Size,
		ContentType: binary.ContentType,
		Metadata:    binary.Metadata,
		Checksum:    binary.Checksum,
		CreatedAt:   timestamppb.New(binary.CreatedAt),
		UpdatedAt:   timestamppb.New(binary.UpdatedAt),
	}

	return response, nil
}

// MapBinaryFromMetadata converts protobuf BinaryMetadata to domain Binary.
// Note: This creates the metadata, actual file data should be handled separately.
func MapBinaryFromMetadata(userID string, metadata *pb.BinaryMetadata) (*Binary, error) {
	if metadata == nil {
		return nil, fmt.Errorf("metadata is nil")
	}

	now := time.Now()
	binary := &Binary{
		UserID:      userID,
		Name:        metadata.Name,
		Filename:    metadata.Filename,
		Size:        metadata.TotalSize,
		ContentType: metadata.ContentType,
		Metadata:    metadata.Metadata,
		Checksum:    metadata.Checksum,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return binary, nil
}

// MapBinaryFromUpdateRequest updates domain Binary from protobuf BinaryUpdateRequest.
func MapBinaryFromUpdateRequest(existing *Binary, req *pb.BinaryUpdateRequest) error {
	if existing == nil {
		return fmt.Errorf("existing binary is nil")
	}
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	existing.Name = req.Name
	existing.Metadata = req.Metadata
	existing.UpdatedAt = time.Now()

	return nil
}

// MapBinaryToListItem converts domain Binary to protobuf Binary for list responses.
func MapBinaryToListItem(binary *Binary) (*pb.Binary, error) {
	if binary == nil {
		return nil, fmt.Errorf("binary is nil")
	}

	response := &pb.Binary{
		Id:          binary.ID,
		Name:        binary.Name,
		Filename:    binary.Filename,
		Size:        binary.Size,
		ContentType: binary.ContentType,
		Checksum:    binary.Checksum,
		CreatedAt:   timestamppb.New(binary.CreatedAt),
		UpdatedAt:   timestamppb.New(binary.UpdatedAt),
		// Metadata is omitted for list views
	}

	return response, nil
}

// GenerateStoragePath generates a storage path for the binary file.
// This should create a unique path to avoid collisions.
func (b *Binary) GenerateStoragePath(baseDir string) {
	// Create path like: baseDir/userID/year/month/filename_uuid.ext
	year, month, _ := b.CreatedAt.Date()
	ext := filepath.Ext(b.Filename)

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

// GetFileExtension returns the file extension from filename.
func (b *Binary) GetFileExtension() string {
	return filepath.Ext(b.Filename)
}

// IsValidFileSize checks if the file size is within acceptable limits.
func (b *Binary) IsValidFileSize(maxSize int64) bool {
	return b.Size > 0 && b.Size <= maxSize
}
