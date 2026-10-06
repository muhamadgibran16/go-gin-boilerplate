// Package pagination provides page/perPage query binding, GORM pagination and
// response metadata for list endpoints.
//
// Typical usage in a module:
//
//	// handler
//	var q pagination.Query
//	if err := c.ShouldBindQuery(&q); err != nil { ... }
//	items, total, err := h.service.List(ctx, q)
//	httpx.SuccessPaginated(c, "...", items, pagination.NewMeta(q, len(items), total))
//
//	// repository: filters with Where/Scopes, ordering with Order or sorting.Apply
//	return pagination.Find[Post](r.db.WithContext(ctx).Order("created_at desc"), q)
//
// Do not add ORDER BY through db.Scopes(): GORM would keep it in the count query,
// which PostgreSQL rejects.
package pagination

import "gorm.io/gorm"

// Defaults used when page or perPage is missing, empty or 0. Change them only here.
const (
	DefaultPage    = 1
	DefaultPerPage = 10
)

// Query is the page/perPage query string of a list endpoint.
// Bind it with c.ShouldBindQuery, or embed it in a module's own query struct.
//
// The binding tags only validate client input (perPage over 100 is rejected with 400).
// Defaults are applied by Normalize, which every method below calls.
type Query struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"perPage" binding:"omitempty,min=1,max=100"`
}

// Normalize returns q with defaults applied to a missing (zero or negative) page or perPage
func (q Query) Normalize() Query {
	if q.Page < 1 {
		q.Page = DefaultPage
	}
	if q.PerPage < 1 {
		q.PerPage = DefaultPerPage
	}
	return q
}

// Offset returns the number of rows to skip
func (q Query) Offset() int {
	q = q.Normalize()
	return (q.Page - 1) * q.PerPage
}

// Limit returns the maximum number of rows to return
func (q Query) Limit() int {
	return q.Normalize().PerPage
}

// Scope is a GORM scope applying LIMIT/OFFSET, e.g. db.Scopes(q.Scope()).Find(&items)
func (q Query) Scope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(q.Limit()).Offset(q.Offset())
	}
}

// Find counts all rows matching db and returns the requested page of them.
// db carries the filters and ordering, e.g. r.db.WithContext(ctx).Where(...).Order(...).
func Find[T any](db *gorm.DB, q Query) ([]T, int64, error) {
	// A new session so the count and the select do not share statement state
	base := db.Session(&gorm.Session{})

	var total int64
	if err := base.Model(new(T)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	items := make([]T, 0)
	if total == 0 {
		return items, 0, nil
	}
	if err := base.Scopes(q.Scope()).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Meta is the pagination metadata returned with list responses
type Meta struct {
	CurrentPage      int `json:"currentPage"`
	PerPage          int `json:"perPage"`
	TotalCurrentPage int `json:"totalCurrentPage"`
	TotalPage        int `json:"totalPage"`
	TotalData        int `json:"totalData"`
}

// NewMeta builds the metadata for a page of count items out of total
func NewMeta(q Query, count int, total int64) Meta {
	q = q.Normalize()
	return Meta{
		CurrentPage:      q.Page,
		PerPage:          q.PerPage,
		TotalCurrentPage: count,
		TotalPage:        TotalPages(total, q.PerPage),
		TotalData:        int(total),
	}
}

// TotalPages returns how many pages are needed to show total items
func TotalPages(total int64, perPage int) int {
	if perPage <= 0 || total <= 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}
