package service

import (
	"context"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/repository/postgres"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/util"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// MaxSessionsPerUser is the maximum number of active sessions per user
	MaxSessionsPerUser = 5
)

type AuthService struct {
	logger     *zap.Logger
	ur         interfaces.UserRepository
	sr         interfaces.SessionRepository
	jwtService *JWTService
}

func NewAuthService(logger *zap.Logger, ur interfaces.UserRepository, jwtService *JWTService, sr interfaces.SessionRepository) *AuthService {
	return &AuthService{
		logger:     logger,
		ur:         ur,
		jwtService: jwtService,
		sr:         sr,
	}
}

func (as *AuthService) Register(ctx context.Context, user entity.User) (*entity.User, error) {
	exists, err := as.ur.ExistsByUsername(ctx, user.Username)
	if err != nil {
		as.logger.Error("Failed to check username existence", zap.Error(err))
		return nil, err
	}
	if exists {
		as.logger.Warn("Registration attempt for existing username",
			zap.String("username", user.Username))
		return nil, models.ErrUserAlreadyExists
	}

	hashedPassword, err := util.HashPassword(user.Password)
	if err != nil {
		as.logger.Error("Failed to hash password during registration",
			zap.Error(err),
			zap.String("username", user.Username))
		return nil, err
	}

	domainUser := &models.User{
		Username:       user.Username,
		Email:          user.Email,
		HashedPassword: hashedPassword,
	}

	createdUser, err := as.ur.Create(ctx, domainUser)
	if err != nil {
		as.logger.Error("Failed to persist user during registration",
			zap.Error(err),
			zap.String("username", user.Username))
		return nil, err
	}

	as.logger.Info("User registered successfully",
		zap.String("user_id", createdUser.ID),
		zap.String("username", createdUser.Username))

	result := &entity.User{
		ID:        createdUser.ID,
		Username:  createdUser.Username,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
	}

	return result, nil
}

// Login authenticates a user and returns access/refresh tokens
func (as *AuthService) Login(ctx context.Context, username, password, ipAddress, userAgent string) (accessToken string, err error) {
	// 1. Verify credentials
	user, err := as.ur.FindByUsername(ctx, username)
	if err != nil {
		as.logger.Error("Failed to get user by username during login",
			zap.Error(err),
			zap.String("username", username))
		return "", models.ErrInvalidCredentials
	}

	valid, err := util.VerifyPassword(password, user.HashedPassword)
	if err != nil {
		as.logger.Error("Failed to verify password",
			zap.Error(err),
			zap.String("username", username))
		return "", models.ErrInvalidCredentials
	}
	if !valid {
		as.logger.Warn("Invalid password during login",
			zap.String("username", username))
		return "", models.ErrInvalidCredentials
	}

	// 2. Check session limit
	count, err := as.sr.CountActiveByUserID(ctx, user.ID)
	if err != nil {
		as.logger.Warn("Failed to count active sessions", zap.Error(err))
		// Continue anyway, non-critical
	} else if count >= MaxSessionsPerUser {
		// Revoke oldest session
		sessions, err := as.sr.FindActiveByUserID(ctx, user.ID)
		if err == nil && len(sessions) > 0 {
			// Sessions are ordered by last_used_at DESC, so revoke the last one
			oldestSession := sessions[len(sessions)-1]
			_ = as.sr.Revoke(ctx, oldestSession.ID, "session_limit_exceeded")
			as.logger.Info("Revoked oldest session due to limit",
				zap.String("user_id", user.ID),
				zap.String("session_id", oldestSession.ID))
		}
	}

	// 3. Generate client_id for this session
	clientID := uuid.New().String()

	// 4. Generate refresh token (stored only in DB)
	refreshToken, err := as.jwtService.GenerateRefreshToken(user)
	if err != nil {
		as.logger.Error("Failed to generate refresh token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		return "", err
	}

	// 5. Save session to database
	var ipAddressPtr *string
	if ipAddress != "" {
		ipAddressPtr = &ipAddress
	}
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}

	session := &models.Session{
		UserID:    user.ID,
		TokenHash: postgres.HashToken(refreshToken),
		ClientID:  clientID,
		IPAddress: ipAddressPtr,
		UserAgent: userAgentPtr,
		ExpiresAt: time.Now().Add(as.jwtService.config.RefreshTokenTTL),
		Revoked:   false,
	}

	_, err = as.sr.Create(ctx, session)
	if err != nil {
		as.logger.Error("Failed to save session",
			zap.Error(err),
			zap.String("user_id", user.ID))
		// Return error, session save is critical
		return "", err
	}

	// 6. Generate access token with client_id
	accessToken, err = as.jwtService.GenerateAccessToken(user, clientID)
	if err != nil {
		as.logger.Error("Failed to generate access token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		// Revoke session if token generation fails
		_ = as.sr.Revoke(ctx, session.ID, "token_generation_failed")
		return "", err
	}

	as.logger.Info("Session created",
		zap.String("session_id", session.ID),
		zap.String("user_id", user.ID),
		zap.String("client_id", clientID))

	as.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID),
		zap.String("username", user.Username))

	// Return only access token, refresh token is stored in DB
	return accessToken, nil
}

