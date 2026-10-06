package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/pagination"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newDryRunRepository returns a repository whose queries are built but never sent to a database
func newDryRunRepository(t *testing.T) *Repository {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=invalid"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return NewRepository(db)
}

// listSQL returns the SELECT (with pagination) and COUNT statements FindAll runs for q
func listSQL(t *testing.T, q ListQuery) (selectSQL, countSQL string, vars []any) {
	t.Helper()
	repo := newDryRunRepository(t)

	db, err := repo.listQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("listQuery(%+v): %v", q, err)
	}
	base := db.Session(&gorm.Session{})

	var total int64
	countSQL = base.Model(&User{}).Count(&total).Statement.SQL.String()
	stmt := base.Scopes(q.Query.Scope()).Find(&[]User{}).Statement
	return stmt.SQL.String(), countSQL, stmt.Vars
}

func TestFindAllSQL(t *testing.T) {
	tests := []struct {
		name       string
		query      ListQuery
		wantSelect string
		wantVars   []any
	}{
		{
			// Newest first, with id as tiebreaker so rows with the same created_at never repeat across pages
			name:       "defaults",
			query:      ListQuery{},
			wantSelect: `SELECT * FROM "users" WHERE "users"."deleted_at" IS NULL ORDER BY "created_at" DESC,"users"."id" LIMIT $1`,
			wantVars:   []any{10},
		},
		{
			name:       "sort, search and page",
			query:      ListQuery{Query: pagination.Query{Page: 3, PerPage: 5}, Sort: "name,-createdAt", Search: "bob"},
			wantSelect: `SELECT * FROM "users" WHERE ("name" ILIKE $1 OR "email" ILIKE $2) AND "users"."deleted_at" IS NULL ORDER BY "name","created_at" DESC,"users"."id" LIMIT $3 OFFSET $4`,
			wantVars:   []any{"%bob%", "%bob%", 5, 10},
		},
		{
			// A sort without any field falls back to the default order
			name:       "empty sort fields",
			query:      ListQuery{Sort: ",,,"},
			wantSelect: `SELECT * FROM "users" WHERE "users"."deleted_at" IS NULL ORDER BY "created_at" DESC,"users"."id" LIMIT $1`,
			wantVars:   []any{10},
		},
		{
			// % and _ typed by the client are matched literally
			name:       "search escapes wildcards",
			query:      ListQuery{Search: "50%_off"},
			wantSelect: `SELECT * FROM "users" WHERE ("name" ILIKE $1 OR "email" ILIKE $2) AND "users"."deleted_at" IS NULL ORDER BY "created_at" DESC,"users"."id" LIMIT $3`,
			wantVars:   []any{`%50\%\_off%`, `%50\%\_off%`, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectSQL, countSQL, vars := listSQL(t, tt.query)

			if selectSQL != tt.wantSelect {
				t.Errorf("select SQL\n got: %s\nwant: %s", selectSQL, tt.wantSelect)
			}
			if len(vars) != len(tt.wantVars) {
				t.Fatalf("vars = %v, want %v", vars, tt.wantVars)
			}
			for i := range vars {
				if vars[i] != tt.wantVars[i] {
					t.Errorf("var %d = %v, want %v", i, vars[i], tt.wantVars[i])
				}
			}

			// PostgreSQL rejects "SELECT count(*) ... ORDER BY", which happens if the
			// order is added through db.Scopes instead of sorting.Apply
			if !strings.HasPrefix(countSQL, `SELECT count(*) FROM "users" WHERE `) {
				t.Errorf("count SQL = %s", countSQL)
			}
			for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET"} {
				if strings.Contains(countSQL, forbidden) {
					t.Errorf("count SQL must not contain %s: %s", forbidden, countSQL)
				}
			}
		})
	}
}

func TestFindAllRejectsUnknownSortField(t *testing.T) {
	repo := newDryRunRepository(t)

	for _, sort := range []string{"password", "name;DROP TABLE users", "created_at"} {
		_, _, err := repo.FindAll(context.Background(), ListQuery{Sort: sort})
		var appErr *apperror.Error
		if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation || appErr.Fields["sort"] == "" {
			t.Errorf("sort %q: want a validation error on sort, got %v", sort, err)
		}
	}
}
