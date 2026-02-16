package interfaces

import (
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
)

type TokenService interface {
	GenerateAccessToken(user *models.User, clientID string) (string, error)
	GenerateRefreshToken(user *models.User) (string, error)
	ValidateAccessToken(tokenString string) (*models.JWTClaims, error)
	ParseToken(tokenString string) (*models.JWTClaims, error)
	ExtractUserID(tokenString string) (string, error)
	ExtractClientID(tokenString string) (string, error)
	ExtractUsername(tokenString string) (string, error)
	IsTokenExpired(tokenString string) bool
	RefreshAccessToken(refreshToken string) (string, error)
	GetRefreshTokenTTL() time.Duration
}
