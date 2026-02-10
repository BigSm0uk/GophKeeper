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
	Pool      *pgxpool.Pool
	Container *helpers.PostgresContainer
	Ctx       context.Context

	GRPCAddress  string
	appContainer *app.Container

	JWTPriv string
	JWTPub  string
}

var globalTestCtx *TestContext

// SetupTestContainer returns the global test context
func SetupTestContainer(t *testing.T) *TestContext {
	t.Helper()
	if globalTestCtx == nil {
		t.Fatal("global TestContext not initialized. Ensure TestMain is running.")
	}
	return globalTestCtx
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := helpers.CreatePostgresContainer(ctx)
	if err != nil {
		panic(err)
	}

	globalTestCtx = &TestContext{
		Pool:      container.Pool,
		Container: container,
		Ctx:       ctx,
	}

	dir, err := os.MkdirTemp("", "gk-test-keys-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	privPath, pubPath := generateTestJWTKeys(dir)
	globalTestCtx.JWTPriv = privPath
	globalTestCtx.JWTPub = pubPath

	globalTestCtx.startServer()

	code := m.Run()

	if globalTestCtx.appContainer != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = globalTestCtx.appContainer.App.Stop(stopCtx)
		cancel()
	}
	_ = container.Cleanup(ctx)

	os.Exit(code)
}

func (tc *TestContext) startServer() {
	cfg := NewTestConfig(tc)
	logger := zap.NewNop()
	c := app.NewContainer(logger, cfg)

	dbConn, err := db.NewPostgresDb(tc.Ctx, cfg.DB, logger)
	if err != nil {
		panic(err)
	}

	c.RegisterDatabase(dbConn)
	c.RegisterRepositories()
	if err := c.RegisterServices(); err != nil {
		panic(err)
	}
	c.RegisterGRPCServer()

	if err := c.App.Start(tc.Ctx); err != nil {
		panic(err)
	}

	tc.GRPCAddress = cfg.GRPCAddress()
	tc.appContainer = c
}

func NewTestConfig(tc *TestContext) *config.ServerConfig {
	cfg := config.NewDefaultServerConfig()
	cfg.Env = config.EnvDevelopment
	cfg.DB.ConnectionString = tc.Container.ConnectionString
	cfg.GRPC.Host = "127.0.0.1"
	cfg.GRPC.Port = 50051 // Use fixed port for shared server
	cfg.JWT.PrivateKeyPath = tc.JWTPriv
	cfg.JWT.PublicKeyPath = tc.JWTPub
	cfg.Storage.BasePath = os.TempDir()

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

func generateTestJWTKeys(dir string) (priv, pub string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	privPath := filepath.Join(dir, "jwt_private.pem")
	pubPath := filepath.Join(dir, "jwt_public.pem")

	if err := os.WriteFile(privPath,
		pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}), 0o600); err != nil {
		panic(err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(pubPath,
		pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubDER,
		}), 0o600); err != nil {
		panic(err)
	}

	return privPath, pubPath
}
