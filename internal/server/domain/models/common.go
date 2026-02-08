package models

// PaginationParams represents pagination parameters.
type PaginationParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
