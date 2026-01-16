package grpc

import (
	"context"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthHandler implements AuthServiceServer
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	logger *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		logger: logger,
	}
}

// Register implements user registration (stub for testing)
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	h.logger.Info("Register called",
		zap.String("username", req.Username),
	)

	// Валидация с использованием сгенерированного метода
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// TODO: Реальная логика регистрации с БД
	// Пока возвращаем stub
	return &pb.RegisterResponse{
		UserId:    "user-1234",
		Username:  req.Username,
		CreatedAt: timestamppb.Now(),
	}, nil
}

// Token implements OAuth2 token endpoint (stub for testing)
func (h *AuthHandler) Token(ctx context.Context, req *pb.TokenRequest) (*pb.TokenResponse, error) {
	h.logger.Info("Token called",
		zap.String("grant_type", req.GrantType.String()),
		zap.String("username", req.Username),
	)

	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// TODO: Реальная логика аутентификации
	// Пока возвращаем stub токены
	return &pb.TokenResponse{
		AccessToken:  "stub-access-token-" + req.Username,
		RefreshToken: "stub-refresh-token-" + req.Username,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        req.Scope,
	}, nil
}

// Revoke implements token revocation (stub for testing)
func (h *AuthHandler) Revoke(ctx context.Context, req *pb.RevokeRequest) (*pb.RevokeResponse, error) {
	h.logger.Info("Revoke called",
		zap.String("token", req.Token[:10]+"..."),
	)

	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// TODO: Реальная логика отзыва токенов
	return &pb.RevokeResponse{
		Revoked: true,
	}, nil
}
