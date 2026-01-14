# Технологический стек GophKeeper

## Выбранный стек технологий

### Server

| Компонент         | Библиотека                       | Назначение                                      |
| -------------------------- | ------------------------------------------ | --------------------------------------------------------- |
| **Configuration**    | `github.com/spf13/viper`                 | Управление конфигурацией           |
| **Protocol**         | `google.golang.org/grpc`                 | gRPC сервер                                         |
| **Proto Generation** | `buf.build/gen/go`                       | Генерация кода из .proto                   |
| **Web Framework**    | `github.com/gofiber/fiber/v2`            | REST API для health checks / metrics                   |
| **Database**         | `github.com/jackc/pgx/v5/pgxpool`        | PostgreSQL connection pool                                |
| **Migrations**       | `github.com/pressly/goose/v3`            | Database migrations                                       |
| **Authentication**   | `github.com/golang-jwt/jwt/v5`           | JWT токены                                          |
| **Password Hashing** | `golang.org/x/crypto/argon2`             | Хеширование паролей                     |
| **Logging**          | `go.uber.org/zap`                        | Структурированное логирование |
| **Validation**       | `github.com/go-playground/validator/v10` | Валидация структур                       |
| **Testing**          | `github.com/stretchr/testify`            | Assertions и моки                                    |
| **Binary Storage**   | `github.com/minio/minio-go/v7`           | S3-совместимое хранилище              |

### Client

| Компонент        | Библиотека                           | Назначение                              |
| ------------------------- | ---------------------------------------------- | ------------------------------------------------- |
| **CLI Framework**   | `github.com/spf13/cobra`                     | Базовая структура CLI             |
| **TUI Framework**   | `github.com/charmbracelet/bubbletea`         | Terminal UI                                       |
| **TUI Styling**     | `github.com/charmbracelet/lipgloss`          | Стили для TUI                             |
| **TUI Components**  | `github.com/charmbracelet/bubbles`           | Готовые UI компоненты            |
| **gRPC Client**     | `google.golang.org/grpc`                     | gRPC клиент                                 |
| **Configuration**   | `github.com/spf13/viper`                     | Управление конфигурацией   |
| **Secrets Storage** | `github.com/99designs/keyring`               | Хранение токенов                   |
| **Config Paths**    | `github.com/adrg/xdg`                        | XDG Base Directory                                |
| **Encryption**      | `crypto/aes` + `crypto/cipher`             | Клиентское шифрование         |
| **Key Derivation**  | `golang.org/x/crypto/scrypt`                 | Генерация ключей из пароля |
| **Progress Bars**   | `github.com/charmbracelet/bubbles/progress`  | Прогресс загрузки файлов    |
| **Spinner**         | `github.com/charmbracelet/bubbles/spinner`   | Индикатор загрузки               |
| **Tables**          | `github.com/charmbracelet/bubbles/table`     | Отображение данных               |
| **Text Input**      | `github.com/charmbracelet/bubbles/textinput` | Ввод данных                             |
| **Testing**         | `github.com/stretchr/testify`                | Тестирование                          |
| **OTP**             | `github.com/pquerna/otp`                     | TOTP/HOTP (опционально)                |

---

## Структура проекта

