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
	server          *grpc.Server
	logger          *zap.Logger
	config          *config.ServerConfig
	authHandler     *grpchandlers.AuthHandler
	binariesHandler *grpchandlers.BinariesHandler
}

func NewGRPCServer(cfg *config.ServerConfig, logger *zap.Logger, authService *service.AuthService, binariesService *service.BinaryService) *GRPCServer {
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

	pb.RegisterAuthServiceServer(server, authHandler)
	pb.RegisterBinariesServiceServer(server, binariesHandler)

	// TODO: Регистрировать другие сервисы
	// pb.RegisterCredentialsServiceServer(server, credHandler)
	// pb.RegisterTextsServiceServer(server, textsHandler)
	// и т.д.

	if cfg.IsDevelopment() {
		reflection.Register(server)
		logger.Info("gRPC reflection enabled (development mode)")
	}

	return &GRPCServer{
		server:          server,
		logger:          logger,
		config:          cfg,
		authHandler:     authHandler,
		binariesHandler: binariesHandler,
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
