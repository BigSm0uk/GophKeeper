package interfaces

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

// CredentialRepository defines the interface for credential data operations.
type CredentialRepository interface {
	// Create creates a new credential and returns the created credential with generated GetID.
	Create(ctx context.Context, user *models.Credential) (*models.Credential, error)

	// FindByID retrieves a credential by its unique identifier.
	FindByID(ctx context.Context, id string) (*models.Credential, error)

	// FindByUserId retrieves all credentials by user GetID.
	FindByUserId(ctx context.Context, id string) ([]*models.Credential, error)

	// Update modifies an existing credential's information.
	Update(ctx context.Context, credential *models.Credential) error

	// Delete removes a credential by its unique identifier.
	Delete(ctx context.Context, id string) error

	// Exists checks if a credential exists by its unique identifier.
	Exists(ctx context.Context, id string) (bool, error)
	// ExistsByUserId checks if credentials exist for a given user ID.
	ExistsByUserId(ctx context.Context, userID string) (bool, error)

	// CountByUserID returns the total number of credentials for a given user ID.
	CountByUserID(ctx context.Context, userID string) (int64, error)
}
