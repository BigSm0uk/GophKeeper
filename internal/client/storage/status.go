package storage

import "time"

// SyncStatus отражает состояние локального элемента относительно сервера.
type SyncStatus string

const (
	StatusSynced    SyncStatus = "synced"
	StatusPending   SyncStatus = "pending"
	StatusUploading SyncStatus = "uploading"
	StatusUpdated   SyncStatus = "updated"
)

// ItemMeta хранит минимальные метаданные для локального кеша.
type ItemMeta struct {
	ID        string
	UpdatedAt time.Time
	Status    SyncStatus
}
