package handlers

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// PaginationParams holds page-based pagination parameters parsed from the query string.
// Supports both page/page_size (preferred) and limit/offset (legacy) styles.
// When page/page_size are absent, the caller can still use Limit/Offset directly.
type PaginationParams struct {
	Page     int `form:"page"`      // 1-based page number (default 1)
	PageSize int `form:"page_size"` // items per page (default 20, max 100)
	// Derived convenience fields
	Limit  int32
	Offset int32
}

// PaginationMeta describes the current page position within the full result set.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
}

// PaginatedResponse wraps any list payload with pagination metadata.
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// ParsePagination reads page/page_size (or legacy limit/offset) from the Gin context
// and returns normalised PaginationParams. It is always safe to call; invalid or
// missing values fall back to sensible defaults.
//
// Priority order:
//  1. page + page_size  → converted to limit/offset
//  2. limit + offset    → used directly (legacy callers)
//  3. defaults          → page 1, page_size 20
func ParsePagination(c *gin.Context) PaginationParams {
	p := PaginationParams{
		Page:     1,
		PageSize: defaultPageSize,
	}

	// Prefer page/page_size
	if pageStr := c.Query("page"); pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v >= 1 {
			p.Page = v
		}
	}
	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if v, err := strconv.Atoi(sizeStr); err == nil && v >= 1 {
			p.PageSize = v
		}
	}

	// Legacy limit/offset — only honoured when page/page_size are both absent
	if c.Query("page") == "" && c.Query("page_size") == "" {
		if limitStr := c.Query("limit"); limitStr != "" {
			if v, err := strconv.Atoi(limitStr); err == nil && v >= 1 {
				p.PageSize = v
			}
		}
		if offsetStr := c.Query("offset"); offsetStr != "" {
			if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
				// Reconstruct page number from offset
				if p.PageSize > 0 {
					p.Page = (v / p.PageSize) + 1
				}
			}
		}
	}

	// Cap page size
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}

	p.Limit = int32(p.PageSize)
	p.Offset = int32((p.Page - 1) * p.PageSize)
	return p
}

// NewPaginatedResponse constructs a PaginatedResponse from a data payload, the total
// item count, and the pagination parameters used to fetch data.
func NewPaginatedResponse(data interface{}, total int64, params PaginationParams) PaginatedResponse {
	totalPages := 0
	if params.PageSize > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.PageSize)))
	}
	return PaginatedResponse{
		Data: data,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalItems: total,
			TotalPages: totalPages,
			HasMore:    int64(params.Page*params.PageSize) < total,
		},
	}
}
