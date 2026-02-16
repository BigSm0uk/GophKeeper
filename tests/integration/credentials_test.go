package integration

import (
	"fmt"
	"testing"

	gophkeepv1 "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCredentialsService_Create_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	url := "https://example.com"
	metadata := "some metadata"

	resp, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
		Name:     "Test Credential",
		Login:    "testlogin",
		Password: "testpassword",
		Url:      &url,
		Metadata: &metadata,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Credential.Id)
	assert.Equal(t, "Test Credential", resp.Credential.Name)
	assert.Equal(t, "testlogin", resp.Credential.Login)
	assert.Equal(t, "testpassword", resp.Credential.Password)
	assert.NotNil(t, resp.Credential.Url)
	assert.Equal(t, url, *resp.Credential.Url)
	assert.NotNil(t, resp.Credential.Metadata)
	assert.Equal(t, metadata, *resp.Credential.Metadata)
	assert.NotNil(t, resp.Credential.CreatedAt)
	assert.NotNil(t, resp.Credential.UpdatedAt)
}

func TestCredentialsService_Create_Unauthenticated(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	_, err := client.Create(tc.Ctx, &gophkeepv1.CredentialCreateRequest{
		Name:     "Test Credential",
		Login:    "testlogin",
		Password: "testpassword",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestCredentialsService_Create_InvalidData(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	testCases := []struct {
		name    string
		request *gophkeepv1.CredentialCreateRequest
	}{
		{
			name: "empty name",
			request: &gophkeepv1.CredentialCreateRequest{
				Name:     "",
				Login:    "testlogin",
				Password: "testpassword",
			},
		},
		{
			name: "empty login",
			request: &gophkeepv1.CredentialCreateRequest{
				Name:     "Test",
				Login:    "",
				Password: "testpassword",
			},
		},
		{
			name: "empty password",
			request: &gophkeepv1.CredentialCreateRequest{
				Name:     "Test",
				Login:    "testlogin",
				Password: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.Create(authCtx, tc.request)

			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}

func TestCredentialsService_Get_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Create a credential
	createResp, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
		Name:     "Get Test",
		Login:    "getlogin",
		Password: "getpassword",
	})
	require.NoError(t, err)

	// Get the credential
	getResp, err := client.Get(authCtx, &gophkeepv1.CredentialGetRequest{
		Id: createResp.Credential.Id,
	})

	require.NoError(t, err)
	require.NotNil(t, getResp)
	assert.Equal(t, createResp.Credential.Id, getResp.Credential.Id)
	assert.Equal(t, "Get Test", getResp.Credential.Name)
	assert.Equal(t, "getlogin", getResp.Credential.Login)
	assert.Equal(t, "getpassword", getResp.Credential.Password)
}

func TestCredentialsService_Get_NotFound(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	_, err := client.Get(authCtx, &gophkeepv1.CredentialGetRequest{
		Id: "00000000-0000-0000-0000-000000000000",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_Get_OtherUserCredential(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	// Create first user and credential
	user1 := CreateTestUser(t, tc.Ctx, GenerateRandomUsername(t), "Password123")
	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx1 := GetAuthContext(tc.Ctx, user1.AccessToken)
	createResp, err := client.Create(authCtx1, &gophkeepv1.CredentialCreateRequest{
		Name:     "User1 Credential",
		Login:    "user1login",
		Password: "user1password",
	})
	require.NoError(t, err)

	// Create second user
	user2 := CreateTestUser(t, tc.Ctx, GenerateRandomUsername(t), "Password456")
	authCtx2 := GetAuthContext(tc.Ctx, user2.AccessToken)

	// User2 tries to access User1's credential
	_, err = client.Get(authCtx2, &gophkeepv1.CredentialGetRequest{
		Id: createResp.Credential.Id,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_List_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Create multiple credentials
	credNames := []string{"Credential 1", "Credential 2", "Credential 3"}
	for _, name := range credNames {
		_, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
			Name:     name,
			Login:    "login",
			Password: "password",
		})
		require.NoError(t, err)
	}

	// List credentials
	resp, err := client.List(authCtx, &gophkeepv1.CredentialListRequest{
		Page: &gophkeepv1.PageRequest{
			Limit:  10,
			Offset: 0,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Items), 3)
	assert.NotNil(t, resp.Page)
	assert.GreaterOrEqual(t, resp.Page.Total, uint32(3))
}

func TestCredentialsService_List_Pagination(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Create 5 credentials
	for i := 0; i < 5; i++ {
		_, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
			Name:     fmt.Sprintf("Credential %d", i),
			Login:    "login",
			Password: "password",
		})
		require.NoError(t, err)
	}

	// Get first page
	page1, err := client.List(authCtx, &gophkeepv1.CredentialListRequest{
		Page: &gophkeepv1.PageRequest{
			Limit:  2,
			Offset: 0,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, len(page1.Items))

	// Get second page
	page2, err := client.List(authCtx, &gophkeepv1.CredentialListRequest{
		Page: &gophkeepv1.PageRequest{
			Limit:  2,
			Offset: 2,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, len(page2.Items))

	// Verify different items
	assert.NotEqual(t, page1.Items[0].Id, page2.Items[0].Id)
}

func TestCredentialsService_Update_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Create credential
	createResp, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
		Name:     "Original Name",
		Login:    "originallogin",
		Password: "originalpassword",
	})
	require.NoError(t, err)

	// Update credential
	newUrl := "https://updated.com"
	newMetadata := "updated metadata"
	updateResp, err := client.Update(authCtx, &gophkeepv1.CredentialUpdateRequest{
		Id:       createResp.Credential.Id,
		Name:     "Updated Name",
		Login:    "updatedlogin",
		Password: "updatedpassword",
		Url:      &newUrl,
		Metadata: &newMetadata,
	})

	require.NoError(t, err)
	require.NotNil(t, updateResp)
	assert.Equal(t, createResp.Credential.Id, updateResp.Credential.Id)
	assert.Equal(t, "Updated Name", updateResp.Credential.Name)
	assert.Equal(t, "updatedlogin", updateResp.Credential.Login)
	assert.Equal(t, "updatedpassword", updateResp.Credential.Password)
	assert.NotNil(t, updateResp.Credential.Url)
	assert.Equal(t, newUrl, *updateResp.Credential.Url)
	assert.NotNil(t, updateResp.Credential.Metadata)
	assert.Equal(t, newMetadata, *updateResp.Credential.Metadata)

	// Verify changes persist
	getResp, err := client.Get(authCtx, &gophkeepv1.CredentialGetRequest{
		Id: createResp.Credential.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", getResp.Credential.Name)
}

func TestCredentialsService_Update_NotFound(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	_, err := client.Update(authCtx, &gophkeepv1.CredentialUpdateRequest{
		Id:       "00000000-0000-0000-0000-000000000000",
		Name:     "Updated",
		Login:    "login",
		Password: "password",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_Delete_Success(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Create credential
	createResp, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
		Name:     "To Delete",
		Login:    "deletelogin",
		Password: "deletepassword",
	})
	require.NoError(t, err)

	// Delete credential
	deleteResp, err := client.Delete(authCtx, &gophkeepv1.CredentialDeleteRequest{
		Id: createResp.Credential.Id,
	})

	require.NoError(t, err)
	require.NotNil(t, deleteResp)
	assert.True(t, deleteResp.Deleted)

	// Verify credential is gone
	_, err = client.Get(authCtx, &gophkeepv1.CredentialGetRequest{
		Id: createResp.Credential.Id,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_Delete_NotFound(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	_, err := client.Delete(authCtx, &gophkeepv1.CredentialDeleteRequest{
		Id: "00000000-0000-0000-0000-000000000000",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestCredentialsService_FullCRUDFlow(t *testing.T) {
	tc := SetupTestContainer(t)
	tc.CleanupTables(t)

	username := GenerateRandomUsername(t)
	password := "SecurePassword123"
	testUser := CreateTestUser(t, tc.Ctx, username, password)

	client, conn := GetAuthenticatedCredentialsClient(t)
	defer conn.Close()

	authCtx := GetAuthContext(tc.Ctx, testUser.AccessToken)

	// Create
	url := "https://example.com"
	createResp, err := client.Create(authCtx, &gophkeepv1.CredentialCreateRequest{
		Name:     "Full Flow Test",
		Login:    "testuser",
		Password: "testpass123",
		Url:      &url,
	})
	require.NoError(t, err)
	credID := createResp.Credential.Id

	// Read
	getResp, err := client.Get(authCtx, &gophkeepv1.CredentialGetRequest{Id: credID})
	require.NoError(t, err)
	assert.Equal(t, "Full Flow Test", getResp.Credential.Name)

	// Update
	newUrl := "https://updated.example.com"
	updateResp, err := client.Update(authCtx, &gophkeepv1.CredentialUpdateRequest{
		Id:       credID,
		Name:     "Updated Flow Test",
		Login:    "updateduser",
		Password: "updatedpass456",
		Url:      &newUrl,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Flow Test", updateResp.Credential.Name)

	// List
	listResp, err := client.List(authCtx, &gophkeepv1.CredentialListRequest{
		Page: &gophkeepv1.PageRequest{Limit: 10, Offset: 0},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Items), 1)

	// Delete
	deleteResp, err := client.Delete(authCtx, &gophkeepv1.CredentialDeleteRequest{Id: credID})
	require.NoError(t, err)
	assert.True(t, deleteResp.Deleted)

	// Verify deletion
	_, err = client.Get(authCtx, &gophkeepv1.CredentialGetRequest{Id: credID})
	require.Error(t, err)
}
