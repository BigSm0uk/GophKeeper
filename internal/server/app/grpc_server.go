package app

import (
	"context"
	"fmt"
	"net"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	grpchandlers "github.com/BigSm0uk/GophKeeper/internal/server/handlers/grpc"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	server      *grpc.Server
	logger      *zap.Logger
	config      *config.ServerConfig
	authHandler *grpchandlers.AuthHandler
}

func NewGRPCServer(cfg *config.ServerConfig, logger *zap.Logger) *GRPCServer {

	server := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)

	authHandler := grpchandlers.NewAuthHandler(logger)
	pb.RegisterAuthServiceServer(server, authHandler)

	// TODO: Регистрировать другие сервисы
	// pb.RegisterCredentialsServiceServer(server, credHandler)
	// pb.RegisterTextsServiceServer(server, textsHandler)
	// и т.д.

	if cfg.IsDevelopment() {
		reflection.Register(server)
		logger.Info("gRPC reflection enabled (development mode)")
	}

	return &GRPCServer{
		server:      server,
		logger:      logger,
		config:      cfg,
		authHandler: authHandler,
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

	// Start serving in a goroutine to make it non-blocking
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil {
			errCh <- fmt.Errorf("failed to serve: %w", err)
		}
	}()

	// Check if server started successfully or failed immediately
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

	// Use GracefulStop with timeout
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
		// Force stop if graceful shutdown takes too long
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

// loggingInterceptor логирует все gRPC вызовы
func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Debug("gRPC call",
			zap.String("method", info.FullMethod),
		)

		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("gRPC call failed",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
		}

		return resp, err
	}
}
