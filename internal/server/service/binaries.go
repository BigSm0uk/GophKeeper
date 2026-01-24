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
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BinaryService struct {
	logger      *zap.Logger
	repo        interfaces.BinaryRepository
	fileService *FileService
	maxFileSize int64
}

// NewBinaryService creates a new binary service
func NewBinaryService(logger *zap.Logger, repo interfaces.BinaryRepository, fileService *FileService, maxFileSize int64) *BinaryService {
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
		s.logger.Error("failed to check for existing file",
			zap.Error(err),
			zap.String("checksum", metadata.Checksum),
		)
		return nil, status.Error(codes.Internal, "failed to check for existing file")
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
		s.logger.Error("failed to create file", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create file")
	}
	defer cleanup()

	var totalReceived int64
	hasher := sha256.New()

	teeReader := io.TeeReader(reader, hasher)

	written, err := io.Copy(file, teeReader)
	if err != nil {
		s.logger.Error("failed to write file", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to write file")
	}

	totalReceived = written

	if totalReceived != metadata.TotalSize {
		s.logger.Error("size mismatch",
			zap.Int64("expected", metadata.TotalSize),
			zap.Int64("received", totalReceived),
		)
		return nil, status.Errorf(codes.InvalidArgument, "size mismatch: expected %d, received %d", metadata.TotalSize, totalReceived)
	}

	calculatedChecksum := hex.EncodeToString(hasher.Sum(nil))
	if calculatedChecksum != metadata.Checksum {
		s.logger.Warn("checksum mismatch",
			zap.String("expected", metadata.Checksum),
			zap.String("calculated", calculatedChecksum),
		)
		return nil, status.Error(codes.InvalidArgument, "checksum verification failed")
	}

	err = file.Close()
	if err != nil {
		s.logger.Error("failed to close file", zap.Error(err))
		return nil, err
	}

	tempPath := storagePath + ".tmp"
	if err := s.fileService.RenameFile(tempPath, storagePath); err != nil {
		s.logger.Error("failed to rename file", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to finalize file")
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
		deleteErr := s.fileService.DeleteFile(storagePath)
		if deleteErr != nil {
			s.logger.Error("failed to delete file", zap.Error(deleteErr))
			return nil, err
		}
		s.logger.Error("failed to save binary metadata", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to save metadata")
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
		if errors.Is(err, models.ErrBinaryNotFound) {
			return 0, status.Error(codes.NotFound, "binary not found")
		}
		s.logger.Error("failed to get binary metadata", zap.Error(err))
		return 0, status.Error(codes.Internal, "failed to get binary metadata")
	}

	if binary.UserID != userID {
		s.logger.Warn("unauthorized access attempt",
			zap.String("user_id", userID),
			zap.String("binary_id", binaryID),
			zap.String("owner_id", binary.UserID),
		)
		return 0, status.Error(codes.PermissionDenied, "access denied")
	}

	s.logger.Info("starting file download",
		zap.String("binary_id", binaryID),
		zap.String("user_id", userID),
		zap.String("filename", binary.Filename),
		zap.Int64("size", binary.Size),
	)

	file, fileSize, err := s.fileService.OpenFileForReading(binary.StoragePath)
	if err != nil {
		s.logger.Error("failed to open file", zap.Error(err))
		return 0, status.Error(codes.Internal, "failed to open file")
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			s.logger.Error("failed to close file", zap.Error(err))
		}
	}(file)

	written, err := io.Copy(writer, file)
	if err != nil {
		s.logger.Error("failed to write file to stream", zap.Error(err))
		return 0, status.Error(codes.Internal, "failed to write file to stream")
	}

	if written != fileSize {
		s.logger.Error("incomplete file transfer",
			zap.Int64("expected", fileSize),
			zap.Int64("written", written),
		)
		return written, status.Error(codes.Internal, "incomplete file transfer")
	}

	s.logger.Info("file downloaded successfully",
		zap.String("binary_id", binaryID),
		zap.String("user_id", userID),
		zap.Int64("size", written),
	)

	return written, nil
}

// GetBinary retrieves binary metadata by ID
func (s *BinaryService) GetBinary(ctx context.Context, userID, binaryID string) (*entity.Binary, error) {
	binary, err := s.repo.FindByID(ctx, binaryID)
	if err != nil {
		if errors.Is(err, models.ErrBinaryNotFound) {
			return nil, status.Error(codes.NotFound, "binary not found")
		}
		return nil, err
	}

	if binary.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
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
		if errors.Is(err, models.ErrBinaryNotFound) {
			return nil, status.Error(codes.NotFound, "binary not found")
		}
		return nil, err
	}

	if binary.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
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
		if errors.Is(err, models.ErrBinaryNotFound) {
			return status.Error(codes.NotFound, "binary not found")
		}
		return err
	}

	if binary.UserID != userID {
		return status.Error(codes.PermissionDenied, "access denied")
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
	if req.UserID == "" {
		return status.Error(codes.InvalidArgument, "user ID is required")
	}
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Filename == "" {
		return status.Error(codes.InvalidArgument, "filename is required")
	}
	if req.ContentType == "" {
		return status.Error(codes.InvalidArgument, "content type is required")
	}
	if req.TotalSize <= 0 {
		return status.Error(codes.InvalidArgument, "total size must be greater than 0")
	}
	if req.TotalSize > s.maxFileSize {
		return status.Errorf(codes.ResourceExhausted, "file size %d exceeds maximum allowed size %d", req.TotalSize, s.maxFileSize)
	}
	if req.Checksum == "" {
		return status.Error(codes.InvalidArgument, "checksum is required")
	}
	if len(req.Checksum) != 64 {
		return status.Error(codes.InvalidArgument, "invalid checksum format (expected SHA256)")
	}

	return nil
}

// generateStoragePath generates a storage path for a binary file
func (s *BinaryService) generateStoragePath(userID, binaryID, filename string) string {
	// Create path like: userID/binary_id_filename
	safeName := fmt.Sprintf("%s_%s", binaryID, filename)
	return fmt.Sprintf("%s/%s", userID, safeName)
}
