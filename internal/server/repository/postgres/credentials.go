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

// Ensure CredentialsRepository implements the CredentialRepository interface
var _ interfaces.CredentialRepository = (*CredentialsRepository)(nil)

type CredentialsRepository struct {
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

func NewCredentialsRepository(logger *zap.Logger, db *db.PostgresDb) *CredentialsRepository {
	return &CredentialsRepository{
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
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
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "login", "password", "url", "metadata", "created_at", "updated_at").
			From("credentials").
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find credential by ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var credential models.Credential
	var url pgtype.Text
	var metadata pgtype.Text

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&credential.ID,
				&credential.UserID,
				&credential.Name,
				&credential.Login,
				&credential.Password,
				&url,
				&metadata,
				&credential.CreatedAt,
				&credential.UpdatedAt,
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
					zap.String("credential_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("credential_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("Credential not found", zap.String("credential_id", id))
			return nil, models.ErrCredentialNotFound
		}
		r.logger.Error("Failed to find credential after retries",
			zap.Error(err),
			zap.String("credential_id", id))
		return nil, err
	}

	// Convert pgtype.Text to *string
	if url.Valid {
		credential.URL = &url.String
	} else {
		credential.URL = nil
	}
	if metadata.Valid {
		credential.Metadata = &metadata.String
	} else {
		credential.Metadata = nil
	}
	return &credential, nil
}

func (r *CredentialsRepository) FindByUserId(ctx context.Context, id string) ([]*models.Credential, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("id", "user_id", "name", "login", "password", "url", "metadata", "created_at", "updated_at").
			From("credentials").
			Where(sq.Eq{"user_id": id}).
			OrderBy("updated_at DESC"),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find credentials by user ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var credentials []*models.Credential

	err = retry.Do(
		func() error {
			rows, err := conn.Query(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			credentials = nil // Reset in case of retry
			for rows.Next() {
				var credential models.Credential
				var url pgtype.Text
				var metadata pgtype.Text

				err := rows.Scan(
					&credential.ID,
					&credential.UserID,
					&credential.Name,
					&credential.Login,
					&credential.Password,
					&url,
					&metadata,
					&credential.CreatedAt,
					&credential.UpdatedAt,
				)
				if err != nil {
					return err
				}

				// Convert pgtype.Text to *string
				if url.Valid {
					credential.URL = &url.String
				} else {
					credential.URL = nil
				}
				if metadata.Valid {
					credential.Metadata = &metadata.String
				} else {
					credential.Metadata = nil
				}

				credentials = append(credentials, &credential)
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
					zap.String("user_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to find credentials after retries",
			zap.Error(err),
			zap.String("user_id", id))
		return nil, err
	}

	if len(credentials) == 0 {
		r.logger.Debug("No credentials found for user", zap.String("user_id", id))
		return []*models.Credential{}, nil
	}

	return credentials, nil
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
	if id == "" {
		return models.ErrInvalidUserID
	}

	// Soft delete: set deleted_at timestamp instead of physical deletion
	query, args, err := sq.Update("credentials").
		Set("deleted_at", time.Now()).
		Where(sq.Eq{"id": id}).
		Where(sq.Expr("deleted_at IS NULL")).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build delete credential query", zap.Error(err))
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
					zap.String("credential_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("credential_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to delete credential after retries",
			zap.Error(err),
			zap.String("credential_id", id))
		return err
	}

	if affectedRows == 0 {
		r.logger.Warn("Credential not found for deletion", zap.String("credential_id", id))
		return models.ErrCredentialNotFound
	}

	r.logger.Info("Credential deleted successfully",
		zap.String("credential_id", id),
		zap.Int64("affected_rows", affectedRows))
	return nil
}

func (r *CredentialsRepository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From("credentials").
			Where(sq.Eq{"id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists credential query", zap.Error(err))
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
			err := conn.QueryRow(ctx, query, args...).Scan(&exists)
			if err == nil {
				return nil
			}
			return err
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("credential_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("credential_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to check credential existence after retries",
			zap.Error(err),
			zap.String("credential_id", id))
		return false, err
	}

	return exists, nil
}

func (r *CredentialsRepository) ExistsByUserId(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*) > 0").
			From("credentials").
			Where(sq.Eq{"user_id": id}),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists credential by user ID query", zap.Error(err))
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
			err := conn.QueryRow(ctx, query, args...).Scan(&exists)
			if err == nil {
				return nil
			}
			return err
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("user_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to check credential existence after retries",
			zap.Error(err),
			zap.String("user_id", id))
		return false, err
	}

	return exists, nil
}

func (r *CredentialsRepository) Count(ctx context.Context) (int64, error) {
	query, args, err := applySoftDeleteFilter(
		sq.Select("COUNT(*)").
			From("credentials"),
	).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build count credentials query", zap.Error(err))
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
		r.logger.Error("Failed to count credentials after retries", zap.Error(err))
		return 0, err
	}

	r.logger.Debug("Credentials count retrieved", zap.Int64("count", count))
	return count, nil
}
