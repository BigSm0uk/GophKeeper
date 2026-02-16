package interfaces

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	// Create creates a new user and returns the created user with generated GetID.
	Create(ctx context.Context, user *models.User) (*models.User, error)

	// FindByID retrieves a user by their unique identifier.
	FindByID(ctx context.Context, id string) (*models.User, error)

	// FindByUsername retrieves a user by their username.
	FindByUsername(ctx context.Context, username string) (*models.User, error)

	// Update modifies an existing user's information.
	Update(ctx context.Context, user *models.User) error

	// Delete removes a user by their unique identifier.
	Delete(ctx context.Context, id string) error

	// Exists checks if a user exists by their unique identifier.
	Exists(ctx context.Context, id string) (bool, error)
	// Exists checks if a user exists by their username.
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// Count returns the total number of users.
	Count(ctx context.Context) (int64, error)
}
