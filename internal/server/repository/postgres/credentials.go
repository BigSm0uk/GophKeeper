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

// Ensure CredentialsRepository implements the CredentialRepository interface
var _ interfaces.CredentialRepository = (*CredentialsRepository)(nil)

type CredentialsRepository struct {
	base            *BaseRepository[*models.Credential]
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

func NewCredentialsRepository(logger *zap.Logger, db *db.PostgresDb) *CredentialsRepository {
	return &CredentialsRepository{
		base:            NewBaseRepository[*models.Credential](db, logger, "credentials", CredentialScanner),
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

func CredentialScanner(row pgx.Row) (*models.Credential, error) {
	var credential models.Credential
	var url pgtype.Text
	var metadata pgtype.Text
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&credential.ID,
		&credential.UserID,
		&credential.Name,
		&credential.Login,
		&credential.Password,
		&url,
		&metadata,
		&credential.CreatedAt,
		&credential.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}

	credential.URL = lo.Ternary(url.Valid, lo.ToPtr(url.String), nil)
	credential.Metadata = lo.Ternary(metadata.Valid, lo.ToPtr(metadata.String), nil)
	credential.DeletedAt = lo.Ternary(deletedAt.Valid, lo.ToPtr(deletedAt.Time), nil)

	return &credential, nil
}

func (r *CredentialsRepository) Create(ctx context.Context, credential *models.Credential) (*models.Credential, error) {
	query, args, err := sq.Insert("credentials").
		Columns("user_id", "name", "login", "password", "url", "metadata").
		Values(credential.UserID, credential.Name, credential.Login, credential.Password, credential.URL, credential.Metadata).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build create credential query", zap.Error(err))
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
					zap.String("user_id", credential.UserID),
					zap.String("name", credential.Name))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", credential.UserID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to create credential after retries",
			zap.Error(err),
			zap.String("user_id", credential.UserID),
			zap.String("name", credential.Name))
		return nil, err
	}

	credential.ID = id
	credential.CreatedAt = createdAt
	credential.UpdatedAt = updatedAt
	return credential, nil
}

func (r *CredentialsRepository) FindByID(ctx context.Context, id string) (*models.Credential, error) {
	credential, err := r.base.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("Credential not found", zap.String("credential_id", id))
			return nil, models.ErrCredentialNotFound
		}
		return nil, err
	}
	return credential, nil
}

func (r *CredentialsRepository) FindByUserId(ctx context.Context, id string, limit, offset int) ([]*models.Credential, error) {
	return r.base.FindByUserID(ctx, id, limit, offset)
}

func (r *CredentialsRepository) Update(ctx context.Context, credential *models.Credential) error {
	if credential == nil || credential.ID == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilterToUpdate(
		sq.Update("credentials").
			Set("name", credential.Name).
			Set("login", credential.Login).
			Set("password", credential.Password).
			Set("url", credential.URL).
			Set("metadata", credential.Metadata).
			Set("updated_at", time.Now()).
			Where(sq.Eq{"id": credential.ID}),
	).
		Suffix("RETURNING updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update credential query", zap.Error(err))
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
					zap.String("credential_id", credential.ID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("credential_id", credential.ID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Credential not found for update", zap.String("credential_id", credential.ID))
			return models.ErrCredentialNotFound
		}
		r.logger.Error("Failed to update credential after retries",
			zap.Error(err),
			zap.String("credential_id", credential.ID))
		return err
	}

	credential.UpdatedAt = updatedAt
	r.logger.Info("Credential updated successfully", zap.String("credential_id", credential.ID))
	return nil
}

func (r *CredentialsRepository) Delete(ctx context.Context, id string) error {
	err := r.base.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Credential not found for deletion", zap.String("credential_id", id))
			return models.ErrCredentialNotFound
		}
		return err
	}
	r.logger.Info("Credential deleted successfully", zap.String("credential_id", id))
	return nil
}

func (r *CredentialsRepository) Exists(ctx context.Context, id string) (bool, error) {
	return r.base.Exists(ctx, id)
}

func (r *CredentialsRepository) ExistsByUserId(ctx context.Context, userID string) (bool, error) {
	return r.base.ExistsByUserID(ctx, userID)
}

func (r *CredentialsRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.base.CountByUserID(ctx, userID)
}
