package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/mocks"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupUserService(t *testing.T) (*UserService, *mocks.MockUserRepository, *mocks.MockSessionRepository) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	svc := NewUserService(zap.NewNop(), userRepo, sessionRepo)
	return svc, userRepo, sessionRepo
}

func validUser(id string) *models.User {
	email := "user@example.com"
	return &models.User{
		ID:             id,
		Username:       "testuser",
		Email:          &email,
		HashedPassword: "",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func validSession(id, userID, clientID string) *models.Session {
	ip := "127.0.0.1"
	return &models.Session{
		ID:         id,
		UserID:     userID,
		ClientID:   clientID,
		IPAddress:  &ip,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
}

// --- GetProfile ---

func TestUserService_GetProfile_Success(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	user := validUser("user-1")
	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)

	got, err := svc.GetProfile(ctx, "user-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "user-1", got.ID)
	assert.Equal(t, "testuser", got.Username)
}

func TestUserService_GetProfile_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	got, err := svc.GetProfile(ctx, "")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	userRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(nil, models.ErrUserNotFound)

	got, err := svc.GetProfile(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserService_GetProfile_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(nil, errors.New("db error"))

	got, err := svc.GetProfile(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- UpdateProfile ---

func TestUserService_UpdateProfile_Success(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	user := validUser("user-1")
	newEmail := "new@example.com"
	updated := validUser("user-1")
	updated.Email = &newEmail

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *models.User) error {
		assert.Equal(t, "user-1", u.ID)
		require.NotNil(t, u.Email)
		assert.Equal(t, "new@example.com", *u.Email)
		return nil
	})
	userRepo.EXPECT().FindByID(ctx, "user-1").Return(updated, nil)

	got, err := svc.UpdateProfile(ctx, "user-1", &newEmail)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "new@example.com", *got.Email)
}

func TestUserService_UpdateProfile_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	email := "e@x.com"
	got, err := svc.UpdateProfile(ctx, "", &email)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	userRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_UpdateProfile_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	email := "e@x.com"
	userRepo.EXPECT().FindByID(ctx, "user-1").Return(nil, models.ErrUserNotFound)

	got, err := svc.UpdateProfile(ctx, "user-1", &email)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserService_UpdateProfile_RepoUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	user := validUser("user-1")
	email := "e@x.com"
	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("update failed"))

	got, err := svc.UpdateProfile(ctx, "user-1", &email)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestUserService_UpdateProfile_RepoFindAfterUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	user := validUser("user-1")
	email := "e@x.com"
	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	userRepo.EXPECT().FindByID(ctx, "user-1").Return(nil, errors.New("db error"))

	got, err := svc.UpdateProfile(ctx, "user-1", &email)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- ChangePassword ---

func TestUserService_ChangePassword_Success(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	oldPass := "oldpass"
	newPass := "newpass"
	hashedOld, err := util.HashPassword(oldPass)
	require.NoError(t, err)

	user := validUser("user-1")
	user.HashedPassword = hashedOld

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *models.User) error {
		assert.Equal(t, "user-1", u.ID)
		assert.NotEqual(t, hashedOld, u.HashedPassword)
		return nil
	})

	err = svc.ChangePassword(ctx, "user-1", oldPass, newPass)
	require.NoError(t, err)
}

func TestUserService_ChangePassword_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	err := svc.ChangePassword(ctx, "", "old", "new")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	userRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_ChangePassword_UserNotFound(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(nil, models.ErrUserNotFound)

	err := svc.ChangePassword(ctx, "user-1", "old", "new")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserService_ChangePassword_InvalidOldPassword(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	hashedOld, err := util.HashPassword("correct")
	require.NoError(t, err)
	user := validUser("user-1")
	user.HashedPassword = hashedOld

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)

	err = svc.ChangePassword(ctx, "user-1", "wrong", "newpass")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_ChangePassword_VerifyPasswordError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	user := validUser("user-1")
	user.HashedPassword = "invalid-hash-format"

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)

	err := svc.ChangePassword(ctx, "user-1", "old", "new")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_ChangePassword_RepoUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	hashedOld, err := util.HashPassword("old")
	require.NoError(t, err)
	user := validUser("user-1")
	user.HashedPassword = hashedOld

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("update failed"))

	err = svc.ChangePassword(ctx, "user-1", "old", "new")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- DeleteAccount ---

