// Package sorting parses a client "sort" query parameter into a safe GORM ORDER BY.
//
// The parameter is a comma-separated list of fields, "-" prefix for descending:
//
//	?sort=-createdAt,name
//
// Only fields present in the allowed map can be used, so clients can never inject
// arbitrary column names:
//
//	allowed := map[string]string{"name": "name", "createdAt": "created_at"}
//	db, err := sorting.Apply(db, q.Sort, allowed, "-createdAt")
//
// The primary key "id" is always appended as the last sort column (unless already
// sorted by it). Without a unique tiebreaker, rows with equal values (same name, same
// created_at) have no guaranteed order, so pages could repeat or skip rows.
//
// Apply adds the ORDER BY immediately instead of returning a GORM scope on purpose:
// GORM drops an immediate ORDER BY when counting, but keeps one added through
// db.Scopes(), which makes "SELECT count(*) ... ORDER BY" fail on PostgreSQL
// (e.g. in pagination.Find).
package sorting

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TiebreakColumn is the unique column appended to every ORDER BY for stable pagination.
// Every table sorted with this package must have it.
const TiebreakColumn = "id"

// Parse converts a sort parameter into ORDER BY columns, ending with TiebreakColumn.
// A sort without any field (empty, or only commas) uses fallback.
// Unknown fields return a validation error listing the allowed fields.
func Parse(sort string, allowed map[string]string, fallback string) ([]clause.OrderByColumn, error) {
	columns, err := parseFields(sort, allowed)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		if columns, err = parseFields(fallback, allowed); err != nil {
			return nil, err
		}
	}

	for _, c := range columns {
		if c.Column.Name == TiebreakColumn {
			return columns, nil
		}
	}
	// Table-qualified so it stays unambiguous when the query joins other tables
	return append(columns, clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: TiebreakColumn},
	}), nil
}

func parseFields(sort string, allowed map[string]string) ([]clause.OrderByColumn, error) {
	var columns []clause.OrderByColumn
	seen := map[string]bool{}
	for _, part := range strings.Split(sort, ",") {
		field := strings.TrimSpace(part)
		if field == "" {
			continue
		}

		desc := strings.HasPrefix(field, "-")
		field = strings.TrimPrefix(field, "-")

		column, ok := allowed[field]
		if !ok {
			return nil, apperror.Validation(map[string]string{
				"sort": fmt.Sprintf("cannot sort by %q, allowed fields: %s", field, allowedFields(allowed)),
			})
		}
		if seen[column] {
			continue
		}
		seen[column] = true

		columns = append(columns, clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: desc})
	}
	return columns, nil
}

// Apply parses sort and returns db with the ORDER BY added
func Apply(db *gorm.DB, sort string, allowed map[string]string, fallback string) (*gorm.DB, error) {
	columns, err := Parse(sort, allowed, fallback)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return db, nil
	}
	return db.Order(clause.OrderBy{Columns: columns}), nil
}

func allowedFields(allowed map[string]string) string {
	fields := make([]string, 0, len(allowed))
	for f := range allowed {
		fields = append(fields, f)
	}
	slices.Sort(fields)
	return strings.Join(fields, ", ")
}
