package integration

import (
	"errors"
	"io"
	"os"
	"testing"

	gophkeepv1 "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBinariesService_UploadStream_SmallFile(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	filePath, checksum := GenerateTestFile(t, 1024*1024)

	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	stream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	metadata := "test metadata"
	err = stream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Small Test File",
				Filename:    "test_1mb.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Metadata:    &metadata,
				Checksum:    checksum,
			},
		},
	})
	require.NoError(t, err)

	buffer := make([]byte, 64*1024) // 64KB chunks
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		err = stream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	resp, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Binary.Id)
	assert.Equal(t, "Small Test File", resp.Binary.Name)
	assert.Equal(t, "test_1mb.bin", resp.Binary.Filename)
	assert.Equal(t, fileInfo.Size(), resp.Binary.Size)
	assert.Equal(t, checksum, resp.Binary.Checksum)
}

func TestBinariesService_UploadStream_LargeFile(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Generate a large test file (25MB - more than 20MB as requested)
	filePath, checksum := GenerateTestFile(t, 25*1024*1024)

	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	stream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	err = stream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Large Test File 25MB",
				Filename:    "test_25mb.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Checksum:    checksum,
			},
		},
	})
	require.NoError(t, err)

	// Send file data in chunks (1MB chunks for large files)
	buffer := make([]byte, 1024*1024)
	totalSent := int64(0)
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		err = stream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		totalSent += int64(n)
	}

	// Close and receive response
	resp, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Binary.Id)
	assert.Equal(t, "Large Test File 25MB", resp.Binary.Name)
	assert.Equal(t, fileInfo.Size(), resp.Binary.Size)
	assert.Equal(t, totalSent, resp.Binary.Size)
	assert.Equal(t, checksum, resp.Binary.Checksum)
}

func TestBinariesService_DownloadStream_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	filePath, originalChecksum := GenerateTestFile(t, 2*1024*1024) // 2MB
	file, err := os.Open(filePath)
	require.NoError(t, err)

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	uploadStream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Download Test",
				Filename:    "download_test.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Checksum:    originalChecksum,
			},
		},
	})
	require.NoError(t, err)

	buffer := make([]byte, 64*1024)
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	file.Close()

	uploadResp, err := uploadStream.CloseAndRecv()
	require.NoError(t, err)
	binaryID := uploadResp.Binary.Id

	downloadStream, err := client.DownloadStream(authCtx, &gophkeepv1.BinaryDownloadRequest{
		Id: binaryID,
	})
	require.NoError(t, err)

	downloadedData := make([]byte, 0)
	var downloadChecksum string
	for {
		chunk, err := downloadStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		downloadedData = append(downloadedData, chunk.ChunkData...)
		if chunk.Checksum != nil {
			downloadChecksum = *chunk.Checksum
		}
	}

	assert.Equal(t, fileInfo.Size(), int64(len(downloadedData)))
	assert.Equal(t, originalChecksum, downloadChecksum)
}

func TestBinariesService_Get_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Upload a file first
	filePath, checksum := GenerateTestFile(t, 512*1024) // 512KB
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	uploadStream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	metadata := "get test metadata"
	err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Get Metadata Test",
				Filename:    "metadata_test.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Metadata:    &metadata,
				Checksum:    checksum,
			},
		},
	})
	require.NoError(t, err)

	buffer := make([]byte, 64*1024)
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	uploadResp, err := uploadStream.CloseAndRecv()
	require.NoError(t, err)

	// Get metadata
	getResp, err := client.Get(authCtx, &gophkeepv1.BinaryGetRequest{
		Id: uploadResp.Binary.Id,
	})

	require.NoError(t, err)
	require.NotNil(t, getResp)
	assert.Equal(t, uploadResp.Binary.Id, getResp.Binary.Id)
	assert.Equal(t, "Get Metadata Test", getResp.Binary.Name)
	assert.Equal(t, "metadata_test.bin", getResp.Binary.Filename)
	assert.Equal(t, fileInfo.Size(), getResp.Binary.Size)
	assert.NotNil(t, getResp.Binary.Metadata)
	assert.Equal(t, metadata, *getResp.Binary.Metadata)
	assert.Equal(t, checksum, getResp.Binary.Checksum)
}

