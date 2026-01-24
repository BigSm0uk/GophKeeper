package app

import (
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/sync"
	"go.uber.org/zap"
)

// Container хранит зависимости клиента.
type Container struct {
	Logger      *zap.Logger
	Config      *config.ClientConfig
	API         *api.Client
	TokenStore  *storage.TokenStore
	LocalDB     *storage.LocalDB
	Encryptor   *crypto.Encryptor // может быть nil, инициализируется при необходимости
	SyncManager *sync.Manager     // менеджер фоновой синхронизации
}

func NewContainer(logger *zap.Logger, cfg *config.ClientConfig, client *api.Client, tokenStore *storage.TokenStore) *Container {
	return &Container{
		Logger:     logger,
		Config:     cfg,
		API:        client,
		TokenStore: tokenStore,
		LocalDB:    nil, // инициализируется lazy
		Encryptor:  nil, // инициализируется lazy
	}
}

// InitEncryptor инициализирует encryptor с заданным master password.
func (c *Container) InitEncryptor(masterPassword string) error {
	// Пробуем загрузить salt из keyring
	salt, err := c.TokenStore.GetEncryptionSalt()
	if err != nil || salt == nil {
		// Генерируем новый salt
		salt, err = crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		// Сохраняем salt в keyring
		if err := c.TokenStore.SaveEncryptionSalt(salt); err != nil {
			return fmt.Errorf("failed to save salt: %w", err)
		}
	}

	// Создаём encryptor
	encryptor, err := crypto.NewEncryptor(masterPassword, salt)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	c.Encryptor = encryptor
	return nil
}

// GetOrInitLocalDB возвращает или инициализирует LocalDB.
func (c *Container) GetOrInitLocalDB() (*storage.LocalDB, error) {
	if c.LocalDB != nil {
		return c.LocalDB, nil
	}

	// Определяем путь к БД через XDG
	dbPath, err := c.Config.GetLocalDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get db path: %w", err)
	}

	db, err := storage.NewLocalDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init local db: %w", err)
	}

	c.LocalDB = db
	return db, nil
}

// InitSyncManager инициализирует менеджер синхронизации.
func (c *Container) InitSyncManager() error {
	if c.SyncManager != nil {
		return nil // уже инициализирован
	}

	if c.LocalDB == nil {
		if _, err := c.GetOrInitLocalDB(); err != nil {
			return fmt.Errorf("failed to init local db: %w", err)
		}
	}

	// Интервал синхронизации по умолчанию - 5 минут
	interval := 5 * time.Minute
	c.SyncManager = sync.NewManager(c.API, c.LocalDB, c.Logger, interval)

	return nil
}

// StartSync запускает фоновую синхронизацию.
func (c *Container) StartSync() error {
	if err := c.InitSyncManager(); err != nil {
		return err
	}

	go c.SyncManager.Start()
	return nil
}

// StopSync останавливает фоновую синхронизацию.
func (c *Container) StopSync() error {
	if c.SyncManager != nil {
		return c.SyncManager.Stop()
	}
	return nil
}
