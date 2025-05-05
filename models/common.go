package models

// PaginationMetadata holds information about the pagination state.
type PaginationMetadata struct {
	TotalItems  int `json:"totalItems"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
	Limit       int `json:"limit"`
}

// PaginatedResponse is a generic wrapper for paginated API responses.
type PaginatedResponse struct {
	Data       interface{}        `json:"data"` // Holds the actual slice of results (e.g., []Investigador, []GrupoWithInvestigadores)
	Pagination PaginationMetadata `json:"pagination"`
}
