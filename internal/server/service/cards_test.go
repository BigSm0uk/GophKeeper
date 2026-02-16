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

func setupCardsService(t *testing.T) (*CardsService, *mocks.MockCardRepository) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockCardRepository(ctrl)
	svc := NewCardsService(zap.NewNop(), repo)
	return svc, repo
}

func validCardEntity() *entity.Card {
	return &entity.Card{
		Name:           "Visa",
		CardNumber:     "4111111111111111",
		CardholderName: "John Doe",
		ExpiryDate:     "12/25",
		CVV:            "123",
	}
}

func validCardModel(id, userID string) *models.Card {
	return &models.Card{
		ID:             id,
		UserID:         userID,
		Name:           "Visa",
		CardNumber:     "4111111111111111",
		CardholderName: "John Doe",
		ExpiryDate:     "12/25",
		CVV:            "123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// --- CreateCard ---

func TestCardsService_CreateCard_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	userID := "user-1"
	card := validCardEntity()
	created := validCardModel("card-1", userID)

	repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Card) (*models.Card, error) {
		assert.Equal(t, userID, m.UserID)
		assert.Equal(t, card.Name, m.Name)
		assert.Equal(t, card.CardNumber, m.CardNumber)
		assert.Equal(t, card.CardholderName, m.CardholderName)
		assert.Equal(t, card.ExpiryDate, m.ExpiryDate)
		assert.Equal(t, card.CVV, m.CVV)
		created.UserID = m.UserID
		created.Name = m.Name
		created.CardNumber = m.CardNumber
		created.CardholderName = m.CardholderName
		created.ExpiryDate = m.ExpiryDate
		created.CVV = m.CVV
		created.CreatedAt = m.CreatedAt
		created.UpdatedAt = m.UpdatedAt
		return created, nil
	})

	got, err := svc.CreateCard(ctx, userID, card)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, card.Name, got.Name)
	assert.Equal(t, card.CardNumber, got.CardNumber)
	assert.Equal(t, card.CardholderName, got.CardholderName)
	assert.Equal(t, card.ExpiryDate, got.ExpiryDate)
	assert.Equal(t, card.CVV, got.CVV)
}

func TestCardsService_CreateCard_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	got, err := svc.CreateCard(ctx, "", card)
	require.Error(t, err)
	assert.Nil(t, got)
	// validateCard uses validation.Err() which returns gRPC status
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_NilCard(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	got, err := svc.CreateCard(ctx, "user-1", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidCard))
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_EmptyName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	card.Name = ""
	got, err := svc.CreateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_EmptyCardNumber(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	card.CardNumber = ""
	got, err := svc.CreateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_EmptyCardholderName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	card.CardholderName = ""
	got, err := svc.CreateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_EmptyExpiryDate(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	card.ExpiryDate = ""
	got, err := svc.CreateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_EmptyCVV(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	card.CVV = ""
	got, err := svc.CreateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_CreateCard_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	repo.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("db error"))

	got, err := svc.CreateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
}

// --- GetCard ---

func TestCardsService_GetCard_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardModel("card-1", "user-1")
	repo.EXPECT().FindByID(ctx, "card-1").Return(card, nil)

	got, err := svc.GetCard(ctx, "user-1", "card-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "card-1", got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, card.Name, got.Name)
	assert.Equal(t, card.CardNumber, got.CardNumber)
}

func TestCardsService_GetCard_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	got, err := svc.GetCard(ctx, "", "card-1")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_GetCard_EmptyCardID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	got, err := svc.GetCard(ctx, "user-1", "")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_GetCard_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	repo.EXPECT().FindByID(ctx, "card-1").Return(nil, models.ErrCardNotFound)

	got, err := svc.GetCard(ctx, "user-1", "card-1")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}

func TestCardsService_GetCard_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardModel("card-1", "user-1")
	repo.EXPECT().FindByID(ctx, "card-1").Return(card, nil)

	got, err := svc.GetCard(ctx, "other-user", "card-1")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}

func TestCardsService_GetCard_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	repo.EXPECT().FindByID(ctx, "card-1").Return(nil, errors.New("db error"))

	got, err := svc.GetCard(ctx, "user-1", "card-1")
	require.Error(t, err)
	assert.Nil(t, got)
}

// --- ListCards ---

func TestCardsService_ListCards_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	list := []*models.Card{
		validCardModel("card-1", "user-1"),
		validCardModel("card-2", "user-1"),
	}
	list[1].Name = "Mastercard"
	repo.EXPECT().FindByUserID(ctx, "user-1", 10, 0).Return(list, nil)
	repo.EXPECT().CountByUserID(ctx, "user-1").Return(int64(2), nil)

	got, total, err := svc.ListCards(ctx, "user-1", 10, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "card-1", got[0].ID)
	assert.Equal(t, "card-2", got[1].ID)
	assert.Equal(t, "Visa", got[0].Name)
	assert.Equal(t, "Mastercard", got[1].Name)
}

func TestCardsService_ListCards_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	got, total, err := svc.ListCards(ctx, "", 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByUserID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_ListCards_RepoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	repo.EXPECT().FindByUserID(ctx, "user-1", 10, 0).Return(nil, errors.New("db error"))

	got, total, err := svc.ListCards(ctx, "user-1", 10, 0)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
}

// --- UpdateCard ---

