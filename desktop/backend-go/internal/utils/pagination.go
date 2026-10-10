package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page    int32 // Current page (1-indexed)
	PerPage int32 // Items per page
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Items   interface{} `json:"items"`
	Total   int64       `json:"total"`
	Page    int32       `json:"page"`
	PerPage int32       `json:"per_page"`
}

// DefaultPaginationParams returns default pagination settings
func DefaultPaginationParams() PaginationParams {
	return PaginationParams{
		Page:    1,
		PerPage: 20,
	}
}

// ExtractPaginationParams extracts pagination parameters from query string
// Query params: ?page=1&per_page=20
// Defaults: page=1, per_page=20, max per_page=100
func ExtractPaginationParams(c *gin.Context) PaginationParams {
	params := DefaultPaginationParams()

	// Extract page
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.ParseInt(pageStr, 10, 32); err == nil && page > 0 {
			params.Page = int32(page)
		}
	}

	// Extract per_page
	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if perPage, err := strconv.ParseInt(perPageStr, 10, 32); err == nil && perPage > 0 {
			params.PerPage = int32(perPage)
		}
	}

	// Enforce max per_page = 100
	if params.PerPage > 100 {
		params.PerPage = 100
	}

	return params
}

// Offset calculates the database offset for LIMIT/OFFSET queries
func (p PaginationParams) Offset() int32 {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the limit value (same as PerPage)
func (p PaginationParams) Limit() int32 {
	return p.PerPage
}

// SetPaginationHeaders sets pagination headers on the response
// Headers: X-Total-Count, X-Page, X-Per-Page
func SetPaginationHeaders(c *gin.Context, total int64, params PaginationParams) {
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	c.Header("X-Page", strconv.FormatInt(int64(params.Page), 10))
	c.Header("X-Per-Page", strconv.FormatInt(int64(params.PerPage), 10))
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(items interface{}, total int64, params PaginationParams) PaginatedResponse {
	return PaginatedResponse{
		Items:   items,
		Total:   total,
		Page:    params.Page,
		PerPage: params.PerPage,
	}
}
