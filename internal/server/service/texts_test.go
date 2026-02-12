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

func setupTextsService(t *testing.T) (*TextsService, *mocks.MockTextRepository) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockTextRepository(ctrl)
	svc := NewTextsService(zap.NewNop(), repo)
	return svc, repo
}

// --- CreateText ---

func TestTextsService_CreateText_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	userID := "user-1"
	text := &entity.Text{
		Name:    "note",
		Content: "hello world",
	}

	created := &models.Text{
		ID:        "text-1",
		UserID:    userID,
		Name:      text.Name,
		Content:   text.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Text) (*models.Text, error) {
		assert.Equal(t, userID, m.UserID)
		assert.Equal(t, text.Name, m.Name)
		assert.Equal(t, text.Content, m.Content)
		created.UserID = m.UserID
		created.Name = m.Name
		created.Content = m.Content
		created.CreatedAt = m.CreatedAt
		created.UpdatedAt = m.UpdatedAt
		return created, nil
	})

	got, err := svc.CreateText(ctx, userID, text)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, text.Name, got.Name)
	assert.Equal(t, text.Content, got.Content)
}

func TestTextsService_CreateText_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{Name: "n", Content: "c"}

	got, err := svc.CreateText(ctx, "", text)
	require.Error(t, err)
	assert.Nil(t, got)
	// validateText uses validation.Err() which returns gRPC status
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_CreateText_NilText(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	got, err := svc.CreateText(ctx, "user-1", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidText))
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_CreateText_EmptyName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{Name: "", Content: "content"}

	got, err := svc.CreateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_CreateText_EmptyContent(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{Name: "name", Content: ""}

	got, err := svc.CreateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_CreateText_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{Name: "n", Content: "c"}
	repo.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("db error"))

	got, err := svc.CreateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
}

// --- GetText ---

func TestTextsService_GetText_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &models.Text{
		ID:        "text-1",
		UserID:    "user-1",
		Name:      "note",
		Content:   "content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.EXPECT().FindByID(ctx, "text-1").Return(text, nil)

	got, err := svc.GetText(ctx, "user-1", "text-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "text-1", got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "note", got.Name)
	assert.Equal(t, "content", got.Content)
}

func TestTextsService_GetText_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	got, err := svc.GetText(ctx, "", "text-1")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_GetText_EmptyTextID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	got, err := svc.GetText(ctx, "user-1", "")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_GetText_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	repo.EXPECT().FindByID(ctx, "text-1").Return(nil, models.ErrTextNotFound)

	got, err := svc.GetText(ctx, "user-1", "text-1")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
}

func TestTextsService_GetText_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &models.Text{ID: "text-1", UserID: "user-1", Name: "n", Content: "c"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(text, nil)

	got, err := svc.GetText(ctx, "other-user", "text-1")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
}

func TestTextsService_GetText_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	repo.EXPECT().FindByID(ctx, "text-1").Return(nil, errors.New("db error"))

	got, err := svc.GetText(ctx, "user-1", "text-1")
	require.Error(t, err)
	assert.Nil(t, got)
}

// --- ListTexts ---

func TestTextsService_ListTexts_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	list := []*models.Text{
		{ID: "t1", UserID: "user-1", Name: "a", Content: "x"},
		{ID: "t2", UserID: "user-1", Name: "b", Content: "y"},
	}
	repo.EXPECT().FindByUserID(ctx, "user-1", 10, 0).Return(list, nil)
	repo.EXPECT().CountByUserID(ctx, "user-1").Return(int64(2), nil)

	got, total, err := svc.ListTexts(ctx, "user-1", 10, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "t1", got[0].ID)
	assert.Equal(t, "t2", got[1].ID)
}

