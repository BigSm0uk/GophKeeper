package service

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/golang-jwt/jwt/v5"
)

// JWTService handles JWT token operations
type JWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	config     config.JWTConfig
}

var _ interfaces.TokenService = (*JWTService)(nil)

// NewJWTService creates a new JWT service
func NewJWTService(cfg config.JWTConfig) (*JWTService, error) {
	privateKeyData, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKeyData, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &JWTService{
		privateKey: privateKey,
		publicKey:  publicKey,
		config:     cfg,
	}, nil
}

// GenerateAccessToken generates an access JWT token for a user
func (s *JWTService) GenerateAccessToken(user *models.User, clientID string) (string, error) {
	if user == nil || user.ID == "" || clientID == "" {
		return "", errors.New("invalid use or client id")
	}

	now := time.Now()
	claims := models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a refresh JWT token for a user
func (s *JWTService) GenerateRefreshToken(user *models.User) (string, error) {
	if user == nil || user.ID == "" {
		return "", errors.New("invalid user")
	}

	now := time.Now()
	claims := models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*models.JWTClaims, error) {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.ExpiresAt.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}

// ParseToken parses a JWT token and returns the claims
func (s *JWTService) ParseToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&models.JWTClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.publicKey, nil
		},
		jwt.WithoutClaimsValidation(),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}

	return claims, nil
}

// ExtractUserID extracts user ID from a valid token
func (s *JWTService) ExtractUserID(tokenString string) (string, error) {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// ExtractClientID extracts client ID from a valid token
func (s *JWTService) ExtractClientID(tokenString string) (string, error) {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.ClientID, nil
}

// ExtractUsername extracts username from a valid token
func (s *JWTService) ExtractUsername(tokenString string) (string, error) {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Username, nil
}

// IsTokenExpired checks if a token is expired
func (s *JWTService) IsTokenExpired(tokenString string) bool {
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return true
	}

	return claims.ExpiresAt.Before(time.Now())
}

// RefreshAccessToken creates a new access token from a valid refresh token
func (s *JWTService) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := s.ValidateAccessToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	user := &models.User{
		ID:       claims.UserID,
		Username: claims.Username,
	}

	return s.GenerateAccessToken(user, claims.ClientID)
}
func (s *JWTService) GetRefreshTokenTTL() time.Duration {
	return s.config.RefreshTokenTTL
}
