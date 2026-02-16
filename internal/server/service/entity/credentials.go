package entity

import (
	"fmt"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Credential represents a login/password entry in the domain layer.
type Credential struct {
	ID        string
	UserID    string
	Name      string
	Login     string
	Password  string
	URL       *string
	Metadata  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MapCredentialToResponse converts domain Credential to protobuf Credential message.
func MapCredentialToResponse(cred *Credential) (*pb.Credential, error) {
	if cred == nil {
		return nil, fmt.Errorf("credential is nil")
	}

	response := &pb.Credential{
		Id:        cred.ID,
		Name:      cred.Name,
		Login:     cred.Login,
		Password:  cred.Password,
		Url:       cred.URL,
		Metadata:  cred.Metadata,
		CreatedAt: timestamppb.New(cred.CreatedAt),
		UpdatedAt: timestamppb.New(cred.UpdatedAt),
	}

	return response, nil
}

// MapCredentialFromRequest converts protobuf CredentialCreateRequest to domain Credential.
func MapCredentialFromRequest(userID string, req *pb.CredentialCreateRequest) (*Credential, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	now := time.Now()
	credential := &Credential{
		UserID:    userID,
		Name:      req.Name,
		Login:     req.Login,
		Password:  req.Password,
		URL:       req.Url,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return credential, nil
}

// MapCredentialFromUpdateRequest updates domain Credential from protobuf CredentialUpdateRequest.
func MapCredentialFromUpdateRequest(existing *Credential, req *pb.CredentialUpdateRequest) error {
	if existing == nil {
		return fmt.Errorf("existing credential is nil")
	}
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	existing.Name = req.Name
	existing.Login = req.Login
	existing.Password = req.Password
	existing.URL = req.Url
	existing.Metadata = req.Metadata
	existing.UpdatedAt = time.Now()

	return nil
}

// MapCredentialToListItem converts domain Credential to protobuf Credential for list responses.
// This version excludes sensitive data like password.
func MapCredentialToListItem(cred *Credential) (*pb.Credential, error) {
	if cred == nil {
		return nil, fmt.Errorf("credential is nil")
	}

	// For list responses, we don't include password and metadata for security
	response := &pb.Credential{
		Id:        cred.ID,
		Name:      cred.Name,
		Login:     cred.Login,
		Url:       cred.URL,
		CreatedAt: timestamppb.New(cred.CreatedAt),
		UpdatedAt: timestamppb.New(cred.UpdatedAt),
		// Password and Metadata are omitted for list views
	}

	return response, nil
}
