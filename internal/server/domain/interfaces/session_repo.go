package interfaces

import (
	"context"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

// SessionRepository defines the interface for session data operations.
type SessionRepository interface {
	// Create creates a new session.
	Create(ctx context.Context, session *models.Session) (*models.Session, error)

	// FindByTokenHash retrieves a session by refresh token hash.
	FindByTokenHash(ctx context.Context, tokenHash string) (*models.Session, error)

	// FindByID retrieves a session by GetID.
	FindByID(ctx context.Context, id string) (*models.Session, error)

	// FindByUserIDAndClientID retrieves an active session by user GetID and client GetID.
	FindByUserIDAndClientID(ctx context.Context, userID, clientID string) (*models.Session, error)

	// FindActiveByUserID retrieves all active sessions for a user.
	FindActiveByUserID(ctx context.Context, userID string) ([]*models.Session, error)

	// UpdateActivity updates last_used_at timestamp.
	UpdateActivity(ctx context.Context, id string) error

	// Revoke revokes a session.
	Revoke(ctx context.Context, id, reason string) error

	// RevokeByTokenHash revokes a session by token hash.
	RevokeByTokenHash(ctx context.Context, tokenHash, reason string) error

	// RevokeAllByUserID revokes all sessions for a user.
	RevokeAllByUserID(ctx context.Context, userID, reason string) error

	// RevokeAllExceptCurrent revokes all sessions except the current one.
	RevokeAllExceptCurrent(ctx context.Context, userID, currentSessionID, reason string) error

	// DeleteExpired deletes expired and revoked sessions.
	DeleteExpired(ctx context.Context, before time.Time) error

	// CountActiveByUserID counts active sessions for a user.
	CountActiveByUserID(ctx context.Context, userID string) (int, error)
}
