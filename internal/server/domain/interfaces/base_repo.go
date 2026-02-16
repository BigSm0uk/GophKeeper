package interfaces

import "context"

type BaseRepository[T Entity] interface {
	// FindByID retrieves an entity by its unique identifier.
	FindByID(ctx context.Context, id string) (T, error)

	// FindByUserID retrieves all entities by user ID.
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]T, error)

	// Delete removes an entity by its unique identifier (soft delete).
	Delete(ctx context.Context, id string) error

	// Exists checks if an entity exists by its unique identifier.
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByUserID checks if entities exist for a given user ID.
	ExistsByUserID(ctx context.Context, userID string) (bool, error)

	// CountByUserID returns the total number of entities for a given user ID.
	CountByUserID(ctx context.Context, userID string) (int64, error)
}
