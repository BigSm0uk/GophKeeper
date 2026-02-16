package storage

import (
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
)

// EncryptedStorage wraps LocalDB and provides automatic encryption/decryption.
type EncryptedStorage struct {
	db        *LocalDB
	encryptor *crypto.Encryptor
}

// NewEncryptedStorage creates a new encrypted storage wrapper.
func NewEncryptedStorage(db *LocalDB, encryptor *crypto.Encryptor) *EncryptedStorage {
	return &EncryptedStorage{
		db:        db,
		encryptor: encryptor,
	}
}

// Close closes the underlying database connection.
func (s *EncryptedStorage) Close() error {
	return s.db.Close()
}

// ====================
// Credentials
// ====================

// SaveCredential encrypts and saves a credential.
func (s *EncryptedStorage) SaveCredential(cred *LocalCredential) error {
	// Encrypt sensitive fields
	encryptedLogin, err := s.encryptor.Encrypt(cred.Login)
	if err != nil {
		return fmt.Errorf("failed to encrypt login: %w", err)
	}

	encryptedPassword, err := s.encryptor.Encrypt(cred.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	var encryptedURL *string
	if cred.URL != nil {
		encrypted, err := s.encryptor.Encrypt(*cred.URL)
		if err != nil {
			return fmt.Errorf("failed to encrypt url: %w", err)
		}
		encryptedURL = &encrypted
	}

	var encryptedMetadata *string
	if cred.Metadata != nil {
		encrypted, err := s.encryptor.Encrypt(*cred.Metadata)
		if err != nil {
			return fmt.Errorf("failed to encrypt metadata: %w", err)
		}
		encryptedMetadata = &encrypted
	}

	// Create encrypted copy
	encryptedCred := &LocalCredential{
		ID:         cred.ID,
		Name:       cred.Name, // Name is not encrypted for search/display
		Login:      encryptedLogin,
		Password:   encryptedPassword,
		URL:        encryptedURL,
		Metadata:   encryptedMetadata,
		SyncStatus: cred.SyncStatus,
		CreatedAt:  cred.CreatedAt,
		UpdatedAt:  cred.UpdatedAt,
		SyncedAt:   cred.SyncedAt,
	}

	return s.db.SaveCredential(encryptedCred)
}

// GetCredential retrieves and decrypts a credential.
func (s *EncryptedStorage) GetCredential(id string) (*LocalCredential, error) {
	cred, err := s.db.GetCredential(id)
	if err != nil || cred == nil {
		return cred, err
	}

	// Decrypt sensitive fields
	decryptedLogin, err := s.encryptor.Decrypt(cred.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt login: %w", err)
	}

	decryptedPassword, err := s.encryptor.Decrypt(cred.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	var decryptedURL *string
	if cred.URL != nil {
		decrypted, err := s.encryptor.Decrypt(*cred.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt url: %w", err)
		}
		decryptedURL = &decrypted
	}

	var decryptedMetadata *string
	if cred.Metadata != nil {
		decrypted, err := s.encryptor.Decrypt(*cred.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
		}
		decryptedMetadata = &decrypted
	}

	cred.Login = decryptedLogin
	cred.Password = decryptedPassword
	cred.URL = decryptedURL
	cred.Metadata = decryptedMetadata

	return cred, nil
}

// ListCredentials returns all credentials with decrypted data.
func (s *EncryptedStorage) ListCredentials() ([]*LocalCredential, error) {
	creds, err := s.db.ListCredentials()
	if err != nil {
		return nil, err
	}

	// Decrypt each credential
	for _, cred := range creds {
		decryptedLogin, err := s.encryptor.Decrypt(cred.Login)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt login for %s: %w", cred.ID, err)
		}

		decryptedPassword, err := s.encryptor.Decrypt(cred.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password for %s: %w", cred.ID, err)
		}

		var decryptedURL *string
		if cred.URL != nil {
			decrypted, err := s.encryptor.Decrypt(*cred.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt url for %s: %w", cred.ID, err)
			}
			decryptedURL = &decrypted
		}

		var decryptedMetadata *string
		if cred.Metadata != nil {
			decrypted, err := s.encryptor.Decrypt(*cred.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt metadata for %s: %w", cred.ID, err)
			}
			decryptedMetadata = &decrypted
		}

		cred.Login = decryptedLogin
		cred.Password = decryptedPassword
		cred.URL = decryptedURL
		cred.Metadata = decryptedMetadata
	}

	return creds, nil
}

// DeleteCredential removes a credential from local storage.
func (s *EncryptedStorage) DeleteCredential(id string) error {
	return s.db.DeleteCredential(id)
}

// ====================
// Cards
// ====================

// SaveCard encrypts and saves a card.
func (s *EncryptedStorage) SaveCard(card *LocalCard) error {
	// Encrypt sensitive fields
	encryptedCardNumber, err := s.encryptor.Encrypt(card.CardNumber)
	if err != nil {
		return fmt.Errorf("failed to encrypt card number: %w", err)
	}

	encryptedExpiryDate, err := s.encryptor.Encrypt(card.ExpiryDate)
	if err != nil {
		return fmt.Errorf("failed to encrypt expiry date: %w", err)
	}

	encryptedCVV, err := s.encryptor.Encrypt(card.CVV)
	if err != nil {
		return fmt.Errorf("failed to encrypt cvv: %w", err)
	}

	var encryptedMetadata *string
	if card.Metadata != nil {
		encrypted, err := s.encryptor.Encrypt(*card.Metadata)
		if err != nil {
			return fmt.Errorf("failed to encrypt metadata: %w", err)
		}
		encryptedMetadata = &encrypted
	}

	encryptedCard := &LocalCard{
		ID:             card.ID,
		Name:           card.Name,
		CardNumber:     encryptedCardNumber,
		CardholderName: card.CardholderName, // Not encrypted for display
		ExpiryDate:     encryptedExpiryDate,
		CVV:            encryptedCVV,
		BankName:       card.BankName,
		Metadata:       encryptedMetadata,
		SyncStatus:     card.SyncStatus,
		CreatedAt:      card.CreatedAt,
		UpdatedAt:      card.UpdatedAt,
		SyncedAt:       card.SyncedAt,
	}

	return s.db.SaveCard(encryptedCard)
}

// GetCard retrieves and decrypts a card.
func (s *EncryptedStorage) GetCard(id string) (*LocalCard, error) {
	card, err := s.db.GetCard(id)
	if err != nil || card == nil {
		return card, err
	}

	// Decrypt sensitive fields
	decryptedCardNumber, err := s.encryptor.Decrypt(card.CardNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt card number: %w", err)
	}

	decryptedExpiryDate, err := s.encryptor.Decrypt(card.ExpiryDate)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt expiry date: %w", err)
	}

	decryptedCVV, err := s.encryptor.Decrypt(card.CVV)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt cvv: %w", err)
	}

	var decryptedMetadata *string
	if card.Metadata != nil {
		decrypted, err := s.encryptor.Decrypt(*card.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
		}
		decryptedMetadata = &decrypted
	}

	card.CardNumber = decryptedCardNumber
	card.ExpiryDate = decryptedExpiryDate
	card.CVV = decryptedCVV
	card.Metadata = decryptedMetadata

	return card, nil
}

