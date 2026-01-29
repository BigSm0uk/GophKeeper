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
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	if cred == nil {
		return nil, status.Error(codes.InvalidArgument, "credential is required")
	}

	// Validate credential data
	if cred.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if cred.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if cred.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
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
		s.logger.Error("Failed to create credential",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("name", cred.Name))
		return nil, status.Error(codes.Internal, "failed to create credential")
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

// GetCredential retrieves a credential by ID
func (s *CredentialsService) GetCredential(ctx context.Context, userID, credentialID string) (*entity.Credential, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}
	if credentialID == "" {
		return nil, status.Error(codes.InvalidArgument, "credential ID is required")
	}

	cred, err := s.repo.FindByID(ctx, credentialID)
	if err != nil {
		if errors.Is(err, models.ErrCredentialNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		s.logger.Error("Failed to find credential",
			zap.Error(err),
			zap.String("credential_id", credentialID))
		return nil, status.Error(codes.Internal, "failed to retrieve credential")
	}

	// Check ownership
	if cred.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
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
func (s *CredentialsService) ListCredentials(ctx context.Context, userID string) ([]*entity.Credential, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	creds, err := s.repo.FindByUserId(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to list credentials",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Error(codes.Internal, "failed to list credentials")
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

	return result, nil
}

// UpdateCredential updates an existing credential
func (s *CredentialsService) UpdateCredential(ctx context.Context, userID string, cred *entity.Credential) (*entity.Credential, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}
	if cred == nil {
		return nil, status.Error(codes.InvalidArgument, "credential is required")
	}
	if cred.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "credential ID is required")
	}

	// Get existing credential to check ownership
	existing, err := s.repo.FindByID(ctx, cred.ID)
	if err != nil {
		if errors.Is(err, models.ErrCredentialNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		s.logger.Error("Failed to find credential for update",
			zap.Error(err),
			zap.String("credential_id", cred.ID))
		return nil, status.Error(codes.Internal, "failed to retrieve credential")
	}

	// Check ownership
	if existing.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Validate credential data
	if cred.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if cred.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if cred.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Update domain credential
	existing.Name = cred.Name
	existing.Login = cred.Login
	existing.Password = cred.Password
	existing.URL = cred.URL
	existing.Metadata = cred.Metadata

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("Failed to update credential",
			zap.Error(err),
			zap.String("credential_id", cred.ID))
		return nil, status.Error(codes.Internal, "failed to update credential")
	}

	s.logger.Info("Credential updated successfully",
		zap.String("credential_id", cred.ID),
		zap.String("user_id", userID))

	// Fetch updated credential to return with updated timestamp
	updated, err := s.repo.FindByID(ctx, cred.ID)
	if err != nil {
		s.logger.Error("Failed to fetch updated credential",
			zap.Error(err),
			zap.String("credential_id", cred.ID))
		return nil, status.Error(codes.Internal, "failed to retrieve updated credential")
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

// DeleteCredential deletes a credential by ID
func (s *CredentialsService) DeleteCredential(ctx context.Context, userID, credentialID string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user ID is required")
	}
	if credentialID == "" {
		return status.Error(codes.InvalidArgument, "credential ID is required")
	}

	// Get credential to check ownership
	cred, err := s.repo.FindByID(ctx, credentialID)
	if err != nil {
		if errors.Is(err, models.ErrCredentialNotFound) {
			return status.Error(codes.NotFound, "credential not found")
		}
		s.logger.Error("Failed to find credential for deletion",
			zap.Error(err),
			zap.String("credential_id", credentialID))
		return status.Error(codes.Internal, "failed to retrieve credential")
	}

	// Check ownership
	if cred.UserID != userID {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	if err := s.repo.Delete(ctx, credentialID); err != nil {
		if errors.Is(err, models.ErrCredentialNotFound) {
			return status.Error(codes.NotFound, "credential not found")
		}
		s.logger.Error("Failed to delete credential",
			zap.Error(err),
			zap.String("credential_id", credentialID))
		return status.Error(codes.Internal, "failed to delete credential")
	}

	s.logger.Info("Credential deleted successfully",
		zap.String("credential_id", credentialID),
		zap.String("user_id", userID))

	return nil
}