```
gophkeeper/
├── api/
│   └── proto/                      # Proto файлы
│       ├── auth.proto
│       ├── credentials.proto
│       ├── texts.proto
│       ├── cards.proto
│       ├── binaries.proto
│       ├── sync.proto
│       └── buf.yaml             # Buf конфигурация
├── cmd/
│   ├── server/                  # gRPC + Fiber сервер
│   │   └── main.go
│   └── client/                  # CLI + TUI клиент
│       └── main.go
├── internal/
│   ├── server/
│   │   ├── grpc/               # gRPC handlers
│   │   │   ├── auth.go
│   │   │   ├── credentials.go
│   │   │   ├── texts.go
│   │   │   ├── cards.go
│   │   │   ├── binaries.go
│   │   │   └── sync.go
│   │   ├── http/               # Fiber handlers (health, metrics)
│   │   │   ├── health.go
│   │   │   └── metrics.go
│   │   ├── service/            # Бизнес-логика
│   │   │   ├── auth.go
│   │   │   ├── credentials.go
│   │   │   ├── texts.go
│   │   │   ├── cards.go
│   │   │   ├── binaries.go
│   │   │   └── sync.go
│   │   ├── repository/         # Работа с БД
│   │   │   ├── postgres/
│   │   │   │   ├── users.go
│   │   │   │   ├── credentials.go
│   │   │   │   ├── texts.go
│   │   │   │   ├── cards.go
│   │   │   │   └── binaries.go
│   │   │   └── minio/
│   │   │       └── storage.go
│   │   ├── middleware/         # gRPC interceptors
│   │   │   ├── auth.go
│   │   │   ├── logging.go
│   │   │   ├── recovery.go
│   │   │   └── ratelimit.go
│   │   ├── models/             # Database models
│   │   └── config/             # Конфигурация
│   │       └── config.go
│   └── client/
│       ├── commands/           # Cobra commands
│       │   ├── root.go
│       │   ├── auth.go         # register, login, logout
│       │   ├── credentials.go  # cred add/list/get/update/delete
│       │   ├── texts.go
│       │   ├── cards.go
│       │   ├── files.go
│       │   ├── sync.go
│       │   └── tui.go          # TUI mode
│       ├── tui/                # Bubble Tea UI
│       │   ├── model.go        # Main TUI model
│       │   ├── views/
│       │   │   ├── login.go
│       │   │   ├── menu.go
│       │   │   ├── credentials_list.go
│       │   │   ├── credentials_form.go
│       │   │   ├── texts_list.go
│       │   │   ├── cards_list.go
│       │   │   └── files_list.go
│       │   ├── components/
│       │   │   ├── header.go
│       │   │   ├── footer.go
│       │   │   └── modal.go
│       │   └── styles/
│       │       └── styles.go
│       ├── api/                # gRPC client wrapper
│       │   ├── client.go
│       │   ├── auth.go
│       │   ├── credentials.go
│       │   ├── texts.go
│       │   ├── cards.go
│       │   ├── binaries.go
│       │   └── sync.go
│       ├── crypto/             # Клиентское шифрование
│       │   ├── encrypt.go
│       │   ├── decrypt.go
│       │   └── keygen.go
│       ├── storage/            # Локальное хранилище
│       │   ├── cache.go        # SQLite кеш
│       │   └── keyring.go      # Работа с keyring
│       └── config/             # Конфигурация клиента
│           └── config.go
├── pkg/                        # Shared code
│   ├── crypto/                 # Общие crypto утилиты
│   ├── models/                 # Общие модели
│   ├── logger/                 # Logger wrapper
│   └── validator/              # Дополнительные валидаторы
├── migrations/                 # Goose миграции
│   ├── 00001_create_users.sql
│   ├── 00002_create_credentials.sql
│   ├── 00003_create_texts.sql
│   ├── 00004_create_cards.sql
│   ├── 00005_create_binaries.sql
│   └── 00006_create_refresh_tokens.sql
├── tests/
│   ├── integration/
│   │   ├── grpc_test.go
│   │   └── sync_test.go
│   └── e2e/
│       └── scenarios_test.go
├── scripts/
│   ├── generate-proto.sh      # Генерация через buf
│   └── build.sh                # Сборка бинарников
├── configs/
│   ├── server.yaml             # Пример конфигурации сервера
│   └── client.yaml             # Пример конфигурации клиента
├── .github/
│   └── workflows/
│       ├── test.yml
│       ├── lint.yml
│       └── release.yml
├── buf.yaml                    # Buf конфигурация
├── buf.gen.yaml                # Buf генерация кода
├── buf.work.yaml               # Buf workspace
├── docker-compose.yml
├── Dockerfile.server
├── Dockerfile.client
├── Makefile
├── go.mod
└── go.sum
```

---

## Детальное описание стека

### 1. gRPC + Buf

#### Установка Buf

```bash
# macOS
brew install bufbuild/buf/buf

# Linux
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf
chmod +x /usr/local/bin/buf
```

#### buf.yaml

```yaml
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

#### buf.gen.yaml

```yaml
version: v1
managed:
  enabled: true
  go_package_prefix:
    default: github.com/yourusername/gophkeeper/gen/go
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative
  - plugin: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false
```

#### Пример proto файла (api/proto/auth.proto)

```protobuf
syntax = "proto3";

