package grpc

import (
	"testing"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
)

func TestRequiresAuth(t *testing.T) {
	cfg := config.NewDefaultAuthConfig()

	tests := []struct {
		name     string
		method   string
		wantAuth bool
	}{
		{
			name:     "public method - register",
			method:   "/gophkeeper.v1.AuthService/Register",
			wantAuth: false,
		},
		{
			name:     "public method - token",
			method:   "/gophkeeper.v1.AuthService/Token",
			wantAuth: false,
		},
		{
			name:     "public method - health check",
			method:   "/grpc.health.v1.Health/Check",
			wantAuth: false,
		},
		{
			name:     "protected method - revoke",
			method:   "/gophkeeper.v1.AuthService/Revoke",
			wantAuth: true,
		},
		{
			name:     "protected method - texts create",
			method:   "/gophkeeper.v1.TextsService/Create",
			wantAuth: true,
		},
		{
			name:     "protected method - credentials list",
			method:   "/gophkeeper.v1.CredentialsService/List",
			wantAuth: true,
		},
		{
			name:     "non-gophkeeper method",
			method:   "/some.other.Service/Method",
			wantAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresAuth(tt.method, cfg)
			if got != tt.wantAuth {
				t.Errorf("requiresAuth(%q) = %v, want %v", tt.method, got, tt.wantAuth)
			}
		})
	}
}

func TestAuthConfig_IsPublicMethod(t *testing.T) {
	cfg := config.AuthConfig{
		PublicMethods: []string{
			"/gophkeeper.v1.AuthService/Register",
			"/gophkeeper.v1.AuthService/Token",
		},
	}

	tests := []struct {
		name     string
		method   string
		isPublic bool
	}{
		{
			name:     "registered public method",
			method:   "/gophkeeper.v1.AuthService/Register",
			isPublic: true,
		},
		{
			name:     "not in list",
			method:   "/gophkeeper.v1.AuthService/Revoke",
			isPublic: false,
		},
		{
			name:     "empty method",
			method:   "",
			isPublic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.IsPublicMethod(tt.method)
			if got != tt.isPublic {
				t.Errorf("IsPublicMethod(%q) = %v, want %v", tt.method, got, tt.isPublic)
			}
		})
	}
}
