package storage

import "time"

// SyncStatus represents the synchronization state of a local item relative to the server.
type SyncStatus string

const (
	StatusSynced    SyncStatus = "synced"
	StatusPending   SyncStatus = "pending"
	StatusUploading SyncStatus = "uploading"
	StatusUpdated   SyncStatus = "updated"
	StatusDeleted   SyncStatus = "deleted"
)

// ItemMeta stores minimal metadata for local cache.
type ItemMeta struct {
	ID        string
	UpdatedAt time.Time
	Status    SyncStatus
}
