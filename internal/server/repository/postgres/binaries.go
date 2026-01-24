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
	"go.uber.org/zap"
)

type BinaryRepository struct {
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

func NewBinaryRepository(logger *zap.Logger, db *db.PostgresDb) *BinaryRepository {
	return &BinaryRepository{
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
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

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var id string
	var createdAt, updatedAt time.Time

	err = retry.Do(
		func() error {
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

// FindByID finds a binary entry by ID.
func (r *BinaryRepository) FindByID(ctx context.Context, id string) (*models.Binary, error) {
	query, args, err := sq.Select("id", "user_id", "name", "filename", "size", "content_type", "metadata", "storage_path", "checksum", "created_at", "updated_at").
		From("binaries").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find binary by ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var binary models.Binary
	var metadata pgtype.Text

	err = retry.Do(
		func() error {
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
					zap.String("binary_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("binary_id", id))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("Binary not found", zap.String("binary_id", id))
			return nil, models.ErrBinaryNotFound
		}
		r.logger.Error("Failed to find binary after retries",
			zap.Error(err),
			zap.String("binary_id", id))
		return nil, err
	}

	// Convert pgtype.Text to *string
	if metadata.Valid {
		binary.Metadata = &metadata.String
	} else {
		binary.Metadata = nil
	}

	return &binary, nil
}

// FindByUserID finds all binary entries for a specific user.
func (r *BinaryRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Binary, error) {
	query, args, err := sq.Select("id", "user_id", "name", "filename", "size", "content_type", "metadata", "storage_path", "checksum", "created_at", "updated_at").
		From("binaries").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find binaries by user ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var rows pgx.Rows
	err = retry.Do(
		func() error {
			var queryErr error
			rows, queryErr = conn.Query(ctx, query, args...)
			return queryErr
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("user_id", userID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", userID))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		r.logger.Error("Failed to find binaries after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}
	defer rows.Close()

	var binaries []*models.Binary
	for rows.Next() {
		var binary models.Binary
		var metadata pgtype.Text

		err := rows.Scan(
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
		if err != nil {
			r.logger.Error("Failed to scan binary row", zap.Error(err))
			return nil, err
		}

		// Convert pgtype.Text to *string
		if metadata.Valid {
			binary.Metadata = &metadata.String
		} else {
			binary.Metadata = nil
		}

		binaries = append(binaries, &binary)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating binary rows", zap.Error(err))
		return nil, err
	}

	r.logger.Debug("Binaries retrieved", zap.String("user_id", userID), zap.Int("count", len(binaries)))
	return binaries, nil
}

// Update updates binary metadata (name and metadata fields).
func (r *BinaryRepository) Update(ctx context.Context, binary *models.Binary) error {
	if binary == nil || binary.ID == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := sq.Update("binaries").
		Set("name", binary.Name).
		Set("metadata", binary.Metadata).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": binary.ID}).
		Suffix("RETURNING updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update binary query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	var updatedAt time.Time

	err = retry.Do(
		func() error {
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
	if id == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := sq.Delete("binaries").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build delete binary query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	var affectedRows int64

	err = retry.Do(
		func() error {
			result, err := conn.Exec(ctx, query, args...)
			if err != nil {
				return err
			}
			affectedRows = result.RowsAffected()
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("binary_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("binary_id", id))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		r.logger.Error("Failed to delete binary after retries",
			zap.Error(err),
			zap.String("binary_id", id))
		return err
	}

	if affectedRows == 0 {
		r.logger.Warn("Binary not found for deletion", zap.String("binary_id", id))
		return models.ErrBinaryNotFound
	}

	r.logger.Info("Binary deleted successfully",
		zap.String("binary_id", id),
		zap.Int64("affected_rows", affectedRows))
	return nil
}

// CountByUserID counts the number of binaries for a specific user.
func (r *BinaryRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	query, args, err := sq.Select("COUNT(*)").
		From("binaries").
		Where(sq.Eq{"user_id": userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build count binaries query", zap.Error(err))
		return 0, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return 0, err
	}
	defer conn.Release()

	var count int64

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(&count)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("user_id", userID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", userID))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		r.logger.Error("Failed to count binaries after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return 0, err
	}

	r.logger.Debug("Binaries count retrieved", zap.String("user_id", userID), zap.Int64("count", count))
	return count, nil
}

// FindByChecksum finds a binary by its checksum (useful for deduplication).
func (r *BinaryRepository) FindByChecksum(ctx context.Context, userID, checksum string) (*models.Binary, error) {
	query, args, err := sq.Select("id", "user_id", "name", "filename", "size", "content_type", "metadata", "storage_path", "checksum", "created_at", "updated_at").
		From("binaries").
		Where(sq.And{
			sq.Eq{"user_id": userID},
			sq.Eq{"checksum": checksum},
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find binary by checksum query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var binary models.Binary
	var metadata pgtype.Text

	err = retry.Do(
		func() error {
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

	// Convert pgtype.Text to *string
	if metadata.Valid {
		binary.Metadata = &metadata.String
	} else {
		binary.Metadata = nil
	}

	return &binary, nil
}
