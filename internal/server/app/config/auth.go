package config

import "slices"

// AuthConfig holds authentication configuration
type AuthConfig struct {
	PublicMethods []string `mapstructure:"public_methods"`
}

// NewDefaultAuthConfig returns default authentication configuration
func NewDefaultAuthConfig() AuthConfig {
	return AuthConfig{
		PublicMethods: []string{
			"/gophkeeper.v1.AuthService/Register",
			"/gophkeeper.v1.AuthService/Token",
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
			"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
		},
	}
}

// IsPublicMethod checks if a method is in the public methods list
func (ac *AuthConfig) IsPublicMethod(fullMethod string) bool {
	return slices.Contains(ac.PublicMethods, fullMethod)
}
