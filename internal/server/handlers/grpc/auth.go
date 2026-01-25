package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthHandler implements AuthServiceServer
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	logger    *zap.Logger
	as        *service.AuthService
	jwtConfig config.JWTConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(logger *zap.Logger, as *service.AuthService, jwtConfig config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		logger:    logger,
		as:        as,
		jwtConfig: jwtConfig,
	}
}

// validateTokenRequest validates token request based on grant type
func (h *AuthHandler) validateTokenRequest(req *pb.TokenRequest) error {
	switch req.GrantType {
	case pb.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD:
		// For password grant, username and password are required
		if req.Username == "" {
			return fmt.Errorf("username is required for password grant")
		}
		if len(req.Username) < 3 || len(req.Username) > 50 {
			return fmt.Errorf("username must be between 3 and 50 characters")
		}
		if req.Password == "" {
			return fmt.Errorf("password is required for password grant")
		}
		if len(req.Password) < 8 || len(req.Password) > 128 {
			return fmt.Errorf("password must be between 8 and 128 characters")
		}
		// client_id is optional but if provided, must be valid
		if req.ClientId != "" && (len(req.ClientId) < 1 || len(req.ClientId) > 255) {
			return fmt.Errorf("client_id must be between 1 and 255 characters if provided")
		}

	case pb.TokenGrantType_TOKEN_GRANT_TYPE_REFRESH_TOKEN:
		// For refresh_token grant, refresh_token is required
		if req.RefreshToken == "" {
			return fmt.Errorf("refresh_token is required for refresh_token grant")
		}
		// client_id is optional but if provided, must be valid
		if req.ClientId != "" && (len(req.ClientId) < 1 || len(req.ClientId) > 255) {
			return fmt.Errorf("client_id must be between 1 and 255 characters if provided")
		}

	default:
		return fmt.Errorf("unsupported grant type")
	}

	return nil
}

// classifyServiceError classifies service errors into appropriate gRPC status codes
// Note: This method only transforms errors, logging is handled at the service layer
func (h *AuthHandler) classifyServiceError(err error) error {
	switch {
	case errors.Is(err, models.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "User with this username already exists")

	case errors.Is(err, models.ErrInvalidUsername):
		return status.Error(codes.InvalidArgument, "Username does not meet requirements")

	case errors.Is(err, models.ErrInvalidPassword):
		return status.Error(codes.InvalidArgument, "Password does not meet requirements")

	case errors.Is(err, models.ErrInvalidUserID):
		return status.Error(codes.InvalidArgument, "Invalid user identifier")

	case errors.Is(err, models.ErrUserNotFound):
		return status.Error(codes.NotFound, "User not found")

	case errors.Is(err, models.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "Invalid credentials")

	case errors.Is(err, models.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, "Invalid token")

	case errors.Is(err, models.ErrTokenExpired):
		return status.Error(codes.Unauthenticated, "Token expired")

	case errors.Is(err, models.ErrInvalidGrantType):
		return status.Error(codes.InvalidArgument, "Invalid grant type")

	default:
		// For any other errors, don't expose internal details to client
		h.logger.Error("Unexpected service error in handler", zap.Error(err))
		return status.Error(codes.Internal, "An internal error occurred")
	}
}

// Register implements user registration
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	h.logger.Info("Register request received",
		zap.String("username", req.Username))

	if err := req.Validate(); err != nil {
		h.logger.Warn("Invalid registration request data",
			zap.Error(err),
			zap.String("username", req.Username))
		// Extract detailed validation error message
		errorMsg := err.Error()
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid registration data: %s", errorMsg))
	}

	user, err := entity.MapUserFromRequest(req)
	if err != nil {
		h.logger.Error("Failed to map request to user entity",
			zap.Error(err),
			zap.String("username", req.Username))
		return nil, status.Error(codes.InvalidArgument, "Invalid user data")
	}

	res, err := h.as.Register(ctx, *user)
	if err != nil {
		return nil, h.classifyServiceError(err)
	}

	h.logger.Info("Registration request completed successfully",
		zap.String("user_id", res.ID),
		zap.String("username", res.Username))

	return &pb.RegisterResponse{
		UserId:    res.ID,
		Username:  res.Username,
		CreatedAt: timestamppb.New(res.CreatedAt),
	}, nil
}

// Token implements OAuth2 token endpoint
func (h *AuthHandler) Token(ctx context.Context, req *pb.TokenRequest) (*pb.TokenResponse, error) {
	h.logger.Info("Token request received",
		zap.String("grant_type", req.GrantType.String()),
		zap.String("username", req.Username),
	)

	if err := h.validateTokenRequest(req); err != nil {
		h.logger.Warn("Invalid token request data", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var grantType string
	switch req.GrantType {
	case pb.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD:
		grantType = "password"
	case pb.TokenGrantType_TOKEN_GRANT_TYPE_REFRESH_TOKEN:
		grantType = "refresh_token"
	default:
		return nil, status.Error(codes.InvalidArgument, "Unsupported grant type")
	}

	accessToken, refreshToken, err := h.as.Token(ctx, grantType, req.Username, req.Password, req.RefreshToken)
	if err != nil {
		return nil, h.classifyServiceError(err)
	}

	h.logger.Info("Token request completed successfully",
		zap.String("grant_type", grantType),
		zap.String("username", req.Username),
	)

	return &pb.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    uint32(h.jwtConfig.AccessTokenTTL.Seconds()),
		Scope:        req.Scope,
	}, nil
}

// Revoke implements token revocation
func (h *AuthHandler) Revoke(ctx context.Context, req *pb.RevokeRequest) (*pb.RevokeResponse, error) {
	tokenPreview := req.Token
	if len(tokenPreview) > 10 {
		tokenPreview = tokenPreview[:10] + "..."
	}

	h.logger.Info("Revoke request received",
		zap.String("token", tokenPreview),
		zap.String("token_type_hint", req.TokenTypeHint.String()),
	)

	if err := req.Validate(); err != nil {
		h.logger.Warn("Invalid revoke request data", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "Invalid revoke request data")
	}

	// For JWT tokens, "revocation" means validating that the token is invalid
	// JWT tokens are stateless, so we just check if they are valid
	_, err := h.as.ValidateToken(ctx, req.Token)
	revoked := err != nil // Token is considered revoked if it's invalid

	h.logger.Info("Revoke request completed",
		zap.Bool("revoked", revoked),
	)

	return &pb.RevokeResponse{
		Revoked: revoked,
	}, nil
}
