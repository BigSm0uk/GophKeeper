package grpc

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type (
	userKey          struct{}
	requestIDKey     struct{}
	requestLoggerKey struct{}
	clientIPKey      struct{}
	userAgentKey     struct{}
	accessTokenKey   struct{}
)

// AuthInterceptor validates JWT tokens for protected gRPC methods.
func AuthInterceptor(authService *service.AuthService, authConfig config.AuthConfig, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if requiresAuth(info.FullMethod, authConfig) {
			user, token, err := authenticateUser(ctx, authService, logger)
			if err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, userKey{}, user)
			ctx = context.WithValue(ctx, accessTokenKey{}, token)
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

// extractTokenFromMetadata extracts Bearer token from gRPC metadata
func extractTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeader, exists := md["authorization"]
	if !exists || len(authHeader) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := authHeader[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid token format")
	}

	return strings.TrimPrefix(token, "Bearer "), nil
}

// authenticateUser extracts and validates JWT token from gRPC metadata
// Returns user, token, and error
func authenticateUser(ctx context.Context, authService *service.AuthService, logger *zap.Logger) (*models.User, string, error) {
	token, err := extractTokenFromMetadata(ctx)
	if err != nil {
		return nil, "", err
	}

	user, err := authService.ValidateToken(ctx, token)
	if err != nil {
		logger.Warn("Token validation failed", zap.Error(err))
		return nil, "", status.Error(codes.Unauthenticated, "invalid token")
	}

	return user, token, nil
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

// RequestIDInterceptor generates unique request GetID for each gRPC call and adds it to context
func RequestIDInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestID := uuid.New().String()

		clientIP := extractClientIP(ctx)
		userAgent := extractUserAgent(ctx)

		ctx = context.WithValue(ctx, requestIDKey{}, requestID)
		ctx = context.WithValue(ctx, clientIPKey{}, clientIP)
		ctx = context.WithValue(ctx, userAgentKey{}, userAgent)

		requestLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		)
		ctx = context.WithValue(ctx, requestLoggerKey{}, requestLogger)

		_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", requestID))

		return handler(ctx, req)
	}
}

func extractClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	addr := p.Addr.String()
	// SplitHostPort correctly handles IPv6 "[::1]:port" -> host "::1" (PostgreSQL inet format)
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	// No port (e.g. "[::1]"): strip brackets for IPv6 so PostgreSQL inet accepts it
	if strings.HasPrefix(addr, "[") {
		if idx := strings.Index(addr, "]"); idx != -1 {
			return addr[1:idx]
		}
	}
	return addr
}

func extractUserAgent(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	// Try different header names for User-Agent
	userAgentHeaders := []string{"user-agent", "User-Agent", "grpcgateway-user-agent"}
	for _, header := range userAgentHeaders {
		if values := md.Get(header); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return ""
}

// LoggingInterceptor logs all gRPC calls with request_id
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// Use request-scoped logger if available
		reqLogger := GetLoggerFromContext(ctx, logger)

		reqLogger.Debug("gRPC call",
			zap.String("method", info.FullMethod),
		)

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		if err != nil {
			reqLogger.Error("gRPC call failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			reqLogger.Debug("gRPC call completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
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

// GetRequestIDFromContext extracts request GetID from context
func GetRequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey{}).(string)
	if !ok || requestID == "" {
		return "unknown"
	}
	return requestID
}

// GetLoggerFromContext extracts request-scoped logger from context
// If not found, returns the provided fallback logger with request_id added
func GetLoggerFromContext(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	logger, ok := ctx.Value(requestLoggerKey{}).(*zap.Logger)
	if ok && logger != nil {
		return logger
	}

	// Fallback: create logger with request_id from context
	requestID := GetRequestIDFromContext(ctx)
	return fallback.With(zap.String("request_id", requestID))
}

// GetClientIPFromContext extracts client IP address from context
func GetClientIPFromContext(ctx context.Context) string {
	ip, ok := ctx.Value(clientIPKey{}).(string)
	if !ok {
		return ""
	}
	return ip
}

// GetUserAgentFromContext extracts User-Agent from context
func GetUserAgentFromContext(ctx context.Context) string {
	ua, ok := ctx.Value(userAgentKey{}).(string)
	if !ok {
		return ""
	}
	return ua
}

// GetAccessTokenFromContext extracts access token from context
func GetAccessTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(accessTokenKey{}).(string)
	if !ok {
		return ""
	}
	return token
}
