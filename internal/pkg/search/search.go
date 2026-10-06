// Package search builds a case-insensitive "contains" filter over several columns.
//
//	db.Scopes(search.Scope(q.Search, "name", "email")).Find(&items)
//
// produces: WHERE ("name" ILIKE '%term%' OR "email" ILIKE '%term%').
// Column names must come from code, never from the client; the term is escaped,
// so "%" and "_" typed by the client are matched literally. Uses ILIKE (PostgreSQL).
package search

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Scope returns a GORM scope filtering rows where any of columns contains term.
// An empty term returns the query unchanged.
func Scope(term string, columns ...string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		term = strings.TrimSpace(term)
		if term == "" || len(columns) == 0 {
			return db
		}

		pattern := "%" + likeEscaper.Replace(term) + "%"
		conditions := make([]clause.Expression, len(columns))
		for i, column := range columns {
			conditions[i] = clause.Expr{SQL: "? ILIKE ?", Vars: []any{clause.Column{Name: column}, pattern}}
		}

		if len(conditions) == 1 {
			return db.Where(conditions[0])
		}
		return db.Where(clause.Or(conditions...))
	}
}