package gophkeeper.auth.v1;

option go_package = "github.com/yourusername/gophkeeper/gen/go/auth/v1;authv1";

import "google/protobuf/timestamp.proto";

service AuthService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  rpc Logout(LogoutRequest) returns (LogoutResponse);
}

message RegisterRequest {
  string username = 1;
  string password = 2;
  optional string email = 3;
}

message RegisterResponse {
  string user_id = 1;
  string username = 2;
  google.protobuf.Timestamp created_at = 3;
}

message LoginRequest {
  string username = 1;
  string password = 2;
}

message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3;
}

message RefreshTokenRequest {
  string refresh_token = 1;
}

message RefreshTokenResponse {
  string access_token = 1;
  int64 expires_in = 2;
}

message LogoutRequest {}

message LogoutResponse {}
```

#### Генерация кода

```bash
# Генерация через buf
buf generate api/proto

# Или через Makefile
make generate-proto
```

#### gRPC Server (internal/server/grpc/auth.go)

```go
package grpc

import (
    "context"
  
    authv1 "github.com/yourusername/gophkeeper/gen/go/auth/v1"
    "github.com/yourusername/gophkeeper/internal/server/service"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
    authv1.UnimplementedAuthServiceServer
    authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
    return &AuthHandler{
        authService: authService,
    }
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
    user, err := h.authService.Register(ctx, req.Username, req.Password, req.Email)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
  
    return &authv1.RegisterResponse{
        UserId:    user.ID,
        Username:  user.Username,
        CreatedAt: timestamppb.New(user.CreatedAt),
    }, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
    tokens, err := h.authService.Login(ctx, req.Username, req.Password)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid credentials")
    }
  
    return &authv1.LoginResponse{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresIn:    tokens.ExpiresIn,
    }, nil
}
```

---

### 2. Fiber для HTTP endpoints

#### Использование

Fiber будет использоваться параллельно с gRPC для:

- Health checks
- Metrics (Prometheus)
- Graceful shutdown endpoint

#### internal/server/http/server.go

```go
package http

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/recover"
    "go.uber.org/zap"
)

type Server struct {
    app    *fiber.App
    logger *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            logger.Error("HTTP error", zap.Error(err))
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": err.Error(),
            })
        },
    })

    // Middleware
    app.Use(recover.New())
    app.Use(cors.New())

    return &Server{
        app:    app,
        logger: logger,
    }
}

func (s *Server) SetupRoutes() {
    // Health check
    s.app.Get("/health", s.healthCheck)
  
    // Metrics
    s.app.Get("/metrics", s.metrics)
  
    // Version
    s.app.Get("/version", s.version)
}

func (s *Server) healthCheck(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "ok",
        "service": "gophkeeper",
    })
}

func (s *Server) Start(addr string) error {
    return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
    return s.app.Shutdown()
}
```

---

### 3. PostgreSQL + pgx + goose

#### Подключение (internal/server/repository/postgres/db.go)

```go
package postgres

import (
    "context"
    "fmt"
  
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
)

type DB struct {
    pool   *pgxpool.Pool
    logger *zap.Logger
}

func New(ctx context.Context, dsn string, logger *zap.Logger) (*DB, error) {
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    // Pool settings
    config.MaxConns = 25
    config.MinConns = 5

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }

    // Ping
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }

    logger.Info("connected to PostgreSQL",
        zap.String("host", config.ConnConfig.Host),
        zap.Uint16("port", config.ConnConfig.Port),
    )

    return &DB{
        pool:   pool,
        logger: logger,
    }, nil
}

func (db *DB) Close() {
    db.pool.Close()
}

func (db *DB) Pool() *pgxpool.Pool {
    return db.pool
}
```

#### Миграции (migrations/00001_create_users.sql)

```sql
-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);

-- +goose Down
DROP TABLE IF EXISTS users;
```

#### Запуск миграций

```bash
# Установка goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Применение миграций
goose -dir ./migrations postgres "postgresql://user:pass@localhost:5432/gophkeeper?sslmode=disable" up

# Откат
goose -dir ./migrations postgres "postgresql://user:pass@localhost:5432/gophkeeper?sslmode=disable" down