func TestTextsService_ListTexts_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	got, total, err := svc.ListTexts(ctx, "", 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByUserID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_ListTexts_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	repo.EXPECT().FindByUserID(ctx, "user-1", 10, 0).Return(nil, errors.New("db error"))

	got, total, err := svc.ListTexts(ctx, "user-1", 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
}

// --- UpdateText ---

func TestTextsService_UpdateText_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	existing := &models.Text{
		ID:        "text-1",
		UserID:    "user-1",
		Name:      "old",
		Content:   "old content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	updated := &models.Text{
		ID:        "text-1",
		UserID:    "user-1",
		Name:      "new name",
		Content:   "new content",
		CreatedAt: existing.CreatedAt,
		UpdatedAt: time.Now(),
	}
	text := &entity.Text{
		ID:      "text-1",
		Name:    "new name",
		Content: "new content",
	}

	repo.EXPECT().FindByID(ctx, "text-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Text) error {
		assert.Equal(t, "new name", m.Name)
		assert.Equal(t, "new content", m.Content)
		return nil
	})
	repo.EXPECT().FindByID(ctx, "text-1").Return(updated, nil)

	got, err := svc.UpdateText(ctx, "user-1", text)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "text-1", got.ID)
	assert.Equal(t, "new name", got.Name)
	assert.Equal(t, "new content", got.Content)
}

func TestTextsService_UpdateText_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{ID: "t1", Name: "n", Content: "c"}

	got, err := svc.UpdateText(ctx, "", text)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_UpdateText_NilText(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	got, err := svc.UpdateText(ctx, "user-1", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidText))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_UpdateText_EmptyTextID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{ID: "", Name: "n", Content: "c"}

	got, err := svc.UpdateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_UpdateText_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &entity.Text{ID: "text-1", Name: "n", Content: "c"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(nil, models.ErrTextNotFound)

	got, err := svc.UpdateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
}

func TestTextsService_UpdateText_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	existing := &models.Text{ID: "text-1", UserID: "user-1", Name: "n", Content: "c"}
	text := &entity.Text{ID: "text-1", Name: "n", Content: "c"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(existing, nil)

	got, err := svc.UpdateText(ctx, "other-user", text)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
}

func TestTextsService_UpdateText_EmptyName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	existing := &models.Text{ID: "text-1", UserID: "user-1", Name: "n", Content: "c"}
	text := &entity.Text{ID: "text-1", Name: "", Content: "c"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(existing, nil)

	got, err := svc.UpdateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
	// validateText uses validation.Err() which returns gRPC status
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_UpdateText_RepoUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	existing := &models.Text{ID: "text-1", UserID: "user-1", Name: "n", Content: "c"}
	text := &entity.Text{ID: "text-1", Name: "n", Content: "c"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("update failed"))

	got, err := svc.UpdateText(ctx, "user-1", text)
	require.Error(t, err)
	assert.Nil(t, got)
}

// --- DeleteText ---

func TestTextsService_DeleteText_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &models.Text{ID: "text-1", UserID: "user-1", Name: "n", Content: "c"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(text, nil)
	repo.EXPECT().Delete(ctx, "text-1").Return(nil)

	err := svc.DeleteText(ctx, "user-1", "text-1")
	require.NoError(t, err)
}

func TestTextsService_DeleteText_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	err := svc.DeleteText(ctx, "", "text-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_DeleteText_EmptyTextID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	err := svc.DeleteText(ctx, "user-1", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_DeleteText_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	repo.EXPECT().FindByID(ctx, "text-1").Return(nil, models.ErrTextNotFound)

	err := svc.DeleteText(ctx, "user-1", "text-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
}

func TestTextsService_DeleteText_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &models.Text{ID: "text-1", UserID: "user-1"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(text, nil)

	err := svc.DeleteText(ctx, "other-user", "text-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrTextNotFound))
	repo.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
}

func TestTextsService_DeleteText_RepoDeleteError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupTextsService(t)

	text := &models.Text{ID: "text-1", UserID: "user-1"}
	repo.EXPECT().FindByID(ctx, "text-1").Return(text, nil)
	repo.EXPECT().Delete(ctx, "text-1").Return(errors.New("delete failed"))

	err := svc.DeleteText(ctx, "user-1", "text-1")
	require.Error(t, err)
}
