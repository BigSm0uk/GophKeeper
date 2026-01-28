package storage

import (
	"database/sql"
	"time"
)

// LocalCard represents a locally stored card.
type LocalCard struct {
	ID             string
	Name           string
	CardNumber     string
	CardholderName string
	ExpiryDate     string
	CVV            string
	BankName       *string
	Metadata       *string
	SyncStatus     SyncStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	SyncedAt       *time.Time
}

// SaveCard saves or updates a card in local storage.
func (l *LocalDB) SaveCard(card *LocalCard) error {
	query := `
		INSERT OR REPLACE INTO cards 
		(id, name, card_number, cardholder_name, expiry_date, cvv, bank_name, metadata, sync_status, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var syncedAt *int64
	if card.SyncedAt != nil {
		ts := card.SyncedAt.Unix()
		syncedAt = &ts
	}

	_, err := l.db.Exec(query,
		card.ID,
		card.Name,
		card.CardNumber,
		card.CardholderName,
		card.ExpiryDate,
		card.CVV,
		card.BankName,
		card.Metadata,
		string(card.SyncStatus),
		card.CreatedAt.Unix(),
		card.UpdatedAt.Unix(),
		syncedAt,
	)

	return err
}

// GetCard retrieves a card by ID.
func (l *LocalDB) GetCard(id string) (*LocalCard, error) {
	query := `
		SELECT id, name, card_number, cardholder_name, expiry_date, cvv, bank_name, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM cards 
		WHERE id = ?
	`

	var card LocalCard
	var createdAtUnix, updatedAtUnix int64
	var syncedAtUnix *int64

	err := l.db.QueryRow(query, id).Scan(
		&card.ID,
		&card.Name,
		&card.CardNumber,
		&card.CardholderName,
		&card.ExpiryDate,
		&card.CVV,
		&card.BankName,
		&card.Metadata,
		&card.SyncStatus,
		&createdAtUnix,
		&updatedAtUnix,
		&syncedAtUnix,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	card.CreatedAt = time.Unix(createdAtUnix, 0)
	card.UpdatedAt = time.Unix(updatedAtUnix, 0)
	if syncedAtUnix != nil {
		t := time.Unix(*syncedAtUnix, 0)
		card.SyncedAt = &t
	}

	return &card, nil
}

// LocalText represents a locally stored text note.
type LocalText struct {
	ID         string
	Name       string
	Content    string
	Metadata   *string
	SyncStatus SyncStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	SyncedAt   *time.Time
}

// SaveText saves or updates a text in local storage.
func (l *LocalDB) SaveText(text *LocalText) error {
	query := `
		INSERT OR REPLACE INTO texts 
		(id, name, content, metadata, sync_status, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var syncedAt *int64
	if text.SyncedAt != nil {
		ts := text.SyncedAt.Unix()
		syncedAt = &ts
	}

	_, err := l.db.Exec(query,
		text.ID,
		text.Name,
		text.Content,
		text.Metadata,
		string(text.SyncStatus),
		text.CreatedAt.Unix(),
		text.UpdatedAt.Unix(),
		syncedAt,
	)

	return err
}

// GetText retrieves a text by ID.
func (l *LocalDB) GetText(id string) (*LocalText, error) {
	query := `
		SELECT id, name, content, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM texts 
		WHERE id = ?
	`

	var text LocalText
	var createdAtUnix, updatedAtUnix int64
	var syncedAtUnix *int64

	err := l.db.QueryRow(query, id).Scan(
		&text.ID,
		&text.Name,
		&text.Content,
		&text.Metadata,
		&text.SyncStatus,
		&createdAtUnix,
		&updatedAtUnix,
		&syncedAtUnix,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	text.CreatedAt = time.Unix(createdAtUnix, 0)
	text.UpdatedAt = time.Unix(updatedAtUnix, 0)
	if syncedAtUnix != nil {
		t := time.Unix(*syncedAtUnix, 0)
		text.SyncedAt = &t
	}

	return &text, nil
}

// ListCards returns all cards.
func (l *LocalDB) ListCards() ([]*LocalCard, error) {
	query := `
		SELECT id, name, card_number, cardholder_name, expiry_date, cvv, bank_name, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM cards 
		ORDER BY updated_at DESC
	`

	rows, err := l.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []*LocalCard
	for rows.Next() {
		var card LocalCard
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&card.ID,
			&card.Name,
			&card.CardNumber,
			&card.CardholderName,
			&card.ExpiryDate,
			&card.CVV,
			&card.BankName,
			&card.Metadata,
			&card.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		card.CreatedAt = time.Unix(createdAtUnix, 0)
		card.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			card.SyncedAt = &t
		}

		cards = append(cards, &card)
	}

	return cards, rows.Err()
}

// ListTexts returns all texts.
func (l *LocalDB) ListTexts() ([]*LocalText, error) {
	query := `
		SELECT id, name, content, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM texts 
		ORDER BY updated_at DESC
	`

	rows, err := l.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var texts []*LocalText
	for rows.Next() {
		var text LocalText
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&text.ID,
			&text.Name,
			&text.Content,
			&text.Metadata,
			&text.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		text.CreatedAt = time.Unix(createdAtUnix, 0)
		text.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			text.SyncedAt = &t
		}

		texts = append(texts, &text)
	}

	return texts, rows.Err()
}

// DeleteCard removes a card from local storage.
func (l *LocalDB) DeleteCard(id string) error {
	_, err := l.db.Exec("DELETE FROM cards WHERE id = ?", id)
	return err
}

// DeleteText removes a text from local storage.
func (l *LocalDB) DeleteText(id string) error {
	_, err := l.db.Exec("DELETE FROM texts WHERE id = ?", id)
	return err
}
