package grpc

import (
	"context"
	"strings"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type userKey struct{}

// AuthInterceptor validates JWT tokens for protected gRPC methods.
func AuthInterceptor(authService *service.AuthService, authConfig config.AuthConfig, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if requiresAuth(info.FullMethod, authConfig) {
			user, err := authenticateUser(ctx, authService, logger)
			if err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, userKey{}, user)
			logger.Debug("User authenticated",
				zap.String("method", info.FullMethod),
				zap.String("user_id", user.ID),
				zap.String("username", user.Username))
		} else {
			logger.Debug("Public method, skipping authentication",
				zap.String("method", info.FullMethod))
		}

		return handler(ctx, req)
	}
}

// authenticateUser extracts and validates JWT token from gRPC metadata
func authenticateUser(ctx context.Context, authService *service.AuthService, logger *zap.Logger) (*models.User, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeader, exists := md["authorization"]
	if !exists || len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := authHeader[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid token format")
	}

	token = strings.TrimPrefix(token, "Bearer ")

	user, err := authService.ValidateToken(ctx, token)
	if err != nil {
		logger.Warn("Token validation failed", zap.Error(err))
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return user, nil
}

// requiresAuth checks if a gRPC method requires authentication
func requiresAuth(fullMethod string, authConfig config.AuthConfig) bool {
	if authConfig.IsPublicMethod(fullMethod) {
		return false
	}

	return strings.HasPrefix(fullMethod, "/gophkeeper.v1.")
}

// RecoveryInterceptor handles panic and converts it to gRPC errors.
func RecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC handler panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.Stack("stack"),
				)
				err = status.Error(codes.Internal, "Internal server error")
			}
		}()

		resp, err = handler(ctx, req)
		return resp, err
	}
}

// LoggingInterceptor logs all gRPC calls
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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

// GetUserFromContext extracts authenticated user from context
func GetUserFromContext(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(userKey{}).(*models.User)
	if !ok || user == nil {
		return nil, status.Error(codes.Unauthenticated, "user not found in context")
	}
	return user, nil
}
