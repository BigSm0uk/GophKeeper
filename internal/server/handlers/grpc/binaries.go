package grpc

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BinariesHandler handles binary file operations via gRPC.
type BinariesHandler struct {
	pb.UnimplementedBinariesServiceServer
	logger        *zap.Logger
	binaryService *service.BinaryService
}

// NewBinariesHandler creates a new binaries handler.
func NewBinariesHandler(logger *zap.Logger, binaryService *service.BinaryService) *BinariesHandler {
	return &BinariesHandler{
		logger:        logger,
		binaryService: binaryService,
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

	uploadReq := &entity.StreamUploadRequest{
		UserID:      user.ID,
		Name:        metadata.Name,
		Filename:    metadata.Filename,
		ContentType: metadata.ContentType,
		TotalSize:   metadata.TotalSize,
		Metadata:    metadata.Metadata,
		Checksum:    metadata.Checksum,
	}

	reader := entity.NewStreamReader(stream)

	result, err := h.binaryService.UploadStream(ctx, uploadReq, reader)
	if err != nil {
		return err
	}

	pbBinary, err := entity.MapBinaryToResponse(result.Binary)
	if err != nil {
		h.logger.Error("failed to map binary to response", zap.Error(err))
		return status.Error(codes.Internal, "failed to map response")
	}

	response := &pb.BinaryUploadResponse{
		Binary: pbBinary,
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

	binary, err := h.binaryService.GetBinary(ctx, user.ID, req.Id)
	if err != nil {
		return err
	}

	writer := newStreamWriter(stream, binary.Size, binary.Checksum)

	_, err = h.binaryService.DownloadStream(ctx, user.ID, req.Id, writer)
	if err != nil {
		return err
	}

	return nil
}

// streamWriter is an adapter that converts io.Writer to gRPC stream
type streamWriter struct {
	stream        pb.BinariesService_DownloadStreamServer
	totalSize     int64
	checksum      string
	offset        int64
	chunkSize     int
	lastChunkSent bool
}

func newStreamWriter(stream pb.BinariesService_DownloadStreamServer, totalSize int64, checksum string) *streamWriter {
	return &streamWriter{
		stream:    stream,
		totalSize: totalSize,
		checksum:  checksum,
		offset:    0,
		chunkSize: 64 * 1024, // 64KB chunks
	}
}

func (sw *streamWriter) Write(p []byte) (n int, err error) {
	totalWritten := 0

	for len(p) > 0 {
		// Determine chunk size
		chunkLen := min(len(p), sw.chunkSize)

		chunk := &pb.BinaryDownloadChunk{
			ChunkData: p[:chunkLen],
			Offset:    sw.offset,
			TotalSize: sw.totalSize,
		}

		// Add checksum to the last chunk
		if sw.offset+int64(chunkLen) == sw.totalSize && !sw.lastChunkSent {
			chunk.Checksum = &sw.checksum
			sw.lastChunkSent = true
		}

		if err := sw.stream.Send(chunk); err != nil {
			return totalWritten, err
		}

		totalWritten += chunkLen
		sw.offset += int64(chunkLen)
		p = p[chunkLen:]
	}

	return totalWritten, nil
}

// Get returns metadata for a binary entry.
func (h *BinariesHandler) Get(ctx context.Context, req *pb.BinaryGetRequest) (*pb.BinaryGetResponse, error) {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	binary, err := h.binaryService.GetBinary(ctx, user.ID, req.Id)
	if err != nil {
		return nil, err
	}

	pbBinary, err := entity.MapBinaryToResponse(binary)
	if err != nil {
		h.logger.Error("failed to map binary to response", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to map response")
	}

	return &pb.BinaryGetResponse{
		Binary: pbBinary,
	}, nil
}

// List returns paginated binary entries.
func (h *BinariesHandler) List(ctx context.Context, req *pb.BinaryListRequest) (*pb.BinaryListResponse, error) {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.Page.Limit)
	offset := int(req.Page.Offset)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	binaries, total, err := h.binaryService.ListBinaries(ctx, user.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.Binary, 0, len(binaries))
	for _, binary := range binaries {
		pbBinary, err := entity.MapBinaryToListItem(binary)
		if err != nil {
			h.logger.Error("failed to map binary to list item", zap.Error(err))
			continue
		}
		items = append(items, pbBinary)
	}

	return &pb.BinaryListResponse{
		Items: items,
		Page: &pb.PageResponse{
			Total:  uint32(total),
			Limit:  uint32(limit),
			Offset: uint32(offset),
		},
	}, nil
}

// Update modifies binary metadata.
func (h *BinariesHandler) Update(ctx context.Context, req *pb.BinaryUpdateRequest) (*pb.BinaryUpdateResponse, error) {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	binary, err := h.binaryService.UpdateBinary(ctx, user.ID, req.Id, req.Name, req.Metadata)
	if err != nil {
		return nil, err
	}

	pbBinary, err := entity.MapBinaryToResponse(binary)
	if err != nil {
		h.logger.Error("failed to map binary to response", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to map response")
	}

	return &pb.BinaryUpdateResponse{
		Binary: pbBinary,
	}, nil
}

// Delete removes a binary entry.
func (h *BinariesHandler) Delete(ctx context.Context, req *pb.BinaryDeleteRequest) (*pb.BinaryDeleteResponse, error) {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.binaryService.DeleteBinary(ctx, user.ID, req.Id); err != nil {
		return nil, err
	}

	return &pb.BinaryDeleteResponse{
		Deleted: true,
	}, nil
}
