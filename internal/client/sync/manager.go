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

// Manager manages data synchronization with the server.
type Manager struct {
	api        *api.Client
	localDB    *storage.LocalDB
	logger     *zap.Logger
	interval   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	stopChan   chan struct{}
	forceChan  chan struct{}
	wasOffline bool
}

// NewManager creates a new sync manager.
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

// Start starts background synchronization.
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

// Stop stops the sync manager.
func (m *Manager) Stop() error {
	m.logger.Info("stopping sync manager")
	m.cancel()
	<-m.stopChan
	return nil
}

// ForceSync triggers an immediate sync cycle.
func (m *Manager) ForceSync() {
	select {
	case m.forceChan <- struct{}{}:
	default:
	}
}

// syncAll runs a full sync cycle for all entity types.
func (m *Manager) syncAll() {
	// Check server availability before syncing
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

	// Server is available
	if m.wasOffline {
		m.logger.Info("connection restored, starting full sync")
		m.wasOffline = false
	}

	m.logger.Debug("starting sync cycle")

	syncedCount := 0

	// Push local changes to server
	if count, err := m.syncCredentials(); err != nil {
		m.logger.Error("failed to sync credentials", zap.Error(err))
	} else {
		syncedCount += count
	}

	if count, err := m.syncCards(); err != nil {
		m.logger.Error("failed to sync cards", zap.Error(err))
	} else {
		syncedCount += count
	}

	if count, err := m.syncTexts(); err != nil {
		m.logger.Error("failed to sync texts", zap.Error(err))
	} else {
		syncedCount += count
	}

	// Sync deletions
	if count, err := m.syncDeletedCredentials(); err != nil {
		m.logger.Error("failed to sync deleted credentials", zap.Error(err))
	} else {
		syncedCount += count
	}

	if count, err := m.syncDeletedCards(); err != nil {
		m.logger.Error("failed to sync deleted cards", zap.Error(err))
	} else {
		syncedCount += count
	}

	if count, err := m.syncDeletedTexts(); err != nil {
		m.logger.Error("failed to sync deleted texts", zap.Error(err))
	} else {
		syncedCount += count
	}

	// Pull server data to local
	if count, err := m.pullCredentials(); err != nil {
		m.logger.Error("failed to pull credentials", zap.Error(err))
	} else {
		syncedCount += count
	}

	if count, err := m.pullCards(); err != nil {
		m.logger.Error("failed to pull cards", zap.Error(err))
	} else {
		syncedCount += count
	}

	if count, err := m.pullTexts(); err != nil {
		m.logger.Error("failed to pull texts", zap.Error(err))
	} else {
		syncedCount += count
	}

	m.logger.Debug("sync cycle completed", zap.Int("synced", syncedCount))
}

// ====================
// Push: Credentials
// ====================

// syncCredentials pushes pending/updated credentials to the server.
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

// syncCredential pushes a single credential to the server.
func (m *Manager) syncCredential(cred *storage.LocalCredential) error {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := m.localDB.UpdateCredentialSyncStatus(cred.ID, storage.StatusUploading); err != nil {
		return fmt.Errorf("update sync status to uploading: %w", err)
	}

	var err error
	switch cred.SyncStatus {
	case storage.StatusPending:
		// New credential — create on server
		req := &pb.CredentialCreateRequest{
			Name:     cred.Name,
			Login:    cred.Login,
			Password: cred.Password,
			Url:      cred.URL,
			Metadata: cred.Metadata,
		}
		_, err = m.api.CreateCredential(ctx, req)

	case storage.StatusUpdated:
		// Updated credential — update on server
		req := &pb.CredentialUpdateRequest{
			Id:       cred.ID,
			Name:     cred.Name,
			Login:    cred.Login,
			Password: cred.Password,
			Url:      cred.URL,
			Metadata: cred.Metadata,
		}
		_, err = m.api.UpdateCredential(ctx, req)
	}

	if err != nil {
		_ = m.localDB.UpdateCredentialSyncStatus(cred.ID, cred.SyncStatus)
		return fmt.Errorf("sync credential to server: %w", err)
	}

	if err := m.localDB.UpdateCredentialSyncStatus(cred.ID, storage.StatusSynced); err != nil {
		return fmt.Errorf("update sync status to synced: %w", err)
	}

	m.logger.Debug("credential synced", zap.String("id", cred.ID))
	return nil
}

