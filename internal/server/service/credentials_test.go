package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/mocks"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupCredentialsService(t *testing.T) (*CredentialsService, *mocks.MockCredentialRepository) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockCredentialRepository(ctrl)
	svc := NewCredentialsService(zap.NewNop(), repo)
	return svc, repo
}

func validCredentialEntity() *entity.Credential {
	return &entity.Credential{
		Name:     "GitHub",
		Login:    "user@example.com",
		Password: "secret",
	}
}

func validCredentialModel(id, userID string) *models.Credential {
	return &models.Credential{
		ID:        id,
		UserID:    userID,
		Name:      "GitHub",
		Login:     "user@example.com",
		Password:  "secret",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// --- CreateCredential ---

func TestCredentialsService_CreateCredential_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	userID := "user-1"
	cred := validCredentialEntity()
	created := validCredentialModel("cred-1", userID)

	repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Credential) (*models.Credential, error) {
		assert.Equal(t, userID, m.UserID)
		assert.Equal(t, cred.Name, m.Name)
		assert.Equal(t, cred.Login, m.Login)
		assert.Equal(t, cred.Password, m.Password)
		created.UserID = m.UserID
		created.Name = m.Name
		created.Login = m.Login
		created.Password = m.Password
		created.CreatedAt = m.CreatedAt
		created.UpdatedAt = m.UpdatedAt
		return created, nil
	})

	got, err := svc.CreateCredential(ctx, userID, cred)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, cred.Name, got.Name)
	assert.Equal(t, cred.Login, got.Login)
	assert.Equal(t, cred.Password, got.Password)
}

