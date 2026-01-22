package models

import "time"

// PaginationParams represents pagination parameters.
type PaginationParams struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Search string `json:"search,omitempty"`
}

// PaginationResult represents pagination result metadata.
type PaginationResult struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// NewPaginationParams creates pagination parameters with defaults.
func NewPaginationParams(limit, offset int, search string) *PaginationParams {
	if limit <= 0 {
		limit = 20 // default limit
	}
	if limit > 100 {
		limit = 100 // max limit
	}
	if offset < 0 {
		offset = 0
	}

	return &PaginationParams{
		Limit:  limit,
		Offset: offset,
		Search: search,
	}
}

// NewPaginationResult creates pagination result metadata.
func NewPaginationResult(total int64, params *PaginationParams) *PaginationResult {
	return &PaginationResult{
		Total:  total,
		Limit:  params.Limit,
		Offset: params.Offset,
	}
}

// HasNextPage checks if there are more pages available.
func (pr *PaginationResult) HasNextPage() bool {
	return int64(pr.Offset+pr.Limit) < pr.Total
}

// HasPreviousPage checks if there are previous pages.
func (pr *PaginationResult) HasPreviousPage() bool {
	return pr.Offset > 0
}

// GetTotalPages calculates total number of pages.
func (pr *PaginationResult) GetTotalPages() int {
	if pr.Limit == 0 {
		return 0
	}
	totalPages := int(pr.Total / int64(pr.Limit))
	if pr.Total%int64(pr.Limit) != 0 {
		totalPages++
	}
	return totalPages
}

// GetCurrentPage calculates current page number (1-based).
func (pr *PaginationResult) GetCurrentPage() int {
	if pr.Limit == 0 {
		return 1
	}
	return (pr.Offset / pr.Limit) + 1
}

// AuditInfo represents audit information for entities.
type AuditInfo struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by,omitempty"`
	UpdatedBy string    `json:"updated_by,omitempty"`
}

// NewAuditInfo creates new audit info with current timestamp.
func NewAuditInfo(userID string) *AuditInfo {
	now := time.Now()
	return &AuditInfo{
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: userID,
		UpdatedBy: userID,
	}
}

// UpdateAuditInfo updates the audit info for modifications.
func (ai *AuditInfo) UpdateAuditInfo(userID string) {
	ai.UpdatedAt = time.Now()
	ai.UpdatedBy = userID
}

// VersionInfo represents version information for optimistic locking.
type VersionInfo struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewVersionInfo creates new version info.
func NewVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   1,
		UpdatedAt: time.Now(),
	}
}

// IncrementVersion increments the version number.
func (vi *VersionInfo) IncrementVersion() {
	vi.Version++
	vi.UpdatedAt = time.Now()
}
