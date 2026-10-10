package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/?"+query, nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

// ---------------------------------------------------------------------------
// ParsePagination
// ---------------------------------------------------------------------------

func TestParsePagination_Defaults(t *testing.T) {
	c := newTestContext("")
	p := ParsePagination(c)
	if p.Page != 1 {
		t.Errorf("expected page=1, got %d", p.Page)
	}
	if p.PageSize != defaultPageSize {
		t.Errorf("expected page_size=%d, got %d", defaultPageSize, p.PageSize)
	}
	if p.Limit != int32(defaultPageSize) {
		t.Errorf("expected limit=%d, got %d", defaultPageSize, p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("expected offset=0, got %d", p.Offset)
	}
}

func TestParsePagination_PageAndPageSize(t *testing.T) {
	c := newTestContext("page=3&page_size=10")
	p := ParsePagination(c)
	if p.Page != 3 {
		t.Errorf("expected page=3, got %d", p.Page)
	}
	if p.PageSize != 10 {
		t.Errorf("expected page_size=10, got %d", p.PageSize)
	}
	if p.Limit != 10 {
		t.Errorf("expected limit=10, got %d", p.Limit)
	}
	if p.Offset != 20 { // (3-1)*10 = 20
		t.Errorf("expected offset=20, got %d", p.Offset)
	}
}

func TestParsePagination_LegacyLimitOffset(t *testing.T) {
	c := newTestContext("limit=5&offset=10")
	p := ParsePagination(c)
	if p.Limit != 5 {
		t.Errorf("expected limit=5, got %d", p.Limit)
	}
	if p.Offset != 10 {
		t.Errorf("expected offset=10, got %d", p.Offset)
	}
	// page should be reconstructed: offset/limit + 1 = 10/5+1 = 3
	if p.Page != 3 {
		t.Errorf("expected page=3, got %d", p.Page)
	}
}

func TestParsePagination_PageSizeCappedAt100(t *testing.T) {
	c := newTestContext("page_size=999")
	p := ParsePagination(c)
	if p.PageSize != maxPageSize {
		t.Errorf("expected page_size capped at %d, got %d", maxPageSize, p.PageSize)
	}
	if p.Limit != int32(maxPageSize) {
		t.Errorf("expected limit=%d, got %d", maxPageSize, p.Limit)
	}
}

func TestParsePagination_InvalidValuesUseDefaults(t *testing.T) {
	c := newTestContext("page=-1&page_size=abc")
	p := ParsePagination(c)
	if p.Page != 1 {
		t.Errorf("expected page=1 on invalid input, got %d", p.Page)
	}
	if p.PageSize != defaultPageSize {
		t.Errorf("expected page_size=%d on invalid input, got %d", defaultPageSize, p.PageSize)
	}
}

func TestParsePagination_PagePreferredOverLimit(t *testing.T) {
	// When page/page_size present, limit/offset should be ignored
	c := newTestContext("page=2&page_size=10&limit=99&offset=99")
	p := ParsePagination(c)
	if p.Page != 2 || p.PageSize != 10 {
		t.Errorf("expected page=2, page_size=10; got page=%d, page_size=%d", p.Page, p.PageSize)
	}
	if p.Offset != 10 { // (2-1)*10
		t.Errorf("expected offset=10, got %d", p.Offset)
	}
}

// ---------------------------------------------------------------------------
// NewPaginatedResponse
// ---------------------------------------------------------------------------

func TestNewPaginatedResponse_Basic(t *testing.T) {
	pg := PaginationParams{Page: 1, PageSize: 10, Limit: 10, Offset: 0}
	resp := NewPaginatedResponse([]string{"a", "b"}, 25, pg)

	meta := resp.Pagination
	if meta.TotalItems != 25 {
		t.Errorf("expected total_items=25, got %d", meta.TotalItems)
	}
	if meta.TotalPages != 3 { // ceil(25/10)
		t.Errorf("expected total_pages=3, got %d", meta.TotalPages)
	}
	if !meta.HasMore {
		t.Error("expected has_more=true for page 1 of 3")
	}
}

func TestNewPaginatedResponse_LastPage(t *testing.T) {
	pg := PaginationParams{Page: 3, PageSize: 10, Limit: 10, Offset: 20}
	resp := NewPaginatedResponse([]string{"y", "z"}, 22, pg)

	meta := resp.Pagination
	if meta.HasMore {
		t.Error("expected has_more=false on last page")
	}
	if meta.TotalPages != 3 { // ceil(22/10)
		t.Errorf("expected total_pages=3, got %d", meta.TotalPages)
	}
}

func TestNewPaginatedResponse_EmptyResult(t *testing.T) {
	pg := PaginationParams{Page: 1, PageSize: 20, Limit: 20, Offset: 0}
	resp := NewPaginatedResponse([]string{}, 0, pg)

	meta := resp.Pagination
	if meta.TotalItems != 0 {
		t.Errorf("expected total_items=0, got %d", meta.TotalItems)
	}
	if meta.TotalPages != 0 {
		t.Errorf("expected total_pages=0 for empty, got %d", meta.TotalPages)
	}
	if meta.HasMore {
		t.Error("expected has_more=false for empty result")
	}
}

func TestNewPaginatedResponse_ExactPage(t *testing.T) {
	// 20 items, page_size=20 → exactly one page, no more
	pg := PaginationParams{Page: 1, PageSize: 20, Limit: 20, Offset: 0}
	resp := NewPaginatedResponse(make([]int, 20), 20, pg)

	if resp.Pagination.HasMore {
		t.Error("expected has_more=false when items == total")
	}
	if resp.Pagination.TotalPages != 1 {
		t.Errorf("expected 1 total page, got %d", resp.Pagination.TotalPages)
	}
}
