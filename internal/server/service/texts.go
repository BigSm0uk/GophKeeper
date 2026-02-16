package service

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/validation"
	"go.uber.org/zap"
)

// TextsService handles text business logic.
type TextsService struct {
	logger *zap.Logger
	repo   interfaces.TextRepository
}

// NewTextsService creates a new texts service.
func NewTextsService(logger *zap.Logger, repo interfaces.TextRepository) *TextsService {
	return &TextsService{
		logger: logger,
		repo:   repo,
	}
}

// CreateText creates a new text entry.
func (s *TextsService) CreateText(ctx context.Context, userID string, text *entity.Text) (*entity.Text, error) {
	if text == nil {
		return nil, models.ErrInvalidText
	}

	if err := s.validateText(userID, text); err != nil {
		return nil, err
	}

	domainText := &models.Text{
		UserID:    userID,
		Name:      text.Name,
		Content:   text.Content,
		Metadata:  text.Metadata,
		CreatedAt: text.CreatedAt,
		UpdatedAt: text.UpdatedAt,
	}

	created, err := s.repo.Create(ctx, domainText)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Text created successfully",
		zap.String("text_id", created.ID),
		zap.String("user_id", userID))

	return textModelToEntity(created), nil
}

// GetText retrieves a text by GetID.
func (s *TextsService) GetText(ctx context.Context, userID, textID string) (*entity.Text, error) {
	if userID == "" {
		return nil, models.ErrInvalidUserID
	}
	if textID == "" {
		return nil, models.ErrTextNotFound
	}

	text, err := s.repo.FindByID(ctx, textID)
	if err != nil {
		return nil, err
	}

	if text.UserID != userID {
		return nil, models.ErrTextNotFound
	}

	return textModelToEntity(text), nil
}

// ListTexts retrieves all texts for a user.
func (s *TextsService) ListTexts(ctx context.Context, userID string, limit, offset int) ([]*entity.Text, int64, error) {
	if userID == "" {
		return nil, 0, models.ErrInvalidUserID
	}

	texts, err := s.repo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*entity.Text, 0, len(texts))
	for _, t := range texts {
		result = append(result, textModelToEntity(t))
	}
	return result, count, nil
}

// UpdateText updates an existing text.
func (s *TextsService) UpdateText(ctx context.Context, userID string, text *entity.Text) (*entity.Text, error) {
	if userID == "" {
		return nil, models.ErrInvalidUserID
	}
	if text == nil {
		return nil, models.ErrInvalidText
	}
	if text.ID == "" {
		return nil, models.ErrTextNotFound
	}

	existing, err := s.repo.FindByID(ctx, text.ID)
	if err != nil {
		return nil, err
	}

	if existing.UserID != userID {
		return nil, models.ErrTextNotFound
	}

	if err := s.validateText(userID, text); err != nil {
		return nil, err
	}

	existing.Name = text.Name
	existing.Content = text.Content
	existing.Metadata = text.Metadata

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	s.logger.Info("Text updated successfully",
		zap.String("text_id", text.ID),
		zap.String("user_id", userID))

	updated, err := s.repo.FindByID(ctx, text.ID)
	if err != nil {
		return nil, err
	}

	return textModelToEntity(updated), nil
}

// DeleteText deletes a text by GetID.
func (s *TextsService) DeleteText(ctx context.Context, userID, textID string) error {
	if userID == "" {
		return models.ErrInvalidUserID
	}
	if textID == "" {
		return models.ErrTextNotFound
	}

	text, err := s.repo.FindByID(ctx, textID)
	if err != nil {
		return err
	}

	if text.UserID != userID {
		return models.ErrTextNotFound
	}

	if err := s.repo.Delete(ctx, textID); err != nil {
		return err
	}

	s.logger.Info("Text deleted successfully",
		zap.String("text_id", textID),
		zap.String("user_id", userID))

	return nil
}

// validateText validates text fields with error accumulation.
func (s *TextsService) validateText(userID string, text *entity.Text) error {
	v := validation.New()

	v.Check(userID != "", "user_id", "required")
	v.Check(text.Name != "", "name", "required")
	v.Check(text.Content != "", "content", "required")

	return v.Err()
}

func textModelToEntity(t *models.Text) *entity.Text {
	if t == nil {
		return nil
	}
	return &entity.Text{
		ID:        t.ID,
		UserID:    t.UserID,
		Name:      t.Name,
		Content:   t.Content,
		Metadata:  t.Metadata,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
