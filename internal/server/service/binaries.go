package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BinaryService struct {
	logger      *zap.Logger
	repo        interfaces.BinariesRepository
	fileService *FileService
	maxFileSize int64
}

// NewBinaryService creates a new binary service
func NewBinaryService(logger *zap.Logger, repo interfaces.BinariesRepository, fileService *FileService, maxFileSize int64) *BinaryService {
	return &BinaryService{
		logger:      logger,
		repo:        repo,
		fileService: fileService,
		maxFileSize: maxFileSize,
	}
}

// UploadStream handles streaming upload of a binary file
func (s *BinaryService) UploadStream(ctx context.Context, metadata *entity.StreamUploadRequest, reader io.Reader) (*entity.StreamUploadResult, error) {
	if err := s.validateUploadRequest(metadata); err != nil {
		return nil, err
	}

	s.logger.Info("starting file upload",
		zap.String("user_id", metadata.UserID),
		zap.String("filename", metadata.Filename),
		zap.Int64("size", metadata.TotalSize),
	)

	existing, err := s.repo.FindByChecksum(ctx, metadata.UserID, metadata.Checksum)
	if err != nil && !errors.Is(err, models.ErrBinaryNotFound) {
		return nil, err
	}

	if err == nil && existing != nil {
		// File with same checksum already exists
		s.logger.Info("file with same checksum already exists, skipping upload",
			zap.String("binary_id", existing.ID),
			zap.String("checksum", metadata.Checksum),
		)

		result := &entity.Binary{
			ID:          existing.ID,
			UserID:      existing.UserID,
			Name:        existing.Name,
			Filename:    existing.Filename,
			Size:        existing.Size,
			ContentType: existing.ContentType,
			Metadata:    existing.Metadata,
			Checksum:    existing.Checksum,
			CreatedAt:   existing.CreatedAt,
			UpdatedAt:   existing.UpdatedAt,
		}
		return &entity.StreamUploadResult{Binary: result, BytesRead: 0}, nil
	}

	// File not found (err == models.ErrBinaryNotFound) or existing == nil, proceed with upload

	binaryID := uuid.New().String()
	storagePath := s.generateStoragePath(metadata.UserID, binaryID, metadata.Filename)

	file, cleanup, err := s.fileService.CreateFileForWriting(storagePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var totalReceived int64
	hasher := sha256.New()

	teeReader := io.TeeReader(reader, hasher)

	written, err := io.Copy(file, teeReader)
	if err != nil {
		return nil, err
	}

	totalReceived = written

	if totalReceived != metadata.TotalSize {
		return nil, models.ErrInvalidFileSize
	}

	calculatedChecksum := hex.EncodeToString(hasher.Sum(nil))
	if calculatedChecksum != metadata.Checksum {
		return nil, models.ErrInvalidBinary
	}

	err = file.Close()
	if err != nil {
		s.logger.Error("failed to close file", zap.Error(err))
		return nil, err
	}

	tempPath := storagePath + ".tmp"
	if err := s.fileService.RenameFile(tempPath, storagePath); err != nil {
		return nil, err
	}

	binary := &models.Binary{
		ID:          binaryID,
		UserID:      metadata.UserID,
		Name:        metadata.Name,
		Filename:    metadata.Filename,
		Size:        totalReceived,
		ContentType: metadata.ContentType,
		Metadata:    metadata.Metadata,
		StoragePath: storagePath,
		Checksum:    calculatedChecksum,
	}

	createdBinary, err := s.repo.Create(ctx, binary)
	if err != nil {
		_ = s.fileService.DeleteFile(storagePath)
		return nil, err
	}

	s.logger.Info("file uploaded successfully",
		zap.String("binary_id", createdBinary.ID),
		zap.String("user_id", metadata.UserID),
		zap.Int64("size", totalReceived),
		zap.String("checksum", calculatedChecksum),
	)

	result := &entity.Binary{
		ID:          createdBinary.ID,
		UserID:      createdBinary.UserID,
		Name:        createdBinary.Name,
		Filename:    createdBinary.Filename,
		Size:        createdBinary.Size,
		ContentType: createdBinary.ContentType,
		Metadata:    createdBinary.Metadata,
		Checksum:    createdBinary.Checksum,
		CreatedAt:   createdBinary.CreatedAt,
		UpdatedAt:   createdBinary.UpdatedAt,
	}

	return &entity.StreamUploadResult{Binary: result, BytesRead: totalReceived}, nil
}

// DownloadStream handles streaming download of a binary file
func (s *BinaryService) DownloadStream(ctx context.Context, userID, binaryID string, writer io.Writer) (int64, error) {
	binary, err := s.repo.FindByID(ctx, binaryID)
	if err != nil {
		return 0, err
	}

	if binary.UserID != userID {
		return 0, models.ErrBinaryNotFound
	}

	s.logger.Info("starting file download",
		zap.String("binary_id", binaryID),
		zap.String("user_id", userID),
		zap.String("filename", binary.Filename),
		zap.Int64("size", binary.Size),
	)

	file, fileSize, err := s.fileService.OpenFileForReading(binary.StoragePath)
	if err != nil {
		return 0, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			s.logger.Error("failed to close file", zap.Error(err))
		}
	}(file)

	written, err := io.Copy(writer, file)
	if err != nil {
		return 0, err
	}

	if written != fileSize {
		return written, models.ErrInvalidFileSize
	}

	s.logger.Info("file downloaded successfully",
		zap.String("binary_id", binaryID),
		zap.String("user_id", userID),
		zap.Int64("size", written),
	)

	return written, nil
}

