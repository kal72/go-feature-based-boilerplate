// Package pagination provides production-grade pagination utilities for Go microservices,
// supporting both traditional Offset-Based Pagination (ideal for Web/CMS dashboards) and
// high-performance Cursor-Based Pagination (ideal for mobile feeds and infinite scrolling).
package pagination

const (
	// DefaultPage specifies the starting page index if omitted or invalid.
	DefaultPage = 1

	// DefaultPageSize defines the default number of items fetched per page.
	DefaultPageSize = 20

	// MaxPageSize restricts the maximum allowed items fetched in a single request to prevent DOS attacks.
	MaxPageSize = 100
)

// Params holds offset-based pagination parameters supplied by the client.
type Params struct {
	// Page is the 1-based page number requested.
	Page int `json:"page"`

	// PageSize is the number of records to return per page.
	PageSize int `json:"page_size"`
}

// Normalize sanitizes and enforces safe bounds on pagination parameters.
// If Page < 1, it defaults to DefaultPage (1).
// If PageSize < 1, it defaults to DefaultPageSize (20).
// If PageSize > MaxPageSize, it is capped at MaxPageSize (100).
func (p *Params) Normalize() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Offset calculates the zero-based SQL row offset for query execution: (Page - 1) * PageSize.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the number of rows to retrieve from the database.
func (p Params) Limit() int {
	return p.PageSize
}

// Result represents a standardized generic paginated response payload.
type Result[T any] struct {
	// Items contains the slice of records retrieved for the current page.
	Items []T `json:"items"`

	// TotalItems is the overall count of records matching the query.
	TotalItems int64 `json:"total_items"`

	// TotalPages is the total number of pages available based on TotalItems and PageSize.
	TotalPages int `json:"total_pages"`

	// Page is the current 1-based page number.
	Page int `json:"page"`

	// PageSize is the number of records requested per page.
	PageSize int `json:"page_size"`

	// HasNext indicates whether there is a subsequent page available.
	HasNext bool `json:"hasNext"`

	// HasPrev indicates whether there is a preceding page available.
	HasPrev bool `json:"hasPrev"`
}

// NewResult constructs a Result value from a slice of items, the total matching records,
// and the pagination parameters used. It automatically computes TotalPages, HasNext, and HasPrev.
func NewResult[T any](items []T, totalItems int64, params Params) Result[T] {
	params.Normalize()

	totalPages := 0
	if params.PageSize > 0 {
		totalPages = int((totalItems + int64(params.PageSize) - 1) / int64(params.PageSize))
	}

	hasNext := params.Page < totalPages
	hasPrev := params.Page > 1 && totalPages > 0

	return Result[T]{
		Items:      items,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       params.Page,
		PageSize:   params.PageSize,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}
}