// Token handles OAuth2 token requests (password grant and refresh token grant)
func (as *AuthService) Token(ctx context.Context, grantType, username, password, accessTokenForRefresh, ipAddress, userAgent string) (accessToken string, err error) {
	switch grantType {
	case "password":
		return as.Login(ctx, username, password, ipAddress, userAgent)
	case "refresh_token":
		// 1. Extract user_id and client_id from access token
		userID, err := as.jwtService.ExtractUserID(accessTokenForRefresh)
		if err != nil {
			as.logger.Error("Failed to extract user ID from access token",
				zap.Error(err))
			return "", models.ErrInvalidToken
		}

		clientID, err := as.jwtService.ExtractClientID(accessTokenForRefresh)
		if err != nil || clientID == "" {
			as.logger.Error("Failed to extract client ID from access token",
				zap.Error(err))
			return "", models.ErrInvalidToken
		}

		// 2. Find session by user_id and client_id
		session, err := as.sr.FindByUserIDAndClientID(ctx, userID, clientID)
		if err != nil {
			as.logger.Error("Failed to find session by user_id and client_id",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("client_id", clientID))
			return "", models.ErrSessionNotFound
		}

		// 3. Check if session is expired
		if session.IsExpired() {
			as.logger.Warn("Session expired",
				zap.String("session_id", session.ID),
				zap.Time("expires_at", session.ExpiresAt))
			return "", models.ErrSessionExpired
		}

		// 4. Check if session is revoked
		if session.Revoked {
			as.logger.Warn("Session revoked",
				zap.String("session_id", session.ID),
				zap.Stringp("reason", session.RevokedReason))
			return "", models.ErrSessionRevoked
		}

		// 5. Get user
		user, err := as.ur.FindByID(ctx, session.UserID)
		if err != nil {
			as.logger.Error("Failed to get user by ID during token refresh",
				zap.Error(err),
				zap.String("user_id", session.UserID))
			return "", err
		}

		// 6. Update session activity
		err = as.sr.UpdateActivity(ctx, session.ID)
		if err != nil {
			as.logger.Warn("Failed to update session activity", zap.Error(err))
			// Continue anyway, non-critical
		}

		// 7. Generate new refresh token (token rotation)
		newRefreshToken, err := as.jwtService.GenerateRefreshToken(user)
		if err != nil {
			as.logger.Error("Failed to generate new refresh token",
				zap.Error(err),
				zap.String("user_id", user.ID))
			return "", err
		}

		// 8. Update session with new token hash (token rotation)
		newTokenHash := postgres.HashToken(newRefreshToken)
		_ = as.sr.Revoke(ctx, session.ID, "token_rotated")

		var ipAddressPtr *string
		if ipAddress != "" {
			ipAddressPtr = &ipAddress
		}
		var userAgentPtr *string
		if userAgent != "" {
			userAgentPtr = &userAgent
		}
		newSession := &models.Session{
			UserID:    user.ID,
			TokenHash: newTokenHash,
			ClientID:  session.ClientID, // Keep same client_id
			IPAddress: ipAddressPtr,
			UserAgent: userAgentPtr,
			ExpiresAt: time.Now().Add(as.jwtService.config.RefreshTokenTTL),
			Revoked:   false,
		}
		_, err = as.sr.Create(ctx, newSession)
		if err != nil {
			as.logger.Warn("Failed to create rotated session", zap.Error(err))
			// Continue anyway
		}

		// 9. Generate new access token with same client_id
		newAccessToken, err := as.jwtService.GenerateAccessToken(user, session.ClientID)
		if err != nil {
			as.logger.Error("Failed to generate new access token",
				zap.Error(err),
				zap.String("user_id", user.ID))
			return "", err
		}

		as.logger.Info("Token refreshed successfully",
			zap.String("user_id", user.ID),
			zap.String("session_id", session.ID),
			zap.String("client_id", clientID))

		// Return only access token, refresh token is stored in DB
		return newAccessToken, nil
	default:
		return "", models.ErrInvalidGrantType
	}
}

// ValidateToken validates an access token and returns user information
func (as *AuthService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	userID, err := as.jwtService.ExtractUserID(token)
	if err != nil {
		as.logger.Error("Failed to extract user ID from token",
			zap.Error(err))
		return nil, models.ErrInvalidToken
	}

	user, err := as.ur.FindByID(ctx, userID)
	if err != nil {
		as.logger.Error("Failed to get user by ID during token validation",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}

	return user, nil
}