// syncDeletedCredentials pushes credential deletions to the server.
func (m *Manager) syncDeletedCredentials() (int, error) {
	deleted, err := m.localDB.GetDeletedCredentials()
	if err != nil {
		return 0, fmt.Errorf("get deleted credentials: %w", err)
	}

	if len(deleted) == 0 {
		return 0, nil
	}

	m.logger.Debug("syncing deleted credentials", zap.Int("count", len(deleted)))

	syncedCount := 0
	for _, cred := range deleted {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		_, err := m.api.DeleteCredential(ctx, cred.ID)
		cancel()

		if err != nil {
			m.logger.Error("failed to delete credential on server",
				zap.String("id", cred.ID),
				zap.Error(err),
			)
			continue
		}

		// Purge from local DB after successful server deletion
		if err := m.localDB.PurgeCredential(cred.ID); err != nil {
			m.logger.Error("failed to purge credential locally",
				zap.String("id", cred.ID),
				zap.Error(err),
			)
		}
		syncedCount++
	}

	return syncedCount, nil
}

// ====================
// Push: Cards
// ====================

// syncCards pushes pending/updated cards to the server.
func (m *Manager) syncCards() (int, error) {
	pending, err := m.localDB.GetPendingCards()
	if err != nil {
		return 0, fmt.Errorf("get pending cards: %w", err)
	}

	if len(pending) == 0 {
		return 0, nil
	}

	m.logger.Debug("syncing cards", zap.Int("count", len(pending)))

	syncedCount := 0
	for _, card := range pending {
		if err := m.syncCard(card); err != nil {
			m.logger.Error("failed to sync card",
				zap.String("id", card.ID),
				zap.Error(err),
			)
			continue
		}
		syncedCount++
	}

	return syncedCount, nil
}

// syncCard pushes a single card to the server.
func (m *Manager) syncCard(card *storage.LocalCard) error {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := m.localDB.UpdateCardSyncStatus(card.ID, storage.StatusUploading); err != nil {
		return fmt.Errorf("update sync status to uploading: %w", err)
	}

	var err error
	switch card.SyncStatus {
	case storage.StatusPending:
		req := &pb.CardCreateRequest{
			Name:           card.Name,
			CardNumber:     card.CardNumber,
			CardholderName: card.CardholderName,
			ExpiryDate:     card.ExpiryDate,
			Cvv:            card.CVV,
			BankName:       card.BankName,
			Metadata:       card.Metadata,
		}
		_, err = m.api.CreateCard(ctx, req)

	case storage.StatusUpdated:
		req := &pb.CardUpdateRequest{
			Id:             card.ID,
			Name:           card.Name,
			CardNumber:     card.CardNumber,
			CardholderName: card.CardholderName,
			ExpiryDate:     card.ExpiryDate,
			Cvv:            card.CVV,
			BankName:       card.BankName,
			Metadata:       card.Metadata,
		}
		_, err = m.api.UpdateCard(ctx, req)
	}

	if err != nil {
		_ = m.localDB.UpdateCardSyncStatus(card.ID, card.SyncStatus)
		return fmt.Errorf("sync card to server: %w", err)
	}

	if err := m.localDB.UpdateCardSyncStatus(card.ID, storage.StatusSynced); err != nil {
		return fmt.Errorf("update sync status to synced: %w", err)
	}

	m.logger.Debug("card synced", zap.String("id", card.ID))
	return nil
}

