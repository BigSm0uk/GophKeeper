package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupJWTServiceWithKeys creates temp RSA keys, writes PEM files, and returns JWTService and the private key
// (private key is needed for tests that build custom tokens, e.g. expired).
func setupJWTServiceWithKeys(t *testing.T) (*JWTService, *rsa.PrivateKey, config.JWTConfig) {
	t.Helper()
	dir := t.TempDir()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	privatePath := filepath.Join(dir, "private.pem")
	require.NoError(t, os.WriteFile(privatePath, privatePEM, 0600))

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicDER,
	})
	publicPath := filepath.Join(dir, "public.pem")
	require.NoError(t, os.WriteFile(publicPath, publicPEM, 0600))

	cfg := config.JWTConfig{
		PrivateKeyPath:  privatePath,
		PublicKeyPath:   publicPath,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	svc, err := NewJWTService(cfg)
	require.NoError(t, err)
	return svc, privateKey, cfg
}

func setupJWTService(t *testing.T) *JWTService {
	svc, _, _ := setupJWTServiceWithKeys(t)
	return svc
}

func jwtTestUser() *models.User {
	email := "u@example.com"
	return &models.User{
		ID:       "user-1",
		Username: "testuser",
		Email:    &email,
	}
}

// --- NewJWTService ---

func TestNewJWTService_Success(t *testing.T) {
	svc, _, cfg := setupJWTServiceWithKeys(t)
	require.NotNil(t, svc)
	assert.Equal(t, cfg.RefreshTokenTTL, svc.GetRefreshTokenTTL())
}

func TestNewJWTService_PrivateKeyFileMissing(t *testing.T) {
	dir := t.TempDir()
	cfg := config.JWTConfig{
		PrivateKeyPath:  filepath.Join(dir, "nonexistent.pem"),
		PublicKeyPath:   filepath.Join(dir, "public.pem"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test",
	}
	// Create public key so we fail on private first
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicDER, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, os.WriteFile(cfg.PublicKeyPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0600))

	svc, err := NewJWTService(cfg)
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "failed to read private key")
}

func TestNewJWTService_PrivateKeyInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	require.NoError(t, os.WriteFile(privatePath, []byte("not pem"), 0600))
	publicPath := filepath.Join(dir, "public.pem")
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicDER, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0600))

	cfg := config.JWTConfig{
		PrivateKeyPath:  privatePath,
		PublicKeyPath:   publicPath,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test",
	}
	svc, err := NewJWTService(cfg)
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestNewJWTService_PublicKeyFileMissing(t *testing.T) {
	_, privateKey, _ := setupJWTServiceWithKeys(t)
	dir := t.TempDir()
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	privatePath := filepath.Join(dir, "private.pem")
	require.NoError(t, os.WriteFile(privatePath, privatePEM, 0600))
	cfg := config.JWTConfig{
		PrivateKeyPath:  privatePath,
		PublicKeyPath:   filepath.Join(dir, "nonexistent.pem"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test",
	}
	svc, err := NewJWTService(cfg)
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "failed to read public key")
}

// --- GenerateAccessToken ---

func TestJWTService_GenerateAccessToken_Success(t *testing.T) {
	svc := setupJWTService(t)
	user := jwtTestUser()
	clientID := "client-1"

	token, err := svc.GenerateAccessToken(user, clientID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, clientID, claims.ClientID)
	assert.Equal(t, "test-issuer", claims.Issuer)
}

func TestJWTService_GenerateAccessToken_NilUser(t *testing.T) {
	svc := setupJWTService(t)

	token, err := svc.GenerateAccessToken(nil, "client-1")
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid")
}

func TestJWTService_GenerateAccessToken_EmptyUserID(t *testing.T) {
	svc := setupJWTService(t)
	user := jwtTestUser()
	user.ID = ""

	token, err := svc.GenerateAccessToken(user, "client-1")
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid")
}

func TestJWTService_GenerateAccessToken_EmptyClientID(t *testing.T) {
	svc := setupJWTService(t)
	user := jwtTestUser()

	token, err := svc.GenerateAccessToken(user, "")
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid")
}

// --- GenerateRefreshToken ---

func TestJWTService_GenerateRefreshToken_Success(t *testing.T) {
	svc := setupJWTService(t)
	user := jwtTestUser()

	token, err := svc.GenerateRefreshToken(user)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Empty(t, claims.ClientID)
}

func TestJWTService_GenerateRefreshToken_NilUser(t *testing.T) {
	svc := setupJWTService(t)

	token, err := svc.GenerateRefreshToken(nil)
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid")
}

func TestJWTService_GenerateRefreshToken_EmptyUserID(t *testing.T) {
	svc := setupJWTService(t)
	user := jwtTestUser()
	user.ID = ""

	token, err := svc.GenerateRefreshToken(user)
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid")
}

// --- ParseToken ---

func TestJWTService_ParseToken_Success(t *testing.T) {
	svc := setupJWTService(t)
	user := jwtTestUser()
	tokenStr, err := svc.GenerateAccessToken(user, "client-1")
	require.NoError(t, err)

	claims, err := svc.ParseToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "client-1", claims.ClientID)
}

