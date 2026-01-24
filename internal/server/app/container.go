package app

import (
	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/db"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	pg_repo "github.com/BigSm0uk/GophKeeper/internal/server/repository/postgres"
	"github.com/BigSm0uk/GophKeeper/internal/server/service"
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

	// Domain Layer
	AuthService *service.AuthService
	JWTService  *service.JWTService
	// CredService     *services.CredentialsService

	// Data Layer
	DB       *db.PostgresDb
	UserRepo interfaces.UserRepository
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
func (c *Container) RegisterGRPCServer() {
	server := NewGRPCServer(c.Config, c.Logger, c.AuthService)
	c.GRPCServer = server
	c.App.AddComponent(server)
}

// RegisterHTTPServer registers the HTTP server component (grpc-gateway).
func (c *Container) RegisterHTTPServer(server *HTTPServer) {
	c.HTTPServer = server
	c.App.AddComponent(server)
}

// TODO: Add methods for registering other components
func (c *Container) RegisterDatabase(db *db.PostgresDb) {
	c.DB = db
	c.App.AddComponent(db)
}

func (c *Container) RegisterRepositories() {
	ur := pg_repo.NewUserRepository(c.Logger, c.DB)
	c.UserRepo = ur
}

func (c *Container) RegisterServices() error {
	jwtSvc, err := service.NewJWTService(c.Config.JWT)
	if err != nil {
		return err
	}
	c.JWTService = jwtSvc

	authSvc := service.NewAuthService(c.Logger, c.UserRepo, c.JWTService)
	c.AuthService = authSvc
	return nil
}
