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

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockBinaryRepository is a mock implementation of BinaryRepository
type MockBinaryRepository struct {
	createFn         func(ctx context.Context, binary *models.Binary) (*models.Binary, error)
	findByIDFn       func(ctx context.Context, id string) (*models.Binary, error)
	findByUserIDFn   func(ctx context.Context, userID string, limit, offset int) ([]*models.Binary, error)
	updateFn         func(ctx context.Context, binary *models.Binary) error
	deleteFn         func(ctx context.Context, id string) error
	countByUserIDFn  func(ctx context.Context, userID string) (int64, error)
	findByChecksumFn func(ctx context.Context, userID, checksum string) (*models.Binary, error)
}

func (m *MockBinaryRepository) Create(ctx context.Context, binary *models.Binary) (*models.Binary, error) {
	if m.createFn != nil {
		return m.createFn(ctx, binary)
	}
	return binary, nil
}

func (m *MockBinaryRepository) FindByID(ctx context.Context, id string) (*models.Binary, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrBinaryNotFound
}

func (m *MockBinaryRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Binary, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (m *MockBinaryRepository) Update(ctx context.Context, binary *models.Binary) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, binary)
	}
	return nil
}

func (m *MockBinaryRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *MockBinaryRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	if m.countByUserIDFn != nil {
		return m.countByUserIDFn(ctx, userID)
	}
	return 0, nil
}

func (m *MockBinaryRepository) FindByChecksum(ctx context.Context, userID, checksum string) (*models.Binary, error) {
	if m.findByChecksumFn != nil {
		return m.findByChecksumFn(ctx, userID, checksum)
	}
	return nil, models.ErrBinaryNotFound
}

