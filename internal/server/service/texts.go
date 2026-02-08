package service

import (
	"context"
	"errors"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	if text == nil {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	if text.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if text.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
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
		s.logger.Error("Failed to create text",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("name", text.Name))
		return nil, status.Error(codes.Internal, "failed to create text")
	}

	s.logger.Info("Text created successfully",
		zap.String("text_id", created.ID),
		zap.String("user_id", userID))

	return textModelToEntity(created), nil
}

// GetText retrieves a text by GetID.
func (s *TextsService) GetText(ctx context.Context, userID, textID string) (*entity.Text, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	if textID == "" {
		return nil, status.Error(codes.InvalidArgument, "text GetID is required")
	}

	text, err := s.repo.FindByID(ctx, textID)
	if err != nil {
		if errors.Is(err, models.ErrTextNotFound) {
			return nil, status.Error(codes.NotFound, "text not found")
		}
		s.logger.Error("Failed to find text",
			zap.Error(err),
			zap.String("text_id", textID))
		return nil, status.Error(codes.Internal, "failed to retrieve text")
	}

	if text.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	return textModelToEntity(text), nil
}

// ListTexts retrieves all texts for a user.
func (s *TextsService) ListTexts(ctx context.Context, userID string) ([]*entity.Text, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}

	texts, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to list texts",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Error(codes.Internal, "failed to list texts")
	}

	result := make([]*entity.Text, 0, len(texts))
	for _, t := range texts {
		result = append(result, textModelToEntity(t))
	}
	return result, nil
}

// UpdateText updates an existing text.
func (s *TextsService) UpdateText(ctx context.Context, userID string, text *entity.Text) (*entity.Text, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	if text == nil {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	if text.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "text GetID is required")
	}

	existing, err := s.repo.FindByID(ctx, text.ID)
	if err != nil {
		if errors.Is(err, models.ErrTextNotFound) {
			return nil, status.Error(codes.NotFound, "text not found")
		}
		s.logger.Error("Failed to find text for update",
			zap.Error(err),
			zap.String("text_id", text.ID))
		return nil, status.Error(codes.Internal, "failed to retrieve text")
	}

	if existing.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	if text.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if text.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	existing.Name = text.Name
	existing.Content = text.Content
	existing.Metadata = text.Metadata

	if err := s.repo.Update(ctx, existing); err != nil {
		if errors.Is(err, models.ErrTextNotFound) {
			return nil, status.Error(codes.NotFound, "text not found")
		}
		s.logger.Error("Failed to update text",
			zap.Error(err),
			zap.String("text_id", text.ID))
		return nil, status.Error(codes.Internal, "failed to update text")
	}

	s.logger.Info("Text updated successfully",
		zap.String("text_id", text.ID),
		zap.String("user_id", userID))

	updated, err := s.repo.FindByID(ctx, text.ID)
	if err != nil {
		s.logger.Error("Failed to fetch updated text",
			zap.Error(err),
			zap.String("text_id", text.ID))
		return nil, status.Error(codes.Internal, "failed to retrieve updated text")
	}

	return textModelToEntity(updated), nil
}

// DeleteText deletes a text by GetID.
func (s *TextsService) DeleteText(ctx context.Context, userID, textID string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user GetID is required")
	}
	if textID == "" {
		return status.Error(codes.InvalidArgument, "text GetID is required")
	}

	text, err := s.repo.FindByID(ctx, textID)
	if err != nil {
		if errors.Is(err, models.ErrTextNotFound) {
			return status.Error(codes.NotFound, "text not found")
		}
		s.logger.Error("Failed to find text for deletion",
			zap.Error(err),
			zap.String("text_id", textID))
		return status.Error(codes.Internal, "failed to retrieve text")
	}

	if text.UserID != userID {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	if err := s.repo.Delete(ctx, textID); err != nil {
		if errors.Is(err, models.ErrTextNotFound) {
			return status.Error(codes.NotFound, "text not found")
		}
		s.logger.Error("Failed to delete text",
			zap.Error(err),
			zap.String("text_id", textID))
		return status.Error(codes.Internal, "failed to delete text")
	}

	s.logger.Info("Text deleted successfully",
		zap.String("text_id", textID),
		zap.String("user_id", userID))

	return nil
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
