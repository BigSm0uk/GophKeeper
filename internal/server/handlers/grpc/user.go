package grpc

import (
	"context"
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	"github.com/BigSm0uk/GophKeeper/internal/server/service/entity"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserHandler implements UserServiceServer.
type UserHandler struct {
	pb.UnimplementedUserServiceServer
	logger       *zap.Logger
	userService  *service.UserService
	jwtService   *service.JWTService
	storageLimit int64
}

// NewUserHandler creates a new user handler.
func NewUserHandler(logger *zap.Logger, userService *service.UserService, jwtService *service.JWTService, storageLimit int64) *UserHandler {
	return &UserHandler{
		logger:       logger,
		userService:  userService,
		jwtService:   jwtService,
		storageLimit: storageLimit,
	}
}

// GetProfile returns current user profile.
func (h *UserHandler) GetProfile(ctx context.Context, _ *pb.UserProfileGetRequest) (*pb.UserProfileGetResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Get profile request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	profileUser, err := h.userService.GetProfile(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	profile, err := entity.MapUserToProfile(profileUser, 0, h.storageLimit)
	if err != nil {
		logger.Error("Failed to map user to profile", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to map profile")
	}

	return &pb.UserProfileGetResponse{Profile: profile}, nil
}

// UpdateProfile updates user profile.
func (h *UserHandler) UpdateProfile(ctx context.Context, req *pb.UserProfileUpdateRequest) (*pb.UserProfileUpdateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Update profile request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid update profile request", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request: %s", err.Error()))
	}

	updatedUser, err := h.userService.UpdateProfile(ctx, user.ID, req.Email)
	if err != nil {
		return nil, err
	}

	profile, err := entity.MapUserToProfile(updatedUser, 0, h.storageLimit)
	if err != nil {
		logger.Error("Failed to map user to profile", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to map profile")
	}

	return &pb.UserProfileUpdateResponse{Profile: profile}, nil
}

// ChangePassword changes user password.
func (h *UserHandler) ChangePassword(ctx context.Context, req *pb.UserPasswordChangeRequest) (*pb.UserPasswordChangeResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Change password request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid change password request", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request: %s", err.Error()))
	}

	if err := h.userService.ChangePassword(ctx, user.ID, req.OldPassword, req.NewPassword); err != nil {
		return nil, err
	}

	return &pb.UserPasswordChangeResponse{Changed: true}, nil
}

// DeleteAccount deletes user account.
func (h *UserHandler) DeleteAccount(ctx context.Context, req *pb.UserAccountDeleteRequest) (*pb.UserAccountDeleteResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Delete account request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid delete account request", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request: %s", err.Error()))
	}

	if req.Confirmation != "DELETE_MY_ACCOUNT" {
		return nil, status.Error(codes.InvalidArgument, "confirmation string is required")
	}

	if err := h.userService.DeleteAccount(ctx, user.ID, req.Password); err != nil {
		return nil, err
	}

	return &pb.UserAccountDeleteResponse{Deleted: true}, nil
}

// GetActiveSessions returns all active sessions for the user.
func (h *UserHandler) GetActiveSessions(ctx context.Context, _ *pb.GetActiveSessionsRequest) (*pb.GetActiveSessionsResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Get active sessions request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	currentClientID := ""
	if token := GetAccessTokenFromContext(ctx); token != "" {
		currentClientID, _ = h.jwtService.ExtractClientID(token)
	}

	sessions, currentSession, err := h.userService.GetActiveSessions(ctx, user.ID, currentClientID)
	if err != nil {
		return nil, err
	}

	currentSessionID := ""
	if currentSession != nil {
		currentSessionID = currentSession.ID
	}

	items := make([]*pb.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		isCurrent := s.ID == currentSessionID
		info, err := entity.MapSessionToSessionInfo(s, isCurrent)
		if err != nil {
			logger.Warn("Failed to map session to info", zap.Error(err), zap.String("session_id", s.ID))
			continue
		}
		items = append(items, info)
	}

	return &pb.GetActiveSessionsResponse{Sessions: items}, nil
}

// RevokeSession revokes a specific session.
func (h *UserHandler) RevokeSession(ctx context.Context, req *pb.RevokeSessionRequest) (*pb.RevokeSessionResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Revoke session request received", zap.String("session_id", req.SessionId))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	if err := h.userService.RevokeSession(ctx, user.ID, req.SessionId); err != nil {
		return nil, err
	}

	return &pb.RevokeSessionResponse{Revoked: true}, nil
}

// RevokeAllSessions revokes all sessions, optionally except current.
func (h *UserHandler) RevokeAllSessions(ctx context.Context, req *pb.RevokeAllSessionsRequest) (*pb.RevokeAllSessionsResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Revoke all sessions request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	currentSessionID := ""
	if req.ExceptCurrent {
		if token := GetAccessTokenFromContext(ctx); token != "" {
			clientID, _ := h.jwtService.ExtractClientID(token)
			_, currentSession, _ := h.userService.GetActiveSessions(ctx, user.ID, clientID)
			if currentSession != nil {
				currentSessionID = currentSession.ID
			}
		}
	}

	revokedCount, err := h.userService.RevokeAllSessions(ctx, user.ID, currentSessionID, req.ExceptCurrent)
	if err != nil {
		return nil, err
	}

	return &pb.RevokeAllSessionsResponse{RevokedCount: revokedCount}, nil
}