# Статус
goose -dir ./migrations postgres "postgresql://user:pass@localhost:5432/gophkeeper?sslmode=disable" status
```

---

### 4. JWT + Argon2

#### internal/server/service/auth.go

```go
package service

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"
  
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/argon2"
)

type AuthService struct {
    jwtSecret      []byte
    accessTokenTTL time.Duration
    refreshTokenTTL time.Duration
}

func NewAuthService(jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
    return &AuthService{
        jwtSecret:       []byte(jwtSecret),
        accessTokenTTL:  accessTTL,
        refreshTokenTTL: refreshTTL,
    }
}

// HashPassword with Argon2
func (s *AuthService) HashPassword(password string) (string, error) {
    // Generate salt
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }

    // Argon2id parameters
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

    // Encode salt + hash
    encoded := fmt.Sprintf("%s$%s",
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    )

    return encoded, nil
}

// VerifyPassword with Argon2
func (s *AuthService) VerifyPassword(password, encoded string) (bool, error) {
    // Decode salt and hash
    var salt, hash []byte
    _, err := fmt.Sscanf(encoded, "%s$%s", &salt, &hash)
    if err != nil {
        return false, err
    }

    // Hash provided password
    computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

    // Compare
    return string(hash) == string(computedHash), nil
}

// GenerateAccessToken
func (s *AuthService) GenerateAccessToken(userID, username string) (string, error) {
    claims := jwt.MapClaims{
        "sub":      userID,
        "username": username,
        "exp":      time.Now().Add(s.accessTokenTTL).Unix(),
        "iat":      time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

// ValidateToken
func (s *AuthService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return &claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}
```

---

### 5. Zap Logger

#### pkg/logger/logger.go

```go
package logger

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func New(level string, isDevelopment bool) (*zap.Logger, error) {
    var config zap.Config

    if isDevelopment {
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    } else {
        config = zap.NewProductionConfig()
    }

    // Parse level
    var zapLevel zapcore.Level
    if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
        zapLevel = zapcore.InfoLevel
    }
    config.Level = zap.NewAtomicLevelAt(zapLevel)

    return config.Build()
}
```

#### Использование

```go
logger, err := logger.New("info", false)
if err != nil {
    panic(err)
}
defer logger.Sync()

logger.Info("server starting",
    zap.String("host", "0.0.0.0"),
    zap.Int("port", 8080),
)

logger.Error("failed to connect",
    zap.Error(err),
    zap.String("dsn", dsn),
)
```

---

### 6. Viper Configuration

#### internal/server/config/config.go

```go
package config

import (
    "fmt"
    "time"

    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    Minio    MinioConfig
    Logger   LoggerConfig
}

type ServerConfig struct {
    GRPCPort string
    HTTPPort string
}

type DatabaseConfig struct {
    DSN          string
    MaxConns     int
    MinConns     int
    MaxConnIdleTime time.Duration
}

type JWTConfig struct {
    Secret           string
    AccessTokenTTL   time.Duration
    RefreshTokenTTL  time.Duration
}

type MinioConfig struct {
    Endpoint        string
    AccessKeyID     string
    SecretAccessKey string
    UseSSL          bool
    BucketName      string
}

type LoggerConfig struct {
    Level       string
    Development bool
}

func Load(configPath string) (*Config, error) {
    viper.SetConfigFile(configPath)
    viper.SetConfigType("yaml")

    // Environment variables
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    return &cfg, nil
}
```

#### configs/server.yaml

```yaml
server:
  grpc_port: ":50051"
  http_port: ":8080"

database:
  dsn: "postgresql://user:password@localhost:5432/gophkeeper?sslmode=disable"
  max_conns: 25
  min_conns: 5
  max_conn_idle_time: 5m

jwt:
  secret: "your-secret-key-change-in-production"
  access_token_ttl: 1h
  refresh_token_ttl: 720h  # 30 days

minio:
  endpoint: "localhost:9000"
  access_key_id: "minioadmin"
  secret_access_key: "minioadmin"
  use_ssl: false
  bucket_name: "gophkeeper"

logger:
  level: "info"
  development: false
```

---

### 7. MinIO для бинарных данных

#### internal/server/repository/minio/storage.go

```go
package minio

import (
    "context"
    "fmt"
    "io"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
    "go.uber.org/zap"
)

type Storage struct {
    client     *minio.Client
    bucketName string
    logger     *zap.Logger
}

func New(endpoint, accessKey, secretKey, bucketName string, useSSL bool, logger *zap.Logger) (*Storage, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        return nil, fmt.Errorf("create minio client: %w", err)
    }

    // Create bucket if not exists
    ctx := context.Background()
    exists, err := client.BucketExists(ctx, bucketName)
    if err != nil {
        return nil, fmt.Errorf("check bucket: %w", err)
    }

    if !exists {
        if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
            return nil, fmt.Errorf("create bucket: %w", err)
        }
        logger.Info("bucket created", zap.String("bucket", bucketName))
    }

    return &Storage{
        client:     client,
        bucketName: bucketName,
        logger:     logger,
    }, nil
}

