package grpc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BinariesHandler handles binary file operations.
type BinariesHandler struct {
	pb.UnimplementedBinariesServiceServer
	logger      *zap.Logger
	baseDir     string
	maxFileSize int64
}

// NewBinariesHandler creates a new binaries handler.
func NewBinariesHandler(logger *zap.Logger, baseDir string, maxFileSize int64) *BinariesHandler {
	return &BinariesHandler{
		logger:      logger,
		baseDir:     baseDir,
		maxFileSize: maxFileSize,
	}
}

// UploadStream handles streaming upload of binary files.
func (h *BinariesHandler) UploadStream(stream pb.BinariesService_UploadStreamServer) error {
	ctx := stream.Context()

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	req, err := stream.Recv()
	if err != nil {
		h.logger.Error("failed to receive first chunk", zap.Error(err))
		return status.Error(codes.InvalidArgument, "failed to receive metadata")
	}

	metadata := req.GetMetadata()
	if metadata == nil {
		return status.Error(codes.InvalidArgument, "first message must contain metadata")
	}

	h.logger.Info("starting file upload",
		zap.String("user_id", user.ID),
		zap.String("filename", metadata.Filename),
		zap.Int64("size", metadata.TotalSize),
	)

	if metadata.TotalSize > h.maxFileSize {
		return status.Errorf(codes.ResourceExhausted, "file size %d exceeds maximum allowed size %d", metadata.TotalSize, h.maxFileSize)
	}

	binaryID := generateUUID() // TODO(Denis): Implement proper UUID generation
	storagePath := h.generateStoragePath(user.ID, binaryID, metadata.Filename)

	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		h.logger.Error("failed to create directory", zap.Error(err))
		return status.Error(codes.Internal, "failed to create storage directory")
	}

	tempPath := storagePath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		h.logger.Error("failed to create file", zap.Error(err))
		return status.Error(codes.Internal, "failed to create file")
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	var totalReceived int64
	hasher := sha256.New()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.logger.Error("failed to receive chunk", zap.Error(err))
			return status.Error(codes.Internal, "failed to receive chunk")
		}

		chunkData := req.GetChunkData()
		if chunkData == nil {
			continue
		}

		n, err := file.Write(chunkData)
		if err != nil {
			h.logger.Error("failed to write chunk", zap.Error(err))
			return status.Error(codes.Internal, "failed to write chunk")
		}

		totalReceived += int64(n)

		hasher.Write(chunkData)

		if totalReceived > metadata.TotalSize {
			return status.Error(codes.InvalidArgument, "received more data than expected")
		}
	}

	if totalReceived != metadata.TotalSize {
		return status.Errorf(codes.InvalidArgument, "size mismatch: expected %d, received %d", metadata.TotalSize, totalReceived)
	}

	calculatedChecksum := hex.EncodeToString(hasher.Sum(nil))
	if calculatedChecksum != metadata.Checksum {
		h.logger.Warn("checksum mismatch",
			zap.String("expected", metadata.Checksum),
			zap.String("calculated", calculatedChecksum),
		)
		return status.Error(codes.InvalidArgument, "checksum verification failed")
	}

	file.Close()

	if err := os.Rename(tempPath, storagePath); err != nil {
		h.logger.Error("failed to rename file", zap.Error(err))
		return status.Error(codes.Internal, "failed to finalize file")
	}

	// TODO: Save metadata to database
	// binary := &entity.Binary{
	// 	ID:          binaryID,
	// 	UserID:      userID,
	// 	Name:        metadata.Name,
	// 	Filename:    metadata.Filename,
	// 	Size:        totalReceived,
	// 	ContentType: metadata.ContentType,
	// 	Metadata:    metadata.Metadata,
	// 	StoragePath: storagePath,
	// 	Checksum:    calculatedChecksum,
	// 	CreatedAt:   time.Now(),
	// 	UpdatedAt:   time.Now(),
	// }
	//
	// if err := h.repo.CreateBinary(ctx, binary); err != nil {
	// 	os.Remove(storagePath)
	// 	return status.Error(codes.Internal, "failed to save metadata")
	// }

	h.logger.Info("file uploaded successfully",
		zap.String("binary_id", binaryID),
		zap.String("user_id", user.ID),
		zap.Int64("size", totalReceived),
		zap.String("checksum", calculatedChecksum),
	)

	response := &pb.BinaryUploadResponse{
		Binary: &pb.Binary{
			Id:          binaryID,
			Name:        metadata.Name,
			Filename:    metadata.Filename,
			Size:        totalReceived,
			ContentType: metadata.ContentType,
			Metadata:    metadata.Metadata,
			Checksum:    calculatedChecksum,
			// CreatedAt and UpdatedAt should be set from database
		},
	}

	return stream.SendAndClose(response)
}

// DownloadStream handles streaming download of binary files.
func (h *BinariesHandler) DownloadStream(req *pb.BinaryDownloadRequest, stream pb.BinariesService_DownloadStreamServer) error {
	ctx := stream.Context()

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	// TODO: Get binary metadata from database
	// binary, err := h.repo.GetBinary(ctx, req.Id)
	// if err != nil {
	// 	return status.Error(codes.NotFound, "binary not found")
	// }
	//
	// // Verify ownership
	// if binary.UserID != userID {
	// 	return status.Error(codes.PermissionDenied, "access denied")
	// }

	// For now, use placeholder
	storagePath := "" // binary.StoragePath
	totalSize := int64(0)
	checksum := ""

	h.logger.Info("starting file download",
		zap.String("binary_id", req.Id),
		zap.String("user_id", user.ID),
	)

	file, err := os.Open(storagePath)
	if err != nil {
		h.logger.Error("failed to open file", zap.Error(err))
		return status.Error(codes.Internal, "failed to open file")
	}
	defer file.Close()

	const chunkSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, chunkSize)
	var offset int64

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			h.logger.Error("failed to read file", zap.Error(err))
			return status.Error(codes.Internal, "failed to read file")
		}

		chunk := &pb.BinaryDownloadChunk{
			ChunkData: buffer[:n],
			Offset:    offset,
			TotalSize: totalSize,
		}

		if offset+int64(n) == totalSize {
			chunk.Checksum = &checksum
		}

		if err := stream.Send(chunk); err != nil {
			h.logger.Error("failed to send chunk", zap.Error(err))
			return status.Error(codes.Internal, "failed to send chunk")
		}

		offset += int64(n)
	}

	h.logger.Info("file downloaded successfully",
		zap.String("binary_id", req.Id),
		zap.String("user_id", user.ID),
		zap.Int64("size", offset),
	)

	return nil
}

// generateStoragePath generates a unique storage path for a file.
func (h *BinariesHandler) generateStoragePath(userID, binaryID, filename string) string {
	// Create path like: baseDir/userID/binary_id_filename
	safeName := filepath.Base(filename)
	return filepath.Join(h.baseDir, userID, fmt.Sprintf("%s_%s", binaryID, safeName))
}

// generateUUID generates a new UUID (placeholder - use a proper UUID library).
func generateUUID() string {
	// TODO: Implement proper UUID generation
	// Use github.com/google/uuid or similar
	return "placeholder-uuid"
}
