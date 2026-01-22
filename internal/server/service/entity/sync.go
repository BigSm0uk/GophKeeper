package entity

import (
	"fmt"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// MapSyncChangeToResponse converts domain SyncChange to protobuf SyncChange.
func MapSyncChangeToResponse(change *SyncChange) (*pb.SyncChange, error) {
	if change == nil {
		return nil, fmt.Errorf("sync change is nil")
	}

	var operation pb.SyncOperation
	switch change.Operation {
	case SyncOperationCreate:
		operation = pb.SyncOperation_SYNC_OPERATION_CREATE
	case SyncOperationUpdate:
		operation = pb.SyncOperation_SYNC_OPERATION_UPDATE
	case SyncOperationDelete:
		operation = pb.SyncOperation_SYNC_OPERATION_DELETE
	default:
		return nil, fmt.Errorf("unknown sync operation: %s", change.Operation)
	}

	response := &pb.SyncChange{
		Id:        change.ID,
		Operation: operation,
		Data:      change.Data,
	}

	return response, nil
}

// MapSyncChangeFromRequest converts protobuf SyncChange to domain SyncChange.
func MapSyncChangeFromRequest(itemType SyncItemType, userID string, req *pb.SyncChange) (*SyncChange, error) {
	if req == nil {
		return nil, fmt.Errorf("sync change request is nil")
	}

	var operation SyncOperation
	switch req.Operation {
	case pb.SyncOperation_SYNC_OPERATION_CREATE:
		operation = SyncOperationCreate
	case pb.SyncOperation_SYNC_OPERATION_UPDATE:
		operation = SyncOperationUpdate
	case pb.SyncOperation_SYNC_OPERATION_DELETE:
		operation = SyncOperationDelete
	default:
		return nil, fmt.Errorf("unknown sync operation: %v", req.Operation)
	}

	change := &SyncChange{
		ID:        req.Id,
		Operation: operation,
		ItemType:  itemType,
		UserID:    userID,
		Data:      req.Data,
		Timestamp: time.Now(),
	}

	return change, nil
}

// MapSyncChangesToResponse converts domain SyncChanges to protobuf SyncPullResponse.
func MapSyncChangesToResponse(changes *SyncChanges, timestamp time.Time) (*pb.SyncPullResponse, error) {
	if changes == nil {
		return nil, fmt.Errorf("sync changes is nil")
	}

	response := &pb.SyncPullResponse{
		Timestamp: timestamppb.New(timestamp),
	}

	// Map credentials changes
	if len(changes.Credentials) > 0 {
		credentials := make([]*pb.SyncChange, 0, len(changes.Credentials))
		for _, change := range changes.Credentials {
			pbChange, err := MapSyncChangeToResponse(change)
			if err != nil {
				return nil, fmt.Errorf("mapping credential change: %w", err)
			}
			credentials = append(credentials, pbChange)
		}
		response.Credentials = credentials
	}

	// Map texts changes
	if len(changes.Texts) > 0 {
		texts := make([]*pb.SyncChange, 0, len(changes.Texts))
		for _, change := range changes.Texts {
			pbChange, err := MapSyncChangeToResponse(change)
			if err != nil {
				return nil, fmt.Errorf("mapping text change: %w", err)
			}
			texts = append(texts, pbChange)
		}
		response.Texts = texts
	}

	// Map binaries changes
	if len(changes.Binaries) > 0 {
		binaries := make([]*pb.SyncChange, 0, len(changes.Binaries))
		for _, change := range changes.Binaries {
			pbChange, err := MapSyncChangeToResponse(change)
			if err != nil {
				return nil, fmt.Errorf("mapping binary change: %w", err)
			}
			binaries = append(binaries, pbChange)
		}
		response.Binaries = binaries
	}

	// Map cards changes
	if len(changes.Cards) > 0 {
		cards := make([]*pb.SyncChange, 0, len(changes.Cards))
		for _, change := range changes.Cards {
			pbChange, err := MapSyncChangeToResponse(change)
			if err != nil {
				return nil, fmt.Errorf("mapping card change: %w", err)
			}
			cards = append(cards, pbChange)
		}
		response.Cards = cards
	}

	return response, nil
}

// MapSyncConflictToResponse converts domain SyncConflict to protobuf SyncConflict.
func MapSyncConflictToResponse(conflict *SyncConflict) (*pb.SyncConflict, error) {
	if conflict == nil {
		return nil, fmt.Errorf("sync conflict is nil")
	}

	response := &pb.SyncConflict{
		EntityType:    string(conflict.ItemType),
		LocalId:       conflict.LocalID,
		ServerId:      conflict.ServerID,
		ServerVersion: conflict.ServerVersion,
	}

	return response, nil
}

// MapSyncPushResponseToResponse converts domain SyncPushResponse to protobuf SyncPushResponse.
func MapSyncPushResponseToResponse(resp *SyncPushResponse) (*pb.SyncPushResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("sync push response is nil")
	}

	response := &pb.SyncPushResponse{
		Timestamp: timestamppb.New(resp.Timestamp),
	}

	// Map conflicts
	if len(resp.Conflicts) > 0 {
		conflicts := make([]*pb.SyncConflict, 0, len(resp.Conflicts))
		for _, conflict := range resp.Conflicts {
			pbConflict, err := MapSyncConflictToResponse(conflict)
			if err != nil {
				return nil, fmt.Errorf("mapping conflict: %w", err)
			}
			conflicts = append(conflicts, pbConflict)
		}
		response.Conflicts = conflicts
	}

	return response, nil
}
