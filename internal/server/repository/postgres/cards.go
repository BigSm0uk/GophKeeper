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
	"github.com/samber/lo"
	"go.uber.org/zap"
)

// Ensure CardRepository implements the CardRepository interface.
var _ interfaces.CardRepository = (*CardRepository)(nil)

type CardRepository struct {
	base            *BaseRepository[*models.Card]
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

// NewCardRepository creates a new card repository.
func NewCardRepository(logger *zap.Logger, db *db.PostgresDb) *CardRepository {
	return &CardRepository{
		base:            NewBaseRepository[*models.Card](db, logger, "cards", CardScanner),
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

func CardScanner(row pgx.Row) (*models.Card, error) {
	var card models.Card
	var bankName pgtype.Text
	var metadata pgtype.Text
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
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
		&deletedAt,
	); err != nil {
		return nil, err
	}

	card.BankName = lo.Ternary(bankName.Valid, lo.ToPtr(bankName.String), nil)
	card.Metadata = lo.Ternary(metadata.Valid, lo.ToPtr(metadata.String), nil)
	card.DeletedAt = lo.Ternary(deletedAt.Valid, lo.ToPtr(deletedAt.Time), nil)

	return &card, nil
}

func (r *CardRepository) FindByID(ctx context.Context, id string) (*models.Card, error) {
	return r.base.FindByID(ctx, id)
}

func (r *CardRepository) FindByUserID(ctx context.Context, userID string) ([]*models.Card, error) {
	return r.base.FindByUserID(ctx, userID)
}

func (r *CardRepository) Delete(ctx context.Context, id string) error {
	return r.base.Delete(ctx, id)
}

func (r *CardRepository) Exists(ctx context.Context, id string) (bool, error) {
	return r.base.Exists(ctx, id)
}

func (r *CardRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	return r.base.ExistsByUserID(ctx, userID)
}

func (r *CardRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.base.CountByUserID(ctx, userID)
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
