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
	DeletedAt *time.Time
}

func (t *Text) GetID() string {
	return t.ID
}

func (t *Text) SetID(id string) {
	t.ID = id
}

func (t *Text) GetUserID() string {
	return t.UserID
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
