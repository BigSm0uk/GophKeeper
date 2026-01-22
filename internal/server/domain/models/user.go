package models

import (
	"time"
)

// User represents a user in the domain layer.
type User struct {
	ID             string
	Username       string
	Email          *string
	HashedPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewUser creates a new user with validation.
func NewUser(username, password string, email *string) (*User, error) {
	if username == "" {
		return nil, ErrInvalidUsername
	}
	if password == "" {
		return nil, ErrInvalidPassword
	}

	now := time.Now()
	user := &User{
		Username:       username,
		Email:          email,
		HashedPassword: password,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return user, nil
}

// UpdateEmail updates the user's email and sets the updated timestamp.
func (u *User) UpdateEmail(email *string) {
	u.Email = email
	u.UpdatedAt = time.Now()
}

// UpdatePassword updates the user's password and sets the updated timestamp.
func (u *User) UpdatePassword(password string) error {
	if password == "" {
		return ErrInvalidPassword
	}
	u.HashedPassword = password
	u.UpdatedAt = time.Now()
	return nil
}

// IsActive checks if the user account is active.
// In this simple implementation, all users are considered active.
func (u *User) IsActive() bool {
	return true
}

// GetDisplayName returns the username as display name.
func (u *User) GetDisplayName() string {
	return u.Username
}

// HasEmail checks if the user has an email address set.
func (u *User) HasEmail() bool {
	return u.Email != nil && *u.Email != ""
}
