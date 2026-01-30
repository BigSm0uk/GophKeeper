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

// Ensure CardRepository implements the CardRepository interface.
var _ interfaces.CardRepository = (*CardRepository)(nil)

type CardRepository struct {
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

// NewCardRepository creates a new card repository.
func NewCardRepository(logger *zap.Logger, db *db.PostgresDb) *CardRepository {
	return &CardRepository{
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

func (r *CardRepository) Create(ctx context.Context, card *models.Card) (*models.Card, error) {
	query, args, err := sq.Insert("cards").
		Columns("user_id", "name", "card_number", "cardholder_name", "expiry_date", "cvv", "bank_name", "metadata").
		Values(card.UserID, card.Name, card.CardNumber, card.CardholderName, card.ExpiryDate, card.CVV, card.BankName, card.Metadata).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build create card query", zap.Error(err))
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
					zap.String("user_id", card.UserID),
					zap.String("name", card.Name))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", card.UserID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to create card after retries",
			zap.Error(err),
			zap.String("user_id", card.UserID),
			zap.String("name", card.Name))
		return nil, err
	}

	card.ID = id
	card.CreatedAt = createdAt
	card.UpdatedAt = updatedAt
	return card, nil
}

func (r *CardRepository) FindByID(ctx context.Context, id string) (*models.Card, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "card_number", "cardholder_name", "expiry_date", "cvv", "bank_name", "metadata", "created_at", "updated_at").
			From("cards").
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find card by ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var card models.Card
	var bankName pgtype.Text
	var metadata pgtype.Text

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&card.ID,
				&card.UserID,
				&card.Name,
				&card.CardNumber,
				&card.CardholderName,
				&card.ExpiryDate,
				&card.CVV,
				&bankName,
				&metadata,
				&card.CreatedAt,
				&card.UpdatedAt,
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
					zap.String("card_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("card_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("Card not found", zap.String("card_id", id))
			return nil, models.ErrCardNotFound
		}
		r.logger.Error("Failed to find card after retries",
			zap.Error(err),
			zap.String("card_id", id))
		return nil, err
	}

	if bankName.Valid {
		card.BankName = &bankName.String
	} else {
		card.BankName = nil
	}
	if metadata.Valid {
		card.Metadata = &metadata.String
	} else {
		card.Metadata = nil
	}
	return &card, nil
}

func (r *CardRepository) FindByUserID(ctx context.Context, userID string) ([]*models.Card, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "card_number", "cardholder_name", "expiry_date", "cvv", "bank_name", "metadata", "created_at", "updated_at").
			From("cards").
			Where(sq.Eq{"user_id": userID}).
			OrderBy("updated_at DESC"),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find cards by user ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var cards []*models.Card

	err = retry.Do(
		func() error {
			rows, err := conn.Query(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			cards = nil
			for rows.Next() {
				var card models.Card
				var bankName pgtype.Text
				var metadata pgtype.Text

				err := rows.Scan(
					&card.ID,
					&card.UserID,
					&card.Name,
					&card.CardNumber,
					&card.CardholderName,
					&card.ExpiryDate,
					&card.CVV,
					&bankName,
					&metadata,
					&card.CreatedAt,
					&card.UpdatedAt,
				)
				if err != nil {
					return err
				}

				if bankName.Valid {
					card.BankName = &bankName.String
				} else {
					card.BankName = nil
				}
				if metadata.Valid {
					card.Metadata = &metadata.String
				} else {
					card.Metadata = nil
				}

				cards = append(cards, &card)
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
		r.logger.Error("Failed to find cards after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}

	if len(cards) == 0 {
		r.logger.Debug("No cards found for user", zap.String("user_id", userID))
		return []*models.Card{}, nil
	}

	return cards, nil
}

func (r *CardRepository) Update(ctx context.Context, card *models.Card) error {
	if card == nil || card.ID == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilterToUpdate(
		sq.Update("cards").
			Set("name", card.Name).
			Set("card_number", card.CardNumber).
			Set("cardholder_name", card.CardholderName).
			Set("expiry_date", card.ExpiryDate).
			Set("cvv", card.CVV).
			Set("bank_name", card.BankName).
			Set("metadata", card.Metadata).
			Set("updated_at", time.Now()).
			Where(sq.Eq{"id": card.ID}),
	).
		Suffix("RETURNING updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update card query", zap.Error(err))
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
					zap.String("card_id", card.ID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("card_id", card.ID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Card not found for update", zap.String("card_id", card.ID))
			return models.ErrCardNotFound
		}
		r.logger.Error("Failed to update card after retries",
			zap.Error(err),
			zap.String("card_id", card.ID))
		return err
	}

	card.UpdatedAt = updatedAt
	r.logger.Info("Card updated successfully", zap.String("card_id", card.ID))
	return nil
}

func (r *CardRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := sq.Update("cards").
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(sq.Expr("deleted_at IS NULL")).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build delete card query", zap.Error(err))
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
					zap.String("card_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("card_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to delete card after retries",
			zap.Error(err),
			zap.String("card_id", id))
		return err
	}

	if affectedRows == 0 {
		r.logger.Warn("Card not found for deletion", zap.String("card_id", id))
		return models.ErrCardNotFound
	}

	r.logger.Info("Card deleted successfully",
		zap.String("card_id", id),
		zap.Int64("affected_rows", affectedRows))
	return nil
}

func (r *CardRepository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From("cards").
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists card query", zap.Error(err))
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
					zap.String("card_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("card_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to check card existence after retries",
			zap.Error(err),
			zap.String("card_id", id))
		return false, err
	}

	return exists, nil
}

func (r *CardRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From("cards").
			Where(sq.Eq{"user_id": userID}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists card by user ID query", zap.Error(err))
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
		r.logger.Error("Failed to check card existence after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return false, err
	}

	return exists, nil
}

func (r *CardRepository) Count(ctx context.Context) (int64, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*)").
			From("cards"),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build count cards query", zap.Error(err))
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
		r.logger.Error("Failed to count cards after retries", zap.Error(err))
		return 0, err
	}

	r.logger.Debug("Cards count retrieved", zap.Int64("count", count))
	return count, nil
}
