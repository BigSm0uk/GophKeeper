package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client обёртка вокруг gRPC клиента.
type Client struct {
	conn          *grpc.ClientConn
	auth          pb.AuthServiceClient
	credentials   pb.CredentialsServiceClient
	cards         pb.CardsServiceClient
	texts         pb.TextsServiceClient
	healthChecker *HealthChecker
	accessToken   string
	timeout       time.Duration
}

const (
	defaultAttempts      = 3
	defaultBackoffMillis = 200
)

// New создаёт новый gRPC клиент.
func New(address string, insecureTLS bool, timeout time.Duration, logger *zap.Logger) (*Client, error) {
	var dialOpts []grpc.DialOption
	if insecureTLS {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	conn, err := grpc.NewClient(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial grpc: %w", err)
	}

	_ = logger // используется для будущих клиентов

	return &Client{
		conn:          conn,
		auth:          pb.NewAuthServiceClient(conn),
		credentials:   pb.NewCredentialsServiceClient(conn),
		cards:         pb.NewCardsServiceClient(conn),
		texts:         pb.NewTextsServiceClient(conn),
		healthChecker: NewHealthChecker(conn),
		timeout:       timeout,
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// GetConn возвращает gRPC соединение для использования в streaming клиентах.
func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}

// SetAccessToken обновляет токен для будущих запросов.
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// IsServerAvailable проверяет доступность сервера через health check.
func (c *Client) IsServerAvailable(ctx context.Context) bool {
	return c.healthChecker.IsHealthy(ctx)
}

// QuickPing делает быстрый пинг сервера для проверки связи.
func (c *Client) QuickPing() bool {
	return c.healthChecker.QuickPing()
}

// CreateCredential создаёт новый credentials entry.
func (c *Client) CreateCredential(ctx context.Context, req *pb.CredentialCreateRequest) (*pb.CredentialCreateResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CredentialCreateResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.credentials.Create(rctx, req)
		return err
	})
	return resp, err
}

// GetCredential получает credentials по ID.
func (c *Client) GetCredential(ctx context.Context, id string) (*pb.CredentialGetResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CredentialGetResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.credentials.Get(rctx, &pb.CredentialGetRequest{Id: id})
		return err
	})
	return resp, err
}

// ListCredentials возвращает список credentials с пагинацией.
func (c *Client) ListCredentials(ctx context.Context, limit, offset uint32) (*pb.CredentialListResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CredentialListResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.credentials.List(rctx, &pb.CredentialListRequest{
			Page: &pb.PageRequest{
				Limit:  limit,
				Offset: offset,
			},
		})
		return err
	})
	return resp, err
}

// UpdateCredential обновляет credentials.
func (c *Client) UpdateCredential(ctx context.Context, req *pb.CredentialUpdateRequest) (*pb.CredentialUpdateResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CredentialUpdateResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.credentials.Update(rctx, req)
		return err
	})
	return resp, err
}

// DeleteCredential удаляет credentials.
func (c *Client) DeleteCredential(ctx context.Context, id string) (*pb.CredentialDeleteResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CredentialDeleteResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.credentials.Delete(rctx, &pb.CredentialDeleteRequest{Id: id})
		return err
	})
	return resp, err
}

// CreateCard создаёт новую карту.
func (c *Client) CreateCard(ctx context.Context, req *pb.CardCreateRequest) (*pb.CardCreateResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CardCreateResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.cards.Create(rctx, req)
		return err
	})
	return resp, err
}

// GetCard получает карту по ID.
func (c *Client) GetCard(ctx context.Context, id string) (*pb.CardGetResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CardGetResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.cards.Get(rctx, &pb.CardGetRequest{Id: id})
		return err
	})
	return resp, err
}

// ListCards возвращает список карт с пагинацией.
func (c *Client) ListCards(ctx context.Context, limit, offset uint32) (*pb.CardListResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CardListResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.cards.List(rctx, &pb.CardListRequest{
			Page: &pb.PageRequest{
				Limit:  limit,
				Offset: offset,
			},
		})
		return err
	})
	return resp, err
}

