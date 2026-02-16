package config

import "time"

type JWTConfig struct {
	PrivateKeyPath  string        `mapstructure:"private_key_path"`
	PublicKeyPath   string        `mapstructure:"public_key_path"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
}

// NewDefaultJWTConfig return default configuraiton
func NewDefaultJWTConfig() JWTConfig {
	return JWTConfig{
		PrivateKeyPath:  "keys/jwt_private.pem",
		PublicKeyPath:   "keys/jwt_public.pem",
		AccessTokenTTL:  15 * time.Minute,   // 15 m
		RefreshTokenTTL: 7 * 24 * time.Hour, // 7 d
		Issuer:          "gophkeeper",
	}
}
