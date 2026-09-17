package pagination_test

import (
	"testing"
	"time"

	"go-feature-based-boilerplate/pkg/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParams_NormalizeAndOffsets(t *testing.T) {
	// Case 1: Negative / zero values get default bounds
	p1 := pagination.Params{Page: -5, PageSize: 0}
	p1.Normalize()
	assert.Equal(t, pagination.DefaultPage, p1.Page)
	assert.Equal(t, pagination.DefaultPageSize, p1.PageSize)
	assert.Equal(t, 0, p1.Offset())
	assert.Equal(t, 20, p1.Limit())

	// Case 2: Exceeding max page size gets capped
	p2 := pagination.Params{Page: 3, PageSize: 500}
	p2.Normalize()
	assert.Equal(t, 3, p2.Page)
	assert.Equal(t, pagination.MaxPageSize, p2.PageSize)
	assert.Equal(t, 200, p2.Offset())
	assert.Equal(t, 100, p2.Limit())
}

func TestNewResult_NavigationFlags(t *testing.T) {
	items := []string{"item1", "item2"}

	// First page of 3 pages: HasNext=true, HasPrev=false
	res1 := pagination.NewResult(items, 50, pagination.Params{Page: 1, PageSize: 20})
	assert.Equal(t, 3, res1.TotalPages)
	assert.True(t, res1.HasNext)
	assert.False(t, res1.HasPrev)

	// Middle page: HasNext=true, HasPrev=true
	res2 := pagination.NewResult(items, 50, pagination.Params{Page: 2, PageSize: 20})
	assert.True(t, res2.HasNext)
	assert.True(t, res2.HasPrev)

	// Last page: HasNext=false, HasPrev=true
	res3 := pagination.NewResult(items, 50, pagination.Params{Page: 3, PageSize: 20})
	assert.False(t, res3.HasNext)
	assert.True(t, res3.HasPrev)

	// Empty dataset: TotalPages=0, HasNext=false, HasPrev=false
	res4 := pagination.NewResult([]string{}, 0, pagination.Params{Page: 1, PageSize: 20})
	assert.Equal(t, 0, res4.TotalPages)
	assert.False(t, res4.HasNext)
	assert.False(t, res4.HasPrev)
}

func TestCursor_EncodeAndDecode(t *testing.T) {
	// Case 1: Encode and decode ID only
	token1 := pagination.EncodeCursor(12345)
	assert.NotEmpty(t, token1)

	decoded1, err := pagination.DecodeCursor(token1)
	require.NoError(t, err)
	assert.Equal(t, "12345", decoded1.ID)
	assert.Nil(t, decoded1.CreatedAt)

	// Case 2: Encode and decode composite ID + Timestamp
	fixedTime := time.Date(2026, 9, 10, 10, 30, 0, 0, time.UTC)
	token2 := pagination.EncodeCursor("user-uuid-99", fixedTime)
	assert.NotEmpty(t, token2)

	decoded2, err := pagination.DecodeCursor(token2)
	require.NoError(t, err)
	assert.Equal(t, "user-uuid-99", decoded2.ID)
	require.NotNil(t, decoded2.CreatedAt)
	assert.True(t, fixedTime.Equal(*decoded2.CreatedAt))

	// Case 3: Empty token returns nil, nil
	decodedEmpty, err := pagination.DecodeCursor("")
	require.NoError(t, err)
	assert.Nil(t, decodedEmpty)

	// Case 4: Invalid token returns ErrInvalidCursor
	_, err = pagination.DecodeCursor("invalid-non-base64-!@#$%^")
	assert.ErrorIs(t, err, pagination.ErrInvalidCursor)
}

func TestNewCursorResult(t *testing.T) {
	type item struct {
		ID int
	}

	// 3 items fetched, limit=2, hasMore=true
	rawItems := []item{{ID: 1}, {ID: 2}, {ID: 3}}
	res := pagination.NewCursorResult(rawItems, 2, true, func(it item) string {
		return pagination.EncodeCursor(it.ID)
	})

	// Must be sliced to limit (2 items)
	assert.Len(t, res.Items, 2)
	assert.Equal(t, 1, res.Items[0].ID)
	assert.Equal(t, 2, res.Items[1].ID)
	assert.True(t, res.HasNext)
	assert.True(t, res.HasPrev)
	assert.NotEmpty(t, res.NextCursor)
	assert.NotEmpty(t, res.PrevCursor)
}

func TestParseSort_And_SQLWhitelistSanitizer(t *testing.T) {
	whitelist := map[string]string{
		"id":         "users.id",
		"name":       "users.name",
		"created_at": "users.created_at",
	}

	// Case 1: Prefixed notation with minus and plus
	sort1 := pagination.ParseSort("-created_at,+name", "id", pagination.OrderDesc)
	sql1 := sort1.SQL(whitelist)
	assert.Equal(t, "users.created_at DESC, users.name ASC", sql1)

	// Case 2: Delimited pair notation (field:desc)
	sort2 := pagination.ParseSort("name:asc,created_at:desc", "id", pagination.OrderDesc)
	sql2 := sort2.SQL(whitelist)
	assert.Equal(t, "users.name ASC, users.created_at DESC", sql2)

	// Case 3: SQL Injection Attack Prevention: unapproved columns are stripped
	injectionQuery := "-created_at,evil_column; DROP TABLE users;--,name:desc"
	sort3 := pagination.ParseSort(injectionQuery, "id", pagination.OrderDesc)
	sql3 := sort3.SQL(whitelist)
	assert.Equal(t, "users.created_at DESC, users.name DESC", sql3)
	assert.NotContains(t, sql3, "DROP TABLE")
	assert.NotContains(t, sql3, "evil_column")

	// Case 4: Empty query falls back to safe default
	sort4 := pagination.ParseSort("", "id", pagination.OrderDesc)
	sql4 := sort4.SQL(whitelist)
	assert.Equal(t, "users.id DESC", sql4)
}
