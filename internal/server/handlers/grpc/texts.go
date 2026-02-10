package grpc

import (
	"context"
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TextsHandler implements TextsServiceServer.
type TextsHandler struct {
	pb.UnimplementedTextsServiceServer
	logger       *zap.Logger
	textsService *service.TextsService
}

// NewTextsHandler creates a new texts handler.
func NewTextsHandler(logger *zap.Logger, textsService *service.TextsService) *TextsHandler {
	return &TextsHandler{
		logger:       logger,
		textsService: textsService,
	}
}

// Create stores a new text entry.
func (h *TextsHandler) Create(ctx context.Context, req *pb.TextCreateRequest) (*pb.TextCreateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Create text request received",
		zap.String("name", req.Name))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid create text request data",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid text data: %s", err.Error()))
	}

	text, err := entity.MapTextFromRequest(user.ID, req)
	if err != nil {
		logger.Error("Failed to map request to text entity",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, status.Error(codes.InvalidArgument, "Invalid text data")
	}

	createdText, err := h.textsService.CreateText(ctx, user.ID, text)
	if err != nil {
		return nil, err
	}

	pbText, err := entity.MapTextToResponse(createdText)
	if err != nil {
		logger.Error("Failed to map text to response",
			zap.Error(err),
			zap.String("text_id", createdText.ID))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Create text request completed successfully",
		zap.String("text_id", createdText.ID),
		zap.String("user_id", user.ID))

	return &pb.TextCreateResponse{
		Text: pbText,
	}, nil
}

// Get returns a single text by id.
func (h *TextsHandler) Get(ctx context.Context, req *pb.TextGetRequest) (*pb.TextGetResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Get text request received",
		zap.String("text_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid get text request data",
			zap.Error(err),
			zap.String("text_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request data: %s", err.Error()))
	}

	text, err := h.textsService.GetText(ctx, user.ID, req.Id)
	if err != nil {
		return nil, err
	}

	pbText, err := entity.MapTextToResponse(text)
	if err != nil {
		logger.Error("Failed to map text to response",
			zap.Error(err),
			zap.String("text_id", req.Id))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Get text request completed successfully",
		zap.String("text_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.TextGetResponse{
		Text: pbText,
	}, nil
}

// List returns paginated text entries.
func (h *TextsHandler) List(ctx context.Context, req *pb.TextListRequest) (*pb.TextListResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("List texts request received")

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

	texts, total, err := h.textsService.ListTexts(ctx, user.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	pbTexts := make([]*pb.Text, 0, len(texts))
	for _, text := range texts {
		pbText, err := entity.MapTextToListItem(text)
		if err != nil {
			logger.Warn("Failed to map text to list item",
				zap.Error(err),
				zap.String("text_id", text.ID))
			continue
		}
		pbTexts = append(pbTexts, pbText)
	}

	logger.Info("List texts request completed successfully",
		zap.String("user_id", user.ID),
		zap.Int("count", len(pbTexts)))

	return &pb.TextListResponse{
		Items: pbTexts,
		Page: &pb.PageResponse{
			Total:  uint32(total),
			Limit:  uint32(limit),
			Offset: uint32(offset),
		},
	}, nil
}

// Update modifies an existing text entry.
func (h *TextsHandler) Update(ctx context.Context, req *pb.TextUpdateRequest) (*pb.TextUpdateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Update text request received",
		zap.String("text_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid update text request data",
			zap.Error(err),
			zap.String("text_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid text data: %s", err.Error()))
	}

	existingText, err := h.textsService.GetText(ctx, user.ID, req.Id)
	if err != nil {
		return nil, err
	}

	if err := entity.MapTextFromUpdateRequest(existingText, req); err != nil {
		logger.Error("Failed to map update request to text",
			zap.Error(err),
			zap.String("text_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, "Invalid text data")
	}

	updatedText, err := h.textsService.UpdateText(ctx, user.ID, existingText)
	if err != nil {
		return nil, err
	}

	pbText, err := entity.MapTextToResponse(updatedText)
	if err != nil {
		logger.Error("Failed to map text to response",
			zap.Error(err),
			zap.String("text_id", req.Id))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Update text request completed successfully",
		zap.String("text_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.TextUpdateResponse{
		Text: pbText,
	}, nil
}

// Delete removes a text entry.
func (h *TextsHandler) Delete(ctx context.Context, req *pb.TextDeleteRequest) (*pb.TextDeleteResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Delete text request received",
		zap.String("text_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid delete text request data",
			zap.Error(err),
			zap.String("text_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request data: %s", err.Error()))
	}

	err = h.textsService.DeleteText(ctx, user.ID, req.Id)
	if err != nil {
		return nil, err
	}

	logger.Info("Delete text request completed successfully",
		zap.String("text_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.TextDeleteResponse{
		Deleted: true,
	}, nil
}
