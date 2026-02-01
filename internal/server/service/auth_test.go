package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/mocks"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/BigSm0uk/GophKeeper/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func setupAuthService(t *testing.T) (*AuthService, *mocks.MockUserRepository, *mocks.MockSessionRepository, *mocks.MockTokenService) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	ur := mocks.NewMockUserRepository(ctrl)
	sr := mocks.NewMockSessionRepository(ctrl)
	jwtService := mocks.NewMockTokenService(ctrl)

	as := NewAuthService(
		zap.NewNop(),
		ur,
		jwtService,
		sr,
	)

	return as, ur, sr, jwtService
}

// --- Register ---

func TestAuthService_Register_Success(t *testing.T) {
	ctx := context.Background()
	as, ur, _, _ := setupAuthService(t)

	username := "testuser"
	password := "securepass123"
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	input := entity.User{Username: username, Password: password, Email: nil}
	createdUser := &models.User{
		ID:             "user-1",
		Username:       username,
		Email:          nil,
		HashedPassword: hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ur.EXPECT().ExistsByUsername(ctx, username).Return(false, nil)
	ur.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *models.User) (*models.User, error) {
		assert.Equal(t, username, u.Username)
		assert.NotEmpty(t, u.HashedPassword)
		createdUser.Username = u.Username
		createdUser.HashedPassword = u.HashedPassword
		createdUser.CreatedAt = u.CreatedAt
		createdUser.UpdatedAt = u.UpdatedAt
		return createdUser, nil
	})

	got, err := as.Register(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, createdUser.ID, got.ID)
	assert.Equal(t, username, got.Username)
}

func TestAuthService_Register_UserAlreadyExists(t *testing.T) {
	ctx := context.Background()
	as, ur, _, _ := setupAuthService(t)

	username := "existing"
	input := entity.User{Username: username, Password: "pass", Email: nil}

	ur.EXPECT().ExistsByUsername(ctx, username).Return(true, nil)

	got, err := as.Register(ctx, input)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, models.ErrUserAlreadyExists)
}

func TestAuthService_Register_ExistsByUsernameError(t *testing.T) {
	ctx := context.Background()
	as, ur, _, _ := setupAuthService(t)

	input := entity.User{Username: "u", Password: "p", Email: nil}
	repoErr := errors.New("db error")

	ur.EXPECT().ExistsByUsername(ctx, "u").Return(false, repoErr)

	got, err := as.Register(ctx, input)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, repoErr)
}

func TestAuthService_Register_CreateError(t *testing.T) {
	ctx := context.Background()
	as, ur, _, _ := setupAuthService(t)

	input := entity.User{Username: "u", Password: "pass123", Email: nil}
	repoErr := errors.New("create failed")

	ur.EXPECT().ExistsByUsername(ctx, "u").Return(false, nil)
	ur.EXPECT().Create(ctx, gomock.Any()).Return(nil, repoErr)

	got, err := as.Register(ctx, input)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, repoErr)
}

// --- Login ---

func TestAuthService_Login_Success(t *testing.T) {
	ctx := context.Background()
	as, ur, sr, jwt := setupAuthService(t)

	username := "user1"
	password := "pass123"
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := &models.User{
		ID:             "uid-1",
		Username:       username,
		HashedPassword: hashedPassword,
	}
	refreshTTL := 24 * time.Hour

	ur.EXPECT().FindByUsername(ctx, username).Return(user, nil)
	sr.EXPECT().CountActiveByUserID(ctx, user.ID).Return(0, nil)
	jwt.EXPECT().GetRefreshTokenTTL().Return(refreshTTL).AnyTimes()
	jwt.EXPECT().GenerateRefreshToken(user).Return("refresh-token", nil)
	sr.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *models.Session) (*models.Session, error) {
		s.ID = "session-1"
		return s, nil
	})
	jwt.EXPECT().GenerateAccessToken(user, gomock.Any()).DoAndReturn(func(u *models.User, cid string) (string, error) {
		assert.Equal(t, user.ID, u.ID)
		return "access-token", nil
	})

	accessToken, err := as.Login(ctx, username, password, "", "")
	require.NoError(t, err)
	assert.Equal(t, "access-token", accessToken)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()
	as, ur, _, _ := setupAuthService(t)

	ur.EXPECT().FindByUsername(ctx, "nobody").Return(nil, errors.New("not found"))

	accessToken, err := as.Login(ctx, "nobody", "pass", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrInvalidCredentials)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	as, ur, _, _ := setupAuthService(t)

	hashedPassword, _ := util.HashPassword("correct")
	user := &models.User{ID: "u1", Username: "u", HashedPassword: hashedPassword}

	ur.EXPECT().FindByUsername(ctx, "u").Return(user, nil)

	accessToken, err := as.Login(ctx, "u", "wrongpassword", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrInvalidCredentials)
}