// ListCards returns all cards with decrypted data.
func (s *EncryptedStorage) ListCards() ([]*LocalCard, error) {
	cards, err := s.db.ListCards()
	if err != nil {
		return nil, err
	}

	for _, card := range cards {
		decryptedCardNumber, err := s.encryptor.Decrypt(card.CardNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt card number for %s: %w", card.ID, err)
		}

		decryptedExpiryDate, err := s.encryptor.Decrypt(card.ExpiryDate)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt expiry date for %s: %w", card.ID, err)
		}

		decryptedCVV, err := s.encryptor.Decrypt(card.CVV)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt cvv for %s: %w", card.ID, err)
		}

		var decryptedMetadata *string
		if card.Metadata != nil {
			decrypted, err := s.encryptor.Decrypt(*card.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt metadata for %s: %w", card.ID, err)
			}
			decryptedMetadata = &decrypted
		}

		card.CardNumber = decryptedCardNumber
		card.ExpiryDate = decryptedExpiryDate
		card.CVV = decryptedCVV
		card.Metadata = decryptedMetadata
	}

	return cards, nil
}

// DeleteCard removes a card from local storage.
func (s *EncryptedStorage) DeleteCard(id string) error {
	return s.db.DeleteCard(id)
}

// ====================
// Texts
// ====================

// SaveText encrypts and saves a text.
func (s *EncryptedStorage) SaveText(text *LocalText) error {
	// Encrypt content
	encryptedContent, err := s.encryptor.Encrypt(text.Content)
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}

	var encryptedMetadata *string
	if text.Metadata != nil {
		encrypted, err := s.encryptor.Encrypt(*text.Metadata)
		if err != nil {
			return fmt.Errorf("failed to encrypt metadata: %w", err)
		}
		encryptedMetadata = &encrypted
	}

	encryptedText := &LocalText{
		ID:         text.ID,
		Name:       text.Name,
		Content:    encryptedContent,
		Metadata:   encryptedMetadata,
		SyncStatus: text.SyncStatus,
		CreatedAt:  text.CreatedAt,
		UpdatedAt:  text.UpdatedAt,
		SyncedAt:   text.SyncedAt,
	}

	return s.db.SaveText(encryptedText)
}

