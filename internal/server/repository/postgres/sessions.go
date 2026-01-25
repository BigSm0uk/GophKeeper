package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	pgerrors "github.com/BigSm0uk/GophKeeper/internal/server/app/db/pgerror"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	sq "github.com/Masterminds/squirrel"
	"github.com/avast/retry-go"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var _ interfaces.SessionRepository = (*SessionRepository)(nil)

type SessionRepository struct {
	logger          *zap.Logger
	db              *db.PostgresDb
	errorClassifier *pgerrors.PostgresErrorClassifier
}

func NewSessionRepository(logger *zap.Logger, db *db.PostgresDb) *SessionRepository {
	return &SessionRepository{
		logger:          logger,
		db:              db,
		errorClassifier: pgerrors.NewPostgresErrorClassifier(),
	}
}

// FindByTokenHash retrieves a session by refresh token hash
func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*models.Session, error) {
	query, args, err := sq.Select(
		"id", "user_id", "token_hash", "client_id",
		"ip_address::text", "user_agent", "created_at", "last_used_at", "expires_at",
		"revoked", "revoked_at", "revoked_reason",
	).
		From("refresh_tokens").
		Where(sq.Eq{"token_hash": tokenHash}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find session by token hash query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	session := &models.Session{}
	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&session.ID, &session.UserID, &session.TokenHash, &session.ClientID,
				&session.IPAddress, &session.UserAgent,
				&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt,
				&session.Revoked, &session.RevokedAt, &session.RevokedReason,
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
					zap.String("token_hash", tokenHash[:10]+"..."))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Session not found by token hash")
			return nil, models.ErrSessionNotFound
		}
		r.logger.Error("Failed to find session by token hash after retries", zap.Error(err))
		return nil, models.ErrSessionNotFound
	}

	return session, nil
}

// FindByID retrieves a session by ID
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*models.Session, error) {
	query, args, err := sq.Select(
		"id", "user_id", "token_hash", "client_id",
		"ip_address::text", "user_agent", "created_at", "last_used_at", "expires_at",
		"revoked", "revoked_at", "revoked_reason",
	).
		From("refresh_tokens").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find session by ID query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	session := &models.Session{}
	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&session.ID, &session.UserID, &session.TokenHash, &session.ClientID,
				&session.IPAddress, &session.UserAgent,
				&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt,
				&session.Revoked, &session.RevokedAt, &session.RevokedReason,
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
					zap.String("session_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("session_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Session not found by ID", zap.String("session_id", id))
			return nil, models.ErrSessionNotFound
		}
		r.logger.Error("Failed to find session by ID after retries",
			zap.Error(err),
			zap.String("session_id", id))
		return nil, models.ErrSessionNotFound
	}

	return session, nil
}

// FindByUserIDAndClientID retrieves an active session by user ID and client ID
func (r *SessionRepository) FindByUserIDAndClientID(ctx context.Context, userID, clientID string) (*models.Session, error) {
	query, args, err := sq.Select(
		"id", "user_id", "token_hash", "client_id",
		"ip_address::text", "user_agent", "created_at", "last_used_at", "expires_at",
		"revoked", "revoked_at", "revoked_reason",
	).
		From("refresh_tokens").
		Where(sq.And{
			sq.Eq{"user_id": userID},
			sq.Eq{"client_id": clientID},
			sq.Eq{"revoked": false},
			sq.Gt{"expires_at": time.Now()},
		}).
		OrderBy("last_used_at DESC").
		Limit(1).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find session by user_id and client_id query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	session := &models.Session{}
	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(
				&session.ID, &session.UserID, &session.TokenHash, &session.ClientID,
				&session.IPAddress, &session.UserAgent,
				&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt,
				&session.Revoked, &session.RevokedAt, &session.RevokedReason,
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
					zap.String("user_id", userID),
					zap.String("client_id", clientID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("client_id", clientID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Session not found by user_id and client_id",
				zap.String("user_id", userID),
				zap.String("client_id", clientID))
			return nil, models.ErrSessionNotFound
		}
		r.logger.Error("Failed to find session by user_id and client_id after retries",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("client_id", clientID))
		return nil, models.ErrSessionNotFound
	}

	return session, nil
}

