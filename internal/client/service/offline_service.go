package service

import (
	"context"
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/google/uuid"
)

// OfflineService управляет локальными операциями в офлайн режиме.
type OfflineService struct {
	storage *storage.EncryptedStorage
}

// NewOfflineService creates a new offline service.
func NewOfflineService(storage *storage.EncryptedStorage) *OfflineService {
	return &OfflineService{
		storage: storage,
	}
}

// ====================
// Credentials
// ====================

// CreateCredential создает credential локально с pending статусом.
func (s *OfflineService) CreateCredential(ctx context.Context, name, login, password string, url, metadata *string) (*storage.LocalCredential, error) {
	id := uuid.New().String()

	cred := storage.CreateCredentialWithEncryption(
		id, name, login, password, url, metadata,
	)

	if err := s.storage.SaveCredential(cred); err != nil {
		return nil, fmt.Errorf("failed to save credential: %w", err)
	}

	return cred, nil
}

// GetCredential получает credential из локального хранилища.
func (s *OfflineService) GetCredential(ctx context.Context, id string) (*storage.LocalCredential, error) {
	cred, err := s.storage.GetCredential(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("credential not found")
	}
	return cred, nil
}

// ListCredentials возвращает все credentials из локального хранилища.
func (s *OfflineService) ListCredentials(ctx context.Context) ([]*storage.LocalCredential, error) {
	creds, err := s.storage.ListCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	return creds, nil
}

// UpdateCredential обновляет credential локально с updated статусом.
func (s *OfflineService) UpdateCredential(ctx context.Context, id, name, login, password string, url, metadata *string) (*storage.LocalCredential, error) {
	// Получаем существующий credential
	existing, err := s.storage.GetCredential(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("credential not found")
	}

	// Обновляем поля
	existing.Name = name
	existing.Login = login
	existing.Password = password
	existing.URL = url
	existing.Metadata = metadata
	existing.SyncStatus = storage.StatusUpdated

	if err := s.storage.SaveCredential(existing); err != nil {
		return nil, fmt.Errorf("failed to update credential: %w", err)
	}

	return existing, nil
}

