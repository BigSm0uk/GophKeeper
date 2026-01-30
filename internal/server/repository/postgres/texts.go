package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	pgerrors "github.com/BigSm0uk/GophKeeper/internal/server/app/db/pgerror"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	sq "github.com/Masterminds/squirrel"
	"github.com/avast/retry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Ensure TextRepository implements the TextRepository interface.
var _ interfaces.TextRepository = (*TextRepository)(nil)

type TextRepository struct {
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

// NewTextRepository creates a new text repository.
func NewTextRepository(logger *zap.Logger, db *db.PostgresDb) *TextRepository {
	return &TextRepository{
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

func (r *TextRepository) Create(ctx context.Context, text *models.Text) (*models.Text, error) {
	query, args, err := sq.Insert("texts").
		Columns("user_id", "name", "content", "metadata").
		Values(text.UserID, text.Name, text.Content, text.Metadata).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build create text query", zap.Error(err))
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
					zap.String("user_id", text.UserID),
					zap.String("name", text.Name))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", text.UserID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to create text after retries",
			zap.Error(err),
			zap.String("user_id", text.UserID),
			zap.String("name", text.Name))
		return nil, err
	}

	text.ID = id
	text.CreatedAt = createdAt
	text.UpdatedAt = updatedAt
	return text, nil
}

func (r *TextRepository) FindByID(ctx context.Context, id string) (*models.Text, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "content", "metadata", "created_at", "updated_at").
			From("texts").
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find text by ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var text models.Text
	var metadata pgtype.Text

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&text.ID,
				&text.UserID,
				&text.Name,
				&text.Content,
				&metadata,
				&text.CreatedAt,
				&text.UpdatedAt,
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
					zap.String("text_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("text_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("Text not found", zap.String("text_id", id))
			return nil, models.ErrTextNotFound
		}
		r.logger.Error("Failed to find text after retries",
			zap.Error(err),
			zap.String("text_id", id))
		return nil, err
	}

	if metadata.Valid {
		text.Metadata = &metadata.String
	} else {
		text.Metadata = nil
	}
	return &text, nil
}

func (r *TextRepository) FindByUserID(ctx context.Context, userID string) ([]*models.Text, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "content", "metadata", "created_at", "updated_at").
			From("texts").
			Where(sq.Eq{"user_id": userID}).
			OrderBy("updated_at DESC"),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find texts by user ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var texts []*models.Text

	err = retry.Do(
		func() error {
			rows, err := conn.Query(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			texts = nil
			for rows.Next() {
				var text models.Text
				var metadata pgtype.Text

				err := rows.Scan(
					&text.ID,
					&text.UserID,
					&text.Name,
					&text.Content,
					&metadata,
					&text.CreatedAt,
					&text.UpdatedAt,
				)
				if err != nil {
					return err
				}

				if metadata.Valid {
					text.Metadata = &metadata.String
				} else {
					text.Metadata = nil
				}

				texts = append(texts, &text)
			}
			return rows.Err()
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
		r.logger.Error("Failed to find texts after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}

	if len(texts) == 0 {
		r.logger.Debug("No texts found for user", zap.String("user_id", userID))
		return []*models.Text{}, nil
	}

	return texts, nil
}

func (r *TextRepository) Update(ctx context.Context, text *models.Text) error {
	if text == nil || text.ID == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilterToUpdate(
		sq.Update("texts").
			Set("name", text.Name).
			Set("content", text.Content).
			Set("metadata", text.Metadata).
			Set("updated_at", time.Now()).
			Where(sq.Eq{"id": text.ID}),
	).
		Suffix("RETURNING updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update text query", zap.Error(err))
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
					zap.String("text_id", text.ID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("text_id", text.ID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Text not found for update", zap.String("text_id", text.ID))
			return models.ErrTextNotFound
		}
		r.logger.Error("Failed to update text after retries",
			zap.Error(err),
			zap.String("text_id", text.ID))
		return err
	}

	text.UpdatedAt = updatedAt
	r.logger.Info("Text updated successfully", zap.String("text_id", text.ID))
	return nil
}

func (r *TextRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := sq.Update("texts").
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(sq.Expr("deleted_at IS NULL")).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build delete text query", zap.Error(err))
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
					zap.String("text_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("text_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to delete text after retries",
			zap.Error(err),
			zap.String("text_id", id))
		return err
	}

	if affectedRows == 0 {
		r.logger.Warn("Text not found for deletion", zap.String("text_id", id))
		return models.ErrTextNotFound
	}

	r.logger.Info("Text deleted successfully",
		zap.String("text_id", id),
		zap.Int64("affected_rows", affectedRows))
	return nil
}

func (r *TextRepository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From("texts").
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists text query", zap.Error(err))
		return false, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return false, err
	}
	defer conn.Release()

	var exists bool

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(&exists)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("text_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("text_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to check text existence after retries",
			zap.Error(err),
			zap.String("text_id", id))
		return false, err
	}

	return exists, nil
}

func (r *TextRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From("texts").
			Where(sq.Eq{"user_id": userID}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists text by user ID query", zap.Error(err))
		return false, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return false, err
	}
	defer conn.Release()

	var exists bool

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(&exists)
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
		r.logger.Error("Failed to check text existence after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return false, err
	}

	return exists, nil
}

func (r *TextRepository) Count(ctx context.Context) (int64, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*)").
			From("texts"),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build count texts query", zap.Error(err))
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
				r.logger.Warn("Retriable database error occurred, will retry", zap.Error(err))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation", zap.Uint("attempt", n+1), zap.Error(err))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to count texts after retries", zap.Error(err))
		return 0, err
	}

	r.logger.Debug("Texts count retrieved", zap.Int64("count", count))
	return count, nil
}
