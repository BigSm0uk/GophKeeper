package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	pgerrors "github.com/BigSm0uk/GophKeeper/internal/server/app/db/pgerror"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	sq "github.com/Masterminds/squirrel"
	"github.com/avast/retry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type BinaryRepository struct {
	base            *BaseRepository[*models.Binary]
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

func NewBinaryRepository(logger *zap.Logger, db *db.PostgresDb) *BinaryRepository {
	return &BinaryRepository{
		base:            NewBaseRepository[*models.Binary](db, logger, "binaries", BinaryScanner),
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

// BinaryScanner scans a row into Binary model, including soft-delete column.
func BinaryScanner(row pgx.Row) (*models.Binary, error) {
	var binary models.Binary
	var metadata pgtype.Text
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&binary.ID,
		&binary.UserID,
		&binary.Name,
		&binary.Filename,
		&binary.Size,
		&binary.ContentType,
		&metadata,
		&binary.StoragePath,
		&binary.Checksum,
		&binary.CreatedAt,
		&binary.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}

	binary.Metadata = lo.Ternary(metadata.Valid, &metadata.String, nil)
	binary.DeletedAt = lo.Ternary(deletedAt.Valid, &deletedAt.Time, nil)

	return &binary, nil
}

// Create creates a new binary entry in the database.
func (r *BinaryRepository) Create(ctx context.Context, binary *models.Binary) (*models.Binary, error) {
	query, args, err := sq.Insert("binaries").
		Columns("user_id", "name", "filename", "size", "content_type", "metadata", "storage_path", "checksum").
		Values(binary.UserID, binary.Name, binary.Filename, binary.Size, binary.ContentType, binary.Metadata, binary.StoragePath, binary.Checksum).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build create binary query", zap.Error(err))
		return nil, err
	}

	var id string
	var createdAt, updatedAt time.Time

	err = retry.Do(
		func() error {
			conn, err := r.db.GetPool().Acquire(ctx)
			if err != nil {
				r.logger.Error("Failed to acquire database connection", zap.Error(err))
				return err
			}
			defer conn.Release()

			return conn.QueryRow(ctx, query, args...).Scan(&id, &createdAt, &updatedAt)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("filename", binary.Filename))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("filename", binary.Filename))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to create binary after retries",
			zap.Error(err),
			zap.String("filename", binary.Filename))
		return nil, err
	}

	binary.ID = id
	binary.CreatedAt = createdAt
	binary.UpdatedAt = updatedAt
	return binary, nil
}

// FindByID finds a binary entry by GetID.
func (r *BinaryRepository) FindByID(ctx context.Context, id string) (*models.Binary, error) {
	return r.base.FindByID(ctx, id)
}

// FindByUserID finds all binary entries for a specific user.
func (r *BinaryRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Binary, error) {
	return r.base.FindByUserID(ctx, userID)
}

// Update updates binary metadata (name and metadata fields).
func (r *BinaryRepository) Update(ctx context.Context, binary *models.Binary) error {
	if binary == nil || binary.ID == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilterToUpdate(
		sq.Update("binaries").
			Set("name", binary.Name).
			Set("metadata", binary.Metadata).
			Set("updated_at", time.Now()).
			Where(sq.Eq{"id": binary.ID}),
	).
		Suffix("RETURNING updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update binary query", zap.Error(err))
		return err
	}

	var updatedAt time.Time

	err = retry.Do(
		func() error {
			conn, err := r.db.GetPool().Acquire(ctx)
			if err != nil {
				r.logger.Error("Failed to acquire database connection", zap.Error(err))
				return err
			}
			defer conn.Release()

			return conn.QueryRow(ctx, query, args...).Scan(&updatedAt)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("binary_id", binary.ID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("binary_id", binary.ID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrBinaryNotFound
		}
		r.logger.Error("Failed to update binary after retries",
			zap.Error(err),
			zap.String("binary_id", binary.ID))
		return err
	}

	binary.UpdatedAt = updatedAt
	r.logger.Info("Binary updated successfully", zap.String("binary_id", binary.ID))
	return nil
}

// Delete deletes a binary entry from the database.
func (r *BinaryRepository) Delete(ctx context.Context, id string) error {
	return r.base.Delete(ctx, id)
}

// CountByUserID counts the number of binaries for a specific user.
func (r *BinaryRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.base.CountByUserID(ctx, userID)
}

// FindByChecksum finds a binary by its checksum (useful for deduplication).
func (r *BinaryRepository) FindByChecksum(ctx context.Context, userID, checksum string) (*models.Binary, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "filename", "size", "content_type", "metadata", "storage_path", "checksum", "created_at", "updated_at").
			From("binaries").
			Where(sq.And{
				sq.Eq{"user_id": userID},
				sq.Eq{"checksum": checksum},
			}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find binary by checksum query", zap.Error(err))
		return nil, err
	}

	var binary models.Binary
	var metadata pgtype.Text

	err = retry.Do(
		func() error {
			conn, err := r.db.GetPool().Acquire(ctx)
			if err != nil {
				r.logger.Error("Failed to acquire database connection", zap.Error(err))
				return err
			}
			defer conn.Release()

			return conn.QueryRow(ctx, query, args...).Scan(
				&binary.ID,
				&binary.UserID,
				&binary.Name,
				&binary.Filename,
				&binary.Size,
				&binary.ContentType,
				&metadata,
				&binary.StoragePath,
				&binary.Checksum,
				&binary.CreatedAt,
				&binary.UpdatedAt,
			)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("checksum", checksum))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("checksum", checksum))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrBinaryNotFound
		}
		r.logger.Error("Failed to find binary by checksum after retries",
			zap.Error(err),
			zap.String("checksum", checksum))
		return nil, err
	}

	if metadata.Valid {
		binary.Metadata = &metadata.String
	} else {
		binary.Metadata = nil
	}

	return &binary, nil
}
