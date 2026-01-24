package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/migrations"
	"github.com/avast/retry-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

type PostgresDb struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

var _ interfaces.Lifecycle = (*PostgresDb)(nil)

func NewPostgresDb(ctx context.Context, cfg config.DBConfig, logger *zap.Logger) (*PostgresDb, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	return &PostgresDb{
		pool:   pool,
		logger: logger,
	}, nil
}

func pingWithRetry(ctx context.Context, pool *pgxpool.Pool, maxRetries int, delay time.Duration) error {
	return retry.Do(func() error {
		return pool.Ping(ctx)
	}, retry.Attempts(uint(maxRetries)), retry.Delay(delay))
}

func (p *PostgresDb) warmUp(n int) {
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			_ = pingWithRetry(context.Background(), p.pool, 3, 100*time.Millisecond)
		})
	}
	wg.Wait()
}

func (p *PostgresDb) GetPool() *pgxpool.Pool {
	return p.pool
}

// Name implements [interfaces.Lifecycle].
func (p *PostgresDb) Name() string {
	return "Postgres DB"
}

// Start implements [interfaces.Lifecycle].
func (p *PostgresDb) Start(ctx context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pingWithRetry(ctxWithTimeout, p.pool, 5, 100*time.Millisecond); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	p.warmUp(10)

	return nil
}

// RunMigrations runs all pending database migrations using goose.
// It creates a separate database/sql connection specifically for migrations.
func RunMigrations(connectionString string, logger *zap.Logger) error {
	// Create a standard database/sql connection for goose
	sqlDB, err := sql.Open("pgx", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	logger.Info("Running database migrations...")
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		logger.Warn("Failed to get migration version", zap.Error(err))
	} else {
		logger.Info("Database migrations completed", zap.Int64("version", version))
	}

	return nil
}

// Stop implements [interfaces.Lifecycle].
func (p *PostgresDb) Stop(_ context.Context) error {
	p.pool.Close()
	return nil
}
