package config

import "time"

type JWTConfig struct {
	// PrivateKeyPath - путь к приватному RSA ключу
	PrivateKeyPath string `mapstructure:"private_key_path"`
	// PublicKeyPath - путь к публичному RSA ключу
	PublicKeyPath string `mapstructure:"public_key_path"`
	// AccessTokenTTL - время жизни access токена
	AccessTokenTTL time.Duration `mapstructure:"access_token_ttl"`
	// RefreshTokenTTL - время жизни refresh токена
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	// Issuer - issuer для JWT токенов
	Issuer string `mapstructure:"issuer"`
}

// NewDefaultJWTConfig возвращает конфигурацию JWT по умолчанию
func NewDefaultJWTConfig() JWTConfig {
	return JWTConfig{
		PrivateKeyPath:  "keys/jwt_private.pem",
		PublicKeyPath:   "keys/jwt_public.pem",
		AccessTokenTTL:  15 * time.Minute,   // 15 минут
		RefreshTokenTTL: 7 * 24 * time.Hour, // 7 дней
		Issuer:          "gophkeeper",
	}
}
