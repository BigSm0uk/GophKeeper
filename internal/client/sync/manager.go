package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"go.uber.org/zap"
)

// Manager управляет синхронизацией данных с сервером.
type Manager struct {
	api       *api.Client
	localDB   *storage.LocalDB
	logger    *zap.Logger
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
	stopChan  chan struct{}
	forceChan chan struct{}
	wasOffline bool // флаг для отслеживания восстановления связи
}

// NewManager создает новый менеджер синхронизации.
func NewManager(api *api.Client, localDB *storage.LocalDB, logger *zap.Logger, interval time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		api:       api,
		localDB:   localDB,
		logger:    logger,
		interval:  interval,
		ctx:       ctx,
		cancel:    cancel,
		stopChan:  make(chan struct{}),
		forceChan: make(chan struct{}, 1),
	}
}

// Start запускает фоновую синхронизацию.
func (m *Manager) Start() {
	m.logger.Info("starting sync manager", zap.Duration("interval", m.interval))

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			close(m.stopChan)
			m.logger.Info("sync manager stopped")
			return

		case <-ticker.C:
			m.syncAll()

		case <-m.forceChan:
			m.syncAll()
		}
	}
}

// Stop останавливает синхронизацию.
func (m *Manager) Stop() error {
	m.logger.Info("stopping sync manager")
	m.cancel()
	<-m.stopChan
	return nil
}

// ForceSync инициирует немедленную синхронизацию.
func (m *Manager) ForceSync() {
	select {
	case m.forceChan <- struct{}{}:
	default:
	}
}

// syncAll синхронизирует все типы данных.
func (m *Manager) syncAll() {
	// Проверяем доступность сервера перед синхронизацией
	checkCtx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
	isOnline := m.api.IsServerAvailable(checkCtx)
	cancel()

	if !isOnline {
		if !m.wasOffline {
			m.logger.Debug("server unavailable, skipping sync")
			m.wasOffline = true
		}
		return
	}

	// Сервер доступен
	if m.wasOffline {
		// Восстановление связи - принудительная синхронизация
		m.logger.Info("connection restored, starting full sync")
		m.wasOffline = false
	}

	m.logger.Debug("starting sync cycle")

	// Синхронизируем все типы данных
	syncedCount := 0

	// Credentials
	if count, err := m.syncCredentials(); err != nil {
		m.logger.Error("failed to sync credentials", zap.Error(err))
	} else {
		syncedCount += count
	}

	// TODO: Cards, Texts, Binaries - будут добавлены позже аналогично

	m.logger.Debug("sync cycle completed", zap.Int("synced", syncedCount))
}

// syncCredentials синхронизирует credentials с сервером.
// Возвращает количество синхронизированных элементов.
func (m *Manager) syncCredentials() (int, error) {
	pending, err := m.localDB.GetPendingCredentials()
	if err != nil {
		return 0, fmt.Errorf("get pending credentials: %w", err)
	}

	if len(pending) == 0 {
		return 0, nil
	}

	m.logger.Debug("syncing credentials", zap.Int("count", len(pending)))

	syncedCount := 0
	for _, cred := range pending {
		if err := m.syncCredential(cred); err != nil {
			m.logger.Error("failed to sync credential",
				zap.String("id", cred.ID),
				zap.Error(err),
			)
			continue
		}
		syncedCount++
	}

	return syncedCount, nil
}

// syncCredential синхронизирует отдельный credential с сервером.
func (m *Manager) syncCredential(cred *storage.LocalCredential) error {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	// Обновляем статус на "в процессе загрузки"
	if err := m.localDB.UpdateCredentialSyncStatus(cred.ID, storage.StatusUploading); err != nil {
		return fmt.Errorf("update sync status to uploading: %w", err)
	}

	// Отправляем на сервер (здесь предполагаем создание, но может быть и обновление)
	req := &pb.CredentialCreateRequest{
		Name:     cred.Name,
		Login:    cred.Login,
		Password: cred.Password,
		Url:      cred.URL,
		Metadata: cred.Metadata,
	}
	_, err := m.api.CreateCredential(ctx, req)
	if err != nil {
		// Возвращаем в pending при ошибке
		_ = m.localDB.UpdateCredentialSyncStatus(cred.ID, storage.StatusPending)
		return fmt.Errorf("create credential on server: %w", err)
	}

	// Помечаем как синхронизированный
	if err := m.localDB.UpdateCredentialSyncStatus(cred.ID, storage.StatusSynced); err != nil {
		return fmt.Errorf("update sync status to synced: %w", err)
	}

	m.logger.Debug("credential synced", zap.String("id", cred.ID))
	return nil
}

// GetSyncStats возвращает статистику pending элементов.
func (m *Manager) GetSyncStats() (credentials, cards, texts, binaries int, err error) {
	pendingCreds, err := m.localDB.GetPendingCredentials()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get pending credentials: %w", err)
	}

	pendingBinaries, err := m.localDB.GetPendingBinaries()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get pending binaries: %w", err)
	}

	// TODO: Добавить cards и texts когда будут реализованы методы GetPendingCards/GetPendingTexts

	return len(pendingCreds), 0, 0, len(pendingBinaries), nil
}
