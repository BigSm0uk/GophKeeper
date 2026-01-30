package app

import (
	"context"
	"fmt"
	"net"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	grpchandlers "github.com/BigSm0uk/GophKeeper/internal/server/handlers/grpc"
	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	server             *grpc.Server
	logger             *zap.Logger
	config             *config.ServerConfig
	authHandler        *grpchandlers.AuthHandler
	binariesHandler    *grpchandlers.BinariesHandler
	credentialsHandler *grpchandlers.CredentialsHandler
	cardsHandler       *grpchandlers.CardsHandler
	textsHandler       *grpchandlers.TextsHandler
	userHandler        *grpchandlers.UserHandler
}

func NewGRPCServer(cfg *config.ServerConfig, logger *zap.Logger, authService *service.AuthService, binariesService *service.BinaryService, credentialsService *service.CredentialsService, cardsService *service.CardsService, textsService *service.TextsService, userService *service.UserService, jwtService *service.JWTService) *GRPCServer {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpchandlers.RequestIDInterceptor(logger),                   // First: generate request ID
			grpchandlers.RecoveryInterceptor(logger),                    // Second: catch panics
			grpchandlers.AuthInterceptor(authService, cfg.Auth, logger), // Third: authenticate
			grpchandlers.LoggingInterceptor(logger),                     // Last: log with all context
		),
	)

	authHandler := grpchandlers.NewAuthHandler(logger, authService, cfg.JWT)
	binariesHandler := grpchandlers.NewBinariesHandler(logger, binariesService)
	credentialsHandler := grpchandlers.NewCredentialsHandler(logger, credentialsService)
	cardsHandler := grpchandlers.NewCardsHandler(logger, cardsService)
	textsHandler := grpchandlers.NewTextsHandler(logger, textsService)
	userHandler := grpchandlers.NewUserHandler(logger, userService, jwtService, cfg.Storage.MaxFileSize)

	pb.RegisterAuthServiceServer(server, authHandler)
	pb.RegisterBinariesServiceServer(server, binariesHandler)
	pb.RegisterCredentialsServiceServer(server, credentialsHandler)
	pb.RegisterCardsServiceServer(server, cardsHandler)
	pb.RegisterTextsServiceServer(server, textsHandler)
	pb.RegisterUserServiceServer(server, userHandler)

	if cfg.IsDevelopment() {
		reflection.Register(server)
		logger.Info("gRPC reflection enabled (development mode)")
	}

	return &GRPCServer{
		server:             server,
		logger:             logger,
		config:             cfg,
		authHandler:        authHandler,
		binariesHandler:    binariesHandler,
		credentialsHandler: credentialsHandler,
		cardsHandler:       cardsHandler,
		textsHandler:       textsHandler,
		userHandler:        userHandler,
	}
}

// Start implements Lifecycle interface
func (s *GRPCServer) Start(ctx context.Context) error {
	address := s.config.GRPCAddress()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	s.logger.Info("Starting gRPC server",
		zap.String("address", address),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil {
			errCh <- fmt.Errorf("failed to serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Stop implements Lifecycle interface
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping gRPC server...")

	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		s.logger.Info("gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("gRPC server graceful stop timeout, forcing stop")
		s.server.Stop()
		return ctx.Err()
	}
}

// Name implements Lifecycle interface
func (s *GRPCServer) Name() string {
	return "gRPC Server"
}

// GetAuthHandler returns the auth handler for in-process gateway registration
func (s *GRPCServer) GetAuthHandler() pb.AuthServiceServer {
	return s.authHandler
}

// GetCredentialsHandler returns the credentials handler for in-process gateway registration
func (s *GRPCServer) GetCredentialsHandler() pb.CredentialsServiceServer {
	return s.credentialsHandler
}

// GetCardsHandler returns the cards handler for in-process gateway registration
func (s *GRPCServer) GetCardsHandler() pb.CardsServiceServer {
	return s.cardsHandler
}

// GetTextsHandler returns the texts handler for in-process gateway registration
func (s *GRPCServer) GetTextsHandler() pb.TextsServiceServer {
	return s.textsHandler
}

// GetUserHandler returns the user handler for in-process gateway registration
func (s *GRPCServer) GetUserHandler() pb.UserServiceServer {
	return s.userHandler
}
