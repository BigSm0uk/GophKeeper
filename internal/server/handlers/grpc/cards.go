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

// CardsHandler implements CardsServiceServer.
type CardsHandler struct {
	pb.UnimplementedCardsServiceServer
	logger       *zap.Logger
	cardsService *service.CardsService
}

// NewCardsHandler creates a new cards handler.
func NewCardsHandler(logger *zap.Logger, cardsService *service.CardsService) *CardsHandler {
	return &CardsHandler{
		logger:       logger,
		cardsService: cardsService,
	}
}

// Create stores a new card entry.
func (h *CardsHandler) Create(ctx context.Context, req *pb.CardCreateRequest) (*pb.CardCreateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Create card request received",
		zap.String("name", req.Name))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid create card request data",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid card data: %s", err.Error()))
	}

	card, err := entity.MapCardFromRequest(user.ID, req)
	if err != nil {
		logger.Error("Failed to map request to card entity",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, status.Error(codes.InvalidArgument, "Invalid card data")
	}

	createdCard, err := h.cardsService.CreateCard(ctx, user.ID, card)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCard, err := entity.MapCardToResponse(createdCard)
	if err != nil {
		logger.Error("Failed to map card to response",
			zap.Error(err),
			zap.String("card_id", createdCard.ID))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Create card request completed successfully",
		zap.String("card_id", createdCard.ID),
		zap.String("user_id", user.ID))

	return &pb.CardCreateResponse{
		Card: pbCard,
	}, nil
}

// Get returns a single card by id.
func (h *CardsHandler) Get(ctx context.Context, req *pb.CardGetRequest) (*pb.CardGetResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Get card request received",
		zap.String("card_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid get card request data",
			zap.Error(err),
			zap.String("card_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request data: %s", err.Error()))
	}

	card, err := h.cardsService.GetCard(ctx, user.ID, req.Id)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCard, err := entity.MapCardToResponse(card)
	if err != nil {
		logger.Error("Failed to map card to response",
			zap.Error(err),
			zap.String("card_id", req.Id))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Get card request completed successfully",
		zap.String("card_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.CardGetResponse{
		Card: pbCard,
	}, nil
}

// List returns paginated card entries.
func (h *CardsHandler) List(ctx context.Context, req *pb.CardListRequest) (*pb.CardListResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("List cards request received")

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

	cards, total, err := h.cardsService.ListCards(ctx, user.ID, limit, offset)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCards := make([]*pb.Card, 0, len(cards))
	for _, card := range cards {
		pbCard, err := entity.MapCardToListItem(card)
		if err != nil {
			logger.Warn("Failed to map card to list item",
				zap.Error(err),
				zap.String("card_id", card.ID))
			continue
		}
		pbCards = append(pbCards, pbCard)
	}

	logger.Info("List cards request completed successfully",
		zap.String("user_id", user.ID),
		zap.Int("count", len(pbCards)))

	return &pb.CardListResponse{
		Items: pbCards,
		Page: &pb.PageResponse{
			Total:  uint32(total),
			Limit:  uint32(limit),
			Offset: uint32(offset),
		},
	}, nil
}

// Update modifies an existing card entry.
func (h *CardsHandler) Update(ctx context.Context, req *pb.CardUpdateRequest) (*pb.CardUpdateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Update card request received",
		zap.String("card_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid update card request data",
			zap.Error(err),
			zap.String("card_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid card data: %s", err.Error()))
	}

	existingCard, err := h.cardsService.GetCard(ctx, user.ID, req.Id)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	if err := entity.MapCardFromUpdateRequest(existingCard, req); err != nil {
		logger.Error("Failed to map update request to card",
			zap.Error(err),
			zap.String("card_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, "Invalid card data")
	}

	updatedCard, err := h.cardsService.UpdateCard(ctx, user.ID, existingCard)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCard, err := entity.MapCardToResponse(updatedCard)
	if err != nil {
		logger.Error("Failed to map card to response",
			zap.Error(err),
			zap.String("card_id", req.Id))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Update card request completed successfully",
		zap.String("card_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.CardUpdateResponse{
		Card: pbCard,
	}, nil
}

// Delete removes a card entry.
func (h *CardsHandler) Delete(ctx context.Context, req *pb.CardDeleteRequest) (*pb.CardDeleteResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Delete card request received",
		zap.String("card_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid delete card request data",
			zap.Error(err),
			zap.String("card_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request data: %s", err.Error()))
	}

	err = h.cardsService.DeleteCard(ctx, user.ID, req.Id)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	logger.Info("Delete card request completed successfully",
		zap.String("card_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.CardDeleteResponse{
		Deleted: true,
	}, nil
}
