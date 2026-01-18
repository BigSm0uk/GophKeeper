package app

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/service"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
)

type HTTPServer struct {
	server      *http.Server
	logger      *zap.Logger
	config      *config.ServerConfig
	grpcServer  *GRPCServer
	fileService *service.FileService
}

// NewHTTPServer creates a new HTTP server with grpc-gateway
func NewHTTPServer(cfg *config.ServerConfig, logger *zap.Logger, grpcServer *GRPCServer) *HTTPServer {
	// Initialize file service for serving static files and documentation
	// Use ./api directory for OpenAPI/Swagger files
	apiDir := filepath.Join(".", "api")
	fileService, err := service.NewFileService(apiDir, logger)
	if err != nil {
		logger.Warn("Failed to initialize file service, static files won't be available",
			zap.Error(err),
			zap.String("api_dir", apiDir),
		)
		fileService = nil
	}

	return &HTTPServer{
		logger:      logger,
		config:      cfg,
		grpcServer:  grpcServer,
		fileService: fileService,
	}
}

// Start implements Lifecycle interface
func (s *HTTPServer) Start(ctx context.Context) error {
	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(s.customErrorHandler),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
	)

	if err := s.registerGatewayHandlers(ctx, mux); err != nil {
		return fmt.Errorf("failed to register gateway handlers: %w", err)
	}

	handler := s.createHandler(mux)

	address := s.config.HTTPAddress()
	s.server = &http.Server{
		Addr:         address,
		Handler:      handler,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("Starting HTTP server",
		zap.String("address", address),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Stop implements Lifecycle interface
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server...")

	if s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", zap.Error(err))
		return err
	}

	s.logger.Info("HTTP server stopped gracefully")
	return nil
}

// Name implements Lifecycle interface
func (s *HTTPServer) Name() string {
	return "HTTP Server (grpc-gateway)"
}

// registerGatewayHandlers registers all gateway handlers using in-process connection
func (s *HTTPServer) registerGatewayHandlers(
	ctx context.Context,
	mux *runtime.ServeMux,
) error {
	if err := pb.RegisterAuthServiceHandlerServer(ctx, mux, s.grpcServer.GetAuthHandler()); err != nil {
		return fmt.Errorf("failed to register AuthService handler: %w", err)
	}

	// TODO: Регистрируем остальные сервисы по мере их реализации
	// if err := pb.RegisterCredentialsServiceHandlerServer(ctx, mux, s.grpcServer.GetCredentialsHandler()); err != nil {
	// 	return fmt.Errorf("failed to register CredentialsService handler: %w", err)
	// }
	// if err := pb.RegisterTextsServiceHandlerServer(ctx, mux, s.grpcServer.GetTextsHandler()); err != nil {
	// 	return fmt.Errorf("failed to register TextsService handler: %w", err)
	// }
	// ... и т.д.

	s.logger.Info("Gateway handlers registered successfully (in-process)")
	return nil
}

// createHandler creates the main HTTP handler with middleware
func (s *HTTPServer) createHandler(gatewayMux *runtime.ServeMux) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", gatewayMux))

	mux.HandleFunc("/health", s.healthCheckHandler)

	// Serve documentation and static files (Swagger UI, OpenAPI spec)
	if s.config.IsDevelopment() {
		mux.HandleFunc("/swagger/", s.swaggerHandler)
	}

	handler := s.loggingMiddleware(mux)
	handler = s.corsMiddleware(handler)

	return handler
}

// healthCheckHandler - простой health check endpoint
func (s *HTTPServer) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// swaggerHandler serves Swagger UI and OpenAPI specification
func (s *HTTPServer) swaggerHandler(w http.ResponseWriter, r *http.Request) {
	if s.fileService == nil {
		http.Error(w, "Documentation not available", http.StatusNotFound)
		return
	}

	// Remove /swagger/ prefix to get the requested file
	requestedPath := strings.TrimPrefix(r.URL.Path, "/swagger/")

	if requestedPath == "" {
		requestedPath = "index.html"
	}

	resp, err := s.fileService.GetFile(requestedPath)
	if err != nil {
		s.logger.Error("Failed to serve Swagger file",
			zap.String("path", requestedPath),
			zap.Error(err),
		)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Content)
}

// loggingMiddleware logging all requests
func (s *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Debug("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", duration),
			zap.String("remote_addr", r.RemoteAddr),
		)
	})
}

// corsMiddleware add CORS headers
func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// customErrorHandler custome grpc-gateway error translator
func (s *HTTPServer) customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)

	s.logger.Error("Gateway error",
		zap.String("path", r.URL.Path),
		zap.String("method", r.Method),
		zap.Error(err),
	)
}

// responseWriter wrapper
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
