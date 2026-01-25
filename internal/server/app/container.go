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
	AuthService     *service.AuthService
	JWTService      *service.JWTService
	BinariesService *service.BinaryService
	FileService     *service.FileService
	// CredService     *services.CredentialsService

	// Data Layer
	DB           *db.PostgresDb
	UserRepo     interfaces.UserRepository
	BinariesRepo interfaces.BinariesRepository
	SessionRepo  interfaces.SessionRepository
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
	server := NewGRPCServer(c.Config, c.Logger, c.AuthService, c.BinariesService)
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
	br := pg_repo.NewBinaryRepository(c.Logger, c.DB)
	sr := pg_repo.NewSessionRepository(c.Logger, c.DB)

	c.UserRepo = ur
	c.BinariesRepo = br
	c.SessionRepo = sr
}

func (c *Container) RegisterServices() error {
	// Initialize FileService with configured storage path
	fSvc, err := service.NewFileService(c.Config.Storage.BasePath, c.Logger)
	if err != nil {
		return err
	}
	c.FileService = fSvc

	// Initialize JWT Service
	jwtSvc, err := service.NewJWTService(c.Config.JWT)
	if err != nil {
		return err
	}
	c.JWTService = jwtSvc

	// Initialize Auth Service
	authSvc := service.NewAuthService(c.Logger, c.UserRepo, c.JWTService, c.SessionRepo)
	c.AuthService = authSvc

	// Initialize Binaries Service
	binSvc := service.NewBinaryService(c.Logger, c.BinariesRepo, c.FileService, c.Config.Storage.MaxFileSize)
	c.BinariesService = binSvc

	return nil
}