func (s *Storage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
    _, err := s.client.PutObject(ctx, s.bucketName, objectName, reader, size, minio.PutObjectOptions{
        ContentType: contentType,
    })
    if err != nil {
        return fmt.Errorf("upload object: %w", err)
    }

    s.logger.Info("file uploaded",
        zap.String("object", objectName),
        zap.Int64("size", size),
    )

    return nil
}

func (s *Storage) Download(ctx context.Context, objectName string) (*minio.Object, error) {
    object, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
    if err != nil {
        return nil, fmt.Errorf("get object: %w", err)
    }

    return object, nil
}

func (s *Storage) Delete(ctx context.Context, objectName string) error {
    err := s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
    if err != nil {
        return fmt.Errorf("delete object: %w", err)
    }

    return nil
}
```

---

## Client Stack

### 1. Cobra CLI

#### cmd/client/main.go

```go
package main

import (
    "fmt"
    "os"

    "github.com/yourusername/gophkeeper/internal/client/commands"
)

var (
    version   = "dev"
    buildDate = "unknown"
    commit    = "unknown"
)

func main() {
    commands.SetVersion(version, buildDate, commit)
  
    if err := commands.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

#### internal/client/commands/root.go

```go
package commands

import (
    "github.com/spf13/cobra"
)

var (
    version   string
    buildDate string
    commit    string
    cfgFile   string
)

var rootCmd = &cobra.Command{
    Use:   "gophkeeper",
    Short: "GophKeeper - secure password manager",
    Long: `GophKeeper is a client-server password manager
that allows you to securely store credentials, text data,
binary files, and bank card information.`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gophkeeper/config.yaml)")
  
    // Subcommands
    rootCmd.AddCommand(versionCmd)
    rootCmd.AddCommand(registerCmd)
    rootCmd.AddCommand(loginCmd)
    rootCmd.AddCommand(logoutCmd)
    rootCmd.AddCommand(credCmd)
    rootCmd.AddCommand(textCmd)
    rootCmd.AddCommand(cardCmd)
    rootCmd.AddCommand(fileCmd)
    rootCmd.AddCommand(syncCmd)
    rootCmd.AddCommand(tuiCmd)  // TUI mode
}

func SetVersion(v, date, c string) {
    version = v
    buildDate = date
    commit = c
}

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("GophKeeper Client\n")
        fmt.Printf("Version: %s\n", version)
        fmt.Printf("Build Date: %s\n", buildDate)
        fmt.Printf("Commit: %s\n", commit)
    },
}
```

---

### 2. Bubble Tea TUI

#### internal/client/tui/model.go

```go
package tui

import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type sessionState int

const (
    stateLogin sessionState = iota
    stateMenu
    stateCredentialsList
    stateCredentialsForm
    stateTextsList
    stateCardsList
    stateFilesList
)

type Model struct {
    state       sessionState
    width       int
    height      int
  
    // Components
    spinner     spinner.Model
    loginForm   LoginForm
    menu        list.Model
    credList    list.Model
    credForm    CredentialsForm
  
    // State
    loading     bool
    err         error
    loggedIn    bool
}

func New() Model {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

    return Model{
        state:    stateLogin,
        spinner:  s,
        loading:  false,
    }
}

func (m Model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
      
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        }
    }

    // Route to current view
    switch m.state {
    case stateLogin:
        return m.updateLogin(msg)
    case stateMenu:
        return m.updateMenu(msg)
    case stateCredentialsList:
        return m.updateCredentialsList(msg)
    // ... other states
    }

    return m, nil
}

func (m Model) View() string {
    if m.width == 0 {
        return "Loading..."
    }

    var content string
    switch m.state {
    case stateLogin:
        content = m.viewLogin()
    case stateMenu:
        content = m.viewMenu()
    case stateCredentialsList:
        content = m.viewCredentialsList()
    // ... other views
    }

    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        content,
    )
}
```

#### internal/client/tui/views/login.go

```go
package views

import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type LoginForm struct {
    username textinput.Model
    password textinput.Model
    focused  int
}

func NewLoginForm() LoginForm {
    username := textinput.New()
    username.Placeholder = "Username"
    username.Focus()
    username.CharLimit = 50
    username.Width = 30

    password := textinput.New()
    password.Placeholder = "Password"
    password.EchoMode = textinput.EchoPassword
    password.CharLimit = 50
    password.Width = 30

    return LoginForm{
        username: username,
        password: password,
        focused:  0,
    }
}

func (f LoginForm) Update(msg tea.Msg) (LoginForm, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab", "shift+tab", "up", "down":
            // Switch focus
            if f.focused == 0 {
                f.focused = 1
                f.username.Blur()
                cmd = f.password.Focus()
            } else {
                f.focused = 0
                f.password.Blur()
                cmd = f.username.Focus()
            }
            return f, cmd
        }
    }

    if f.focused == 0 {
        f.username, cmd = f.username.Update(msg)
    } else {
        f.password, cmd = f.password.Update(msg)
    }

    return f, cmd
}

func (f LoginForm) View() string {
    style := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")).
        Padding(1, 2)

    title := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("205")).
        Render("🔐 GophKeeper Login")

    content := lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        "",
        f.username.View(),
        f.password.View(),
        "",
        lipgloss.NewStyle().Faint(true).Render("Press Enter to login, Ctrl+C to quit"),
    )

    return style.Render(content)
}
```

#### internal/client/tui/views/credentials_list.go

```go
package views

import (
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/table"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("240"))

func NewCredentialsTable(credentials []Credential) table.Model {
    columns := []table.Column{
        {Title: "Name", Width: 20},
        {Title: "Login", Width: 20},
        {Title: "URL", Width: 30},
        {Title: "Updated", Width: 20},
    }

    rows := make([]table.Row, len(credentials))
    for i, cred := range credentials {
        rows[i] = table.Row{
            cred.Name,
            cred.Login,
            cred.URL,
            cred.UpdatedAt.Format("2006-01-02 15:04"),
        }
    }

    t := table.New(
        table.WithColumns(columns),
        table.WithRows(rows),
        table.WithFocused(true),
        table.WithHeight(10),
    )

    s := table.DefaultStyles()
    s.Header = s.Header.
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("240")).
        BorderBottom(true).
        Bold(false)
    s.Selected = s.Selected.
        Foreground(lipgloss.Color("229")).
        Background(lipgloss.Color("57")).
        Bold(false)
    t.SetStyles(s)

    return t
}
```

#### internal/client/tui/styles/styles.go

```go
package styles

import "github.com/charmbracelet/lipgloss"

var (
    // Colors
    Primary   = lipgloss.Color("205")
    Secondary = lipgloss.Color("63")
    Success   = lipgloss.Color("42")
    Error     = lipgloss.Color("196")
    Warning   = lipgloss.Color("214")
    Muted     = lipgloss.Color("240")

    // Title styles
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Primary).
        MarginBottom(1)

    // Box styles
    BoxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Secondary).
        Padding(1, 2)

    // Error style
    ErrorStyle = lipgloss.NewStyle().
        Foreground(Error).
        Bold(true)

    // Success style
    SuccessStyle = lipgloss.NewStyle().
        Foreground(Success).
        Bold(true)

    // Help style
    HelpStyle = lipgloss.NewStyle().
        Foreground(Muted).
        Italic(true)
)
```

---

### 3. gRPC Client API

#### internal/client/api/client.go

```go
package api

import (
    "context"
    "crypto/tls"
    "fmt"
    "time"

    authv1 "github.com/yourusername/gophkeeper/gen/go/auth/v1"
    credv1 "github.com/yourusername/gophkeeper/gen/go/credentials/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
)

type Client struct {
    conn           *grpc.ClientConn
    authClient     authv1.AuthServiceClient
    credClient     credv1.CredentialsServiceClient
    // ... other service clients
  
    accessToken    string
}

func New(serverAddr string, useTLS bool) (*Client, error) {
    var opts []grpc.DialOption

    if useTLS {
        opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
    } else {
        opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
    }

    conn, err := grpc.Dial(serverAddr, opts...)
    if err != nil {
        return nil, fmt.Errorf("dial: %w", err)
    }

    return &Client{
        conn:       conn,
        authClient: authv1.NewAuthServiceClient(conn),
        credClient: credv1.NewCredentialsServiceClient(conn),
    }, nil
}

func (c *Client) Close() error {
    return c.conn.Close()
}

func (c *Client) SetAccessToken(token string) {
    c.accessToken = token
}

// withAuth adds authentication metadata to context
func (c *Client) withAuth(ctx context.Context) context.Context {
    if c.accessToken == "" {
        return ctx
    }
    return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.accessToken)
}

// Register
func (c *Client) Register(ctx context.Context, username, password, email string) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    req := &authv1.RegisterRequest{
        Username: username,
        Password: password,
    }
    if email != "" {
        req.Email = &email
    }

    _, err := c.authClient.Register(ctx, req)
    return err
}

// Login
func (c *Client) Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    resp, err := c.authClient.Login(ctx, &authv1.LoginRequest{
        Username: username,
        Password: password,
    })
    if err != nil {
        return "", "", err
    }

    c.accessToken = resp.AccessToken
    return resp.AccessToken, resp.RefreshToken, nil
}
```

---

### 4. Client-side Encryption

#### internal/client/crypto/encrypt.go

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"

    "golang.org/x/crypto/scrypt"
)

