package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

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

func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func setupBinaryService(t *testing.T) (*BinaryService, *mocks.MockBinariesRepository, *FileService, string) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockBinariesRepository(ctrl)
	tempDir := t.TempDir()
	logger := zap.NewNop()

	fileService, err := NewFileService(tempDir, logger)
	require.NoError(t, err)

	svc := NewBinaryService(logger, repo, fileService, 10*1024*1024)
	return svc, repo, fileService, tempDir
}

// --- UploadStream ---

func TestBinaryService_UploadStream_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	content := []byte("test content")
	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.pdf",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		TotalSize:   int64(len(content)),
		Checksum:    calculateChecksum(content),
	}

	repo.EXPECT().FindByChecksum(ctx, "user-123", metadata.Checksum).Return(nil, models.ErrBinaryNotFound)
	repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, b *models.Binary) (*models.Binary, error) {
		b.ID = "binary-123"
		return b, nil
	})

	result, err := svc.UploadStream(ctx, metadata, bytes.NewReader(content))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "binary-123", result.Binary.ID)
	assert.Equal(t, "user-123", result.Binary.UserID)
	assert.Equal(t, int64(len(content)), result.BytesRead)
}

func TestBinaryService_UploadStream_ChecksumMismatch(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.txt",
		Filename:    "test.txt",
		ContentType: "text/plain",
		TotalSize:   4,
		Checksum:    "0000000000000000000000000000000000000000000000000000000000000000",
	}

	repo.EXPECT().FindByChecksum(ctx, "user-123", metadata.Checksum).Return(nil, models.ErrBinaryNotFound)

	_, err := svc.UploadStream(ctx, metadata, bytes.NewReader([]byte("test")))
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBinaryService_UploadStream_SizeMismatchReceivedMore(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	content := []byte("test")
	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.txt",
		Filename:    "test.txt",
		ContentType: "text/plain",
		TotalSize:   2,
		Checksum:    calculateChecksum(content),
	}

	repo.EXPECT().FindByChecksum(ctx, "user-123", metadata.Checksum).Return(nil, models.ErrBinaryNotFound)

	_, err := svc.UploadStream(ctx, metadata, bytes.NewReader(content))
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBinaryService_UploadStream_SizeMismatchReceivedLess(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	content := []byte("test")
	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.txt",
		Filename:    "test.txt",
		ContentType: "text/plain",
		TotalSize:   100,
		Checksum:    calculateChecksum(content),
	}

	repo.EXPECT().FindByChecksum(ctx, "user-123", metadata.Checksum).Return(nil, models.ErrBinaryNotFound)

	_, err := svc.UploadStream(ctx, metadata, bytes.NewReader(content))
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBinaryService_UploadStream_DeduplicationExistingChecksum(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	content := []byte("test")
	checksum := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.txt",
		Filename:    "test.txt",
		ContentType: "text/plain",
		TotalSize:   int64(len(content)),
		Checksum:    checksum,
	}

	existing := &models.Binary{
		ID:       "existing-binary-456",
		UserID:   "user-123",
		Name:     "existing.txt",
		Filename: "existing.txt",
		Checksum: checksum,
	}
	repo.EXPECT().FindByChecksum(ctx, "user-123", checksum).Return(existing, nil)

	result, err := svc.UploadStream(ctx, metadata, bytes.NewReader(content))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "existing-binary-456", result.Binary.ID)
	assert.Equal(t, int64(0), result.BytesRead)
}

