package models

import "time"

// Session represents a user session (refresh token).
type Session struct {
	ID            string
	UserID        string
	TokenHash     string  // SHA-256 hash of refresh token
	ClientID      string  // Client identifier (stored in access token)
	IPAddress     *string // Last IP address
	UserAgent     *string // Client info
	CreatedAt     time.Time
	LastUsedAt    time.Time
	ExpiresAt     time.Time
	Revoked       bool
	RevokedAt     *time.Time
	RevokedReason *string
}

// IsExpired checks if the session is expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsActive checks if the session is active.
func (s *Session) IsActive() bool {
	return !s.Revoked && !s.IsExpired()
}

// Revoke revokes the session with a reason.
func (s *Session) Revoke(reason string) {
	s.Revoked = true
	now := time.Now()
	s.RevokedAt = &now
	s.RevokedReason = &reason
}

// UpdateActivity updates last used timestamp.
func (s *Session) UpdateActivity() {
	s.LastUsedAt = time.Now()
}