const (
    keyLen   = 32 // AES-256
    saltLen  = 32
    nonceLen = 12 // GCM standard nonce size
)

// DeriveKey from password using scrypt
func DeriveKey(password, salt []byte) ([]byte, error) {
    return scrypt.Key(password, salt, 32768, 8, 1, keyLen)
}

// GenerateSalt creates a random salt
func GenerateSalt() ([]byte, error) {
    salt := make([]byte, saltLen)
    if _, err := rand.Read(salt); err != nil {
        return nil, err
    }
    return salt, nil
}

// Encrypt data with AES-256-GCM
func Encrypt(plaintext []byte, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("create gcm: %w", err)
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("generate nonce: %w", err)
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt data with AES-256-GCM
func Decrypt(ciphertext string, key []byte) ([]byte, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return nil, fmt.Errorf("decode base64: %w", err)
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("create gcm: %w", err)
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decrypt: %w", err)
    }

    return plaintext, nil
}
```

---

### 5. Keyring для хранения токенов

#### internal/client/storage/keyring.go

```go
package storage

import (
    "fmt"

    "github.com/99designs/keyring"
)

const (
    serviceName = "gophkeeper"
)

type KeyringStore struct {
    ring keyring.Keyring
}

func NewKeyringStore() (*KeyringStore, error) {
    ring, err := keyring.Open(keyring.Config{
        ServiceName: serviceName,
        // macOS: использует Keychain
        // Linux: использует Secret Service (gnome-keyring/kwallet)
        // Windows: использует Windows Credential Manager
    })
    if err != nil {
        return nil, fmt.Errorf("open keyring: %w", err)
    }

    return &KeyringStore{ring: ring}, nil
}

