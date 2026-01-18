package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client обёртка вокруг gRPC клиента.
type Client struct {
	conn        *grpc.ClientConn
	auth        pb.AuthServiceClient
	accessToken string
	timeout     time.Duration
}

// New создаёт новый gRPC клиент.
func New(address string, insecureTLS bool, timeout time.Duration) (*Client, error) {
	var dialOpts []grpc.DialOption
	if insecureTLS {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial grpc: %w", err)
	}

	return &Client{
		conn:    conn,
		auth:    pb.NewAuthServiceClient(conn),
		timeout: timeout,
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// SetAccessToken обновляет токен для будущих запросов.
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

func (c *Client) withAuth(ctx context.Context) context.Context {
	if c.accessToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.accessToken)
}

func (c *Client) timeoutCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

// Register вызывает AuthService.Register.
func (c *Client) Register(ctx context.Context, username, password, email string) (*pb.RegisterResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()

	req := &pb.RegisterRequest{
		Username: username,
		Password: password,
	}
	if email != "" {
		req.Email = &email
	}

	return c.auth.Register(ctx, req)
}

// PasswordToken запрашивает access/refresh токены по паролю.
func (c *Client) PasswordToken(ctx context.Context, username, password, clientID, clientSecret, scope string) (*pb.TokenResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()

	req := &pb.TokenRequest{
		GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_PASSWORD,
		Username:  username,
		Password:  password,
		ClientId:  clientID,
	}
	if clientSecret != "" {
		req.ClientSecret = &clientSecret
	}
	if scope != "" {
		req.Scope = &scope
	}

	return c.auth.Token(ctx, req)
}

// RefreshToken запрашивает новый access токен по refresh токену.
func (c *Client) RefreshToken(ctx context.Context, refreshToken, clientID, clientSecret string) (*pb.TokenResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()

	req := &pb.TokenRequest{
		GrantType:    pb.TokenGrantType_TOKEN_GRANT_TYPE_REFRESH_TOKEN,
		RefreshToken: refreshToken,
		ClientId:     clientID,
	}
	if clientSecret != "" {
		req.ClientSecret = &clientSecret
	}

	return c.auth.Token(ctx, req)
}

// Revoke отзывает access/refresh токен.
func (c *Client) Revoke(ctx context.Context, token, clientID, clientSecret string, hint pb.TokenTypeHint) (*pb.RevokeResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()

	req := &pb.RevokeRequest{
		Token:         token,
		TokenTypeHint: hint,
		ClientId:      clientID,
	}
	if clientSecret != "" {
		req.ClientSecret = &clientSecret
	}

	ctx = c.withAuth(ctx)
	return c.auth.Revoke(ctx, req)
}
