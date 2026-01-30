package interfaces

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

// CardRepository defines the interface for card data operations.
type CardRepository interface {
	// Create creates a new card and returns the created card with generated ID.
	Create(ctx context.Context, card *models.Card) (*models.Card, error)

	// FindByID retrieves a card by its unique identifier.
	FindByID(ctx context.Context, id string) (*models.Card, error)

	// FindByUserID retrieves all cards by user ID.
	FindByUserID(ctx context.Context, userID string) ([]*models.Card, error)

	// Update modifies an existing card's information.
	Update(ctx context.Context, card *models.Card) error

	// Delete removes a card by its unique identifier (soft delete).
	Delete(ctx context.Context, id string) error

	// Exists checks if a card exists by its unique identifier.
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByUserID checks if cards exist for a given user ID.
	ExistsByUserID(ctx context.Context, userID string) (bool, error)

	// Count returns the total number of cards.
	Count(ctx context.Context) (int64, error)
}
