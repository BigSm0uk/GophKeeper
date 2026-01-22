package postgres

import (
	"context"
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

// Ensure UserRepository implements the UserRepository interface
var _ interfaces.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

func NewUserRepository(logger *zap.Logger, db *db.PostgresDb) *UserRepository {
	return &UserRepository{
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	query, args, err := sq.Insert("users").
		Columns("username", "email", "hashed_password").
		Values(user.Username, user.Email, user.HashedPassword).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build create user query", zap.Error(err))
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
					zap.String("username", user.Username))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("username", user.Username))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		r.logger.Error("Failed to create user after retries",
			zap.Error(err),
			zap.String("username", user.Username))
		return nil, err
	}

	user.ID = id
	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	query, args, err := sq.Select("id", "username", "email", "hashed_password", "created_at", "updated_at").
		From("users").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find user by ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var user models.User
	var email pgtype.Text
	var hashedPassword string

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&user.ID,
				&user.Username,
				&email,
				&hashedPassword,
				&user.CreatedAt,
				&user.UpdatedAt,
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
		if err == pgx.ErrNoRows {
			r.logger.Info("User not found", zap.String("user_id", id))
			return nil, models.ErrUserNotFound
		}
		r.logger.Error("Failed to find user after retries",
			zap.Error(err),
			zap.String("user_id", id))
		return nil, err
	}

	// Convert pgtype.Text to *string
	if email.Valid {
		user.Email = &email.String
	} else {
		user.Email = nil
	}
	user.HashedPassword = hashedPassword
	return &user, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query, args, err := sq.Select("id", "username", "email", "hashed_password", "created_at", "updated_at").
		From("users").
		Where(sq.Eq{"username": username}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find user by username query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var user models.User
	var email pgtype.Text
	var hashedPassword string

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&user.ID,
				&user.Username,
				&email,
				&hashedPassword,
				&user.CreatedAt,
				&user.UpdatedAt,
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
					zap.String("username", username))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("username", username))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			r.logger.Info("User not found", zap.String("username", username))
			return nil, models.ErrUserNotFound
		}
		r.logger.Error("Failed to find user after retries",
			zap.Error(err),
			zap.String("username", username))
		return nil, err
	}

	// Convert pgtype.Text to *string
	if email.Valid {
		user.Email = &email.String
	} else {
		user.Email = nil
	}
	user.HashedPassword = hashedPassword
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	if user == nil || user.ID == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := sq.Update("users").
		Set("username", user.Username).
		Set("email", user.Email).
		Set("hashed_password", user.HashedPassword).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": user.ID}).
		Suffix("RETURNING updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update user query", zap.Error(err))
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
					zap.String("user_id", user.ID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", user.ID))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		r.logger.Error("Failed to update user after retries",
			zap.Error(err),
			zap.String("user_id", user.ID))
		return err
	}

	user.UpdatedAt = updatedAt
	r.logger.Info("User updated successfully", zap.String("user_id", user.ID))
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return models.ErrInvalidUserID
	}

	query, args, err := sq.Delete("users").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build delete user query", zap.Error(err))
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
		r.logger.Error("Failed to delete user after retries",
			zap.Error(err),
			zap.String("user_id", id))
		return err
	}

	if affectedRows == 0 {
		r.logger.Warn("User not found for deletion", zap.String("user_id", id))
		return models.ErrUserNotFound
	}

	r.logger.Info("User deleted successfully",
		zap.String("user_id", id),
		zap.Int64("affected_rows", affectedRows))
	return nil
}

func (r *UserRepository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, models.ErrInvalidUserID
	}

	query, args, err := sq.Select("COUNT(*) > 0").
		From("users").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists user query", zap.Error(err))
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
		r.logger.Error("Failed to check user existence after retries",
			zap.Error(err),
			zap.String("user_id", id))
		return false, err
	}

	return exists, nil
}
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if username == "" {
		return false, models.ErrInvalidUsername
	}

	query, args, err := sq.Select("COUNT(*) > 0").
		From("users").
		Where(sq.Eq{"username": username}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build exists user query", zap.Error(err))
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
					zap.String("username", username))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("username", username))
		}),
		retry.Context(ctx),
	)

	if err != nil {
		r.logger.Error("Failed to check user existence after retries",
			zap.Error(err),
			zap.String("username", username))
		return false, err
	}

	return exists, nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	query, args, err := sq.Select("COUNT(*)").
		From("users").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build count users query", zap.Error(err))
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
		r.logger.Error("Failed to count users after retries", zap.Error(err))
		return 0, err
	}

	r.logger.Debug("Users count retrieved", zap.Int64("count", count))
	return count, nil
}
