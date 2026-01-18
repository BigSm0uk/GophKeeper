package app

import (
	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"go.uber.org/zap"
)

// Container is a dependency injection container for the application.
// It follows the Dependency Injection pattern from Clean Architecture.
type Container struct {
	// Infrastructure
	Logger *zap.Logger
	Config *config.ServerConfig

	// Application
	App *Application

	// Presentation Layer (gRPC & HTTP)
	GRPCServer *GRPCServer
	HTTPServer *HTTPServer

	// TODO: Add when implementing
	// Domain Layer
	// AuthService     *services.AuthService
	// CredService     *services.CredentialsService

	// Data Layer
	// DB              *pgxpool.Pool
	// UserRepo        *repositories.UserRepository
	// CredRepo        *repositories.CredentialsRepository
	// MinioClient     *minio.Client
}

// NewContainer creates a new dependency injection container.
func NewContainer(logger *zap.Logger, cfg *config.ServerConfig) *Container {
	return &Container{
		Logger: logger,
		Config: cfg,
		App:    NewApplication(logger),
	}
}

// RegisterGRPCServer registers the gRPC server component.
func (c *Container) RegisterGRPCServer(server *GRPCServer) {
	c.GRPCServer = server
	c.App.AddComponent(server)
}

// RegisterHTTPServer registers the HTTP server component (grpc-gateway).
func (c *Container) RegisterHTTPServer(server *HTTPServer) {
	c.HTTPServer = server
	c.App.AddComponent(server)
}

// TODO: Add methods for registering other components
// func (c *Container) RegisterDatabase(db *pgxpool.Pool) { ... }
// func (c *Container) RegisterRepositories() { ... }
// func (c *Container) RegisterServices() { ... }