func TestBinaryService_UploadStream_FindByChecksumError(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	content := []byte("test")
	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.txt",
		Filename:    "test.txt",
		ContentType: "text/plain",
		TotalSize:   int64(len(content)),
		Checksum:    calculateChecksum(content),
	}

	repo.EXPECT().FindByChecksum(ctx, "user-123", metadata.Checksum).Return(nil, errors.New("database error"))

	_, err := svc.UploadStream(ctx, metadata, bytes.NewReader(content))
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBinaryService_UploadStream_CreateError(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	content := []byte("test")
	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "test.txt",
		Filename:    "test.txt",
		ContentType: "text/plain",
		TotalSize:   int64(len(content)),
		Checksum:    calculateChecksum(content),
	}

	repo.EXPECT().FindByChecksum(ctx, "user-123", metadata.Checksum).Return(nil, models.ErrBinaryNotFound)
	repo.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("insert failed"))

	_, err := svc.UploadStream(ctx, metadata, bytes.NewReader(content))
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBinaryService_UploadStream_EmptyFileRejectedByValidation(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := setupBinaryService(t)

	metadata := &entity.StreamUploadRequest{
		UserID:      "user-123",
		Name:        "empty.txt",
		Filename:    "empty.txt",
		ContentType: "text/plain",
		TotalSize:   0,
		Checksum:    calculateChecksum([]byte("")),
	}

	// validateUploadRequest runs first and rejects TotalSize <= 0, so no repo calls
	_, err := svc.UploadStream(ctx, metadata, bytes.NewReader(nil))
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// --- DownloadStream ---

func TestBinaryService_DownloadStream_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, tempDir := setupBinaryService(t)

	content := []byte("test content")
	storagePath := "user-123/binary-123_test.txt"
	fullPath := filepath.Join(tempDir, storagePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, content, 0o644))

	binary := &models.Binary{
		ID:          "binary-123",
		UserID:      "user-123",
		Filename:    "test.txt",
		Size:        int64(len(content)),
		StoragePath: storagePath,
	}
	repo.EXPECT().FindByID(ctx, "binary-123").Return(binary, nil)

	var buf bytes.Buffer
	written, err := svc.DownloadStream(ctx, "user-123", "binary-123", &buf)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), written)
	assert.Equal(t, "test content", buf.String())
}

func TestBinaryService_DownloadStream_BinaryNotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByID(ctx, "non-existent").Return(nil, models.ErrBinaryNotFound)

	var buf bytes.Buffer
	_, err := svc.DownloadStream(ctx, "user-123", "non-existent", &buf)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBinaryService_DownloadStream_PermissionDeniedWrongUser(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{
		ID:       "binary-123",
		UserID:   "user-123",
		Filename: "test.txt",
	}
	repo.EXPECT().FindByID(ctx, "binary-123").Return(binary, nil)

	var buf bytes.Buffer
	_, err := svc.DownloadStream(ctx, "user-456", "binary-123", &buf)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestBinaryService_DownloadStream_DatabaseError(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByID(ctx, "binary-123").Return(nil, errors.New("db error"))

	var buf bytes.Buffer
	_, err := svc.DownloadStream(ctx, "user-123", "binary-123", &buf)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBinaryService_DownloadStream_FileNotFoundOnDisk(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{
		ID:          "binary-123",
		UserID:      "user-123",
		Filename:    "test.txt",
		StoragePath: "nonexistent/path.txt",
	}
	repo.EXPECT().FindByID(ctx, "binary-123").Return(binary, nil)

	var buf bytes.Buffer
	_, err := svc.DownloadStream(ctx, "user-123", "binary-123", &buf)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// --- GetBinary ---

func TestBinaryService_GetBinary_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{
		ID:       "bin-1",
		UserID:   "user-1",
		Name:     "file",
		Filename: "file.txt",
		Size:     100,
		Checksum: "abc",
	}
	repo.EXPECT().FindByID(ctx, "bin-1").Return(binary, nil)

	got, err := svc.GetBinary(ctx, "user-1", "bin-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "bin-1", got.ID)
	assert.Equal(t, "user-1", got.UserID)
}

func TestBinaryService_GetBinary_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByID(ctx, "bin-1").Return(nil, models.ErrBinaryNotFound)

	got, err := svc.GetBinary(ctx, "user-1", "bin-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBinaryService_GetBinary_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{ID: "bin-1", UserID: "user-1"}
	repo.EXPECT().FindByID(ctx, "bin-1").Return(binary, nil)

	got, err := svc.GetBinary(ctx, "other-user", "bin-1")
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

// --- ListBinaries ---

func TestBinaryService_ListBinaries_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	list := []*models.Binary{
		{ID: "b1", UserID: "u1", Name: "a", Filename: "a.txt", Size: 10},
	}
	repo.EXPECT().FindByUserID(ctx, "u1", 10, 0).Return(list, nil)
	repo.EXPECT().CountByUserID(ctx, "u1").Return(int64(1), nil)

	items, count, err := svc.ListBinaries(ctx, "u1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, "b1", items[0].ID)
}

func TestBinaryService_ListBinaries_FindByUserIDError(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByUserID(ctx, "u1", 10, 0).Return(nil, errors.New("db error"))

	_, _, err := svc.ListBinaries(ctx, "u1", 10, 0)
	require.Error(t, err)
}

func TestBinaryService_ListBinaries_CountByUserIDError(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByUserID(ctx, "u1", 10, 0).Return(nil, nil)
	repo.EXPECT().CountByUserID(ctx, "u1").Return(int64(0), errors.New("count error"))

	_, _, err := svc.ListBinaries(ctx, "u1", 10, 0)
	require.Error(t, err)
}

// --- UpdateBinary ---

func TestBinaryService_UpdateBinary_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{
		ID:       "bin-1",
		UserID:   "user-1",
		Name:     "old",
		Filename: "f.txt",
	}
	meta := "new-meta"
	repo.EXPECT().FindByID(ctx, "bin-1").Return(binary, nil)
	repo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, b *models.Binary) error {
		assert.Equal(t, "new-name", b.Name)
		assert.Equal(t, &meta, b.Metadata)
		return nil
	})

	got, err := svc.UpdateBinary(ctx, "user-1", "bin-1", "new-name", &meta)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "bin-1", got.ID)
	assert.Equal(t, "new-name", got.Name)
}

