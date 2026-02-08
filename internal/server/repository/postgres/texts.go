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
	base            *BaseRepository[*models.Text]
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

// NewTextRepository creates a new text repository.
func NewTextRepository(logger *zap.Logger, db *db.PostgresDb) *TextRepository {
	return &TextRepository{
		base:            NewBaseRepository[*models.Text](db, logger, "texts", TextScanner),
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

// TextScanner scans a row into Text model, including soft-delete column.
func TextScanner(row pgx.Row) (*models.Text, error) {
	var text models.Text
	var metadata pgtype.Text
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&text.ID,
		&text.UserID,
		&text.Name,
		&text.Content,
		&metadata,
		&text.CreatedAt,
		&text.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}

	if metadata.Valid {
		text.Metadata = &metadata.String
	} else {
		text.Metadata = nil
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		text.DeletedAt = &t
	} else {
		text.DeletedAt = nil
	}

	return &text, nil
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
	text, err := r.base.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("Text not found", zap.String("text_id", id))
			return nil, models.ErrTextNotFound
		}
		return nil, err
	}
	return text, nil
}

func (r *TextRepository) FindByUserID(ctx context.Context, userID string) ([]*models.Text, error) {
	texts, err := r.base.FindByUserID(ctx, userID)
	if err != nil {
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
	err := r.base.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Text not found for deletion", zap.String("text_id", id))
			return models.ErrTextNotFound
		}
		return err
	}
	r.logger.Info("Text deleted successfully", zap.String("text_id", id))
	return nil
}

func (r *TextRepository) Exists(ctx context.Context, id string) (bool, error) {
	return r.base.Exists(ctx, id)
}

func (r *TextRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	return r.base.ExistsByUserID(ctx, userID)
}

func (r *TextRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.base.CountByUserID(ctx, userID)
}
