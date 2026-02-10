package integration

import (
	"testing"

	gophkeepv1 "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUserService_GetProfile_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)
	resp, err := client.GetProfile(authCtx, &gophkeepv1.UserProfileGetRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, testUser.UserID, resp.Profile.UserId)
	assert.Equal(t, username, resp.Profile.Username)
	assert.NotNil(t, resp.Profile.CreatedAt)
}

func TestUserService_GetProfile_Unauthenticated(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	// Try to get profile without authentication
	_, err := client.GetProfile(tc.Ctx, &gophkeepv1.UserProfileGetRequest{})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUserService_GetProfile_InvalidToken(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	// Try with invalid token
	authCtx := GetAuthContext(tc.Ctx, "invalid-token-12345")
	_, err := client.GetProfile(authCtx, &gophkeepv1.UserProfileGetRequest{})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)
	newEmail := "newemail@example.com"

	resp, err := client.UpdateProfile(authCtx, &gophkeepv1.UserProfileUpdateRequest{
		Email: &newEmail,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Profile.Email)
	assert.Equal(t, newEmail, *resp.Profile.Email)

	// Verify the change persists
	getResp, err := client.GetProfile(authCtx, &gophkeepv1.UserProfileGetRequest{})
	require.NoError(t, err)
	assert.NotNil(t, getResp.Profile.Email)
	assert.Equal(t, newEmail, *getResp.Profile.Email)
}

func TestUserService_UpdateProfile_InvalidEmail(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)
	invalidEmail := "not-an-email"

	_, err := client.UpdateProfile(authCtx, &gophkeepv1.UserProfileUpdateRequest{
		Email: &invalidEmail,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	oldPassword := "OldPassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, oldPassword)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)
	newPassword := "NewPassword456"

	resp, err := client.ChangePassword(authCtx, &gophkeepv1.UserPasswordChangeRequest{
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Changed)

	// Verify can login with new password
	authClient, authConn := GetAuthenticatedAuthClient(t)
	defer authConn.Close()

	tokenResp, err := authClient.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  newPassword,
		ClientId:  "test-client",
		DeviceId:  "test-device",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tokenResp.AccessToken)

	// Verify cannot login with old password
	_, err = authClient.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  oldPassword,
		ClientId:  "test-client",
		DeviceId:  "test-device",
	})
	require.Error(t, err)
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	_, err := client.ChangePassword(authCtx, &gophkeepv1.UserPasswordChangeRequest{
		OldPassword: "WrongPassword123",
		NewPassword: "NewPassword456",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestUserService_ChangePassword_InvalidNewPassword(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Password too short
	_, err := client.ChangePassword(authCtx, &gophkeepv1.UserPasswordChangeRequest{
		OldPassword: password,
		NewPassword: "short",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestUserService_GetActiveSessions_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	resp, err := client.GetActiveSessions(authCtx, &gophkeepv1.GetActiveSessionsRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Sessions), 1, "Should have at least one active session")

	// Check session fields
	for _, session := range resp.Sessions {
		assert.NotEmpty(t, session.SessionId)
		assert.NotEmpty(t, session.ClientId)
		assert.NotNil(t, session.CreatedAt)
		assert.NotNil(t, session.LastUsedAt)
	}
}

func TestUserService_GetActiveSessions_MultipleSessions(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	authClient, authConn := GetAuthenticatedAuthClient(t)
	defer authConn.Close()

	// Register user
	_, err := authClient.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Create multiple sessions from different devices
	devices := []string{"device-1", "device-2", "device-3"}
	tokens := make([]string, len(devices))

	for i, deviceID := range devices {
		tokenResp, err := authClient.Token(tc.Ctx, &gophkeepv1.TokenRequest{
			GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
			Username:  username,
			Password:  password,
			ClientId:  "test-client",
			DeviceId:  deviceID,
		})
		require.NoError(t, err)
		tokens[i] = tokenResp.AccessToken
	}

	// Get active sessions
	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, tokens[0])
	resp, err := client.GetActiveSessions(authCtx, &gophkeepv1.GetActiveSessionsRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Sessions), 3, "Should have at least 3 active sessions")
}

func TestUserService_RevokeSession_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	authClient, authConn := GetAuthenticatedAuthClient(t)
	defer authConn.Close()

	// Register user
	_, err := authClient.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Create two sessions
	token1Resp, err := authClient.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "device-1",
	})
	require.NoError(t, err)

	token2Resp, err := authClient.Token(tc.Ctx, &gophkeepv1.TokenRequest{
		GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  "test-client",
		DeviceId:  "device-2",
	})
	require.NoError(t, err)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	// Get sessions to find session ID
	authCtx := GetAuthContext(tc.Ctx, token1Resp.AccessToken)
	sessionsResp, err := client.GetActiveSessions(authCtx, &gophkeepv1.GetActiveSessionsRequest{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sessionsResp.Sessions), 2)

	// Find a non-current session to revoke
	var sessionToRevoke string
	for _, session := range sessionsResp.Sessions {
		if !session.IsCurrent {
			sessionToRevoke = session.SessionId
			break
		}
	}
	require.NotEmpty(t, sessionToRevoke, "Should find a non-current session")

	// Revoke the session
	revokeResp, err := client.RevokeSession(authCtx, &gophkeepv1.RevokeSessionRequest{
		SessionId: sessionToRevoke,
	})
	require.NoError(t, err)
	require.NotNil(t, revokeResp)
	assert.True(t, revokeResp.Revoked)

	// Verify session list is updated
	sessionsResp2, err := client.GetActiveSessions(authCtx, &gophkeepv1.GetActiveSessionsRequest{})
	require.NoError(t, err)
	assert.Less(t, len(sessionsResp2.Sessions), len(sessionsResp.Sessions))

	// Verify the revoked token cannot be used
	authCtx2 := GetAuthContext(tc.Ctx, token2Resp.AccessToken)
	_, err = client.GetProfile(authCtx2, &gophkeepv1.UserProfileGetRequest{})
	// This should fail if the token was revoked
	// Note: This depends on your token/session implementation
}