func TestBinariesService_List_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	fileCount := 3
	for i := range fileCount {
		filePath, checksum := GenerateTestFile(t, 100*1024) // 100KB
		file, err := os.Open(filePath)
		require.NoError(t, err)

		fileInfo, err := file.Stat()
		require.NoError(t, err)

		uploadStream, err := client.UploadStream(authCtx)
		require.NoError(t, err)

		err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_Metadata{
				Metadata: &gophkeepv1.BinaryMetadata{
					Name:        "List Test File " + string(rune('A'+i)),
					Filename:    "list_test.bin",
					ContentType: "application/octet-stream",
					TotalSize:   fileInfo.Size(),
					Checksum:    checksum,
				},
			},
		})
		require.NoError(t, err)

		buffer := make([]byte, 64*1024)
		for {
			n, err := file.Read(buffer)
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
				Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
					ChunkData: buffer[:n],
				},
			})
			require.NoError(t, err)
		}
		file.Close()

		_, err = uploadStream.CloseAndRecv()
		require.NoError(t, err)
	}

	listResp, err := client.List(authCtx, &gophkeepv1.BinaryListRequest{
		Page: &gophkeepv1.PageRequest{
			Limit:  10,
			Offset: 0,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, listResp)
	assert.GreaterOrEqual(t, len(listResp.Items), fileCount)
	assert.NotNil(t, listResp.Page)
	assert.GreaterOrEqual(t, listResp.Page.Total, uint32(fileCount))
}

func TestBinariesService_Update_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	filePath, checksum := GenerateTestFile(t, 256*1024)
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	uploadStream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Original Name",
				Filename:    "original.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Checksum:    checksum,
			},
		},
	})
	require.NoError(t, err)

	buffer := make([]byte, 64*1024)
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	uploadResp, err := uploadStream.CloseAndRecv()
	require.NoError(t, err)

	newMetadata := "updated metadata"
	updateResp, err := client.Update(authCtx, &gophkeepv1.BinaryUpdateRequest{
		Id:       uploadResp.Binary.Id,
		Name:     "Updated Name",
		Metadata: &newMetadata,
	})

	require.NoError(t, err)
	require.NotNil(t, updateResp)
	assert.Equal(t, "Updated Name", updateResp.Binary.Name)
	assert.NotNil(t, updateResp.Binary.Metadata)
	assert.Equal(t, newMetadata, *updateResp.Binary.Metadata)

	getResp, err := client.Get(authCtx, &gophkeepv1.BinaryGetRequest{
		Id: uploadResp.Binary.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", getResp.Binary.Name)
}

func TestBinariesService_Delete_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	filePath, checksum := GenerateTestFile(t, 128*1024)
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	uploadStream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "To Delete",
				Filename:    "delete_test.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Checksum:    checksum,
			},
		},
	})
	require.NoError(t, err)

	buffer := make([]byte, 64*1024)
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	uploadResp, err := uploadStream.CloseAndRecv()
	require.NoError(t, err)

	// Delete the binary
	deleteResp, err := client.Delete(authCtx, &gophkeepv1.BinaryDeleteRequest{
		Id: uploadResp.Binary.Id,
	})

	require.NoError(t, err)
	require.NotNil(t, deleteResp)
	assert.True(t, deleteResp.Deleted)

	// Verify file is deleted
	_, err = client.Get(authCtx, &gophkeepv1.BinaryGetRequest{
		Id: uploadResp.Binary.Id,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBinariesService_Upload_VeryLargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Generate a very large file (50MB)
	filePath, checksum := GenerateTestFile(t, 50*1024*1024)

	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fileInfo, err := file.Stat()
	require.NoError(t, err)

	uploadStream, err := client.UploadStream(authCtx)
	require.NoError(t, err)

	err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Very Large File 50MB",
				Filename:    "test_50mb.bin",
				ContentType: "application/octet-stream",
				TotalSize:   fileInfo.Size(),
				Checksum:    checksum,
			},
		},
	})
	require.NoError(t, err)

	// Use 1MB chunks for very large files
	buffer := make([]byte, 1024*1024)
	for {
		n, err := file.Read(buffer)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		err = uploadStream.Send(&gophkeepv1.BinaryUploadChunk{
			Data: &gophkeepv1.BinaryUploadChunk_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	resp, err := uploadStream.CloseAndRecv()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, fileInfo.Size(), resp.Binary.Size)
	assert.Equal(t, checksum, resp.Binary.Checksum)
}

func TestBinariesService_Unauthenticated(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedBinariesClient(t)
	defer conn.Close()

	// Try to upload without authentication
	stream, err := client.UploadStream(tc.Ctx)
	require.NoError(t, err)

	err = stream.Send(&gophkeepv1.BinaryUploadChunk{
		Data: &gophkeepv1.BinaryUploadChunk_Metadata{
			Metadata: &gophkeepv1.BinaryMetadata{
				Name:        "Unauthorized",
				Filename:    "test.bin",
				ContentType: "application/octet-stream",
				TotalSize:   1024,
				Checksum:    "0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
	})
	require.NoError(t, err)

	_, err = stream.CloseAndRecv()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}
