package integration

import (
	"testing"

	gophkeepv1 "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthService_Register_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	resp, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
		Email:    lo.ToPtr("test@example.com"),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.UserId)
	assert.Equal(t, username, resp.Username)
	assert.NotNil(t, resp.CreatedAt)
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	// Register first time
	_, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Try to register again with same username
	_, err = client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestAuthService_Register_InvalidUsername(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "username too short",
			username: "ab",
			password: "SecurePassword123",
		},
		{
			name:     "username with invalid characters",
			username: "user@invalid",
			password: "SecurePassword123",
		},
		{
			name:     "empty username",
			username: "",
			password: "SecurePassword123",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
				Username: testCase.username,
				Password: testCase.password,
			})

			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}

func TestAuthService_Register_InvalidPassword(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)

	// Password too short (less than 8 characters)
	_, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: "short",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAuthService_Token_PasswordGrant_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	// Register user first
	_, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Get token using password grant
	tokenResp, err := client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "test-device-123",
	})

	require.NoError(t, err)
	require.NotNil(t, tokenResp)
	assert.NotEmpty(t, tokenResp.AccessToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Greater(t, tokenResp.ExpiresIn, uint32(0))
}

func TestAuthService_Token_PasswordGrant_InvalidCredentials(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	// Register user first
	_, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Try to get token with wrong password
	_, err = client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  "WrongPassword123",
		ClientId:  "test-client",
		DeviceId:  "test-device-123",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthService_Token_PasswordGrant_NonExistentUser(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	// Try to get token for non-existent user
	_, err := client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  "nonexistent_user_12345",
		Password:  "SomePassword123",
		ClientId:  "test-client",
		DeviceId:  "test-device-123",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthService_Token_InvalidGrantType(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	// Try to use unspecified grant type
	_, err := client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_UNSPECIFIED,
		Username:  "testuser",
		Password:  "password",
		ClientId:  "test-client",
		DeviceId:  "test-device-123",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAuthService_FullAuthFlow(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	email := "fullflow@example.com"

	// Step 1: Register
	registerResp, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
		Email:    &email,
	})
	require.NoError(t, err)
	require.NotNil(t, registerResp)
	assert.NotEmpty(t, registerResp.UserId)

	// Step 2: Get initial token
	tokenResp, err := client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "test-device-flow",
	})
	require.NoError(t, err)
	require.NotNil(t, tokenResp)
	assert.NotEmpty(t, tokenResp.AccessToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)

	// Verify we can use the token (will be tested in user service tests)
	assert.Greater(t, tokenResp.ExpiresIn, uint32(0))
}

func TestAuthService_MultipleDevices(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedAuthClient(t)
	defer conn.Close()

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	// Register user
	_, err := client.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Login from device 1
	token1, err := client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "device-1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, token1.AccessToken)

	// Login from device 2
	token2, err := client.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "device-2",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, token2.AccessToken)

	// Tokens should be different
	assert.NotEqual(t, token1.AccessToken, token2.AccessToken)
}
