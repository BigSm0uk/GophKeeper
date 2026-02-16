package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gophkeepv1 "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TestUser holds user credentials and tokens
type TestUser struct {
	Username     string
	Password     string
	UserID       string
	AccessToken  string
	RefreshToken string
}

// CreateTestUser registers a new test user and returns authentication tokens
func CreateTestUser(t *testing.T, ctx context.Context, username, password string) *TestUser {
	t.Helper()

	address := SetupTestContainer(t).GRPCAddress

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create gRPC client")
	defer conn.Close()

	authClient := gophkeepv1.NewAuthServiceClient(conn)

	// Register user
	registerResp, err := authClient.Register(ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err, "Failed to register user")
	require.NotNil(t, registerResp)

	// Login to get tokens
	tokenResp, err := authClient.Token(ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "test-device",
	})
	require.NoError(t, err, "Failed to get token")
	require.NotNil(t, tokenResp)

	return &TestUser{
		Username:    username,
		Password:    password,
		UserID:      registerResp.UserId,
		AccessToken: tokenResp.AccessToken,
	}
}

// GetAuthContext returns a context with authentication metadata
func GetAuthContext(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

// GetAuthenticatedAuthClient returns an auth client configured with authentication
func GetAuthenticatedAuthClient(t *testing.T) (gophkeepv1.AuthServiceClient, *grpc.ClientConn) {
	t.Helper()
	address := SetupTestContainer(t).GRPCAddress

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create gRPC client")

	return gophkeepv1.NewAuthServiceClient(conn), conn
}

// GetAuthenticatedUserClient returns a user client configured with authentication
func GetAuthenticatedUserClient(t *testing.T) (gophkeepv1.UserServiceClient, *grpc.ClientConn) {
	t.Helper()
	address := SetupTestContainer(t).GRPCAddress

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create gRPC client")

	return gophkeepv1.NewUserServiceClient(conn), conn
}

// GetAuthenticatedCredentialsClient returns a credentials client configured with authentication
func GetAuthenticatedCredentialsClient(t *testing.T) (gophkeepv1.CredentialsServiceClient, *grpc.ClientConn) {
	t.Helper()
	address := SetupTestContainer(t).GRPCAddress

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create gRPC client")

	return gophkeepv1.NewCredentialsServiceClient(conn), conn
}

// GetAuthenticatedBinariesClient returns a binaries client configured with authentication
func GetAuthenticatedBinariesClient(t *testing.T) (gophkeepv1.BinariesServiceClient, *grpc.ClientConn) {
	t.Helper()
	address := SetupTestContainer(t).GRPCAddress

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to create gRPC client")

	return gophkeepv1.NewBinariesServiceClient(conn), conn
}

// GenerateRandomUsername generates a random username for testing
func GenerateRandomUsername(t *testing.T) string {
	t.Helper()

	b := make([]byte, 8)
	_, err := rand.Read(b)
	require.NoError(t, err)

	return fmt.Sprintf("user_%x", b)
}

// GenerateTestFile creates a test file with specified size in bytes
// Returns the file path and SHA256 checksum
func GenerateTestFile(t *testing.T, sizeBytes int64) (filePath, checksum string) {
	t.Helper()

	dir := t.TempDir()
	filePath = filepath.Join(dir, fmt.Sprintf("testfile_%d.bin", sizeBytes))

	file, err := os.Create(filePath)
	require.NoError(t, err, "Failed to create test file")
	defer file.Close()

	// Create hash for checksum calculation
	hash := sha256.New()

	// Generate random data in chunks to avoid memory issues with large files
	const chunkSize = 1024 * 1024 // 1MB chunks
	buffer := make([]byte, chunkSize)

	remaining := sizeBytes
	for remaining > 0 {
		writeSize := chunkSize
		if remaining < int64(chunkSize) {
			writeSize = int(remaining)
			buffer = buffer[:writeSize]
		}

		// Fill buffer with random data
		_, err := rand.Read(buffer)
		require.NoError(t, err, "Failed to generate random data")

		// Write to file
		n, err := file.Write(buffer)
		require.NoError(t, err, "Failed to write to file")
		require.Equal(t, writeSize, n, "Written size mismatch")

		// Update hash
		_, err = hash.Write(buffer)
		require.NoError(t, err, "Failed to update hash")

		remaining -= int64(n)
	}

	checksum = hex.EncodeToString(hash.Sum(nil))
	return filePath, checksum
}

// CalculateFileChecksum calculates SHA256 checksum for a file
func CalculateFileChecksum(t *testing.T, filePath string) string {
	t.Helper()

	file, err := os.Open(filePath)
	require.NoError(t, err, "Failed to open file")
	defer file.Close()

	hash := sha256.New()
	buffer := make([]byte, 1024*1024) // 1MB buffer

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			_, hashErr := hash.Write(buffer[:n])
			require.NoError(t, hashErr, "Failed to update hash")
		}
		if err != nil {
			break
		}
	}

	return hex.EncodeToString(hash.Sum(nil))
}