// GetText retrieves and decrypts a text.
func (s *EncryptedStorage) GetText(id string) (*LocalText, error) {
	text, err := s.db.GetText(id)
	if err != nil || text == nil {
		return text, err
	}

	// Decrypt content
	decryptedContent, err := s.encryptor.Decrypt(text.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt content: %w", err)
	}

	var decryptedMetadata *string
	if text.Metadata != nil {
		decrypted, err := s.encryptor.Decrypt(*text.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
		}
		decryptedMetadata = &decrypted
	}

	text.Content = decryptedContent
	text.Metadata = decryptedMetadata

	return text, nil
}

// ListTexts returns all texts with decrypted data.
func (s *EncryptedStorage) ListTexts() ([]*LocalText, error) {
	texts, err := s.db.ListTexts()
	if err != nil {
		return nil, err
	}

	for _, text := range texts {
		decryptedContent, err := s.encryptor.Decrypt(text.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt content for %s: %w", text.ID, err)
		}

		var decryptedMetadata *string
		if text.Metadata != nil {
			decrypted, err := s.encryptor.Decrypt(*text.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt metadata for %s: %w", text.ID, err)
			}
			decryptedMetadata = &decrypted
		}

		text.Content = decryptedContent
		text.Metadata = decryptedMetadata
	}

	return texts, nil
}

// DeleteText removes a text from local storage.
func (s *EncryptedStorage) DeleteText(id string) error {
	return s.db.DeleteText(id)
}

// ====================
// Binaries
// ====================

// SaveBinary saves binary metadata (file path is stored locally).
func (s *EncryptedStorage) SaveBinary(binary *LocalBinary) error {
	var encryptedMetadata *string
	if binary.Metadata != nil {
		encrypted, err := s.encryptor.Encrypt(*binary.Metadata)
		if err != nil {
			return fmt.Errorf("failed to encrypt metadata: %w", err)
		}
		encryptedMetadata = &encrypted
	}

	encryptedBinary := &LocalBinary{
		ID:          binary.ID,
		Name:        binary.Name,
		Filename:    binary.Filename,
		FilePath:    binary.FilePath,
		Size:        binary.Size,
		ContentType: binary.ContentType,
		Checksum:    binary.Checksum,
		Metadata:    encryptedMetadata,
		SyncStatus:  binary.SyncStatus,
		CreatedAt:   binary.CreatedAt,
		UpdatedAt:   binary.UpdatedAt,
		SyncedAt:    binary.SyncedAt,
	}

	return s.db.SaveBinary(encryptedBinary)
}

// GetBinary retrieves binary metadata.
func (s *EncryptedStorage) GetBinary(id string) (*LocalBinary, error) {
	binary, err := s.db.GetBinary(id)
	if err != nil || binary == nil {
		return binary, err
	}

	var decryptedMetadata *string
	if binary.Metadata != nil {
		decrypted, err := s.encryptor.Decrypt(*binary.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
		}
		decryptedMetadata = &decrypted
	}

	binary.Metadata = decryptedMetadata
	return binary, nil
}

// ListBinaries returns all binaries with decrypted metadata.
func (s *EncryptedStorage) ListBinaries() ([]*LocalBinary, error) {
	binaries, err := s.db.ListBinaries()
	if err != nil {
		return nil, err
	}

	for _, binary := range binaries {
		var decryptedMetadata *string
		if binary.Metadata != nil {
			decrypted, err := s.encryptor.Decrypt(*binary.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt metadata for %s: %w", binary.ID, err)
			}
			decryptedMetadata = &decrypted
		}
		binary.Metadata = decryptedMetadata
	}

	return binaries, nil
}

// DeleteBinary removes a binary from local storage.
func (s *EncryptedStorage) DeleteBinary(id string) error {
	return s.db.DeleteBinary(id)
}

// ====================
// Sync Status
// ====================

// UpdateCredentialSyncStatus updates the sync status of a credential.
func (s *EncryptedStorage) UpdateCredentialSyncStatus(id string, status SyncStatus) error {
	return s.db.UpdateCredentialSyncStatus(id, status)
}

// UpdateCardSyncStatus updates the sync status of a card.
func (s *EncryptedStorage) UpdateCardSyncStatus(id string, status SyncStatus) error {
	return s.db.UpdateCardSyncStatus(id, status)
}

