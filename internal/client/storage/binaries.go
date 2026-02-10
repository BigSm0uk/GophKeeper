package storage

import (
	"database/sql"
	"time"
)

// LocalBinary represents a locally stored binary file metadata.
type LocalBinary struct {
	ID          string
	Name        string
	Filename    string
	FilePath    string
	Size        int64
	ContentType string
	Checksum    string
	Metadata    *string
	SyncStatus  SyncStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	SyncedAt    *time.Time
}

// SaveBinary saves or updates a binary in local storage.
func (l *LocalDB) SaveBinary(binary *LocalBinary) error {
	query := `
		INSERT OR REPLACE INTO binaries 
		(id, name, filename, file_path, size, content_type, checksum, metadata, sync_status, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var syncedAt *int64
	if binary.SyncedAt != nil {
		ts := binary.SyncedAt.Unix()
		syncedAt = &ts
	}

	_, err := l.db.Exec(query,
		binary.ID,
		binary.Name,
		binary.Filename,
		binary.FilePath,
		binary.Size,
		binary.ContentType,
		binary.Checksum,
		binary.Metadata,
		string(binary.SyncStatus),
		binary.CreatedAt.Unix(),
		binary.UpdatedAt.Unix(),
		syncedAt,
	)

	return err
}

// GetBinary retrieves a binary by GetID.
func (l *LocalDB) GetBinary(id string) (*LocalBinary, error) {
	query := `
		SELECT id, name, filename, file_path, size, content_type, checksum, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM binaries 
		WHERE id = ?
	`

	var binary LocalBinary
	var createdAtUnix, updatedAtUnix int64
	var syncedAtUnix *int64

	err := l.db.QueryRow(query, id).Scan(
		&binary.ID,
		&binary.Name,
		&binary.Filename,
		&binary.FilePath,
		&binary.Size,
		&binary.ContentType,
		&binary.Checksum,
		&binary.Metadata,
		&binary.SyncStatus,
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

	binary.CreatedAt = time.Unix(createdAtUnix, 0)
	binary.UpdatedAt = time.Unix(updatedAtUnix, 0)
	if syncedAtUnix != nil {
		t := time.Unix(*syncedAtUnix, 0)
		binary.SyncedAt = &t
	}

	return &binary, nil
}

// ListBinaries returns all active (non-deleted) binaries.
func (l *LocalDB) ListBinaries() ([]*LocalBinary, error) {
	query := `
		SELECT id, name, filename, file_path, size, content_type, checksum, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM binaries 
		WHERE sync_status != ?
		ORDER BY updated_at DESC
	`

	rows, err := l.db.Query(query, string(StatusDeleted))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var binaries []*LocalBinary
	for rows.Next() {
		var binary LocalBinary
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&binary.ID,
			&binary.Name,
			&binary.Filename,
			&binary.FilePath,
			&binary.Size,
			&binary.ContentType,
			&binary.Checksum,
			&binary.Metadata,
			&binary.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		binary.CreatedAt = time.Unix(createdAtUnix, 0)
		binary.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			binary.SyncedAt = &t
		}

		binaries = append(binaries, &binary)
	}

	return binaries, rows.Err()
}

// DeleteBinary marks a binary as deleted for sync.
func (l *LocalDB) DeleteBinary(id string) error {
	_, err := l.db.Exec(
		"UPDATE binaries SET sync_status = ?, updated_at = ? WHERE id = ?",
		string(StatusDeleted), time.Now().Unix(), id,
	)
	return err
}

// PurgeBinary physically removes a binary from local storage after sync.
func (l *LocalDB) PurgeBinary(id string) error {
	_, err := l.db.Exec("DELETE FROM binaries WHERE id = ?", id)
	return err
}

// GetDeletedBinaries returns binaries marked as deleted that need sync.
func (l *LocalDB) GetDeletedBinaries() ([]*LocalBinary, error) {
	query := `
		SELECT id, name, filename, file_path, size, content_type, checksum, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM binaries 
		WHERE sync_status = ?
		ORDER BY updated_at ASC
	`

	rows, err := l.db.Query(query, string(StatusDeleted))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var binaries []*LocalBinary
	for rows.Next() {
		var binary LocalBinary
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&binary.ID,
			&binary.Name,
			&binary.Filename,
			&binary.FilePath,
			&binary.Size,
			&binary.ContentType,
			&binary.Checksum,
			&binary.Metadata,
			&binary.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		binary.CreatedAt = time.Unix(createdAtUnix, 0)
		binary.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			binary.SyncedAt = &t
		}

		binaries = append(binaries, &binary)
	}

	return binaries, rows.Err()
}

// UpdateBinarySyncStatus updates the sync status of a binary.
func (l *LocalDB) UpdateBinarySyncStatus(id string, status SyncStatus) error {
	query := `UPDATE binaries SET sync_status = ?, synced_at = ? WHERE id = ?`

	var syncedAt *int64
	if status == StatusSynced {
		now := time.Now().Unix()
		syncedAt = &now
	}

	_, err := l.db.Exec(query, string(status), syncedAt, id)
	return err
}

// GetPendingBinaries returns binaries that need to be synced.
func (l *LocalDB) GetPendingBinaries() ([]*LocalBinary, error) {
	query := `
		SELECT id, name, filename, file_path, size, content_type, checksum, metadata, sync_status,
		       created_at, updated_at, synced_at
		FROM binaries 
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
	`

	rows, err := l.db.Query(query, string(StatusPending), string(StatusUpdated))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var binaries []*LocalBinary
	for rows.Next() {
		var binary LocalBinary
		var createdAtUnix, updatedAtUnix int64
		var syncedAtUnix *int64

		err := rows.Scan(
			&binary.ID,
			&binary.Name,
			&binary.Filename,
			&binary.FilePath,
			&binary.Size,
			&binary.ContentType,
			&binary.Checksum,
			&binary.Metadata,
			&binary.SyncStatus,
			&createdAtUnix,
			&updatedAtUnix,
			&syncedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		binary.CreatedAt = time.Unix(createdAtUnix, 0)
		binary.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if syncedAtUnix != nil {
			t := time.Unix(*syncedAtUnix, 0)
			binary.SyncedAt = &t
		}

		binaries = append(binaries, &binary)
	}

	return binaries, rows.Err()
}
