package models

import (
	"time"
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
	DeletedAt *time.Time
}

func (c *Credential) GetID() string {
	return c.ID
}

func (c *Credential) SetID(id string) {
	c.ID = id
}

func (c *Credential) GetUserID() string {
	return c.UserID
}

// UpdatePassword updates the password and sets the updated timestamp.
func (c *Credential) UpdatePassword(password string) error {
	if password == "" {
		return ErrInvalidPassword
	}
	c.Password = password
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateMetadata updates the metadata and sets the updated timestamp.
func (c *Credential) UpdateMetadata(metadata *string) {
	c.Metadata = metadata
	c.UpdatedAt = time.Now()
}

// UpdateURL updates the URL and sets the updated timestamp.
func (c *Credential) UpdateURL(url *string) {
	c.URL = url
	c.UpdatedAt = time.Now()
}