func TestCredentialsService_CreateCredential_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialEntity()
	got, err := svc.CreateCredential(ctx, "", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_CreateCredential_NilCredential(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	got, err := svc.CreateCredential(ctx, "user-1", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_CreateCredential_EmptyName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialEntity()
	cred.Name = ""
	got, err := svc.CreateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_CreateCredential_EmptyLogin(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialEntity()
	cred.Login = ""
	got, err := svc.CreateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_CreateCredential_EmptyPassword(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialEntity()
	cred.Password = ""
	got, err := svc.CreateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_CreateCredential_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialEntity()
	repo.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("db error"))

	got, err := svc.CreateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- GetCredential ---

func TestCredentialsService_GetCredential_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialModel("cred-1", "user-1")
	repo.EXPECT().FindByID(ctx, "cred-1").Return(cred, nil)

	got, err := svc.GetCredential(ctx, "user-1", "cred-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "cred-1", got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, cred.Name, got.Name)
	assert.Equal(t, cred.Login, got.Login)
	assert.Equal(t, cred.Password, got.Password)
}

func TestCredentialsService_GetCredential_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	got, err := svc.GetCredential(ctx, "", "cred-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_GetCredential_EmptyCredentialID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	got, err := svc.GetCredential(ctx, "user-1", "")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_GetCredential_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	repo.EXPECT().FindByID(ctx, "cred-1").Return(nil, models.ErrCredentialNotFound)

	got, err := svc.GetCredential(ctx, "user-1", "cred-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_GetCredential_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialModel("cred-1", "user-1")
	repo.EXPECT().FindByID(ctx, "cred-1").Return(cred, nil)

	got, err := svc.GetCredential(ctx, "other-user", "cred-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestCredentialsService_GetCredential_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	repo.EXPECT().FindByID(ctx, "cred-1").Return(nil, errors.New("db error"))

	got, err := svc.GetCredential(ctx, "user-1", "cred-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- ListCredentials ---

func TestCredentialsService_ListCredentials_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	list := []*models.Credential{
		validCredentialModel("cred-1", "user-1"),
		validCredentialModel("cred-2", "user-1"),
	}
	list[1].Name = "GitLab"
	repo.EXPECT().FindByUserId(ctx, "user-1", 10, 0).Return(list, nil)
	repo.EXPECT().CountByUserID(ctx, "user-1").Return(int64(2), nil)

	got, total, err := svc.ListCredentials(ctx, "user-1", 10, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "cred-1", got[0].ID)
	assert.Equal(t, "cred-2", got[1].ID)
	assert.Equal(t, "GitHub", got[0].Name)
	assert.Equal(t, "GitLab", got[1].Name)
}

func TestCredentialsService_ListCredentials_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	got, total, err := svc.ListCredentials(ctx, "", 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByUserId(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_ListCredentials_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	repo.EXPECT().FindByUserId(ctx, "user-1", 10, 0).Return(nil, errors.New("db error"))

	got, total, err := svc.ListCredentials(ctx, "user-1", 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- UpdateCredential ---

func TestCredentialsService_UpdateCredential_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	updated := validCredentialModel("cred-1", "user-1")
	updated.Name = "GitHub Updated"
	updated.Login = "new@example.com"
	updated.Password = "newsecret"
	updated.UpdatedAt = time.Now()

	cred := &entity.Credential{
		ID:       "cred-1",
		Name:     "GitHub Updated",
		Login:    "new@example.com",
		Password: "newsecret",
	}

	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Credential) error {
		assert.Equal(t, "GitHub Updated", m.Name)
		assert.Equal(t, "new@example.com", m.Login)
		assert.Equal(t, "newsecret", m.Password)
		return nil
	})
	repo.EXPECT().FindByID(ctx, "cred-1").Return(updated, nil)

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "cred-1", got.ID)
	assert.Equal(t, "GitHub Updated", got.Name)
	assert.Equal(t, "new@example.com", got.Login)
	assert.Equal(t, "newsecret", got.Password)
}

func TestCredentialsService_UpdateCredential_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "l", Password: "p"}

	got, err := svc.UpdateCredential(ctx, "", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_UpdateCredential_NilCredential(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	got, err := svc.UpdateCredential(ctx, "user-1", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_UpdateCredential_EmptyCredentialID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialEntity()
	cred.ID = ""

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_UpdateCredential_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "l", Password: "p"}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(nil, models.ErrCredentialNotFound)

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_UpdateCredential_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "l", Password: "p"}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)

	got, err := svc.UpdateCredential(ctx, "other-user", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestCredentialsService_UpdateCredential_EmptyName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	cred := &entity.Credential{ID: "cred-1", Name: "", Login: "l", Password: "p"}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_UpdateCredential_EmptyLogin(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "", Password: "p"}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_UpdateCredential_EmptyPassword(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "l", Password: ""}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_UpdateCredential_RepoUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "l", Password: "p"}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("update failed"))

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestCredentialsService_UpdateCredential_RepoFindAfterUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	existing := validCredentialModel("cred-1", "user-1")
	cred := &entity.Credential{ID: "cred-1", Name: "n", Login: "l", Password: "p"}
	repo.EXPECT().FindByID(ctx, "cred-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().FindByID(ctx, "cred-1").Return(nil, errors.New("db error"))

	got, err := svc.UpdateCredential(ctx, "user-1", cred)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- DeleteCredential ---

func TestCredentialsService_DeleteCredential_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialModel("cred-1", "user-1")
	repo.EXPECT().FindByID(ctx, "cred-1").Return(cred, nil)
	repo.EXPECT().Delete(ctx, "cred-1").Return(nil)

	err := svc.DeleteCredential(ctx, "user-1", "cred-1")
	require.NoError(t, err)
}

func TestCredentialsService_DeleteCredential_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	err := svc.DeleteCredential(ctx, "", "cred-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_DeleteCredential_EmptyCredentialID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	err := svc.DeleteCredential(ctx, "user-1", "")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_DeleteCredential_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	repo.EXPECT().FindByID(ctx, "cred-1").Return(nil, models.ErrCredentialNotFound)

	err := svc.DeleteCredential(ctx, "user-1", "cred-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_DeleteCredential_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialModel("cred-1", "user-1")
	repo.EXPECT().FindByID(ctx, "cred-1").Return(cred, nil)

	err := svc.DeleteCredential(ctx, "other-user", "cred-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	repo.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
}

func TestCredentialsService_DeleteCredential_RepoDeleteError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialModel("cred-1", "user-1")
	repo.EXPECT().FindByID(ctx, "cred-1").Return(cred, nil)
	repo.EXPECT().Delete(ctx, "cred-1").Return(errors.New("delete failed"))

	err := svc.DeleteCredential(ctx, "user-1", "cred-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestCredentialsService_DeleteCredential_RepoDeleteErrNotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCredentialsService(t)

	cred := validCredentialModel("cred-1", "user-1")
	repo.EXPECT().FindByID(ctx, "cred-1").Return(cred, nil)
	repo.EXPECT().Delete(ctx, "cred-1").Return(models.ErrCredentialNotFound)

	err := svc.DeleteCredential(ctx, "user-1", "cred-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}
