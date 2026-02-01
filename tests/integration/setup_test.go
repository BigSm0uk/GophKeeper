package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	"github.com/BigSm0uk/GophKeeper/tests/helpers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestContext holds common resources for integration tests
type TestContext struct {
	Pool         *pgxpool.Pool
	Container    *helpers.PostgresContainer
	Ctx          context.Context
	GRPCAddress  string
	appContainer *app.Container
}

// SetupTestContainer creates and configures PostgreSQL test container
func SetupTestContainer(t *testing.T) *TestContext {
	t.Helper()

	ctx := context.Background()

	container, err := helpers.CreatePostgresContainer(ctx)
	require.NoError(t, err, "Failed to create postgres container")

	t.Cleanup(func() {
		if err := container.Cleanup(ctx); err != nil {
			t.Logf("Failed to cleanup container: %v", err)
		}
	})

	return &TestContext{
		Pool:      container.Pool,
		Container: container,
		Ctx:       ctx,
	}
}

func (tc *TestContext) StartTestGRPCServer(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	priv, pub := CreateTestJWTKeys(t, dir)

	cfg := NewTestConfig(t, tc, priv, pub)

	app := StartTestApp(t, tc.Ctx, cfg)

	tc.GRPCAddress = cfg.GRPCAddress()
	tc.appContainer = app

	return tc.GRPCAddress
}
func StartTestApp(
	t *testing.T,
	ctx context.Context,
	cfg *config.ServerConfig,
) *app.Container {

	logger := zap.NewNop()
	c := app.NewContainer(logger, cfg)

	db, err := db.NewPostgresDb(ctx, cfg.DB, logger)
	require.NoError(t, err)

	c.RegisterDatabase(db)
	c.RegisterRepositories()
	require.NoError(t, c.RegisterServices())
	c.RegisterGRPCServer()

	require.NoError(t, c.App.Start(ctx))

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.App.Stop(stopCtx)
	})

	return c
}
func NewTestConfig(
	t *testing.T,
	tc *TestContext,
	jwtPriv, jwtPub string,
) *config.ServerConfig {

	cfg := config.NewDefaultServerConfig()
	cfg.Env = config.EnvDevelopment
	cfg.DB.ConnectionString = tc.Container.ConnectionString
	cfg.GRPC.Host = "127.0.0.1"
	cfg.GRPC.Port = freePort(t)
	cfg.JWT.PrivateKeyPath = jwtPriv
	cfg.JWT.PublicKeyPath = jwtPub
	cfg.Storage.BasePath = t.TempDir()

	return &cfg
}

// CleanupTables truncates all tables between tests while preserving schema
func (tc *TestContext) CleanupTables(t *testing.T) {
	t.Helper()
	rows, err := tc.Pool.Query(tc.Ctx, "SELECT tablename FROM pg_tables WHERE schemaname = 'public';")
	require.NoError(t, err, "Failed to query tables")
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tablename string
		require.NoError(t, rows.Scan(&tablename), "Failed to scan tablename")
		tables = append(tables, tablename)
	}
	require.NoError(t, rows.Err(), "Error iterating table rows")

	for _, table := range tables {
		_, err := tc.Pool.Exec(tc.Ctx, "TRUNCATE TABLE "+table+" CASCADE")
		require.NoError(t, err, "Failed to truncate table: "+table)
	}
}

// SeedTestData executes test data queries
func (tc *TestContext) SeedTestData(t *testing.T, queries ...string) {
	t.Helper()

	for _, query := range queries {
		_, err := tc.Pool.Exec(tc.Ctx, query)
		require.NoError(t, err, "Failed to execute seed query")
	}
}
func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to find free port")

	defer func() {
		_ = l.Close()
	}()

	addr := l.Addr().(*net.TCPAddr)
	return addr.Port
}
func CreateTestJWTKeys(t *testing.T, dir string) (priv, pub string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privPath := filepath.Join(dir, "jwt_private.pem")
	pubPath := filepath.Join(dir, "jwt_public.pem")

	require.NoError(t, os.WriteFile(privPath,
		pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}), 0600))

	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, os.WriteFile(pubPath,
		pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubDER,
		}), 0600))

	return privPath, pubPath
}
