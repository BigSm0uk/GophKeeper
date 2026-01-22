package entity

import (
	"fmt"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Text represents a text entry in the domain layer.
type Text struct {
	ID        string
	UserID    string
	Name      string
	Content   string
	Metadata  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MapTextToResponse converts domain Text to protobuf Text message.
func MapTextToResponse(text *Text) (*pb.Text, error) {
	if text == nil {
		return nil, fmt.Errorf("text is nil")
	}

	response := &pb.Text{
		Id:        text.ID,
		Name:      text.Name,
		Content:   text.Content,
		Metadata:  text.Metadata,
		CreatedAt: timestamppb.New(text.CreatedAt),
		UpdatedAt: timestamppb.New(text.UpdatedAt),
	}

	return response, nil
}

// MapTextFromRequest converts protobuf TextCreateRequest to domain Text.
func MapTextFromRequest(userID string, req *pb.TextCreateRequest) (*Text, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	now := time.Now()
	text := &Text{
		UserID:    userID,
		Name:      req.Name,
		Content:   req.Content,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return text, nil
}

// MapTextFromUpdateRequest updates domain Text from protobuf TextUpdateRequest.
func MapTextFromUpdateRequest(existing *Text, req *pb.TextUpdateRequest) error {
	if existing == nil {
		return fmt.Errorf("existing text is nil")
	}
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	existing.Name = req.Name
	existing.Content = req.Content
	existing.Metadata = req.Metadata
	existing.UpdatedAt = time.Now()

	return nil
}

// MapTextToListItem converts domain Text to protobuf Text for list responses.
// This version excludes content for list views.
func MapTextToListItem(text *Text) (*pb.Text, error) {
	if text == nil {
		return nil, fmt.Errorf("text is nil")
	}

	// For list responses, we don't include content for brevity
	response := &pb.Text{
		Id:        text.ID,
		Name:      text.Name,
		CreatedAt: timestamppb.New(text.CreatedAt),
		UpdatedAt: timestamppb.New(text.UpdatedAt),
		// Content and Metadata are omitted for list views
	}

	return response, nil
}