// syncDeletedCards pushes card deletions to the server.
func (m *Manager) syncDeletedCards() (int, error) {
	deleted, err := m.localDB.GetDeletedCards()
	if err != nil {
		return 0, fmt.Errorf("get deleted cards: %w", err)
	}

	if len(deleted) == 0 {
		return 0, nil
	}

	m.logger.Debug("syncing deleted cards", zap.Int("count", len(deleted)))

	syncedCount := 0
	for _, card := range deleted {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		_, err := m.api.DeleteCard(ctx, card.ID)
		cancel()

		if err != nil {
			m.logger.Error("failed to delete card on server",
				zap.String("id", card.ID),
				zap.Error(err),
			)
			continue
		}

		if err := m.localDB.PurgeCard(card.ID); err != nil {
			m.logger.Error("failed to purge card locally",
				zap.String("id", card.ID),
				zap.Error(err),
			)
		}
		syncedCount++
	}

	return syncedCount, nil
}

// ====================
// Push: Texts
// ====================

// syncTexts pushes pending/updated texts to the server.
func (m *Manager) syncTexts() (int, error) {
	pending, err := m.localDB.GetPendingTexts()
	if err != nil {
		return 0, fmt.Errorf("get pending texts: %w", err)
	}

	if len(pending) == 0 {
		return 0, nil
	}

	m.logger.Debug("syncing texts", zap.Int("count", len(pending)))

	syncedCount := 0
	for _, text := range pending {
		if err := m.syncText(text); err != nil {
			m.logger.Error("failed to sync text",
				zap.String("id", text.ID),
				zap.Error(err),
			)
			continue
		}
		syncedCount++
	}

	return syncedCount, nil
}

// syncText pushes a single text to the server.
func (m *Manager) syncText(text *storage.LocalText) error {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := m.localDB.UpdateTextSyncStatus(text.ID, storage.StatusUploading); err != nil {
		return fmt.Errorf("update sync status to uploading: %w", err)
	}

	var err error
	switch text.SyncStatus {
	case storage.StatusPending:
		req := &pb.TextCreateRequest{
			Name:     text.Name,
			Content:  text.Content,
			Metadata: text.Metadata,
		}
		_, err = m.api.CreateText(ctx, req)

	case storage.StatusUpdated:
		req := &pb.TextUpdateRequest{
			Id:       text.ID,
			Name:     text.Name,
			Content:  text.Content,
			Metadata: text.Metadata,
		}
		_, err = m.api.UpdateText(ctx, req)
	}

	if err != nil {
		_ = m.localDB.UpdateTextSyncStatus(text.ID, text.SyncStatus)
		return fmt.Errorf("sync text to server: %w", err)
	}

	if err := m.localDB.UpdateTextSyncStatus(text.ID, storage.StatusSynced); err != nil {
		return fmt.Errorf("update sync status to synced: %w", err)
	}

	m.logger.Debug("text synced", zap.String("id", text.ID))
	return nil
}

// syncDeletedTexts pushes text deletions to the server.
func (m *Manager) syncDeletedTexts() (int, error) {
	deleted, err := m.localDB.GetDeletedTexts()
	if err != nil {
		return 0, fmt.Errorf("get deleted texts: %w", err)
	}

	if len(deleted) == 0 {
		return 0, nil
	}

	m.logger.Debug("syncing deleted texts", zap.Int("count", len(deleted)))

	syncedCount := 0
	for _, text := range deleted {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		_, err := m.api.DeleteText(ctx, text.ID)
		cancel()

		if err != nil {
			m.logger.Error("failed to delete text on server",
				zap.String("id", text.ID),
				zap.Error(err),
			)
			continue
		}

		if err := m.localDB.PurgeText(text.ID); err != nil {
			m.logger.Error("failed to purge text locally",
				zap.String("id", text.ID),
				zap.Error(err),
			)
		}
		syncedCount++
	}

	return syncedCount, nil
}