func TestAuthService_Login_GenerateRefreshTokenError(t *testing.T) {
	ctx := context.Background()
	as, ur, sr, jwt := setupAuthService(t)

	password := "pass"
	hashedPassword, _ := util.HashPassword(password)
	user := &models.User{ID: "u1", Username: "u", HashedPassword: hashedPassword}

	ur.EXPECT().FindByUsername(ctx, "u").Return(user, nil)
	sr.EXPECT().CountActiveByUserID(ctx, user.ID).Return(0, nil)
	jwt.EXPECT().GetRefreshTokenTTL().Return(time.Hour).AnyTimes()
	jwt.EXPECT().GenerateRefreshToken(user).Return("", errors.New("jwt error"))

	accessToken, err := as.Login(ctx, "u", password, "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
}

func TestAuthService_Login_CreateSessionError(t *testing.T) {
	ctx := context.Background()
	as, ur, sr, jwt := setupAuthService(t)

	password := "pass"
	hashedPassword, _ := util.HashPassword(password)
	user := &models.User{ID: "u1", Username: "u", HashedPassword: hashedPassword}

	ur.EXPECT().FindByUsername(ctx, "u").Return(user, nil)
	sr.EXPECT().CountActiveByUserID(ctx, user.ID).Return(0, nil)
	jwt.EXPECT().GetRefreshTokenTTL().Return(time.Hour).AnyTimes()
	jwt.EXPECT().GenerateRefreshToken(user).Return("refresh", nil)
	sr.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("session create failed"))

	accessToken, err := as.Login(ctx, "u", password, "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
}

// --- Token ---

func TestAuthService_Token_PasswordGrant_DelegatesToLogin(t *testing.T) {
	ctx := context.Background()
	as, ur, sr, jwt := setupAuthService(t)

	username := "u"
	password := "p"
	hashedPassword, _ := util.HashPassword(password)
	user := &models.User{ID: "uid", Username: username, HashedPassword: hashedPassword}

	ur.EXPECT().FindByUsername(ctx, username).Return(user, nil)
	sr.EXPECT().CountActiveByUserID(ctx, user.ID).Return(0, nil)
	jwt.EXPECT().GetRefreshTokenTTL().Return(time.Hour).AnyTimes()
	jwt.EXPECT().GenerateRefreshToken(user).Return("refresh", nil)
	sr.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *models.Session) (*models.Session, error) {
		s.ID = "s1"
		return s, nil
	})
	jwt.EXPECT().GenerateAccessToken(user, gomock.Any()).Return("access", nil)

	accessToken, err := as.Token(ctx, "password", username, password, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, "access", accessToken)
}

func TestAuthService_Token_InvalidGrantType(t *testing.T) {
	ctx := context.Background()
	as, _, _, _ := setupAuthService(t)

	accessToken, err := as.Token(ctx, "invalid_grant", "", "", "", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrInvalidGrantType)
}

func TestAuthService_Token_RefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	as, ur, sr, jwt := setupAuthService(t)

	userID := "uid-1"
	clientID := "client-1"
	accessTokenForRefresh := "expired-access-token"
	user := &models.User{ID: userID, Username: "u"}
	session := &models.Session{
		ID:        "sess-1",
		UserID:    userID,
		ClientID:  clientID,
		Revoked:   false,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	jwt.EXPECT().ExtractUserID(accessTokenForRefresh).Return(userID, nil)
	jwt.EXPECT().ExtractClientID(accessTokenForRefresh).Return(clientID, nil)
	sr.EXPECT().FindByUserIDAndClientID(ctx, userID, clientID).Return(session, nil)
	ur.EXPECT().FindByID(ctx, userID).Return(user, nil)
	sr.EXPECT().UpdateActivity(ctx, session.ID).Return(nil)
	jwt.EXPECT().GenerateRefreshToken(user).Return("new-refresh", nil)
	sr.EXPECT().Revoke(ctx, session.ID, "token_rotated").Return(nil)
	sr.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *models.Session) (*models.Session, error) {
		return s, nil
	})
	jwt.EXPECT().GetRefreshTokenTTL().Return(time.Hour).AnyTimes()
	jwt.EXPECT().GenerateAccessToken(user, clientID).Return("new-access", nil)

	accessToken, err := as.Token(ctx, "refresh_token", "", "", accessTokenForRefresh, "", "")
	require.NoError(t, err)
	assert.Equal(t, "new-access", accessToken)
}