func TestBinaryService_UpdateBinary_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByID(ctx, "bin-1").Return(nil, models.ErrBinaryNotFound)

	got, err := svc.UpdateBinary(ctx, "user-1", "bin-1", "name", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBinaryService_UpdateBinary_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{ID: "bin-1", UserID: "user-1"}
	repo.EXPECT().FindByID(ctx, "bin-1").Return(binary, nil)

	got, err := svc.UpdateBinary(ctx, "other-user", "bin-1", "name", nil)
	require.Error(t, err)
	assert.Nil(t, got)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

// --- DeleteBinary ---

func TestBinaryService_DeleteBinary_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, tempDir := setupBinaryService(t)

	storagePath := "user-1/bin-1_file.txt"
	fullPath := filepath.Join(tempDir, storagePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte("x"), 0o644))

	binary := &models.Binary{
		ID:          "bin-1",
		UserID:      "user-1",
		StoragePath: storagePath,
	}
	repo.EXPECT().FindByID(ctx, "bin-1").Return(binary, nil)
	repo.EXPECT().Delete(ctx, "bin-1").Return(nil)

	err := svc.DeleteBinary(ctx, "user-1", "bin-1")
	require.NoError(t, err)
	_, err = os.Stat(fullPath)
	require.True(t, os.IsNotExist(err))
}

func TestBinaryService_DeleteBinary_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	repo.EXPECT().FindByID(ctx, "bin-1").Return(nil, models.ErrBinaryNotFound)

	err := svc.DeleteBinary(ctx, "user-1", "bin-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBinaryService_DeleteBinary_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setupBinaryService(t)

	binary := &models.Binary{ID: "bin-1", UserID: "user-1"}
	repo.EXPECT().FindByID(ctx, "bin-1").Return(binary, nil)

	err := svc.DeleteBinary(ctx, "other-user", "bin-1")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

// --- ValidateUploadRequest ---

func TestBinaryService_ValidateUploadRequest(t *testing.T) {
	_, repo, _, _ := setupBinaryService(t)
	// Use a minimal service just for validation; repo not used in validateUploadRequest
	logger := zap.NewNop()
	tempDir := t.TempDir()
	fs, _ := NewFileService(tempDir, logger)
	svc := NewBinaryService(logger, repo, fs, 1024*1024)

	validChecksum := "1234567890123456789012345678901234567890123456789012345678901234"

	tests := []struct {
		name        string
		req         *entity.StreamUploadRequest
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "valid request",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
				Checksum:    validChecksum,
			},
			wantErr: false,
		},
		{
			name: "missing user id",
			req: &entity.StreamUploadRequest{
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
				Checksum:    validChecksum,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing name",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
				Checksum:    validChecksum,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing filename",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
				Checksum:    validChecksum,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing content type",
			req: &entity.StreamUploadRequest{
				UserID:    "user-123",
				Name:      "test.txt",
				Filename:  "test.txt",
				TotalSize: 100,
				Checksum:  validChecksum,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "zero size",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   0,
				Checksum:    validChecksum,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "file too large",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   10 * 1024 * 1024,
				Checksum:    validChecksum,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing checksum",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "invalid checksum format",
			req: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
				Checksum:    "short",
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateUploadRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != codes.OK {
					st, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, st.Code())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
