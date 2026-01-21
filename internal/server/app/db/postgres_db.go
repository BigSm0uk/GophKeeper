package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/avast/retry-go"
	"github.com/jackc/pgx/v5/pgxpool"
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
			pingWithRetry(context.Background(), p.pool, 3, 100*time.Millisecond)
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

// Stop implements [interfaces.Lifecycle].
func (p *PostgresDb) Stop(ctx context.Context) error {
	p.pool.Close()
	return nil
}
