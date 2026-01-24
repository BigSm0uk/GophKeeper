package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	ConnectionString string
	Pool             *pgxpool.Pool
}

func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	pgContainer, err := postgres.Run(ctx, "postgres:18.1-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	if err := runMigrations(connStr); err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connStr,
		Pool:              pool,
	}, nil
}

// runMigrations применяет миграции goose к тестовой БД
func runMigrations(connectionString string) error {
	// Создаём соединение для goose (требуется database/sql)
	sqlDB, err := sql.Open("pgx", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	// Настраиваем goose с embed миграциями
	goose.SetBaseFS(migrations.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Применяем миграции
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// Cleanup закрывает соединения и останавливает контейнер
func (c *PostgresContainer) Cleanup(ctx context.Context) error {
	if c.Pool != nil {
		c.Pool.Close()
	}
	return c.Terminate(ctx)
}