// Helper function to calculate checksum
func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func TestBinaryService_UploadStream(t *testing.T) {
	tests := []struct {
		name        string
		metadata    *entity.StreamUploadRequest
		fileContent []byte
		setupMocks  func(*MockBinaryRepository)
		wantErr     bool
		wantErrCode codes.Code
		checkResult func(*testing.T, *entity.StreamUploadResult)
	}{
		{
			name: "successful upload",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.pdf",
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				TotalSize:   12,
				Checksum:    calculateChecksum([]byte("test content")),
			},
			fileContent: []byte("test content"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
				repo.createFn = func(ctx context.Context, binary *models.Binary) (*models.Binary, error) {
					binary.ID = "binary-123"
					return binary, nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *entity.StreamUploadResult) {
				assert.Equal(t, "binary-123", result.Binary.ID)
				assert.Equal(t, "user-123", result.Binary.UserID)
				assert.Equal(t, int64(12), result.BytesRead)
			},
		},
		{
			name: "checksum mismatch",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   4,
				Checksum:    "0000000000000000000000000000000000000000000000000000000000000000",
			},
			fileContent: []byte("test"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "size mismatch - received more",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   2,
				Checksum:    calculateChecksum([]byte("test")),
			},
			fileContent: []byte("test"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "size mismatch - received less",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   100,
				Checksum:    calculateChecksum([]byte("test")),
			},
			fileContent: []byte("test"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "deduplication - file with same checksum exists",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   4,
				Checksum:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			fileContent: []byte("test"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return &models.Binary{
						ID:       "existing-binary-456",
						UserID:   "user-123",
						Name:     "existing.txt",
						Filename: "existing.txt",
						Checksum: checksum,
					}, nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *entity.StreamUploadResult) {
				assert.Equal(t, "existing-binary-456", result.Binary.ID)
				assert.Equal(t, int64(0), result.BytesRead, "should skip upload if file exists")
			},
		},
		{
			name: "database error during checksum check",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   4,
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
			},
			fileContent: []byte("test"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, errors.New("database connection failed")
				}
			},
			wantErr:     true,
			wantErrCode: codes.Internal,
		},
		{
			name: "database error during create",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "test.txt",
				Filename:    "test.txt",
				ContentType: "text/plain",
				TotalSize:   4,
				Checksum:    calculateChecksum([]byte("test")),
			},
			fileContent: []byte("test"),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
				repo.createFn = func(ctx context.Context, binary *models.Binary) (*models.Binary, error) {
					return nil, errors.New("database insert failed")
				}
			},
			wantErr:     true,
			wantErrCode: codes.Internal,
		},
		{
			name: "empty file",
			metadata: &entity.StreamUploadRequest{
				UserID:      "user-123",
				Name:        "empty.txt",
				Filename:    "empty.txt",
				ContentType: "text/plain",
				TotalSize:   0,
				Checksum:    calculateChecksum([]byte("")),
			},
			fileContent: []byte(""),
			setupMocks: func(repo *MockBinaryRepository) {
				repo.findByChecksumFn = func(ctx context.Context, userID, checksum string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tempDir := t.TempDir()
			logger := zap.NewNop()

			fileService, err := NewFileService(tempDir, logger)
			require.NoError(t, err)

			mockRepo := &MockBinaryRepository{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo)
			}

			service := NewBinaryService(logger, mockRepo, fileService, 10*1024*1024)

			reader := bytes.NewReader(tt.fileContent)

			// Act
			result, err := service.UploadStream(context.Background(), tt.metadata, reader)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != codes.OK {
					st, ok := status.FromError(err)
					require.True(t, ok, "error should be a gRPC status error")
					assert.Equal(t, tt.wantErrCode, st.Code())
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestBinaryService_DownloadStream(t *testing.T) {
	tests := []struct {
		name         string
		binaryID     string
		userID       string
		setupMocks   func(*MockBinaryRepository, string)
		setupFile    func(string) []byte
		wantErr      bool
		wantErrCode  codes.Code
		checkContent func(*testing.T, []byte)
	}{
		{
			name:     "successful download",
			binaryID: "binary-123",
			userID:   "user-123",
			setupMocks: func(repo *MockBinaryRepository, storagePath string) {
				repo.findByIDFn = func(ctx context.Context, id string) (*models.Binary, error) {
					return &models.Binary{
						ID:          "binary-123",
						UserID:      "user-123",
						Filename:    "test.txt",
						Size:        12,
						StoragePath: storagePath,
					}, nil
				}
			},
			setupFile: func(storagePath string) []byte {
				content := []byte("test content")
				return content
			},
			wantErr: false,
			checkContent: func(t *testing.T, content []byte) {
				assert.Equal(t, "test content", string(content))
			},
		},
		{
			name:     "binary not found",
			binaryID: "non-existent",
			userID:   "user-123",
			setupMocks: func(repo *MockBinaryRepository, storagePath string) {
				repo.findByIDFn = func(ctx context.Context, id string) (*models.Binary, error) {
					return nil, models.ErrBinaryNotFound
				}
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name:     "permission denied - wrong user",
			binaryID: "binary-123",
			userID:   "user-456",
			setupMocks: func(repo *MockBinaryRepository, storagePath string) {
				repo.findByIDFn = func(ctx context.Context, id string) (*models.Binary, error) {
					return &models.Binary{
						ID:          "binary-123",
						UserID:      "user-123",
						Filename:    "test.txt",
						StoragePath: storagePath,
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name:     "database error",
			binaryID: "binary-123",
			userID:   "user-123",
			setupMocks: func(repo *MockBinaryRepository, storagePath string) {
				repo.findByIDFn = func(ctx context.Context, id string) (*models.Binary, error) {
					return nil, errors.New("database connection failed")
				}
			},
			wantErr:     true,
			wantErrCode: codes.Internal,
		},
		{
			name:     "file not found on disk",
			binaryID: "binary-123",
			userID:   "user-123",
			setupMocks: func(repo *MockBinaryRepository, storagePath string) {
				repo.findByIDFn = func(ctx context.Context, id string) (*models.Binary, error) {
					return &models.Binary{
						ID:          "binary-123",
						UserID:      "user-123",
						Filename:    "test.txt",
						StoragePath: "nonexistent/path.txt",
					}, nil
				}
			},
			wantErr:     true,
			wantErrCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tempDir := t.TempDir()
			logger := zap.NewNop()

			fileService, err := NewFileService(tempDir, logger)
			require.NoError(t, err)

			// Setup file if needed
			var storagePath string
			if tt.setupFile != nil {
				storagePath = filepath.Join("user-123", "test.txt")
				fullPath := filepath.Join(tempDir, storagePath)
				err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				content := tt.setupFile(storagePath)
				err = os.WriteFile(fullPath, content, 0o644)
				require.NoError(t, err)
			}

			mockRepo := &MockBinaryRepository{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo, storagePath)
			}

			service := NewBinaryService(logger, mockRepo, fileService, 10*1024*1024)

			var buf bytes.Buffer

			// Act
			written, err := service.DownloadStream(context.Background(), tt.userID, tt.binaryID, &buf)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != codes.OK {
					st, ok := status.FromError(err)
					require.True(t, ok, "error should be a gRPC status error")
					assert.Equal(t, tt.wantErrCode, st.Code())
				}
				return
			}

			require.NoError(t, err)
			assert.Greater(t, written, int64(0))

			if tt.checkContent != nil {
				tt.checkContent(t, buf.Bytes())
			}
		})
	}
}

func TestBinaryService_ValidateUploadRequest(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	fileService, _ := NewFileService(tempDir, logger)
	mockRepo := &MockBinaryRepository{}
	service := NewBinaryService(logger, mockRepo, fileService, 1024*1024)

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
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
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
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
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
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
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
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
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
				Checksum:  "1234567890123456789012345678901234567890123456789012345678901234",
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
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
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
				Checksum:    "1234567890123456789012345678901234567890123456789012345678901234",
			},
			wantErr:     true,
			wantErrCode: codes.ResourceExhausted,
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
			err := service.validateUploadRequest(tt.req)

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
