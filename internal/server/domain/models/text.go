package models

import (
	"time"
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

// NewText creates a new text entry with validation.
func NewText(userID, name, content string, metadata *string) (*Text, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if name == "" {
		return nil, ErrInvalidName
	}
	if content == "" {
		return nil, ErrInvalidContent
	}

	now := time.Now()
	text := &Text{
		UserID:    userID,
		Name:      name,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return text, nil
}

// UpdateContent updates the text content and sets the updated timestamp.
func (t *Text) UpdateContent(content string) error {
	if content == "" {
		return ErrInvalidContent
	}
	t.Content = content
	t.UpdatedAt = time.Now()
	return nil
}

// UpdateMetadata updates the metadata and sets the updated timestamp.
func (t *Text) UpdateMetadata(metadata *string) {
	t.Metadata = metadata
	t.UpdatedAt = time.Now()
}

// IsOwnedBy checks if the text is owned by the specified user.
func (t *Text) IsOwnedBy(userID string) bool {
	return t.UserID == userID
}

// GetContentLength returns the length of the content in bytes.
func (t *Text) GetContentLength() int {
	return len(t.Content)
}

// IsEmpty checks if the text content is empty.
func (t *Text) IsEmpty() bool {
	return t.Content == ""
}

// TruncateContent truncates content to specified length and adds ellipsis.
func (t *Text) TruncateContent(maxLength int) string {
	if len(t.Content) <= maxLength {
		return t.Content
	}
	if maxLength <= 3 {
		return t.Content[:maxLength]
	}
	return t.Content[:maxLength-3] + "..."
}
