package grpc

import (
	"context"
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
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
		// For refresh_token grant, user is identified by Bearer token in Authorization header (username not required)
		// client_id is optional but if provided, must be valid
		if req.ClientId != "" && (len(req.ClientId) < 1 || len(req.ClientId) > 255) {
			return fmt.Errorf("client_id must be between 1 and 255 characters if provided")
		}

	default:
		return fmt.Errorf("unsupported grant type")
	}

	return nil
}

// Register implements user registration
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Register request received",
		zap.String("username", req.Username))

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid registration request data",
			zap.Error(err),
			zap.String("username", req.Username))
		// Extract detailed validation error message
		errorMsg := err.Error()
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid registration data: %s", errorMsg))
	}

	user, err := entity.MapUserFromRequest(req)
	if err != nil {
		logger.Error("Failed to map request to user entity",
			zap.Error(err),
			zap.String("username", req.Username))
		return nil, status.Error(codes.InvalidArgument, "Invalid user data")
	}

	res, err := h.as.Register(ctx, *user)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	logger.Info("Registration request completed successfully",
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
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Token request received",
		zap.String("grant_type", req.GrantType.String()),
		zap.String("username", req.Username),
	)

	if err := h.validateTokenRequest(req); err != nil {
		logger.Warn("Invalid token request data", zap.Error(err))
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

	ipAddress := GetClientIPFromContext(ctx)
	userAgent := GetUserAgentFromContext(ctx)

	// For refresh_token grant, extract access token from context or Authorization header
	var accessTokenForRefresh string
	if grantType == "refresh_token" {
		accessTokenForRefresh = GetAccessTokenFromContext(ctx)

		if accessTokenForRefresh == "" {
			token, err := extractTokenFromMetadata(ctx)
			if err != nil {
				logger.Warn("Failed to extract token for refresh_token grant", zap.Error(err))
				return nil, err
			}
			accessTokenForRefresh = token
		}
	}

	accessToken, err := h.as.Token(ctx, grantType, req.Username, req.Password, accessTokenForRefresh, ipAddress, userAgent)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	logger.Info("Token request completed successfully",
		zap.String("grant_type", grantType),
		zap.String("username", req.Username),
	)

	return &pb.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   uint32(h.jwtConfig.AccessTokenTTL.Seconds()),
		Scope:       req.Scope,
	}, nil
}