// GetBinary retrieves binary metadata by GetID
func (s *BinaryService) GetBinary(ctx context.Context, userID, binaryID string) (*entity.Binary, error) {
	binary, err := s.repo.FindByID(ctx, binaryID)
	if err != nil {
		return nil, err
	}

	if binary.UserID != userID {
		return nil, models.ErrBinaryNotFound
	}

	return &entity.Binary{
		ID:          binary.ID,
		UserID:      binary.UserID,
		Name:        binary.Name,
		Filename:    binary.Filename,
		Size:        binary.Size,
		ContentType: binary.ContentType,
		Metadata:    binary.Metadata,
		Checksum:    binary.Checksum,
		CreatedAt:   binary.CreatedAt,
		UpdatedAt:   binary.UpdatedAt,
	}, nil
}

// ListBinaries retrieves a list of binaries for a user
func (s *BinaryService) ListBinaries(ctx context.Context, userID string, limit, offset int) ([]*entity.Binary, int64, error) {
	binaries, err := s.repo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*entity.Binary, 0, len(binaries))
	for _, b := range binaries {
		result = append(result, &entity.Binary{
			ID:          b.ID,
			UserID:      b.UserID,
			Name:        b.Name,
			Filename:    b.Filename,
			Size:        b.Size,
			ContentType: b.ContentType,
			Metadata:    b.Metadata,
			Checksum:    b.Checksum,
			CreatedAt:   b.CreatedAt,
			UpdatedAt:   b.UpdatedAt,
		})
	}

	return result, count, nil
}

// UpdateBinary updates binary metadata
func (s *BinaryService) UpdateBinary(ctx context.Context, userID, binaryID, name string, metadata *string) (*entity.Binary, error) {
	binary, err := s.repo.FindByID(ctx, binaryID)
	if err != nil {
		return nil, err
	}

	if binary.UserID != userID {
		return nil, models.ErrBinaryNotFound
	}

	binary.Name = name
	binary.Metadata = metadata

	if err := s.repo.Update(ctx, binary); err != nil {
		return nil, err
	}

	return &entity.Binary{
		ID:          binary.ID,
		UserID:      binary.UserID,
		Name:        binary.Name,
		Filename:    binary.Filename,
		Size:        binary.Size,
		ContentType: binary.ContentType,
		Metadata:    binary.Metadata,
		Checksum:    binary.Checksum,
		CreatedAt:   binary.CreatedAt,
		UpdatedAt:   binary.UpdatedAt,
	}, nil
}

// DeleteBinary deletes a binary and its file
func (s *BinaryService) DeleteBinary(ctx context.Context, userID, binaryID string) error {
	binary, err := s.repo.FindByID(ctx, binaryID)
	if err != nil {
		return err
	}

	if binary.UserID != userID {
		return models.ErrBinaryNotFound
	}

	if err := s.fileService.DeleteFile(binary.StoragePath); err != nil {
		s.logger.Warn("failed to delete file from filesystem",
			zap.Error(err),
			zap.String("binary_id", binaryID),
			zap.String("storage_path", binary.StoragePath),
		)
		// Continue with database deletion even if file deletion fails
	}

	if err := s.repo.Delete(ctx, binaryID); err != nil {
		return err
	}

	s.logger.Info("binary deleted successfully",
		zap.String("binary_id", binaryID),
		zap.String("user_id", userID),
	)

	return nil
}

// validateUploadRequest validates the upload request
func (s *BinaryService) validateUploadRequest(req *entity.StreamUploadRequest) error {
	v := validation.New()

	v.Check(req.UserID != "", "user_id", "required")
	v.Check(req.Name != "", "name", "required")
	v.Check(req.Filename != "", "filename", "required")
	v.Check(req.ContentType != "", "content_type", "required")
	v.Check(req.TotalSize > 0, "total_size", "must be greater than 0")
	v.Check(req.TotalSize <= s.maxFileSize, "total_size", fmt.Sprintf("exceeds max size %d", s.maxFileSize))
	v.Check(len(req.Checksum) == 64, "checksum", "must be 64 characters (SHA256)")

	return v.Err()
}

// generateStoragePath generates a storage path for a binary file
func (s *BinaryService) generateStoragePath(userID, binaryID, filename string) string {
	safeName := fmt.Sprintf("%s_%s", binaryID, filename)
	return fmt.Sprintf("%s/%s", userID, safeName)
}