// FindActiveByUserID retrieves all active sessions for a user
func (r *SessionRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	query, args, err := sq.Select(
		"id", "user_id", "token_hash", "client_id",
		"ip_address::text", "user_agent", "created_at", "last_used_at", "expires_at",
		"revoked", "revoked_at", "revoked_reason",
	).
		From("refresh_tokens").
		Where(sq.And{
			sq.Eq{"user_id": userID},
			sq.Eq{"revoked": false},
			sq.Gt{"expires_at": time.Now()},
		}).
		OrderBy("last_used_at DESC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build find active sessions query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var sessions []*models.Session
	err = retry.Do(
		func() error {
			rows, err := conn.Query(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			sessions = nil // Reset in case of retry
			for rows.Next() {
				session := &models.Session{}
				err := rows.Scan(
					&session.ID, &session.UserID, &session.TokenHash, &session.ClientID,
					&session.IPAddress, &session.UserAgent,
					&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt,
					&session.Revoked, &session.RevokedAt, &session.RevokedReason,
				)
				if err != nil {
					return err
				}
				sessions = append(sessions, session)
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
		r.logger.Error("Failed to find active sessions after retries",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}

	return sessions, nil
}

// UpdateActivity updates last_used_at timestamp
func (r *SessionRepository) UpdateActivity(ctx context.Context, id string) error {
	query, args, err := sq.Update("refresh_tokens").
		Set("last_used_at", time.Now()).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build update activity query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	err = retry.Do(
		func() error {
			_, err := conn.Exec(ctx, query, args...)
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
					zap.String("session_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("session_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to update session activity after retries",
			zap.Error(err),
			zap.String("session_id", id))
	}
	return err
}

// Revoke revokes a session
func (r *SessionRepository) Revoke(ctx context.Context, id, reason string) error {
	now := time.Now()
	query, args, err := sq.Update("refresh_tokens").
		Set("revoked", true).
		Set("revoked_at", now).
		Set("revoked_reason", reason).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build revoke session query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	err = retry.Do(
		func() error {
			_, err := conn.Exec(ctx, query, args...)
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
					zap.String("session_id", id))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("session_id", id))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to revoke session after retries",
			zap.Error(err),
			zap.String("session_id", id))
		return err
	}

	r.logger.Info("Session revoked", zap.String("id", id), zap.String("reason", reason))
	return nil
}

// RevokeByTokenHash revokes a session by token hash
func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash, reason string) error {
	now := time.Now()
	query, args, err := sq.Update("refresh_tokens").
		Set("revoked", true).
		Set("revoked_at", now).
		Set("revoked_reason", reason).
		Where(sq.Eq{"token_hash": tokenHash}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build revoke by token hash query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	err = retry.Do(
		func() error {
			_, err := conn.Exec(ctx, query, args...)
			return err
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err))
		}),
		retry.Context(ctx),
	)
	return err
}

// RevokeAllByUserID revokes all sessions for a user
func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID, reason string) error {
	now := time.Now()
	query, args, err := sq.Update("refresh_tokens").
		Set("revoked", true).
		Set("revoked_at", now).
		Set("revoked_reason", reason).
		Where(sq.Eq{"user_id": userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build revoke all sessions query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	err = retry.Do(
		func() error {
			_, err := conn.Exec(ctx, query, args...)
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
		r.logger.Error("Failed to revoke all sessions after retries",
			zap.Error(err),
			zap.String("user_id", userID))
	}
	return err
}

// RevokeAllExceptCurrent revokes all sessions except the current one
func (r *SessionRepository) RevokeAllExceptCurrent(ctx context.Context, userID, currentSessionID, reason string) error {
	now := time.Now()
	query, args, err := sq.Update("refresh_tokens").
		Set("revoked", true).
		Set("revoked_at", now).
		Set("revoked_reason", reason).
		Where(sq.And{
			sq.Eq{"user_id": userID},
			sq.NotEq{"id": currentSessionID},
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build revoke all except current query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	err = retry.Do(
		func() error {
			_, err := conn.Exec(ctx, query, args...)
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
					zap.String("user_id", userID),
					zap.String("current_session_id", currentSessionID))
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
		r.logger.Error("Failed to revoke all sessions except current after retries",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("current_session_id", currentSessionID))
	}
	return err
}

// DeleteExpired deletes expired and revoked sessions
func (r *SessionRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	query, args, err := sq.Delete("refresh_tokens").
		Where(sq.Or{
			sq.Lt{"expires_at": before},
			sq.And{
				sq.Eq{"revoked": true},
				sq.Lt{"revoked_at": before.AddDate(0, 0, -30)}, // keep revoked for 30 days
			},
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build delete expired sessions query", zap.Error(err))
		return err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return err
	}
	defer conn.Release()

	var rowsAffected int64
	err = retry.Do(
		func() error {
			cmdTag, err := conn.Exec(ctx, query, args...)
			if err != nil {
				return err
			}
			rowsAffected = cmdTag.RowsAffected()
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to delete expired sessions after retries", zap.Error(err))
		return err
	}

	r.logger.Info("Expired sessions deleted", zap.Int64("count", rowsAffected))
	return nil
}

// CountActiveByUserID counts active sessions for a user
func (r *SessionRepository) CountActiveByUserID(ctx context.Context, userID string) (int, error) {
	query, args, err := sq.Select("COUNT(*)").
		From("refresh_tokens").
		Where(sq.And{
			sq.Eq{"user_id": userID},
			sq.Eq{"revoked": false},
			sq.Gt{"expires_at": time.Now()},
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build count active sessions query", zap.Error(err))
		return 0, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return 0, err
	}
	defer conn.Release()

	var count int
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
		r.logger.Error("Failed to count active sessions after retries",
			zap.Error(err),
			zap.String("user_id", userID))
	}
	return count, err
}

// Create creates a new session in the database
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) (*models.Session, error) {
	query, args, err := sq.Insert("refresh_tokens").
		Columns("user_id", "token_hash", "client_id",
			"ip_address", "user_agent", "expires_at").
		Values(session.UserID, session.TokenHash, session.ClientID, session.IPAddress, session.UserAgent, session.ExpiresAt).
		Suffix("RETURNING id, created_at, last_used_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		r.logger.Error("Failed to build create session query", zap.Error(err))
		return nil, err
	}

	conn, err := r.db.GetPool().Acquire(ctx)
	if err != nil {
		r.logger.Error("Failed to acquire database connection", zap.Error(err))
		return nil, err
	}
	defer conn.Release()

	var id string
	var createdAt, lastUsedAt time.Time

	err = retry.Do(
		func() error {
			return conn.QueryRow(ctx, query, args...).Scan(&id, &createdAt, &lastUsedAt)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxDelay(1*time.Second),
		retry.RetryIf(func(err error) bool {
			classification := r.errorClassifier.Classify(err)
			if classification == pgerrors.Retriable {
				r.logger.Warn("Retriable database error occurred, will retry",
					zap.Error(err),
					zap.String("user_id", session.UserID))
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			r.logger.Info("Retrying database operation",
				zap.Uint("attempt", n+1),
				zap.Error(err),
				zap.String("user_id", session.UserID))
		}),
		retry.Context(ctx),
	)
	if err != nil {
		r.logger.Error("Failed to create session after retries",
			zap.Error(err),
			zap.String("user_id", session.UserID))
		return nil, err
	}

	session.ID = id
	session.CreatedAt = createdAt
	session.LastUsedAt = lastUsedAt

	r.logger.Info("Session created successfully",
		zap.String("session_id", id),
		zap.String("user_id", session.UserID))

	return session, nil
}

// HashToken creates SHA-256 hash of refresh token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