func TestAuthService_Token_RefreshToken_ExtractUserIDError(t *testing.T) {
	ctx := context.Background()
	as, _, _, jwt := setupAuthService(t)

	jwt.EXPECT().ExtractUserID(gomock.Any()).Return("", errors.New("parse error"))

	accessToken, err := as.Token(ctx, "refresh_token", "", "", "bad-token", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrInvalidToken)
}

func TestAuthService_Token_RefreshToken_ExtractClientIDError(t *testing.T) {
	ctx := context.Background()
	as, _, _, jwt := setupAuthService(t)

	jwt.EXPECT().ExtractUserID(gomock.Any()).Return("uid", nil)
	jwt.EXPECT().ExtractClientID(gomock.Any()).Return("", errors.New("no client"))

	accessToken, err := as.Token(ctx, "refresh_token", "", "", "token", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrInvalidToken)
}

func TestAuthService_Token_RefreshToken_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	as, _, sr, jwt := setupAuthService(t)

	jwt.EXPECT().ExtractUserID(gomock.Any()).Return("uid", nil)
	jwt.EXPECT().ExtractClientID(gomock.Any()).Return("cid", nil)
	sr.EXPECT().FindByUserIDAndClientID(ctx, "uid", "cid").Return(nil, errors.New("not found"))

	accessToken, err := as.Token(ctx, "refresh_token", "", "", "token", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrSessionNotFound)
}

func TestAuthService_Token_RefreshToken_SessionExpired(t *testing.T) {
	ctx := context.Background()
	as, _, sr, jwt := setupAuthService(t)

	session := &models.Session{
		ID:        "s1",
		UserID:    "uid",
		ClientID:  "cid",
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	jwt.EXPECT().ExtractUserID(gomock.Any()).Return("uid", nil)
	jwt.EXPECT().ExtractClientID(gomock.Any()).Return("cid", nil)
	sr.EXPECT().FindByUserIDAndClientID(ctx, "uid", "cid").Return(session, nil)

	accessToken, err := as.Token(ctx, "refresh_token", "", "", "token", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrSessionExpired)
}

func TestAuthService_Token_RefreshToken_SessionRevoked(t *testing.T) {
	ctx := context.Background()
	as, _, sr, jwt := setupAuthService(t)

	reason := "revoked"
	session := &models.Session{
		ID:            "s1",
		UserID:        "uid",
		ClientID:      "cid",
		Revoked:       true,
		RevokedReason: &reason,
		ExpiresAt:     time.Now().Add(time.Hour),
	}

	jwt.EXPECT().ExtractUserID(gomock.Any()).Return("uid", nil)
	jwt.EXPECT().ExtractClientID(gomock.Any()).Return("cid", nil)
	sr.EXPECT().FindByUserIDAndClientID(ctx, "uid", "cid").Return(session, nil)

	accessToken, err := as.Token(ctx, "refresh_token", "", "", "token", "", "")
	require.Error(t, err)
	assert.Empty(t, accessToken)
	assert.ErrorIs(t, err, models.ErrSessionRevoked)
}

// --- ValidateToken ---

func TestAuthService_ValidateToken_Success(t *testing.T) {
	ctx := context.Background()
	as, ur, _, jwt := setupAuthService(t)

	userID := "uid-1"
	user := &models.User{ID: userID, Username: "u"}

	jwt.EXPECT().ExtractUserID("valid-token").Return(userID, nil)
	ur.EXPECT().FindByID(ctx, userID).Return(user, nil)

	got, err := as.ValidateToken(ctx, "valid-token")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, userID, got.ID)
}

func TestAuthService_ValidateToken_ExtractUserIDError(t *testing.T) {
	ctx := context.Background()
	as, _, _, jwt := setupAuthService(t)

	jwt.EXPECT().ExtractUserID("bad-token").Return("", errors.New("invalid"))

	got, err := as.ValidateToken(ctx, "bad-token")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, models.ErrInvalidToken)
}

func TestAuthService_ValidateToken_UserNotFound(t *testing.T) {
	ctx := context.Background()
	as, ur, _, jwt := setupAuthService(t)

	jwt.EXPECT().ExtractUserID("token").Return("uid", nil)
	ur.EXPECT().FindByID(ctx, "uid").Return(nil, errors.New("user not found"))

	got, err := as.ValidateToken(ctx, "token")
	require.Error(t, err)
	assert.Nil(t, got)
}
