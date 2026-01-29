package grpc

import (
	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	case errors.Is(err, models.ErrSessionNotFound):
		return status.Error(codes.Unauthenticated, "Session not found or expired")

	case errors.Is(err, models.ErrSessionExpired):
		return status.Error(codes.Unauthenticated, "Session expired")

	case errors.Is(err, models.ErrSessionRevoked):
		return status.Error(codes.Unauthenticated, "Session revoked")

	case errors.Is(err, models.ErrDeviceMismatch):
		return status.Error(codes.PermissionDenied, "Device mismatch detected - potential security breach")

	case errors.Is(err, models.ErrTooManySessions):
		return status.Error(codes.ResourceExhausted, "Too many active sessions")

	default:
		// For any other errors, don't expose internal details to client
		h.logger.Error("Unexpected service error in handler", zap.Error(err))
		return status.Error(codes.Internal, "An internal error occurred")
	}
}
