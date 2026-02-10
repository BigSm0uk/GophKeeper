package service

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/validation"
	"go.uber.org/zap"
)

// CardsService handles card business logic.
type CardsService struct {
	logger *zap.Logger
	repo   interfaces.CardRepository
}

// NewCardsService creates a new cards service.
func NewCardsService(logger *zap.Logger, repo interfaces.CardRepository) *CardsService {
	return &CardsService{
		logger: logger,
		repo:   repo,
	}
}

// CreateCard creates a new card entry.
func (s *CardsService) CreateCard(ctx context.Context, userID string, card *entity.Card) (*entity.Card, error) {
	if card == nil {
		return nil, models.ErrInvalidCard
	}

	if err := s.validateCard(userID, card); err != nil {
		return nil, err
	}

	domainCard := &models.Card{
		UserID:         userID,
		Name:           card.Name,
		CardNumber:     card.CardNumber,
		CardholderName: card.CardholderName,
		ExpiryDate:     card.ExpiryDate,
		CVV:            card.CVV,
		BankName:       card.BankName,
		Metadata:       card.Metadata,
		CreatedAt:      card.CreatedAt,
		UpdatedAt:      card.UpdatedAt,
	}

	created, err := s.repo.Create(ctx, domainCard)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Card created successfully",
		zap.String("card_id", created.ID),
		zap.String("user_id", userID))

	return cardModelToEntity(created), nil
}

// GetCard retrieves a card by GetID.
func (s *CardsService) GetCard(ctx context.Context, userID, cardID string) (*entity.Card, error) {
	if userID == "" {
		return nil, models.ErrInvalidUserID
	}
	if cardID == "" {
		return nil, models.ErrCardNotFound
	}

	card, err := s.repo.FindByID(ctx, cardID)
	if err != nil {
		return nil, err
	}

	if card.UserID != userID {
		return nil, models.ErrCardNotFound
	}

	return cardModelToEntity(card), nil
}

// ListCards retrieves all cards for a user.
func (s *CardsService) ListCards(ctx context.Context, userID string, limit, offset int) ([]*entity.Card, int64, error) {
	if userID == "" {
		return nil, 0, models.ErrInvalidUserID
	}

	cards, err := s.repo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*entity.Card, 0, len(cards))
	for _, c := range cards {
		result = append(result, cardModelToEntity(c))
	}
	return result, count, nil
}

// UpdateCard updates an existing card.
func (s *CardsService) UpdateCard(ctx context.Context, userID string, card *entity.Card) (*entity.Card, error) {
	if userID == "" {
		return nil, models.ErrInvalidUserID
	}
	if card == nil {
		return nil, models.ErrInvalidCard
	}
	if card.ID == "" {
		return nil, models.ErrCardNotFound
	}

	existing, err := s.repo.FindByID(ctx, card.ID)
	if err != nil {
		return nil, err
	}

	if existing.GetUserID() != userID {
		return nil, models.ErrCardNotFound
	}

	if err := s.validateCard(userID, card); err != nil {
		return nil, err
	}

	existing.Name = card.Name
	existing.CardNumber = card.CardNumber
	existing.CardholderName = card.CardholderName
	existing.ExpiryDate = card.ExpiryDate
	existing.CVV = card.CVV
	existing.BankName = card.BankName
	existing.Metadata = card.Metadata

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	s.logger.Info("Card updated successfully",
		zap.String("card_id", card.ID),
		zap.String("user_id", userID))

	updated, err := s.repo.FindByID(ctx, card.ID)
	if err != nil {
		return nil, err
	}

	return cardModelToEntity(updated), nil
}

// DeleteCard deletes a card by GetID.
func (s *CardsService) DeleteCard(ctx context.Context, userID, cardID string) error {
	if userID == "" {
		return models.ErrInvalidUserID
	}
	if cardID == "" {
		return models.ErrCardNotFound
	}

	card, err := s.repo.FindByID(ctx, cardID)
	if err != nil {
		return err
	}

	if card.GetUserID() != userID {
		return models.ErrCardNotFound
	}

	if err := s.repo.Delete(ctx, cardID); err != nil {
		return err
	}

	s.logger.Info("Card deleted successfully",
		zap.String("card_id", cardID),
		zap.String("user_id", userID))

	return nil
}

// validateCard validates card fields with error accumulation.
func (s *CardsService) validateCard(userID string, card *entity.Card) error {
	v := validation.New()

	v.Check(userID != "", "user_id", "required")
	v.Check(card.Name != "", "name", "required")
	v.Check(card.CardNumber != "", "card_number", "required")
	v.Check(card.CardholderName != "", "cardholder_name", "required")
	v.Check(card.ExpiryDate != "", "expiry_date", "required")
	v.Check(card.CVV != "", "cvv", "required")

	return v.Err()
}

func cardModelToEntity(c *models.Card) *entity.Card {
	if c == nil {
		return nil
	}
	return &entity.Card{
		ID:             c.GetID(),
		UserID:         c.GetUserID(),
		Name:           c.Name,
		CardNumber:     c.CardNumber,
		CardholderName: c.CardholderName,
		ExpiryDate:     c.ExpiryDate,
		CVV:            c.CVV,
		BankName:       c.BankName,
		Metadata:       c.Metadata,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
