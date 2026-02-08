package api

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthChecker provides methods to check server availability.
type HealthChecker struct {
	conn   *grpc.ClientConn
	client grpc_health_v1.HealthClient
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(conn *grpc.ClientConn) *HealthChecker {
	return &HealthChecker{
		conn:   conn,
		client: grpc_health_v1.NewHealthClient(conn),
	}
}

// IsHealthy checks if the server is available and healthy.
// Returns true if server is reachable and serving.
func (h *HealthChecker) IsHealthy(ctx context.Context) bool {
	// Проверяем состояние соединения
	state := h.conn.GetState()
	if state == connectivity.Shutdown || state == connectivity.TransientFailure {
		return false
	}

	// Делаем health check запрос с коротким таймаутом
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := h.client.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{
		Service: "", // пустой означает проверку всего сервера
	})
	if err != nil {
		return false
	}

	return resp.Status == grpc_health_v1.HealthCheckResponse_SERVING
}

// QuickPing делает быстрый пинг сервера для проверки доступности.
// Более легковесный, чем полный health check.
func (h *HealthChecker) QuickPing() bool {
	state := h.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// WaitForReady ждет, пока соединение не станет готовым или не истечет таймаут.
func (h *HealthChecker) WaitForReady(ctx context.Context) bool {
	return h.conn.WaitForStateChange(ctx, connectivity.TransientFailure)
}