func TestJWTService_ParseToken_InvalidString(t *testing.T) {
	svc := setupJWTService(t)

	claims, err := svc.ParseToken("invalid.jwt.here")
	require.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ParseToken_WrongSignature(t *testing.T) {
	svc := setupJWTService(t)
	// Create token signed with another key
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := models.JWTClaims{
		UserID:   "user-1",
		Username: "user",
		ClientID: "c1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test",
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(otherKey)
	require.NoError(t, err)

	parsed, err := svc.ParseToken(tokenStr)
	require.Error(t, err)
	assert.Nil(t, parsed)
}

// --- ValidateAccessToken ---

func TestJWTService_ValidateAccessToken_Success(t *testing.T) {
	svc := setupJWTService(t)
	tokenStr, err := svc.GenerateAccessToken(jwtTestUser(), "client-1")
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-1", claims.UserID)
	assert.False(t, claims.ExpiresAt.Before(time.Now()))
}

func TestJWTService_ValidateAccessToken_Expired(t *testing.T) {
	svc, privateKey, cfg := setupJWTServiceWithKeys(t)
	now := time.Now()
	claims := models.JWTClaims{
		UserID:   "user-1",
		Username: "user",
		ClientID: "c1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(privateKey)
	require.NoError(t, err)

	parsed, err := svc.ValidateAccessToken(tokenStr)
	require.Error(t, err)
	assert.Nil(t, parsed)
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestJWTService_ValidateAccessToken_InvalidToken(t *testing.T) {
	svc := setupJWTService(t)

	claims, err := svc.ValidateAccessToken("bad.token.here")
	require.Error(t, err)
	assert.Nil(t, claims)
}

// --- ExtractUserID, ExtractClientID, ExtractUsername ---

func TestJWTService_ExtractUserID_Success(t *testing.T) {
	svc := setupJWTService(t)
	tokenStr, _ := svc.GenerateAccessToken(jwtTestUser(), "client-1")

	userID, err := svc.ExtractUserID(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestJWTService_ExtractUserID_InvalidToken(t *testing.T) {
	svc := setupJWTService(t)

	userID, err := svc.ExtractUserID("invalid")
	require.Error(t, err)
	assert.Empty(t, userID)
}

func TestJWTService_ExtractClientID_Success(t *testing.T) {
	svc := setupJWTService(t)
	tokenStr, _ := svc.GenerateAccessToken(jwtTestUser(), "my-client")

	clientID, err := svc.ExtractClientID(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "my-client", clientID)
}

func TestJWTService_ExtractClientID_InvalidToken(t *testing.T) {
	svc := setupJWTService(t)

	clientID, err := svc.ExtractClientID("invalid")
	require.Error(t, err)
	assert.Empty(t, clientID)
}

func TestJWTService_ExtractUsername_Success(t *testing.T) {
	svc := setupJWTService(t)
	tokenStr, _ := svc.GenerateAccessToken(jwtTestUser(), "c1")

	username, err := svc.ExtractUsername(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "testuser", username)
}

func TestJWTService_ExtractUsername_InvalidToken(t *testing.T) {
	svc := setupJWTService(t)

	username, err := svc.ExtractUsername("invalid")
	require.Error(t, err)
	assert.Empty(t, username)
}

// --- IsTokenExpired ---

func TestJWTService_IsTokenExpired_ValidToken(t *testing.T) {
	svc := setupJWTService(t)
	tokenStr, _ := svc.GenerateAccessToken(jwtTestUser(), "c1")

	expired := svc.IsTokenExpired(tokenStr)
	assert.False(t, expired)
}

func TestJWTService_IsTokenExpired_ExpiredToken(t *testing.T) {
	svc, privateKey, cfg := setupJWTServiceWithKeys(t)
	now := time.Now()
	claims := models.JWTClaims{
		UserID:   "user-1",
		Username: "user",
		ClientID: "c1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, _ := token.SignedString(privateKey)

	expired := svc.IsTokenExpired(tokenStr)
	assert.True(t, expired)
}

func TestJWTService_IsTokenExpired_InvalidToken(t *testing.T) {
	svc := setupJWTService(t)

	expired := svc.IsTokenExpired("invalid")
	assert.True(t, expired)
}

// --- RefreshAccessToken ---

func TestJWTService_RefreshAccessToken_Success(t *testing.T) {
	svc := setupJWTService(t)
	// Use access token as input (has ClientID); in real flow refresh might carry client_id from login
	accessToken, err := svc.GenerateAccessToken(jwtTestUser(), "client-1")
	require.NoError(t, err)

	newToken, err := svc.RefreshAccessToken(accessToken)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)

	claims, err := svc.ParseToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "client-1", claims.ClientID)
}

func TestJWTService_RefreshAccessToken_InvalidToken(t *testing.T) {
	svc := setupJWTService(t)

	newToken, err := svc.RefreshAccessToken("invalid.refresh.token")
	require.Error(t, err)
	assert.Empty(t, newToken)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestJWTService_RefreshAccessToken_RefreshTokenWithoutClientID(t *testing.T) {
	svc := setupJWTService(t)
	// Refresh token from GenerateRefreshToken has no ClientID
	refreshToken, err := svc.GenerateRefreshToken(jwtTestUser())
	require.NoError(t, err)

	newToken, err := svc.RefreshAccessToken(refreshToken)
	require.Error(t, err)
	assert.Empty(t, newToken)
	assert.Contains(t, err.Error(), "invalid")
}

// --- GetRefreshTokenTTL ---

func TestJWTService_GetRefreshTokenTTL(t *testing.T) {
	_, _, cfg := setupJWTServiceWithKeys(t)
	svc := setupJWTService(t)

	ttl := svc.GetRefreshTokenTTL()
	assert.Equal(t, cfg.RefreshTokenTTL, ttl)
	assert.Equal(t, 7*24*time.Hour, ttl)
}
