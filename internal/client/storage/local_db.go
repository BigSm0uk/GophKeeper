package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// LocalDB provides local SQLite storage for offline data and sync management.
type LocalDB struct {
	db *sql.DB
}

// NewLocalDB creates a new local database at the specified path.
func NewLocalDB(dbPath string) (*LocalDB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	localDB := &LocalDB{db: db}

	// Initialize schema
	if err := localDB.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return localDB, nil
}

// Close closes the database connection.
func (l *LocalDB) Close() error {
	return l.db.Close()
}

// initSchema creates the database schema.
func (l *LocalDB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		login TEXT NOT NULL,
		password TEXT NOT NULL,
		url TEXT,
		metadata TEXT,
		sync_status TEXT NOT NULL DEFAULT 'pending',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		synced_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS cards (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		card_number TEXT NOT NULL,
		cardholder_name TEXT NOT NULL,
		expiry_date TEXT NOT NULL,
		cvv TEXT NOT NULL,
		bank_name TEXT,
		metadata TEXT,
		sync_status TEXT NOT NULL DEFAULT 'pending',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		synced_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS texts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		sync_status TEXT NOT NULL DEFAULT 'pending',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		synced_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS binaries (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		filename TEXT NOT NULL,
		file_path TEXT NOT NULL,
		size INTEGER NOT NULL,
		content_type TEXT NOT NULL,
		checksum TEXT NOT NULL,
		metadata TEXT,
		sync_status TEXT NOT NULL DEFAULT 'pending',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		synced_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS sync_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_type TEXT NOT NULL,
		item_id TEXT NOT NULL,
		operation TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		last_error TEXT,
		UNIQUE(item_type, item_id, operation)
	);

	CREATE INDEX IF NOT EXISTS idx_credentials_sync ON credentials(sync_status);
	CREATE INDEX IF NOT EXISTS idx_cards_sync ON cards(sync_status);
	CREATE INDEX IF NOT EXISTS idx_texts_sync ON texts(sync_status);
	CREATE INDEX IF NOT EXISTS idx_binaries_sync ON binaries(sync_status);
	CREATE INDEX IF NOT EXISTS idx_sync_queue_type ON sync_queue(item_type);
	`

	_, err := l.db.Exec(schema)
	return err
}

// LocalCredential represents a locally stored credential.
type LocalCredential struct {
	ID         string
	Name       string
	Login      string
	Password   string
	URL        *string
	Metadata   *string
	SyncStatus SyncStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	SyncedAt   *time.Time
}

// SaveCredential saves or updates a credential in local storage.
func (l *LocalDB) SaveCredential(cred *LocalCredential) error {
	query := `
		INSERT OR REPLACE INTO credentials 
		(id, name, login, password, url, metadata, sync_status, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var syncedAt *int64
	if cred.SyncedAt != nil {
		ts := cred.SyncedAt.Unix()
		syncedAt = &ts
	}

	_, err := l.db.Exec(query,
		cred.ID,
		cred.Name,
		cred.Login,
		cred.Password,
		cred.URL,
		cred.Metadata,
		string(cred.SyncStatus),
		cred.CreatedAt.Unix(),
		cred.UpdatedAt.Unix(),
		syncedAt,
	)

	return err
}

// GetCredential retrieves a credential by ID.
func (l *LocalDB) GetCredential(id string) (*LocalCredential, error) {
	query := `
		SELECT id, name, login, password, url, metadata, sync_status, 
		       created_at, updated_at, synced_at
		FROM credentials 
		WHERE id = ?
	`

	var cred LocalCredential
	var createdAtUnix, updatedAtUnix int64
	var syncedAtUnix *int64

	err := l.db.QueryRow(query, id).Scan(
		&cred.ID,
		&cred.Name,
		&cred.Login,
		&cred.Password,
		&cred.URL,
		&cred.Metadata,
		&cred.SyncStatus,
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

	cred.CreatedAt = time.Unix(createdAtUnix, 0)
	cred.UpdatedAt = time.Unix(updatedAtUnix, 0)
	if syncedAtUnix != nil {
		t := time.Unix(*syncedAtUnix, 0)
		cred.SyncedAt = &t
	}

	return &cred, nil
}

// ListCredentials returns all credentials.
func (l *LocalDB) ListCredentials() ([]*LocalCredential, error) {
	query := `
		SELECT id, name, login, password, url, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM credentials 
		ORDER BY updated_at DESC
	`

	rows, err := l.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*LocalCredential
	for rows.Next() {
		var cred LocalCredential
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&cred.ID,
			&cred.Name,
			&cred.Login,
			&cred.Password,
			&cred.URL,
			&cred.Metadata,
			&cred.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		cred.CreatedAt = time.Unix(createdAtUnix, 0)
		cred.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			cred.SyncedAt = &t
		}

		creds = append(creds, &cred)
	}

	return creds, rows.Err()
}

// DeleteCredential removes a credential from local storage.
func (l *LocalDB) DeleteCredential(id string) error {
	_, err := l.db.Exec("DELETE FROM credentials WHERE id = ?", id)
	return err
}

// UpdateCredentialSyncStatus updates the sync status of a credential.
func (l *LocalDB) UpdateCredentialSyncStatus(id string, status SyncStatus) error {
	query := `UPDATE credentials SET sync_status = ?, synced_at = ? WHERE id = ?`

	var syncedAt *int64
	if status == StatusSynced {
		now := time.Now().Unix()
		syncedAt = &now
	}

	_, err := l.db.Exec(query, string(status), syncedAt, id)
	return err
}

// GetPendingCredentials returns credentials that need to be synced.
func (l *LocalDB) GetPendingCredentials() ([]*LocalCredential, error) {
	query := `
		SELECT id, name, login, password, url, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM credentials 
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
	`

	rows, err := l.db.Query(query, string(StatusPending), string(StatusUpdated))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*LocalCredential
	for rows.Next() {
		var cred LocalCredential
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&cred.ID,
			&cred.Name,
			&cred.Login,
			&cred.Password,
			&cred.URL,
			&cred.Metadata,
			&cred.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		cred.CreatedAt = time.Unix(createdAtUnix, 0)
		cred.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			cred.SyncedAt = &t
		}

		creds = append(creds, &cred)
	}

	return creds, rows.Err()
}
