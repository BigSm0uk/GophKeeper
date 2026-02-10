package service

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/validation"
	"go.uber.org/zap"
)

type CredentialsService struct {
	logger *zap.Logger
	repo   interfaces.CredentialRepository
}

// NewCredentialsService creates a new credentials service
func NewCredentialsService(logger *zap.Logger, repo interfaces.CredentialRepository) *CredentialsService {
	return &CredentialsService{
		logger: logger,
		repo:   repo,
	}
}

// CreateCredential creates a new credential entry
func (s *CredentialsService) CreateCredential(ctx context.Context, userID string, cred *entity.Credential) (*entity.Credential, error) {
	if cred == nil {
		return nil, models.ErrInvalidCredential
	}

	if err := s.validateCredential(userID, cred); err != nil {
		return nil, err
	}

	// Convert entity.Credential to domain models.Credential
	domainCred := &models.Credential{
		UserID:    userID,
		Name:      cred.Name,
		Login:     cred.Login,
		Password:  cred.Password,
		URL:       cred.URL,
		Metadata:  cred.Metadata,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	}

	createdCred, err := s.repo.Create(ctx, domainCred)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Credential created successfully",
		zap.String("credential_id", createdCred.ID),
		zap.String("user_id", userID))

	// Convert domain models.Credential back to entity.Credential
	return &entity.Credential{
		ID:        createdCred.ID,
		UserID:    createdCred.UserID,
		Name:      createdCred.Name,
		Login:     createdCred.Login,
		Password:  createdCred.Password,
		URL:       createdCred.URL,
		Metadata:  createdCred.Metadata,
		CreatedAt: createdCred.CreatedAt,
		UpdatedAt: createdCred.UpdatedAt,
	}, nil
}

// GetCredential retrieves a credential by GetID
func (s *CredentialsService) GetCredential(ctx context.Context, userID, credentialID string) (*entity.Credential, error) {
	if userID == "" {
		return nil, models.ErrInvalidUserID
	}
	if credentialID == "" {
		return nil, models.ErrCredentialNotFound
	}

	cred, err := s.repo.FindByID(ctx, credentialID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if cred.UserID != userID {
		return nil, models.ErrCredentialNotFound
	}

	return &entity.Credential{
		ID:        cred.ID,
		UserID:    cred.UserID,
		Name:      cred.Name,
		Login:     cred.Login,
		Password:  cred.Password,
		URL:       cred.URL,
		Metadata:  cred.Metadata,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	}, nil
}

// ListCredentials retrieves all credentials for a user
func (s *CredentialsService) ListCredentials(ctx context.Context, userID string, limit, offset int) ([]*entity.Credential, int64, error) {
	if userID == "" {
		return nil, 0, models.ErrInvalidUserID
	}

	creds, err := s.repo.FindByUserId(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*entity.Credential, 0, len(creds))
	for _, cred := range creds {
		result = append(result, &entity.Credential{
			ID:        cred.ID,
			UserID:    cred.UserID,
			Name:      cred.Name,
			Login:     cred.Login,
			Password:  cred.Password,
			URL:       cred.URL,
			Metadata:  cred.Metadata,
			CreatedAt: cred.CreatedAt,
			UpdatedAt: cred.UpdatedAt,
		})
	}

	return result, count, nil
}

// UpdateCredential updates an existing credential
func (s *CredentialsService) UpdateCredential(ctx context.Context, userID string, cred *entity.Credential) (*entity.Credential, error) {
	if userID == "" {
		return nil, models.ErrInvalidUserID
	}
	if cred == nil {
		return nil, models.ErrInvalidCredential
	}
	if cred.ID == "" {
		return nil, models.ErrCredentialNotFound
	}

	// Get existing credential to check ownership
	existing, err := s.repo.FindByID(ctx, cred.ID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if existing.UserID != userID {
		return nil, models.ErrCredentialNotFound
	}

	if err := s.validateCredential(userID, cred); err != nil {
		return nil, err
	}

	// Update domain credential
	existing.Name = cred.Name
	existing.Login = cred.Login
	existing.Password = cred.Password
	existing.URL = cred.URL
	existing.Metadata = cred.Metadata

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	s.logger.Info("Credential updated successfully",
		zap.String("credential_id", cred.ID),
		zap.String("user_id", userID))

	// Fetch updated credential to return with updated timestamp
	updated, err := s.repo.FindByID(ctx, cred.ID)
	if err != nil {
		return nil, err
	}

	return &entity.Credential{
		ID:        updated.ID,
		UserID:    updated.UserID,
		Name:      updated.Name,
		Login:     updated.Login,
		Password:  updated.Password,
		URL:       updated.URL,
		Metadata:  updated.Metadata,
		CreatedAt: updated.CreatedAt,
		UpdatedAt: updated.UpdatedAt,
	}, nil
}

// DeleteCredential deletes a credential by GetID
func (s *CredentialsService) DeleteCredential(ctx context.Context, userID, credentialID string) error {
	if userID == "" {
		return models.ErrInvalidUserID
	}
	if credentialID == "" {
		return models.ErrCredentialNotFound
	}

	cred, err := s.repo.FindByID(ctx, credentialID)
	if err != nil {
		return err
	}

	if cred.UserID != userID {
		return models.ErrCredentialNotFound
	}

	if err := s.repo.Delete(ctx, credentialID); err != nil {
		return err
	}

	s.logger.Info("Credential deleted successfully",
		zap.String("credential_id", credentialID),
		zap.String("user_id", userID))

	return nil
}

// validateCredential validates credential fields with error accumulation.
func (s *CredentialsService) validateCredential(userID string, cred *entity.Credential) error {
	v := validation.New()

	v.Check(userID != "", "user_id", "required")
	v.Check(cred.Name != "", "name", "required")
	v.Check(cred.Login != "", "login", "required")
	v.Check(cred.Password != "", "password", "required")

	return v.Err()
}