func (s *KeyringStore) SaveAccessToken(username, token string) error {
    return s.ring.Set(keyring.Item{
        Key:  fmt.Sprintf("%s:access_token", username),
        Data: []byte(token),
    })
}

func (s *KeyringStore) GetAccessToken(username string) (string, error) {
    item, err := s.ring.Get(fmt.Sprintf("%s:access_token", username))
    if err != nil {
        return "", err
    }
    return string(item.Data), nil
}

func (s *KeyringStore) SaveRefreshToken(username, token string) error {
    return s.ring.Set(keyring.Item{
        Key:  fmt.Sprintf("%s:refresh_token", username),
        Data: []byte(token),
    })
}

func (s *KeyringStore) GetRefreshToken(username string) (string, error) {
    item, err := s.ring.Get(fmt.Sprintf("%s:refresh_token", username))
    if err != nil {
        return "", err
    }
    return string(item.Data), nil
}

func (s *KeyringStore) DeleteTokens(username string) error {
    _ = s.ring.Remove(fmt.Sprintf("%s:access_token", username))
    _ = s.ring.Remove(fmt.Sprintf("%s:refresh_token", username))
    return nil
}
```

---

## Makefile

```makefile
.PHONY: help build test lint proto migrate-up migrate-down docker-up docker-down