// UpdateCard обновляет карту.
func (c *Client) UpdateCard(ctx context.Context, req *pb.CardUpdateRequest) (*pb.CardUpdateResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CardUpdateResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.cards.Update(rctx, req)
		return err
	})
	return resp, err
}

// DeleteCard удаляет карту.
func (c *Client) DeleteCard(ctx context.Context, id string) (*pb.CardDeleteResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.CardDeleteResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.cards.Delete(rctx, &pb.CardDeleteRequest{Id: id})
		return err
	})
	return resp, err
}

// CreateText создаёт новый текст.
func (c *Client) CreateText(ctx context.Context, req *pb.TextCreateRequest) (*pb.TextCreateResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.TextCreateResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.texts.Create(rctx, req)
		return err
	})
	return resp, err
}

// GetText получает текст по ID.
func (c *Client) GetText(ctx context.Context, id string) (*pb.TextGetResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.TextGetResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.texts.Get(rctx, &pb.TextGetRequest{Id: id})
		return err
	})
	return resp, err
}

// ListTexts возвращает список текстов с пагинацией.
func (c *Client) ListTexts(ctx context.Context, limit, offset uint32) (*pb.TextListResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.TextListResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.texts.List(rctx, &pb.TextListRequest{
			Page: &pb.PageRequest{
				Limit:  limit,
				Offset: offset,
			},
		})
		return err
	})
	return resp, err
}

// UpdateText обновляет текст.
func (c *Client) UpdateText(ctx context.Context, req *pb.TextUpdateRequest) (*pb.TextUpdateResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.TextUpdateResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.texts.Update(rctx, req)
		return err
	})
	return resp, err
}

// DeleteText удаляет текст.
func (c *Client) DeleteText(ctx context.Context, id string) (*pb.TextDeleteResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	ctx = c.withAuth(ctx)

	var resp *pb.TextDeleteResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.texts.Delete(rctx, &pb.TextDeleteRequest{Id: id})
		return err
	})
	return resp, err
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

	var resp *pb.RegisterResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.auth.Register(rctx, req)
		return err
	})
	return resp, err
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

	var resp *pb.TokenResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.auth.Token(rctx, req)
		return err
	})
	return resp, err
}

// RefreshToken запрашивает новый access токен по refresh токену.
func (c *Client) RefreshToken(ctx context.Context, refreshToken, clientID, clientSecret string) (*pb.TokenResponse, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()

	req := &pb.TokenRequest{
		GrantType: pb.TokenGrantType_TOKEN_GRANT_TYPE_REFRESH_TOKEN,
		ClientId:  clientID,
	}
	if clientSecret != "" {
		req.ClientSecret = &clientSecret
	}

	var resp *pb.TokenResponse
	err := c.callWithRetry(ctx, func(rctx context.Context) error {
		var err error
		resp, err = c.auth.Token(rctx, req)
		return err
	})
	return resp, err
}

// // Revoke отзывает access/refresh токен.
// func (c *Client) Revoke(ctx context.Context, token, clientID, clientSecret string, hint pb.TokenTypeHint) (*pb.RevokeResponse, error) {
// 	ctx, cancel := c.timeoutCtx(ctx)
// 	defer cancel()

// 	req := &pb.RevokeRequest{
// 		Token:         token,
// 		TokenTypeHint: hint,
// 		ClientId:      clientID,
// 	}
// 	if clientSecret != "" {
// 		req.ClientSecret = &clientSecret
// 	}

// 	ctx = c.withAuth(ctx)
// 	var resp *pb.RevokeResponse
// 	err := c.callWithRetry(ctx, func(rctx context.Context) error {
// 		var err error
// 		resp, err = c.auth.Revoke(rctx, req)
// 		return err
// 	})
// 	return resp, err
// }

// callWithRetry выполняет fn с простым повтором при ошибке.
func (c *Client) callWithRetry(ctx context.Context, fn func(context.Context) error) error {
	backoff := time.Duration(defaultBackoffMillis) * time.Millisecond
	for attempt := 0; attempt < defaultAttempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if attempt == defaultAttempts-1 {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return nil
}