// DeleteCredential удаляет credential из локального хранилища.
func (s *OfflineService) DeleteCredential(ctx context.Context, id string) error {
	if err := s.storage.DeleteCredential(id); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

// ====================
// Cards
// ====================

// CreateCard создает card локально с pending статусом.
func (s *OfflineService) CreateCard(ctx context.Context, name, cardNumber, cardholderName, expiryDate, cvv string, bankName, metadata *string) (*storage.LocalCard, error) {
	id := uuid.New().String()

	card := storage.CreateCardWithEncryption(
		id, name, cardNumber, cardholderName, expiryDate, cvv, bankName, metadata,
	)

	if err := s.storage.SaveCard(card); err != nil {
		return nil, fmt.Errorf("failed to save card: %w", err)
	}

	return card, nil
}

// GetCard получает card из локального хранилища.
func (s *OfflineService) GetCard(ctx context.Context, id string) (*storage.LocalCard, error) {
	card, err := s.storage.GetCard(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	if card == nil {
		return nil, fmt.Errorf("card not found")
	}
	return card, nil
}

// ListCards возвращает все cards из локального хранилища.
func (s *OfflineService) ListCards(ctx context.Context) ([]*storage.LocalCard, error) {
	cards, err := s.storage.ListCards()
	if err != nil {
		return nil, fmt.Errorf("failed to list cards: %w", err)
	}
	return cards, nil
}

// UpdateCard обновляет card локально с updated статусом.
func (s *OfflineService) UpdateCard(ctx context.Context, id, name, cardNumber, cardholderName, expiryDate, cvv string, bankName, metadata *string) (*storage.LocalCard, error) {
	existing, err := s.storage.GetCard(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("card not found")
	}

	existing.Name = name
	existing.CardNumber = cardNumber
	existing.CardholderName = cardholderName
	existing.ExpiryDate = expiryDate
	existing.CVV = cvv
	existing.BankName = bankName
	existing.Metadata = metadata
	existing.SyncStatus = storage.StatusUpdated

	if err := s.storage.SaveCard(existing); err != nil {
		return nil, fmt.Errorf("failed to update card: %w", err)
	}

	return existing, nil
}

// DeleteCard удаляет card из локального хранилища.
func (s *OfflineService) DeleteCard(ctx context.Context, id string) error {
	if err := s.storage.DeleteCard(id); err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}
	return nil
}

// ====================
// Texts
// ====================

// CreateText создает text локально с pending статусом.
func (s *OfflineService) CreateText(ctx context.Context, name, content string, metadata *string) (*storage.LocalText, error) {
	id := uuid.New().String()

	text := storage.CreateTextWithEncryption(
		id, name, content, metadata,
	)

	if err := s.storage.SaveText(text); err != nil {
		return nil, fmt.Errorf("failed to save text: %w", err)
	}

	return text, nil
}

// GetText получает text из локального хранилища.
func (s *OfflineService) GetText(ctx context.Context, id string) (*storage.LocalText, error) {
	text, err := s.storage.GetText(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}
	if text == nil {
		return nil, fmt.Errorf("text not found")
	}
	return text, nil
}

// ListTexts возвращает все texts из локального хранилища.
func (s *OfflineService) ListTexts(ctx context.Context) ([]*storage.LocalText, error) {
	texts, err := s.storage.ListTexts()
	if err != nil {
		return nil, fmt.Errorf("failed to list texts: %w", err)
	}
	return texts, nil
}

// UpdateText обновляет text локально с updated статусом.
func (s *OfflineService) UpdateText(ctx context.Context, id, name, content string, metadata *string) (*storage.LocalText, error) {
	existing, err := s.storage.GetText(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("text not found")
	}

	existing.Name = name
	existing.Content = content
	existing.Metadata = metadata
	existing.SyncStatus = storage.StatusUpdated

	if err := s.storage.SaveText(existing); err != nil {
		return nil, fmt.Errorf("failed to update text: %w", err)
	}

	return existing, nil
}

// DeleteText удаляет text из локального хранилища.
func (s *OfflineService) DeleteText(ctx context.Context, id string) error {
	if err := s.storage.DeleteText(id); err != nil {
		return fmt.Errorf("failed to delete text: %w", err)
	}
	return nil
}

// ====================
// Binaries
// ====================

// CreateBinary создает binary metadata локально с pending статусом.
func (s *OfflineService) CreateBinary(ctx context.Context, name, filename, filePath string, size int64, contentType, checksum string, metadata *string) (*storage.LocalBinary, error) {
	id := uuid.New().String()

	binary := storage.CreateBinaryWithEncryption(
		id, name, filename, filePath, size, contentType, checksum, metadata,
	)

	if err := s.storage.SaveBinary(binary); err != nil {
		return nil, fmt.Errorf("failed to save binary: %w", err)
	}

	return binary, nil
}

// GetBinary получает binary metadata из локального хранилища.
func (s *OfflineService) GetBinary(ctx context.Context, id string) (*storage.LocalBinary, error) {
	binary, err := s.storage.GetBinary(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get binary: %w", err)
	}
	if binary == nil {
		return nil, fmt.Errorf("binary not found")
	}
	return binary, nil
}

// ListBinaries возвращает все binaries из локального хранилища.
func (s *OfflineService) ListBinaries(ctx context.Context) ([]*storage.LocalBinary, error) {
	binaries, err := s.storage.ListBinaries()
	if err != nil {
		return nil, fmt.Errorf("failed to list binaries: %w", err)
	}
	return binaries, nil
}

// DeleteBinary удаляет binary metadata из локального хранилища.
func (s *OfflineService) DeleteBinary(ctx context.Context, id string) error {
	if err := s.storage.DeleteBinary(id); err != nil {
		return fmt.Errorf("failed to delete binary: %w", err)
	}
	return nil
}
