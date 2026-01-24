package integration

import (
	"context"
	"testing"

	"github.com/BigSm0uk/GophKeeper/tests/helpers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestContext holds common resources for integration tests
type TestContext struct {
	Pool      *pgxpool.Pool
	Container *helpers.PostgresContainer
	Ctx       context.Context
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

// CleanupTables truncates all tables between tests while preserving schema
func (tc *TestContext) CleanupTables(t *testing.T) {
	t.Helper()

	tables := []string{
		"sync_changelog",
		"binaries",
		"cards",
		"credentials",
		"texts",
		"users",
	}

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
