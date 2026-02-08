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

// NewCredential creates a new credential with validation.
func NewCredential(userID, name, login, password string, url, metadata *string) (*Credential, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if name == "" {
		return nil, ErrInvalidName
	}
	if login == "" {
		return nil, ErrInvalidLogin
	}
	if password == "" {
		return nil, ErrInvalidPassword
	}

	now := time.Now()
	credential := &Credential{
		UserID:    userID,
		Name:      name,
		Login:     login,
		Password:  password,
		URL:       url,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return credential, nil
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

// IsOwnedBy checks if the credential is owned by the specified user.
func (c *Credential) IsOwnedBy(userID string) bool {
	return c.UserID == userID
}

// MaskSensitiveData returns a copy of the credential with sensitive data masked.
// Useful for logging or displaying in lists.
func (c *Credential) MaskSensitiveData() *Credential {
	masked := *c
	masked.Password = "***masked***"
	return &masked
}
