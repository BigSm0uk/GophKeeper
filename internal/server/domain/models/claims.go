package models

import "github.com/golang-jwt/jwt/v5"

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	ClientID string `json:"client_id"`
	jwt.RegisteredClaims
}
