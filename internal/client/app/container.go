package app

import (
	"context"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
	"github.com/BigSm0uk/GophKeeper/internal/client/service"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/sync"
	"go.uber.org/zap"
)

// Container хранит зависимости клиента.
type Container struct {
	Logger          *zap.Logger
	Config          *config.ClientConfig
	API             *api.Client
	TokenStore      *storage.TokenStore
	LocalDB         *storage.LocalDB
	Encryptor       *crypto.Encryptor        // может быть nil, инициализируется при необходимости
	StorageManager  *storage.StorageManager  // полный стек локального хранилища с шифрованием
	SyncManager     *sync.Manager            // менеджер фоновой синхронизации
	IsOnline        bool                     // текущий статус подключения к серверу
	LastHealthCheck time.Time                // время последней проверки доступности
	offlineMode     bool                     // принудительный офлайн режим
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

// InitEncryptor инициализирует encryptor с заданным master password для пользователя.
func (c *Container) InitEncryptor(username, masterPassword string) error {
	if c.Encryptor != nil {
		return nil // уже инициализирован
	}

	// Пробуем загрузить salt из keyring
	salt, err := c.TokenStore.GetEncryptionSalt(username)
	if err != nil || salt == nil || len(salt) == 0 {
		// Генерируем новый salt
		salt, err = crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		// Сохраняем salt в keyring
		if err := c.TokenStore.SaveEncryptionSalt(username, salt); err != nil {
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

// CheckOnlineStatus проверяет доступность сервера и обновляет статус.
func (c *Container) CheckOnlineStatus(ctx context.Context) bool {
	if c.API == nil {
		c.IsOnline = false
		c.LastHealthCheck = time.Now()
		return false
	}

	isOnline := c.API.IsServerAvailable(ctx)
	c.IsOnline = isOnline
	c.LastHealthCheck = time.Now()

	return isOnline
}

// GetOnlineStatus возвращает текущий статус подключения.
// Если проверка устарела (более 30 секунд), делает новую проверку.
func (c *Container) GetOnlineStatus(ctx context.Context) bool {
	if time.Since(c.LastHealthCheck) > 30*time.Second {
		return c.CheckOnlineStatus(ctx)
	}
	return c.IsOnline
}

// Close закрывает все ресурсы Container.
func (c *Container) Close() error {
	var errs []error

	// Останавливаем синхронизацию
	if err := c.StopSync(); err != nil {
		errs = append(errs, fmt.Errorf("stop sync: %w", err))
	}

	// Закрываем StorageManager
	if c.StorageManager != nil {
		if err := c.StorageManager.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close storage manager: %w", err))
		}
	}

	// Закрываем LocalDB если она была инициализирована отдельно
	if c.LocalDB != nil && c.StorageManager == nil {
		if err := c.LocalDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close local db: %w", err))
		}
	}

	// Закрываем API соединение
	if c.API != nil {
		if err := c.API.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close api: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close container: %v", errs)
	}

	return nil
}

// SetOfflineMode включает принудительный офлайн режим.
func (c *Container) SetOfflineMode(offline bool) {
	c.offlineMode = offline
}

// IsOfflineMode возвращает true если работаем в офлайн режиме.
// Учитывает как принудительный offline, так и недоступность сервера.
func (c *Container) IsOfflineMode() bool {
	return c.offlineMode || !c.IsOnline
}

// GetOfflineService возвращает сервис для офлайн операций.
func (c *Container) GetOfflineService() *service.OfflineService {
	if c.StorageManager == nil || c.StorageManager.Encrypted == nil {
		return nil
	}
	return service.NewOfflineService(c.StorageManager.Encrypted)
}

// UpdateOnlineStatus обновляет статус подключения при запуске.
func (c *Container) UpdateOnlineStatus(ctx context.Context) {
	c.CheckOnlineStatus(ctx)
	if c.Logger != nil {
		status := "offline"
		if c.IsOnline {
			status = "online"
		}
		c.Logger.Info("connection status", zap.String("mode", status))
	}
}
