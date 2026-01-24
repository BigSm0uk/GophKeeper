package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/logger"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver for database/sql
	"go.uber.org/zap"
)

// bootstrap initializes and wires up all application components.
// This follows the Dependency Injection and Inversion of Control patterns.
func bootstrap() (*app.Container, error) {
	// 1. Infrastructure Layer: Configuration
	cfg, err := config.ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 2. Infrastructure Layer: Logger
	log, err := logger.NewZapLogger(cfg.Logger.Level, cfg.IsDevelopment())
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	log.Info("Initializing GophKeeper server",
		zap.String("env", cfg.Env),
		zap.String("grpc_address", cfg.GRPCAddress()),
		zap.String("http_address", cfg.HTTPAddress()),
	)

	// 3. Create Dependency Injection Container
	container := app.NewContainer(log, cfg)

	// 4. Infrastructure Layer: Database
	postgresDb, err := db.NewPostgresDb(context.Background(), cfg.DB, log)
	if err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	container.RegisterDatabase(postgresDb)

	// 4.1. Run database migrations
	if err := db.RunMigrations(cfg.DB.ConnectionString, log); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// 5. Data Layer: Repositories (TODO: Denis)
	container.RegisterRepositories()

	// 6. Domain Layer: Services (TODO: Denis)
	err = container.RegisterServices()
	if err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// 7. Presentation Layer: gRPC Server
	container.RegisterGRPCServer()

	// 8. Presentation Layer: HTTP Server (grpc-gateway)
	// ВАЖНО: HTTP сервер должен стартовать ПОСЛЕ gRPC сервера,
	// так как он делает forwarding запросов к gRPC
	httpServer := app.NewHTTPServer(cfg, log, container.GRPCServer)
	container.RegisterHTTPServer(httpServer)

	// 9. Configure shutdown timeouts
	container.App.SetShutdownTimeout(30 * time.Second)
	container.App.SetStartupTimeout(10 * time.Second)

	log.Info("Application bootstrap completed successfully")
	return container, nil
}

// run starts the application and handles graceful shutdown.
// This is the application entry point that coordinates the lifecycle.
func run(container *app.Container) error {
	// Create context that listens for interrupt signals
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	container.Logger.Info("Starting application...")

	// Run the application (starts all components and waits for shutdown)
	if err := container.App.Run(ctx); err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	container.Logger.Info("Application shutdown completed")
	return nil
}
