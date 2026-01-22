package models

import (
	"time"
)

// SyncOperation represents the type of synchronization operation.
type SyncOperation string

const (
	SyncOperationCreate SyncOperation = "create"
	SyncOperationUpdate SyncOperation = "update"
	SyncOperationDelete SyncOperation = "delete"
)

// SyncItemType represents the type of synchronized item.
type SyncItemType string

const (
	SyncItemTypeCredential SyncItemType = "credentials"
	SyncItemTypeText       SyncItemType = "texts"
	SyncItemTypeBinary     SyncItemType = "binaries"
	SyncItemTypeCard       SyncItemType = "cards"
)

// SyncChange represents a single change in synchronization.
type SyncChange struct {
	ID        string
	Operation SyncOperation
	ItemType  SyncItemType
	UserID    string
	Data      []byte // serialized entity data
	Timestamp time.Time
}

// SyncChanges represents all changes grouped by type.
type SyncChanges struct {
	Credentials []*SyncChange
	Texts       []*SyncChange
	Binaries    []*SyncChange
	Cards       []*SyncChange
}

// SyncConflict represents a conflict between local and server data.
type SyncConflict struct {
	ItemType      SyncItemType
	LocalID       string
	ServerID      string
	ConflictType  string // e.g., "concurrent_update"
	ServerVersion []byte
}

// SyncPullResponse represents the server's response to sync pull request.
type SyncPullResponse struct {
	Timestamp time.Time
	Changes   *SyncChanges
}

// SyncPushResponse represents the server's response to sync push request.
type SyncPushResponse struct {
	Timestamp time.Time
	Conflicts []*SyncConflict
	SyncedIDs map[string]string // local_id -> server_id mapping
}

// NewSyncChange creates a new sync change.
func NewSyncChange(id string, operation SyncOperation, itemType SyncItemType, userID string, data []byte) *SyncChange {
	return &SyncChange{
		ID:        id,
		Operation: operation,
		ItemType:  itemType,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewSyncChanges creates an empty SyncChanges instance.
func NewSyncChanges() *SyncChanges {
	return &SyncChanges{
		Credentials: make([]*SyncChange, 0),
		Texts:       make([]*SyncChange, 0),
		Binaries:    make([]*SyncChange, 0),
		Cards:       make([]*SyncChange, 0),
	}
}

// AddCredentialChange adds a credential change to the sync changes.
func (sc *SyncChanges) AddCredentialChange(change *SyncChange) {
	if change.ItemType == SyncItemTypeCredential {
		sc.Credentials = append(sc.Credentials, change)
	}
}

// AddTextChange adds a text change to the sync changes.
func (sc *SyncChanges) AddTextChange(change *SyncChange) {
	if change.ItemType == SyncItemTypeText {
		sc.Texts = append(sc.Texts, change)
	}
}

// AddBinaryChange adds a binary change to the sync changes.
func (sc *SyncChanges) AddBinaryChange(change *SyncChange) {
	if change.ItemType == SyncItemTypeBinary {
		sc.Binaries = append(sc.Binaries, change)
	}
}

// AddCardChange adds a card change to the sync changes.
func (sc *SyncChanges) AddCardChange(change *SyncChange) {
	if change.ItemType == SyncItemTypeCard {
		sc.Cards = append(sc.Cards, change)
	}
}

// IsEmpty checks if there are any changes.
func (sc *SyncChanges) IsEmpty() bool {
	return len(sc.Credentials) == 0 &&
		len(sc.Texts) == 0 &&
		len(sc.Binaries) == 0 &&
		len(sc.Cards) == 0
}

// GetTotalChanges returns the total number of changes.
func (sc *SyncChanges) GetTotalChanges() int {
	return len(sc.Credentials) + len(sc.Texts) + len(sc.Binaries) + len(sc.Cards)
}

// NewSyncConflict creates a new sync conflict.
func NewSyncConflict(itemType SyncItemType, localID, serverID, conflictType string, serverVersion []byte) *SyncConflict {
	return &SyncConflict{
		ItemType:      itemType,
		LocalID:       localID,
		ServerID:      serverID,
		ConflictType:  conflictType,
		ServerVersion: serverVersion,
	}
}

// NewSyncPullResponse creates a new sync pull response.
func NewSyncPullResponse(timestamp time.Time, changes *SyncChanges) *SyncPullResponse {
	return &SyncPullResponse{
		Timestamp: timestamp,
		Changes:   changes,
	}
}

// NewSyncPushResponse creates a new sync push response.
func NewSyncPushResponse(timestamp time.Time, conflicts []*SyncConflict, syncedIDs map[string]string) *SyncPushResponse {
	return &SyncPushResponse{
		Timestamp: timestamp,
		Conflicts: conflicts,
		SyncedIDs: syncedIDs,
	}
}

// HasConflicts checks if there are any conflicts.
func (spr *SyncPushResponse) HasConflicts() bool {
	return len(spr.Conflicts) > 0
}

// GetConflictCount returns the number of conflicts.
func (spr *SyncPushResponse) GetConflictCount() int {
	return len(spr.Conflicts)
}