// ====================
// Pull: Server → Local
// ====================

// pullCredentials downloads credentials from server and merges with local storage.
func (m *Manager) pullCredentials() (int, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	resp, err := m.api.ListCredentials(ctx, 1000, 0)
	if err != nil {
		return 0, fmt.Errorf("list credentials from server: %w", err)
	}

	if resp == nil || len(resp.Items) == 0 {
		return 0, nil
	}

	pulledCount := 0
	for _, item := range resp.Items {
		existing, err := m.localDB.GetCredential(item.Id)
		if err != nil {
			m.logger.Error("failed to check local credential", zap.String("id", item.Id), zap.Error(err))
			continue
		}

		// Skip if local version is newer or has pending changes
		if existing != nil &&
			(existing.SyncStatus == storage.StatusPending ||
				existing.SyncStatus == storage.StatusUpdated ||
				existing.SyncStatus == storage.StatusDeleted) {
			continue
		}

		// Server version is newer or doesn't exist locally — save it
		serverUpdatedAt := time.Now()
		if item.UpdatedAt != nil {
			serverUpdatedAt = item.UpdatedAt.AsTime()
		}

		if existing != nil && existing.SyncStatus == storage.StatusSynced {
			// Check if server version is actually newer
			if !serverUpdatedAt.After(existing.UpdatedAt) {
				continue
			}
		}

		serverCreatedAt := time.Now()
		if item.CreatedAt != nil {
			serverCreatedAt = item.CreatedAt.AsTime()
		}

		now := time.Now()
		cred := &storage.LocalCredential{
			ID:         item.Id,
			Name:       item.Name,
			Login:      item.Login,
			Password:   item.Password,
			URL:        item.Url,
			Metadata:   item.Metadata,
			SyncStatus: storage.StatusSynced,
			CreatedAt:  serverCreatedAt,
			UpdatedAt:  serverUpdatedAt,
			SyncedAt:   &now,
		}

		if err := m.localDB.SaveCredential(cred); err != nil {
			m.logger.Error("failed to save pulled credential", zap.String("id", item.Id), zap.Error(err))
			continue
		}

		pulledCount++
	}

	if pulledCount > 0 {
		m.logger.Debug("pulled credentials from server", zap.Int("count", pulledCount))
	}

	return pulledCount, nil
}

// pullCards downloads cards from server and merges with local storage.
func (m *Manager) pullCards() (int, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	resp, err := m.api.ListCards(ctx, 1000, 0)
	if err != nil {
		return 0, fmt.Errorf("list cards from server: %w", err)
	}

	if resp == nil || len(resp.Items) == 0 {
		return 0, nil
	}

	pulledCount := 0
	for _, item := range resp.Items {
		existing, err := m.localDB.GetCard(item.Id)
		if err != nil {
			m.logger.Error("failed to check local card", zap.String("id", item.Id), zap.Error(err))
			continue
		}

		if existing != nil &&
			(existing.SyncStatus == storage.StatusPending ||
				existing.SyncStatus == storage.StatusUpdated ||
				existing.SyncStatus == storage.StatusDeleted) {
			continue
		}

		serverUpdatedAt := time.Now()
		if item.UpdatedAt != nil {
			serverUpdatedAt = item.UpdatedAt.AsTime()
		}

		if existing != nil && existing.SyncStatus == storage.StatusSynced {
			if !serverUpdatedAt.After(existing.UpdatedAt) {
				continue
			}
		}

		serverCreatedAt := time.Now()
		if item.CreatedAt != nil {
			serverCreatedAt = item.CreatedAt.AsTime()
		}

		now := time.Now()
		card := &storage.LocalCard{
			ID:             item.Id,
			Name:           item.Name,
			CardNumber:     item.CardNumber,
			CardholderName: item.CardholderName,
			ExpiryDate:     item.ExpiryDate,
			CVV:            item.Cvv,
			BankName:       item.BankName,
			Metadata:       item.Metadata,
			SyncStatus:     storage.StatusSynced,
			CreatedAt:      serverCreatedAt,
			UpdatedAt:      serverUpdatedAt,
			SyncedAt:       &now,
		}

		if err := m.localDB.SaveCard(card); err != nil {
			m.logger.Error("failed to save pulled card", zap.String("id", item.Id), zap.Error(err))
			continue
		}

		pulledCount++
	}

	if pulledCount > 0 {
		m.logger.Debug("pulled cards from server", zap.Int("count", pulledCount))
	}

	return pulledCount, nil
}