func TestUserService_DeleteAccount_Success(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	pass := "mypassword"
	hashed, err := util.HashPassword(pass)
	require.NoError(t, err)
	user := validUser("user-1")
	user.HashedPassword = hashed

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Delete(ctx, "user-1").Return(nil)

	err = svc.DeleteAccount(ctx, "user-1", pass)
	require.NoError(t, err)
}

func TestUserService_DeleteAccount_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	err := svc.DeleteAccount(ctx, "", "pass")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	userRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_DeleteAccount_UserNotFound(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(nil, models.ErrUserNotFound)

	err := svc.DeleteAccount(ctx, "user-1", "pass")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserService_DeleteAccount_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	hashed, err := util.HashPassword("correct")
	require.NoError(t, err)
	user := validUser("user-1")
	user.HashedPassword = hashed

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)

	err = svc.DeleteAccount(ctx, "user-1", "wrong")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	userRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_DeleteAccount_VerifyPasswordError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	user := validUser("user-1")
	user.HashedPassword = "invalid"

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)

	err := svc.DeleteAccount(ctx, "user-1", "pass")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	userRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_DeleteAccount_RepoDeleteError(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _ := setupUserService(t)

	hashed, err := util.HashPassword("pass")
	require.NoError(t, err)
	user := validUser("user-1")
	user.HashedPassword = hashed

	userRepo.EXPECT().FindByID(ctx, "user-1").Return(user, nil)
	userRepo.EXPECT().Delete(ctx, "user-1").Return(errors.New("delete failed"))

	err = svc.DeleteAccount(ctx, "user-1", "pass")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- GetActiveSessions ---

func TestUserService_GetActiveSessions_Success(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessions := []*models.Session{
		validSession("s1", "user-1", "client-1"),
		validSession("s2", "user-1", "client-2"),
	}
	sessionRepo.EXPECT().FindActiveByUserID(ctx, "user-1").Return(sessions, nil)

	list, current, err := svc.GetActiveSessions(ctx, "user-1", "")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Nil(t, current)
	assert.Equal(t, "s1", list[0].ID)
	assert.Equal(t, "s2", list[1].ID)
}

func TestUserService_GetActiveSessions_WithCurrentClient(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessions := []*models.Session{
		validSession("s1", "user-1", "client-1"),
	}
	currentSession := validSession("s1", "user-1", "client-1")
	sessionRepo.EXPECT().FindActiveByUserID(ctx, "user-1").Return(sessions, nil)
	sessionRepo.EXPECT().FindByUserIDAndClientID(ctx, "user-1", "client-1").Return(currentSession, nil)

	list, current, err := svc.GetActiveSessions(ctx, "user-1", "client-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, current)
	assert.Equal(t, "s1", current.ID)
	assert.Equal(t, "client-1", current.ClientID)
}

func TestUserService_GetActiveSessions_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	list, current, err := svc.GetActiveSessions(ctx, "", "client-1")
	require.Error(t, err)
	assert.Nil(t, list)
	assert.Nil(t, current)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	sessionRepo.EXPECT().FindActiveByUserID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_GetActiveSessions_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().FindActiveByUserID(ctx, "user-1").Return(nil, errors.New("db error"))

	list, current, err := svc.GetActiveSessions(ctx, "user-1", "")
	require.Error(t, err)
	assert.Nil(t, list)
	assert.Nil(t, current)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- RevokeSession ---

func TestUserService_RevokeSession_Success(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	session := validSession("s1", "user-1", "client-1")
	sessionRepo.EXPECT().FindByID(ctx, "s1").Return(session, nil)
	sessionRepo.EXPECT().Revoke(ctx, "s1", "user_revoked").Return(nil)

	err := svc.RevokeSession(ctx, "user-1", "s1")
	require.NoError(t, err)
}

func TestUserService_RevokeSession_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	err := svc.RevokeSession(ctx, "", "s1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	sessionRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_RevokeSession_EmptySessionID(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	err := svc.RevokeSession(ctx, "user-1", "")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	sessionRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_RevokeSession_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().FindByID(ctx, "s1").Return(nil, models.ErrSessionNotFound)

	err := svc.RevokeSession(ctx, "user-1", "s1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserService_RevokeSession_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	session := validSession("s1", "other-user", "client-1")
	sessionRepo.EXPECT().FindByID(ctx, "s1").Return(session, nil)

	err := svc.RevokeSession(ctx, "user-1", "s1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	sessionRepo.EXPECT().Revoke(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_RevokeSession_RepoRevokeError(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	session := validSession("s1", "user-1", "client-1")
	sessionRepo.EXPECT().FindByID(ctx, "s1").Return(session, nil)
	sessionRepo.EXPECT().Revoke(ctx, "s1", "user_revoked").Return(errors.New("revoke failed"))

	err := svc.RevokeSession(ctx, "user-1", "s1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- RevokeAllSessions ---

func TestUserService_RevokeAllSessions_All(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().CountActiveByUserID(ctx, "user-1").Return(3, nil)
	sessionRepo.EXPECT().RevokeAllByUserID(ctx, "user-1", "user_revoked").Return(nil)

	count, err := svc.RevokeAllSessions(ctx, "user-1", "", false)
	require.NoError(t, err)
	assert.Equal(t, int32(3), count)
}

func TestUserService_RevokeAllSessions_ExceptCurrent(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().CountActiveByUserID(ctx, "user-1").Return(3, nil)
	sessionRepo.EXPECT().RevokeAllExceptCurrent(ctx, "user-1", "current-s1", "user_revoked").Return(nil)

	count, err := svc.RevokeAllSessions(ctx, "user-1", "current-s1", true)
	require.NoError(t, err)
	assert.Equal(t, int32(2), count)
}

func TestUserService_RevokeAllSessions_ExceptCurrentSingleSession(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().CountActiveByUserID(ctx, "user-1").Return(1, nil)
	sessionRepo.EXPECT().RevokeAllExceptCurrent(ctx, "user-1", "current-s1", "user_revoked").Return(nil)

	count, err := svc.RevokeAllSessions(ctx, "user-1", "current-s1", true)
	require.NoError(t, err)
	assert.Equal(t, int32(0), count)
}

func TestUserService_RevokeAllSessions_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	count, err := svc.RevokeAllSessions(ctx, "", "s1", true)
	require.Error(t, err)
	assert.Equal(t, int32(0), count)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	sessionRepo.EXPECT().CountActiveByUserID(gomock.Any(), gomock.Any()).Times(0)
}

func TestUserService_RevokeAllSessions_CountErrorUsesZero(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().CountActiveByUserID(ctx, "user-1").Return(0, errors.New("count failed"))
	sessionRepo.EXPECT().RevokeAllByUserID(ctx, "user-1", "user_revoked").Return(nil)

	count, err := svc.RevokeAllSessions(ctx, "user-1", "", false)
	require.NoError(t, err)
	assert.Equal(t, int32(0), count)
}

func TestUserService_RevokeAllSessions_RevokeAllError(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().CountActiveByUserID(ctx, "user-1").Return(2, nil)
	sessionRepo.EXPECT().RevokeAllByUserID(ctx, "user-1", "user_revoked").Return(errors.New("revoke failed"))

	count, err := svc.RevokeAllSessions(ctx, "user-1", "", false)
	require.Error(t, err)
	assert.Equal(t, int32(0), count)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestUserService_RevokeAllSessions_RevokeExceptCurrentError(t *testing.T) {
	ctx := context.Background()
	svc, _, sessionRepo := setupUserService(t)

	sessionRepo.EXPECT().CountActiveByUserID(ctx, "user-1").Return(2, nil)
	sessionRepo.EXPECT().RevokeAllExceptCurrent(ctx, "user-1", "current-s1", "user_revoked").Return(errors.New("revoke failed"))

	count, err := svc.RevokeAllSessions(ctx, "user-1", "current-s1", true)
	require.Error(t, err)
	assert.Equal(t, int32(0), count)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}
