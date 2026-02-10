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

// CredentialsHandler implements CredentialServiceServer
type CredentialsHandler struct {
	pb.UnimplementedCredentialsServiceServer
	logger             *zap.Logger
	credentialsService *service.CredentialsService
}

// NewCredentialsHandler creates a new credentials handler
func NewCredentialsHandler(logger *zap.Logger, credentialsService *service.CredentialsService) *CredentialsHandler {
	return &CredentialsHandler{
		logger:             logger,
		credentialsService: credentialsService,
	}
}

// Create stores a new credential entry
func (h *CredentialsHandler) Create(ctx context.Context, req *pb.CredentialCreateRequest) (*pb.CredentialCreateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Create credential request received",
		zap.String("name", req.Name))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid create credential request data",
			zap.Error(err),
			zap.String("name", req.Name))
		errorMsg := err.Error()
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid credential data: %s", errorMsg))
	}

	cred, err := entity.MapCredentialFromRequest(user.ID, req)
	if err != nil {
		logger.Error("Failed to map request to credential entity",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, status.Error(codes.InvalidArgument, "Invalid credential data")
	}

	createdCred, err := h.credentialsService.CreateCredential(ctx, user.ID, cred)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCred, err := entity.MapCredentialToResponse(createdCred)
	if err != nil {
		logger.Error("Failed to map credential to response",
			zap.Error(err),
			zap.String("credential_id", createdCred.ID))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Create credential request completed successfully",
		zap.String("credential_id", createdCred.ID),
		zap.String("user_id", user.ID))

	return &pb.CredentialCreateResponse{
		Credential: pbCred,
	}, nil
}

// Get returns a single credential by id
func (h *CredentialsHandler) Get(ctx context.Context, req *pb.CredentialGetRequest) (*pb.CredentialGetResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Get credential request received",
		zap.String("credential_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid get credential request data",
			zap.Error(err),
			zap.String("credential_id", req.Id))
		errorMsg := err.Error()
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request data: %s", errorMsg))
	}

	cred, err := h.credentialsService.GetCredential(ctx, user.ID, req.Id)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCred, err := entity.MapCredentialToResponse(cred)
	if err != nil {
		logger.Error("Failed to map credential to response",
			zap.Error(err),
			zap.String("credential_id", req.Id))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Get credential request completed successfully",
		zap.String("credential_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.CredentialGetResponse{
		Credential: pbCred,
	}, nil
}

// List returns paginated credentials
func (h *CredentialsHandler) List(ctx context.Context, req *pb.CredentialListRequest) (*pb.CredentialListResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("List credentials request received")

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.Page.Limit)
	offset := int(req.Page.Offset)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	creds, total, err := h.credentialsService.ListCredentials(ctx, user.ID, limit, offset)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCreds := make([]*pb.Credential, 0, len(creds))
	for _, cred := range creds {
		// Use MapCredentialToListItem to exclude sensitive data in list view
		pbCred, err := entity.MapCredentialToListItem(cred)
		if err != nil {
			logger.Warn("Failed to map credential to list item",
				zap.Error(err),
				zap.String("credential_id", cred.ID))
			continue
		}
		pbCreds = append(pbCreds, pbCred)
	}

	logger.Info("List credentials request completed successfully",
		zap.String("user_id", user.ID),
		zap.Int("count", len(pbCreds)))

	return &pb.CredentialListResponse{
		Items: pbCreds,
		Page: &pb.PageResponse{
			Total:  uint32(total),
			Limit:  uint32(limit),
			Offset: uint32(offset),
		},
	}, nil
}

// Update modifies an existing credential
func (h *CredentialsHandler) Update(ctx context.Context, req *pb.CredentialUpdateRequest) (*pb.CredentialUpdateResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Update credential request received",
		zap.String("credential_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid update credential request data",
			zap.Error(err),
			zap.String("credential_id", req.Id))
		errorMsg := err.Error()
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid credential data: %s", errorMsg))
	}

	// Get existing credential first
	existingCred, err := h.credentialsService.GetCredential(ctx, user.ID, req.Id)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	// Update fields from request
	if err := entity.MapCredentialFromUpdateRequest(existingCred, req); err != nil {
		logger.Error("Failed to map update request to credential",
			zap.Error(err),
			zap.String("credential_id", req.Id))
		return nil, status.Error(codes.InvalidArgument, "Invalid credential data")
	}

	updatedCred, err := h.credentialsService.UpdateCredential(ctx, user.ID, existingCred)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	pbCred, err := entity.MapCredentialToResponse(updatedCred)
	if err != nil {
		logger.Error("Failed to map credential to response",
			zap.Error(err),
			zap.String("credential_id", req.Id))
		return nil, status.Error(codes.Internal, "Failed to map response")
	}

	logger.Info("Update credential request completed successfully",
		zap.String("credential_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.CredentialUpdateResponse{
		Credential: pbCred,
	}, nil
}

// Delete removes a credential entry
func (h *CredentialsHandler) Delete(ctx context.Context, req *pb.CredentialDeleteRequest) (*pb.CredentialDeleteResponse, error) {
	logger := GetLoggerFromContext(ctx, h.logger)

	logger.Info("Delete credential request received",
		zap.String("credential_id", req.Id))

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		logger.Warn("Invalid delete credential request data",
			zap.Error(err),
			zap.String("credential_id", req.Id))
		errorMsg := err.Error()
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid request data: %s", errorMsg))
	}

	err = h.credentialsService.DeleteCredential(ctx, user.ID, req.Id)
	if err != nil {
		return nil, classifyServiceError(h.logger, err)
	}

	logger.Info("Delete credential request completed successfully",
		zap.String("credential_id", req.Id),
		zap.String("user_id", user.ID))

	return &pb.CredentialDeleteResponse{
		Deleted: true,
	}, nil
}
