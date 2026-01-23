package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
)

const (
	// ChunkSize is the size of each chunk for streaming (64KB)
	ChunkSize = 64 * 1024
)

// BinariesClient provides methods for working with binary files.
type BinariesClient struct {
	client pb.BinariesServiceClient
	logger *zap.Logger
}

// NewBinariesClient creates a new binaries client.
func NewBinariesClient(client pb.BinariesServiceClient, logger *zap.Logger) *BinariesClient {
	return &BinariesClient{
		client: client,
		logger: logger,
	}
}

// UploadFile uploads a file using streaming.
func (c *BinariesClient) UploadFile(ctx context.Context, filePath, name string, metadata *string) (*pb.Binary, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	filename := filepath.Base(filePath)
	totalSize := fileInfo.Size()

	c.logger.Info("starting file upload",
		zap.String("filename", filename),
		zap.Int64("size", totalSize),
	)

	// Calculate checksum
	checksum, err := c.calculateChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	c.logger.Debug("file checksum calculated", zap.String("checksum", checksum))

	// Start streaming
	stream, err := c.client.UploadStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start upload stream: %w", err)
	}

	// Send metadata in first message
	metadataMsg := &pb.BinaryMetadata{
		Name:        name,
		Filename:    filename,
		ContentType: detectContentType(filename),
		TotalSize:   totalSize,
		Metadata:    metadata,
		Checksum:    checksum,
	}

	if err := stream.Send(&pb.BinaryUploadChunk{
		Data: &pb.BinaryUploadChunk_Metadata{
			Metadata: metadataMsg,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send metadata: %w", err)
	}

	// Send file data in chunks
	buffer := make([]byte, ChunkSize)
	var totalSent int64

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		if err := stream.Send(&pb.BinaryUploadChunk{
			Data: &pb.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to send chunk: %w", err)
		}

		totalSent += int64(n)

		// Log progress every 10 MB
		if totalSent%(10*1024*1024) == 0 {
			progress := float64(totalSent) / float64(totalSize) * 100
			c.logger.Debug("upload progress",
				zap.Int64("sent", totalSent),
				zap.Int64("total", totalSize),
				zap.Float64("percent", progress),
			)
		}
	}

	// Close stream and receive response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("failed to close stream: %w", err)
	}

	c.logger.Info("file uploaded successfully",
		zap.String("binary_id", resp.Binary.Id),
		zap.String("filename", resp.Binary.Filename),
		zap.Int64("size", resp.Binary.Size),
	)

	return resp.Binary, nil
}

// DownloadFile downloads a file using streaming.
func (c *BinariesClient) DownloadFile(ctx context.Context, binaryID, destinationPath string) error {
	c.logger.Info("starting file download",
		zap.String("binary_id", binaryID),
		zap.String("destination", destinationPath),
	)

	// Start streaming
	stream, err := c.client.DownloadStream(ctx, &pb.BinaryDownloadRequest{
		Id: binaryID,
	})
	if err != nil {
		return fmt.Errorf("failed to start download stream: %w", err)
	}

	// Create temporary file
	tempPath := destinationPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		file.Close()
		// Clean up temp file if download failed
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Receive chunks and write to file
	var totalReceived int64
	var totalSize int64
	var serverChecksum string
	hasher := sha256.New()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to receive chunk: %w", err)
		}

		// Write to file
		n, err := file.Write(chunk.ChunkData)
		if err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}

		totalReceived += int64(n)
		totalSize = chunk.TotalSize

		// Update hash
		hasher.Write(chunk.ChunkData)

		// Get checksum from last chunk
		if chunk.Checksum != nil {
			serverChecksum = *chunk.Checksum
		}

		// Log progress every 10 MB
		if totalReceived%(10*1024*1024) == 0 && totalSize > 0 {
			progress := float64(totalReceived) / float64(totalSize) * 100
			c.logger.Debug("download progress",
				zap.Int64("received", totalReceived),
				zap.Int64("total", totalSize),
				zap.Float64("percent", progress),
			)
		}
	}

	// Verify checksum
	calculatedChecksum := hex.EncodeToString(hasher.Sum(nil))
	if serverChecksum != "" && calculatedChecksum != serverChecksum {
		return fmt.Errorf("checksum verification failed: expected %s, got %s", serverChecksum, calculatedChecksum)
	}

	// Close file before rename
	file.Close()

	// Rename temp file to final name
	if err := os.Rename(tempPath, destinationPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	c.logger.Info("file downloaded successfully",
		zap.String("binary_id", binaryID),
		zap.Int64("size", totalReceived),
		zap.String("checksum", calculatedChecksum),
	)

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file.
func (c *BinariesClient) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// detectContentType detects content type based on file extension.
func detectContentType(filename string) string {
	ext := filepath.Ext(filename)
	
	// Map common extensions to MIME types
	contentTypes := map[string]string{
		".txt":  "text/plain",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}

	return "application/octet-stream"
}
