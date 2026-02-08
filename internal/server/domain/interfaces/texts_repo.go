package interfaces

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

// TextRepository defines the interface for text data operations.
type TextRepository interface {
	// Create creates a new text and returns the created text with generated GetID.
	Create(ctx context.Context, text *models.Text) (*models.Text, error)

	// FindByID retrieves a text by its unique identifier.
	FindByID(ctx context.Context, id string) (*models.Text, error)

	// FindByUserID retrieves all texts by user GetID.
	FindByUserID(ctx context.Context, userID string) ([]*models.Text, error)

	// Update modifies an existing text's information.
	Update(ctx context.Context, text *models.Text) error

	// Delete removes a text by its unique identifier (soft delete).
	Delete(ctx context.Context, id string) error

	// Exists checks if a text exists by its unique identifier.
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByUserID checks if texts exist for a given user ID.
	ExistsByUserID(ctx context.Context, userID string) (bool, error)

	// CountByUserID returns the total number of texts for a given user ID.
	CountByUserID(ctx context.Context, userID string) (int64, error)
}