func TestCardsService_UpdateCard_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	existing := validCardModel("card-1", "user-1")
	updated := validCardModel("card-1", "user-1")
	updated.Name = "Updated Visa"
	updated.CardNumber = "4111111111111112"
	updated.UpdatedAt = time.Now()

	card := &entity.Card{
		ID:             "card-1",
		Name:           "Updated Visa",
		CardNumber:     "4111111111111112",
		CardholderName: existing.CardholderName,
		ExpiryDate:     existing.ExpiryDate,
		CVV:            existing.CVV,
	}

	repo.EXPECT().FindByID(ctx, "card-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Card) error {
		assert.Equal(t, "Updated Visa", m.Name)
		assert.Equal(t, "4111111111111112", m.CardNumber)
		return nil
	})
	repo.EXPECT().FindByID(ctx, "card-1").Return(updated, nil)

	got, err := svc.UpdateCard(ctx, "user-1", card)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "card-1", got.ID)
	assert.Equal(t, "Updated Visa", got.Name)
	assert.Equal(t, "4111111111111112", got.CardNumber)
}

func TestCardsService_UpdateCard_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := &entity.Card{ID: "card-1", Name: "n", CardNumber: "4111111111111111", CardholderName: "J", ExpiryDate: "12/25", CVV: "123"}

	got, err := svc.UpdateCard(ctx, "", card)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_UpdateCard_NilCard(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	got, err := svc.UpdateCard(ctx, "user-1", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrInvalidCard))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_UpdateCard_EmptyCardID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardEntity()
	card.ID = ""

	got, err := svc.UpdateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_UpdateCard_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := &entity.Card{ID: "card-1", Name: "n", CardNumber: "4111111111111111", CardholderName: "J", ExpiryDate: "12/25", CVV: "123"}
	repo.EXPECT().FindByID(ctx, "card-1").Return(nil, models.ErrCardNotFound)

	got, err := svc.UpdateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}

func TestCardsService_UpdateCard_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	existing := validCardModel("card-1", "user-1")
	card := &entity.Card{ID: "card-1", Name: "n", CardNumber: "4111111111111111", CardholderName: "J", ExpiryDate: "12/25", CVV: "123"}
	repo.EXPECT().FindByID(ctx, "card-1").Return(existing, nil)

	got, err := svc.UpdateCard(ctx, "other-user", card)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}

func TestCardsService_UpdateCard_EmptyName(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	existing := validCardModel("card-1", "user-1")
	card := &entity.Card{ID: "card-1", Name: "", CardNumber: "4111111111111111", CardholderName: "J", ExpiryDate: "12/25", CVV: "123"}
	repo.EXPECT().FindByID(ctx, "card-1").Return(existing, nil)

	got, err := svc.UpdateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	// validateCard uses validation.Err() which returns gRPC status
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_UpdateCard_RepoUpdateError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	existing := validCardModel("card-1", "user-1")
	card := &entity.Card{ID: "card-1", Name: "n", CardNumber: "4111111111111111", CardholderName: "J", ExpiryDate: "12/25", CVV: "123"}
	repo.EXPECT().FindByID(ctx, "card-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("update failed"))

	got, err := svc.UpdateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestCardsService_UpdateCard_RepoUpdateErrNotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	existing := validCardModel("card-1", "user-1")
	card := &entity.Card{ID: "card-1", Name: "n", CardNumber: "4111111111111111", CardholderName: "J", ExpiryDate: "12/25", CVV: "123"}
	repo.EXPECT().FindByID(ctx, "card-1").Return(existing, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).Return(models.ErrCardNotFound)

	got, err := svc.UpdateCard(ctx, "user-1", card)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}

// --- DeleteCard ---

func TestCardsService_DeleteCard_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardModel("card-1", "user-1")
	repo.EXPECT().FindByID(ctx, "card-1").Return(card, nil)
	repo.EXPECT().Delete(ctx, "card-1").Return(nil)

	err := svc.DeleteCard(ctx, "user-1", "card-1")
	require.NoError(t, err)
}

func TestCardsService_DeleteCard_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	err := svc.DeleteCard(ctx, "", "card-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrInvalidUserID))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_DeleteCard_EmptyCardID(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	err := svc.DeleteCard(ctx, "user-1", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_DeleteCard_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	repo.EXPECT().FindByID(ctx, "card-1").Return(nil, models.ErrCardNotFound)

	err := svc.DeleteCard(ctx, "user-1", "card-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}

func TestCardsService_DeleteCard_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardModel("card-1", "user-1")
	repo.EXPECT().FindByID(ctx, "card-1").Return(card, nil)

	err := svc.DeleteCard(ctx, "other-user", "card-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
	repo.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
}

func TestCardsService_DeleteCard_RepoDeleteError(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardModel("card-1", "user-1")
	repo.EXPECT().FindByID(ctx, "card-1").Return(card, nil)
	repo.EXPECT().Delete(ctx, "card-1").Return(errors.New("delete failed"))

	err := svc.DeleteCard(ctx, "user-1", "card-1")
	require.Error(t, err)
}

func TestCardsService_DeleteCard_RepoDeleteErrNotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo := setupCardsService(t)

	card := validCardModel("card-1", "user-1")
	repo.EXPECT().FindByID(ctx, "card-1").Return(card, nil)
	repo.EXPECT().Delete(ctx, "card-1").Return(models.ErrCardNotFound)

	err := svc.DeleteCard(ctx, "user-1", "card-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrCardNotFound))
}