// UpdateTextSyncStatus updates the sync status of a text.
func (s *EncryptedStorage) UpdateTextSyncStatus(id string, status SyncStatus) error {
	return s.db.UpdateTextSyncStatus(id, status)
}

// UpdateBinarySyncStatus updates the sync status of a binary.
func (s *EncryptedStorage) UpdateBinarySyncStatus(id string, status SyncStatus) error {
	return s.db.UpdateBinarySyncStatus(id, status)
}

// GetPendingCredentials returns credentials that need to be synced (encrypted).
func (s *EncryptedStorage) GetPendingCredentials() ([]*LocalCredential, error) {
	return s.db.GetPendingCredentials()
}

// GetPendingCards returns cards that need to be synced (encrypted).
func (s *EncryptedStorage) GetPendingCards() ([]*LocalCard, error) {
	return s.db.GetPendingCards()
}

// GetPendingTexts returns texts that need to be synced (encrypted).
func (s *EncryptedStorage) GetPendingTexts() ([]*LocalText, error) {
	return s.db.GetPendingTexts()
}

// GetPendingBinaries returns binaries that need to be synced.
func (s *EncryptedStorage) GetPendingBinaries() ([]*LocalBinary, error) {
	return s.db.GetPendingBinaries()
}

// GetDeletedCredentials returns credentials marked as deleted.
func (s *EncryptedStorage) GetDeletedCredentials() ([]*LocalCredential, error) {
	return s.db.GetDeletedCredentials()
}

// GetDeletedCards returns cards marked as deleted.
func (s *EncryptedStorage) GetDeletedCards() ([]*LocalCard, error) {
	return s.db.GetDeletedCards()
}

// GetDeletedTexts returns texts marked as deleted.
func (s *EncryptedStorage) GetDeletedTexts() ([]*LocalText, error) {
	return s.db.GetDeletedTexts()
}

// GetDeletedBinaries returns binaries marked as deleted.
func (s *EncryptedStorage) GetDeletedBinaries() ([]*LocalBinary, error) {
	return s.db.GetDeletedBinaries()
}

// PurgeCredential physically removes a credential after successful sync.
func (s *EncryptedStorage) PurgeCredential(id string) error {
	return s.db.PurgeCredential(id)
}

// PurgeCard physically removes a card after successful sync.
func (s *EncryptedStorage) PurgeCard(id string) error {
	return s.db.PurgeCard(id)
}

// PurgeText physically removes a text after successful sync.
func (s *EncryptedStorage) PurgeText(id string) error {
	return s.db.PurgeText(id)
}

// PurgeBinary physically removes a binary after successful sync.
func (s *EncryptedStorage) PurgeBinary(id string) error {
	return s.db.PurgeBinary(id)
}

// ====================
// Utility
// ====================

// CreateCredentialWithEncryption is a helper to create a new encrypted credential.
func CreateCredentialWithEncryption(id, name, login, password string, url, metadata *string) *LocalCredential {
	now := time.Now()
	return &LocalCredential{
		ID:         id,
		Name:       name,
		Login:      login,
		Password:   password,
		URL:        url,
		Metadata:   metadata,
		SyncStatus: StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateCardWithEncryption is a helper to create a new encrypted card.
func CreateCardWithEncryption(id, name, cardNumber, cardholderName, expiryDate, cvv string, bankName, metadata *string) *LocalCard {
	now := time.Now()
	return &LocalCard{
		ID:             id,
		Name:           name,
		CardNumber:     cardNumber,
		CardholderName: cardholderName,
		ExpiryDate:     expiryDate,
		CVV:            cvv,
		BankName:       bankName,
		Metadata:       metadata,
		SyncStatus:     StatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateTextWithEncryption is a helper to create a new encrypted text.
func CreateTextWithEncryption(id, name, content string, metadata *string) *LocalText {
	now := time.Now()
	return &LocalText{
		ID:         id,
		Name:       name,
		Content:    content,
		Metadata:   metadata,
		SyncStatus: StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateBinaryWithEncryption is a helper to create a new binary metadata entry.
func CreateBinaryWithEncryption(id, name, filename, filePath string, size int64, contentType, checksum string, metadata *string) *LocalBinary {
	now := time.Now()
	return &LocalBinary{
		ID:          id,
		Name:        name,
		Filename:    filename,
		FilePath:    filePath,
		Size:        size,
		ContentType: contentType,
		Checksum:    checksum,
		Metadata:    metadata,
		SyncStatus:  StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
