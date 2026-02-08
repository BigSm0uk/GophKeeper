package service

import (
	"context"
	"errors"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/interfaces"
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/BigSm0uk/GophKeeper/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService handles user profile and session operations.
type UserService struct {
	logger      *zap.Logger
	userRepo    interfaces.UserRepository
	sessionRepo interfaces.SessionRepository
}

// NewUserService creates a new user service.
func NewUserService(logger *zap.Logger, userRepo interfaces.UserRepository, sessionRepo interfaces.SessionRepository) *UserService {
	return &UserService{
		logger:      logger,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// GetProfile returns the current user profile.
func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("Failed to find user for profile",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Error(codes.Internal, "failed to retrieve profile")
	}
	return user, nil
}

// UpdateProfile updates user profile (email).
func (s *UserService) UpdateProfile(ctx context.Context, userID string, email *string) (*models.User, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("Failed to find user for update",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Error(codes.Internal, "failed to retrieve profile")
	}
	user.UpdateEmail(email)
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user profile",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Error(codes.Internal, "failed to update profile")
	}
	s.logger.Info("User profile updated",
		zap.String("user_id", userID))
	updated, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve updated profile")
	}
	return updated, nil
}

// ChangePassword changes user password.
func (s *UserService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user GetID is required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("Failed to find user for password change",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to retrieve user")
	}
	valid, err := util.VerifyPassword(oldPassword, user.HashedPassword)
	if err != nil {
		s.logger.Error("Failed to verify password",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to verify password")
	}
	if !valid {
		return status.Error(codes.Unauthenticated, "invalid old password")
	}
	hashed, err := util.HashPassword(newPassword)
	if err != nil {
		s.logger.Error("Failed to hash new password",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to update password")
	}
	if err := user.UpdatePassword(hashed); err != nil {
		return status.Error(codes.InvalidArgument, "invalid new password")
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update password",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to update password")
	}
	s.logger.Info("User password changed",
		zap.String("user_id", userID))
	return nil
}

// DeleteAccount soft-deletes user account after password confirmation.
func (s *UserService) DeleteAccount(ctx context.Context, userID, password string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user GetID is required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("Failed to find user for account deletion",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to retrieve user")
	}
	valid, err := util.VerifyPassword(password, user.HashedPassword)
	if err != nil {
		s.logger.Error("Failed to verify password for account deletion",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to verify password")
	}
	if !valid {
		return status.Error(codes.Unauthenticated, "invalid password")
	}
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		s.logger.Error("Failed to delete user account",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to delete account")
	}
	s.logger.Info("User account deleted",
		zap.String("user_id", userID))
	return nil
}

// GetActiveSessions returns all active sessions for the user.
func (s *UserService) GetActiveSessions(ctx context.Context, userID, currentClientID string) ([]*models.Session, *models.Session, error) {
	if userID == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	sessions, err := s.sessionRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to list active sessions",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, nil, status.Error(codes.Internal, "failed to list sessions")
	}
	var currentSession *models.Session
	if currentClientID != "" {
		currentSession, _ = s.sessionRepo.FindByUserIDAndClientID(ctx, userID, currentClientID)
	}
	return sessions, currentSession, nil
}

// RevokeSession revokes a specific session (must belong to user).
func (s *UserService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if userID == "" || sessionID == "" {
		return status.Error(codes.InvalidArgument, "user GetID and session GetID are required")
	}
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			return status.Error(codes.NotFound, "session not found")
		}
		s.logger.Error("Failed to find session for revoke",
			zap.Error(err),
			zap.String("session_id", sessionID))
		return status.Error(codes.Internal, "failed to retrieve session")
	}
	if session.UserID != userID {
		return status.Error(codes.PermissionDenied, "access denied")
	}
	if err := s.sessionRepo.Revoke(ctx, sessionID, "user_revoked"); err != nil {
		s.logger.Error("Failed to revoke session",
			zap.Error(err),
			zap.String("session_id", sessionID))
		return status.Error(codes.Internal, "failed to revoke session")
	}
	s.logger.Info("Session revoked",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID))
	return nil
}

// RevokeAllSessions revokes all sessions, optionally except the current one.
func (s *UserService) RevokeAllSessions(ctx context.Context, userID, currentSessionID string, exceptCurrent bool) (int32, error) {
	if userID == "" {
		return 0, status.Error(codes.InvalidArgument, "user GetID is required")
	}
	countBefore, err := s.sessionRepo.CountActiveByUserID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to count active sessions", zap.Error(err), zap.String("user_id", userID))
		countBefore = 0
	}
	var revoked int32
	if exceptCurrent && currentSessionID != "" {
		if err := s.sessionRepo.RevokeAllExceptCurrent(ctx, userID, currentSessionID, "user_revoked"); err != nil {
			s.logger.Error("Failed to revoke all sessions except current",
				zap.Error(err),
				zap.String("user_id", userID))
			return 0, status.Error(codes.Internal, "failed to revoke sessions")
		}
		if countBefore > 1 {
			revoked = int32(countBefore - 1)
		}
		s.logger.Info("Sessions revoked except current",
			zap.String("user_id", userID),
			zap.Int32("revoked_count", revoked))
		return revoked, nil
	}
	if err := s.sessionRepo.RevokeAllByUserID(ctx, userID, "user_revoked"); err != nil {
		s.logger.Error("Failed to revoke all sessions",
			zap.Error(err),
			zap.String("user_id", userID))
		return 0, status.Error(codes.Internal, "failed to revoke sessions")
	}
	revoked = int32(countBefore)
	s.logger.Info("All sessions revoked",
		zap.String("user_id", userID),
		zap.Int32("revoked_count", revoked))
	return revoked, nil
}