// pullTexts downloads texts from server and merges with local storage.
func (m *Manager) pullTexts() (int, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	resp, err := m.api.ListTexts(ctx, 1000, 0)
	if err != nil {
		return 0, fmt.Errorf("list texts from server: %w", err)
	}

	if resp == nil || len(resp.Items) == 0 {
		return 0, nil
	}

	pulledCount := 0
	for _, item := range resp.Items {
		existing, err := m.localDB.GetText(item.Id)
		if err != nil {
			m.logger.Error("failed to check local text", zap.String("id", item.Id), zap.Error(err))
			continue
		}

		if existing != nil &&
			(existing.SyncStatus == storage.StatusPending ||
				existing.SyncStatus == storage.StatusUpdated ||
				existing.SyncStatus == storage.StatusDeleted) {
			continue
		}

		serverUpdatedAt := time.Now()
		if item.UpdatedAt != nil {
			serverUpdatedAt = item.UpdatedAt.AsTime()
		}

		if existing != nil && existing.SyncStatus == storage.StatusSynced {
			if !serverUpdatedAt.After(existing.UpdatedAt) {
				continue
			}
		}

		serverCreatedAt := time.Now()
		if item.CreatedAt != nil {
			serverCreatedAt = item.CreatedAt.AsTime()
		}

		now := time.Now()
		text := &storage.LocalText{
			ID:         item.Id,
			Name:       item.Name,
			Content:    item.Content,
			Metadata:   item.Metadata,
			SyncStatus: storage.StatusSynced,
			CreatedAt:  serverCreatedAt,
			UpdatedAt:  serverUpdatedAt,
			SyncedAt:   &now,
		}

		if err := m.localDB.SaveText(text); err != nil {
			m.logger.Error("failed to save pulled text", zap.String("id", item.Id), zap.Error(err))
			continue
		}

		pulledCount++
	}

	if pulledCount > 0 {
		m.logger.Debug("pulled texts from server", zap.Int("count", pulledCount))
	}

	return pulledCount, nil
}

// ====================
// Statistics
// ====================

// GetSyncStats returns counts of pending items for each entity type.
func (m *Manager) GetSyncStats() (credentials, cards, texts, binaries int, err error) {
	pendingCreds, err := m.localDB.GetPendingCredentials()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get pending credentials: %w", err)
	}

	deletedCreds, err := m.localDB.GetDeletedCredentials()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get deleted credentials: %w", err)
	}

	pendingCards, err := m.localDB.GetPendingCards()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get pending cards: %w", err)
	}

	deletedCards, err := m.localDB.GetDeletedCards()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get deleted cards: %w", err)
	}

	pendingTexts, err := m.localDB.GetPendingTexts()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get pending texts: %w", err)
	}

	deletedTexts, err := m.localDB.GetDeletedTexts()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get deleted texts: %w", err)
	}

	pendingBinaries, err := m.localDB.GetPendingBinaries()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get pending binaries: %w", err)
	}

	deletedBinaries, err := m.localDB.GetDeletedBinaries()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("get deleted binaries: %w", err)
	}

	return len(pendingCreds) + len(deletedCreds),
		len(pendingCards) + len(deletedCards),
		len(pendingTexts) + len(deletedTexts),
		len(pendingBinaries) + len(deletedBinaries),
		nil
}
