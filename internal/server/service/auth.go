package service

import (
	"context"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/util"
	"go.uber.org/zap"
)

type AuthService struct {
	logger     *zap.Logger
	ur         interfaces.UserRepository
	jwtService *JWTService
}

func NewAuthService(logger *zap.Logger, ur interfaces.UserRepository, jwtService *JWTService) *AuthService {
	return &AuthService{
		logger:     logger,
		ur:         ur,
		jwtService: jwtService,
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
func (as *AuthService) Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error) {
	user, err := as.ur.FindByUsername(ctx, username)
	if err != nil {
		as.logger.Error("Failed to get user by username during login",
			zap.Error(err),
			zap.String("username", username))
		return "", "", models.ErrInvalidCredentials
	}

	valid, err := util.VerifyPassword(password, user.HashedPassword)
	if err != nil {
		as.logger.Error("Failed to verify password",
			zap.Error(err),
			zap.String("username", username))
		return "", "", models.ErrInvalidCredentials
	}
	if !valid {
		as.logger.Warn("Invalid password during login",
			zap.String("username", username))
		return "", "", models.ErrInvalidCredentials
	}

	accessToken, err = as.jwtService.GenerateAccessToken(user)
	if err != nil {
		as.logger.Error("Failed to generate access token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		return "", "", err
	}

	refreshToken, err = as.jwtService.GenerateRefreshToken(user)
	if err != nil {
		as.logger.Error("Failed to generate refresh token",
			zap.Error(err),
			zap.String("user_id", user.ID))
		return "", "", err
	}

	as.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID),
		zap.String("username", user.Username))

	return accessToken, refreshToken, nil
}

// Token handles OAuth2 token requests (password grant and refresh token grant)
func (as *AuthService) Token(ctx context.Context, grantType, username, password, refreshToken string) (accessToken, refreshTokenOut string, err error) {
	switch grantType {
	case "password":
		return as.Login(ctx, username, password)
	case "refresh_token":
		newAccessToken, err := as.jwtService.RefreshAccessToken(refreshToken)
		if err != nil {
			as.logger.Error("Failed to refresh access token",
				zap.Error(err))
			return "", "", models.ErrInvalidCredentials
		}

		userID, err := as.jwtService.ExtractUserID(refreshToken)
		if err != nil {
			as.logger.Error("Failed to extract user ID from refresh token",
				zap.Error(err))
			return "", "", models.ErrInvalidCredentials
		}

		user, err := as.ur.FindByID(ctx, userID)
		if err != nil {
			as.logger.Error("Failed to get user by ID during token refresh",
				zap.Error(err),
				zap.String("user_id", userID))
			return "", "", err
		}

		newRefreshToken, err := as.jwtService.GenerateRefreshToken(user)
		if err != nil {
			as.logger.Error("Failed to generate new refresh token",
				zap.Error(err),
				zap.String("user_id", user.ID))
			return "", "", err
		}

		as.logger.Info("Token refreshed successfully",
			zap.String("user_id", user.ID))

		return newAccessToken, newRefreshToken, nil
	default:
		return "", "", models.ErrInvalidGrantType
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
