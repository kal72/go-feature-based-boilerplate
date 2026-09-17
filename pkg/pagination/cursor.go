package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidCursor is returned when an opaque cursor string cannot be parsed or decoded.
	ErrInvalidCursor = errors.New("pagination: invalid cursor format")
)

// CursorParams holds cursor-based pagination parameters supplied by the client.
type CursorParams struct {
	// Cursor is an opaque Base64-encoded token pointing to the boundary record of the previous page.
	Cursor string `json:"cursor"`

	// Limit specifies the maximum number of records to return.
	Limit int `json:"limit"`
}

// Normalize sanitizes and enforces safe bounds on CursorParams.
func (p *CursorParams) Normalize() {
	if p.Limit < 1 {
		p.Limit = DefaultPageSize
	}
	if p.Limit > MaxPageSize {
		p.Limit = MaxPageSize
	}
}

// CursorData represents the decoded contents of an opaque cursor token.
type CursorData struct {
	// ID holds the unique record identifier (e.g. integer or UUID string).
	ID string `json:"id"`

	// CreatedAt holds an optional timestamp for composite sorting (e.g., created_at DESC, id DESC).
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// EncodeCursor creates an opaque Base64 URL-safe cursor token from an ID and optional timestamp.
//
// Example usage:
//
//	token := pagination.EncodeCursor(order.ID, order.CreatedAt)
func EncodeCursor(id any, createdAt ...time.Time) string {
	data := CursorData{
		ID: fmt.Sprintf("%v", id),
	}
	if len(createdAt) > 0 && !createdAt[0].IsZero() {
		t := createdAt[0].UTC()
		data.CreatedAt = &t
	}

	raw, _ := json.Marshal(data)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeCursor parses an opaque Base64 URL-safe cursor token into structured CursorData.
// If cursorToken is empty, it returns nil, nil without error.
//
// Example usage:
//
//	cursorData, err := pagination.DecodeCursor(req.Cursor)
//	if err != nil {
//	    return nil, err
//	}
//	if cursorData != nil {
//	    query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", cursorData.CreatedAt, cursorData.CreatedAt, cursorData.ID)
//	}
func DecodeCursor(cursorToken string) (*CursorData, error) {
	if cursorToken == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursorToken)
	if err != nil {
		// Fallback for standard padding if supplied by external clients
		raw, err = base64.URLEncoding.DecodeString(cursorToken)
		if err != nil {
			return nil, ErrInvalidCursor
		}
	}

	var data CursorData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, ErrInvalidCursor
	}

	return &data, nil
}

// CursorResult represents a paginated response using cursor pointers.
type CursorResult[T any] struct {
	// Items contains the records retrieved for the current window.
	Items []T `json:"items"`

	// NextCursor is the opaque token to pass in subsequent requests to fetch the next page.
	NextCursor string `json:"next_cursor,omitempty"`

	// PrevCursor is the opaque token to pass in requests to fetch the previous page.
	PrevCursor string `json:"prev_cursor,omitempty"`

	// HasNext indicates whether further records exist after this window.
	HasNext bool `json:"has_next"`

	// HasPrev indicates whether records exist before this window.
	HasPrev bool `json:"has_prev"`

	// Limit is the page size limit used for this query.
	Limit int `json:"limit"`
}

// NewCursorResult constructs a CursorResult from fetched items.
// If the caller fetched (limit + 1) items from the database to detect hasMore,
// NewCursorResult slices off the extra probe record automatically.
//
// Example usage:
//
//	// In Repository: Fetch (limit + 1) items
//	db.Limit(params.Limit + 1).Find(&orders)
//	hasMore := len(orders) > params.Limit
//	return pagination.NewCursorResult(orders, params.Limit, hasMore, func(o Order) string {
//	    return pagination.EncodeCursor(o.ID, o.CreatedAt)
//	})
func NewCursorResult[T any](items []T, limit int, hasMore bool, getCursorFn func(item T) string) CursorResult[T] {
	if limit < 1 {
		limit = DefaultPageSize
	}

	var nextCursor string
	var prevCursor string

	// If extra item was fetched to detect more rows, slice it off
	actualItems := items
	if hasMore && len(actualItems) > limit {
		actualItems = actualItems[:limit]
	}

	if len(actualItems) > 0 && getCursorFn != nil {
		prevCursor = getCursorFn(actualItems[0])
		if hasMore {
			nextCursor = getCursorFn(actualItems[len(actualItems)-1])
		}
	}

	return CursorResult[T]{
		Items:      actualItems,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasMore,
		HasPrev:    prevCursor != "",
		Limit:      limit,
	}
}
