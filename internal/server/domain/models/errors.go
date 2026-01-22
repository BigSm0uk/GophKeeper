package models

import "errors"

// Domain errors for validation
var (
	ErrInvalidUserID         = errors.New("invalid user ID")
	ErrInvalidUsername       = errors.New("invalid username: cannot be empty")
	ErrInvalidName           = errors.New("invalid name: cannot be empty")
	ErrInvalidLogin          = errors.New("invalid login: cannot be empty")
	ErrInvalidPassword       = errors.New("invalid password: cannot be empty")
	ErrInvalidContent        = errors.New("invalid content: cannot be empty")
	ErrInvalidCardNumber     = errors.New("invalid card number: cannot be empty")
	ErrInvalidCardholderName = errors.New("invalid cardholder name: cannot be empty")
	ErrInvalidExpiryDate     = errors.New("invalid expiry date: cannot be empty")
	ErrInvalidCVV            = errors.New("invalid CVV: cannot be empty")
	ErrInvalidFilename       = errors.New("invalid filename: cannot be empty")
	ErrInvalidFileSize       = errors.New("invalid file size: must be greater than 0")
	ErrInvalidContentType    = errors.New("invalid content type: cannot be empty")
	ErrInvalidURL            = errors.New("invalid URL format")
	ErrCredentialNotFound    = errors.New("credential not found")
	ErrTextNotFound          = errors.New("text not found")
	ErrCardNotFound          = errors.New("card not found")
	ErrBinaryNotFound        = errors.New("binary not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrAccessDenied          = errors.New("access denied: item not owned by user")
	ErrUserAlreadyExists     = errors.New("user already exist")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalidToken          = errors.New("invalid token")
	ErrTokenExpired          = errors.New("token expired")
	ErrInvalidGrantType      = errors.New("invalid grant type")
)