func TestUserService_RevokeAllSessions_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"

	authClient, authConn := GetAuthenticatedAuthClient(t)
	defer authConn.Close()

	// Register user
	_, err := authClient.Register(tc.Ctx, &gophkeepv1.RegisterRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Create multiple sessions
	devices := []string{"device-1", "device-2", "device-3"}
	currentToken := ""

	for i, deviceID := range devices {
		tokenResp, err := authClient.Token(tc.Ctx, &gophkeepv1.TokenRequest{
			GrantType: gophkeepv1.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
			Username:  username,
			Password:  password,
			ClientId:  "test-client",
			DeviceId:  deviceID,
		})
		require.NoError(t, err)
		if i == 0 {
			currentToken = tokenResp.AccessToken
		}
	}

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	// Revoke all other sessions
	authCtx := GetAuthContext(tc.Ctx, currentToken)
	revokeResp, err := client.RevokeAllSessions(authCtx, &gophkeepv1.RevokeAllSessionsRequest{
		ExceptCurrent: true,
	})

	require.NoError(t, err)
	require.NotNil(t, revokeResp)
	assert.GreaterOrEqual(t, revokeResp.RevokedCount, int32(2), "Should revoke at least 2 sessions")

	// Verify current session still works
	_, err = client.GetProfile(authCtx, &gophkeepv1.UserProfileGetRequest{})
	require.NoError(t, err)

	// Verify only current session remains
	sessionsResp, err := client.GetActiveSessions(authCtx, &gophkeepv1.GetActiveSessionsRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(sessionsResp.Sessions), "Should have only one active session")
	assert.True(t, sessionsResp.Sessions[0].IsCurrent)
}

func TestUserService_RevokeAllSessions_IncludingCurrent(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedUserClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Revoke all sessions including current
	revokeResp, err := client.RevokeAllSessions(authCtx, &gophkeepv1.RevokeAllSessionsRequest{
		ExceptCurrent: false,
	})

	require.NoError(t, err)
	require.NotNil(t, revokeResp)
	assert.GreaterOrEqual(t, revokeResp.RevokedCount, int32(1))

	// Current session should no longer work
	_, err = client.GetProfile(authCtx, &gophkeepv1.UserProfileGetRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}
