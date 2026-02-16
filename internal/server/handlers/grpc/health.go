package grpc

import (
	"context"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
)

var (
	from = time.Now()
)

// HealthHandler implements HealthServiceServer.
type HealthHandler struct {
	pb.UnimplementedHealthServiceServer
	logger *zap.Logger
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

// Check implements HealthServiceServer.
func (h *HealthHandler) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)
	logger.Info("Health check request received")

	return &pb.HealthCheckResponse{
		Status:        "healthy",
		UptimeSeconds: int64(time.Since(from).Seconds()),
	}, nil
}
