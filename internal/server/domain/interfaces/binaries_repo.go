package interfaces

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

// BinariesRepository defines methods for working with binary metadata in the database
type BinariesRepository interface {
	Create(ctx context.Context, binary *models.Binary) (*models.Binary, error)
	FindByID(ctx context.Context, id string) (*models.Binary, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Binary, error)
	Update(ctx context.Context, binary *models.Binary) error
	Delete(ctx context.Context, id string) error
	CountByUserID(ctx context.Context, userID string) (int64, error)
	FindByChecksum(ctx context.Context, userID, checksum string) (*models.Binary, error)
}
