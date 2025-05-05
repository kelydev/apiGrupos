package utils

import (
	"net/http"
	"strconv"
)

// GetPaginationParams parses page and limit query parameters from a request.
// Returns page (default 1) and limit (default 6, max 100).
func GetPaginationParams(r *http.Request) (page, limit int) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1 // Default to page 1
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 6 // Default to 6 items per page
	}
	if limit > 100 { // Optional: Max limit
		limit = 100
	}
	return page, limit
}