# Variables
SERVER_BINARY=bin/server
CLIENT_BINARY=bin/client
MIGRATIONS_DIR=migrations
POSTGRES_DSN=postgresql://user:password@localhost:5432/gophkeeper?sslmode=disable

# Version info
VERSION?=dev
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.commit=$(COMMIT)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: proto ## Build server and client
	@echo "Building server..."
	@go build $(LDFLAGS) -o $(SERVER_BINARY) ./cmd/server
	@echo "Building client..."
	@go build $(LDFLAGS) -o $(CLIENT_BINARY) ./cmd/client
	@echo "Done!"

proto: ## Generate code from proto files
	@echo "Generating proto files..."
	@buf generate api/proto
	@echo "Done!"

test: ## Run tests
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

coverage: test ## Show test coverage
	@go tool cover -html=coverage.out

lint: ## Run linters
	@golangci-lint run ./...

migrate-up: ## Run database migrations up
	@goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

migrate-down: ## Rollback last migration
	@goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

migrate-status: ## Show migration status
	@goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" status

docker-up: ## Start docker-compose services
	@docker-compose up -d
	@echo "Waiting for PostgreSQL..."
	@sleep 3
	@make migrate-up

docker-down: ## Stop docker-compose services
	@docker-compose down

docker-logs: ## Show docker logs
	@docker-compose logs -f

run-server: ## Run server locally
	@go run ./cmd/server

run-client: ## Run client locally
	@go run ./cmd/client

clean: ## Clean build artifacts
	@rm -rf bin/
	@rm -f coverage.out

install-tools: ## Install development tools
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@brew install bufbuild/buf/buf  # macOS

release: ## Build release binaries for all platforms
	@goreleaser release --snapshot --clean
```

---

## docker-compose.yml

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: gophkeeper-postgres
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: gophkeeper
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d gophkeeper"]
      interval: 5s
      timeout: 3s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: gophkeeper-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  postgres_data:
  minio_data:
```

---

## .goreleaser.yaml

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - make proto

builds:
  - id: server
    main: ./cmd/server
    binary: gophkeeper-server
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildDate={{.Date}}
      - -X main.commit={{.Commit}}

  - id: client
    main: ./cmd/client
    binary: gophkeeper
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildDate={{.Date}}
      - -X main.commit={{.Commit}}

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
```

---

## Следующие шаги

1. **Создать структуру проекта**

   ```bash
   mkdir -p cmd/{server,client}
   mkdir -p internal/{server,client}/{grpc,service,repository,commands,tui}
   mkdir -p api/proto
   mkdir -p migrations
   mkdir -p pkg/{crypto,logger}
   ```
2. **Инициализировать Go модуль**

   ```bash
   go mod init github.com/yourusername/gophkeeper
   ```
3. **Установить зависимости**

   ```bash
   # Server
   go get github.com/spf13/viper
   go get google.golang.org/grpc
   go get github.com/gofiber/fiber/v2
   go get github.com/jackc/pgx/v5/pgxpool
   go get github.com/pressly/goose/v3
   go get github.com/golang-jwt/jwt/v5
   go get golang.org/x/crypto/argon2
   go get go.uber.org/zap
   go get github.com/go-playground/validator/v10
   go get github.com/minio/minio-go/v7

   # Client
   go get github.com/spf13/cobra
   go get github.com/charmbracelet/bubbletea
   go get github.com/charmbracelet/lipgloss
   go get github.com/charmbracelet/bubbles
   go get github.com/99designs/keyring
   go get github.com/adrg/xdg

   # Testing
   go get github.com/stretchr/testify
   ```
4. **Создать proto файлы и настроить buf**
5. **Создать миграции БД**
6. **Начать разработку с аутентификации**
