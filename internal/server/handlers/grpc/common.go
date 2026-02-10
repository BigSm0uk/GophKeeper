package grpc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// classifyServiceError classifies service errors into appropriate gRPC status codes
// Note: This function only transforms errors, logging is handled at the service layer
func classifyServiceError(logger *zap.Logger, err error) error {
	// Check for pgx.ErrNoRows which indicates entity not found
	errMsg := err.Error()
	if errors.Is(err, pgx.ErrNoRows) || strings.Contains(errMsg, "no rows in result set") {
		return status.Error(codes.NotFound, "Resource not found")
	}

	switch {
	case errors.Is(err, models.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "User with this username already exists")

	case errors.Is(err, models.ErrInvalidUsername):
		return status.Error(codes.InvalidArgument, "Username does not meet requirements")

	case errors.Is(err, models.ErrInvalidPassword):
		return status.Error(codes.InvalidArgument, "Password does not meet requirements")
	case errors.Is(err, models.ErrInvalidCredential):
		return status.Error(codes.InvalidArgument, "Invalid credential data")
	case errors.Is(err, models.ErrInvalidText):
		return status.Error(codes.InvalidArgument, "Invalid text data")
	case errors.Is(err, models.ErrInvalidCard):
		return status.Error(codes.InvalidArgument, "Invalid card data")
	case errors.Is(err, models.ErrInvalidBinary):
		return status.Error(codes.InvalidArgument, "Invalid binary data")

	case errors.Is(err, models.ErrInvalidUserID):
		return status.Error(codes.InvalidArgument, "Invalid user identifier")

	case errors.Is(err, models.ErrUserNotFound):
		return status.Error(codes.NotFound, "User not found")
	case errors.Is(err, models.ErrBinaryNotFound):
		return status.Error(codes.NotFound, "Binary not found")
	case errors.Is(err, models.ErrTextNotFound):
		return status.Error(codes.NotFound, "Text not found")
	case errors.Is(err, models.ErrCardNotFound):
		return status.Error(codes.NotFound, "Card not found")
	case errors.Is(err, models.ErrCredentialNotFound):
		return status.Error(codes.NotFound, "Credential not found")
	case errors.Is(err, models.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, "Access denied")

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
		logger.Error("Unexpected service error in handler",
			zap.Error(err),
			zap.String("type", fmt.Sprintf("%T", err)))
		return status.Error(codes.Internal, "An internal error occurred")
	}
}
