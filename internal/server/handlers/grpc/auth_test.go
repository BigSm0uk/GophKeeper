package grpc

import (
	"testing"

	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
)

func TestValidateTokenRequest(t *testing.T) {
	logger := zap.NewNop()
	handler := NewAuthHandler(logger, nil, config.NewDefaultJWTConfig())

	tests := []struct {
		name    string
		req     *pb.TokenRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid password grant",
			req: &pb.TokenRequest{
				GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
				Username:  "testuser",
				Password:  "password123",
				ClientId:  "test-client",
			},
			wantErr: false,
		},
		{
			name: "password grant without username",
			req: &pb.TokenRequest{
				GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
				Password:  "password123",
			},
			wantErr: true,
			errMsg:  "username is required for password grant",
		},
		{
			name: "password grant without password",
			req: &pb.TokenRequest{
				GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
				Username:  "testuser",
			},
			wantErr: true,
			errMsg:  "password is required for password grant",
		},
		{
			name: "valid refresh_token grant",
			req: &pb.TokenRequest{
				GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_REFRESH_TOKEN,
				Username:  "testuser",
				ClientId:  "test-client",
			},
			wantErr: false,
		},
		{
			name: "refresh_token grant without username",
			req: &pb.TokenRequest{
				GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_REFRESH_TOKEN,
			},
			wantErr: false,
		},
		{
			name: "unsupported grant type",
			req: &pb.TokenRequest{
				GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_UNSPECIFIED,
			},
			wantErr: true,
			errMsg:  "unsupported grant type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateTokenRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTokenRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validateTokenRequest() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
