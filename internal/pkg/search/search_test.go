package search

import (
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type item struct{ ID int }

func TestScope(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=invalid"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		term     string
		columns  []string
		wantSQL  string
		wantVars []any
	}{
		{"", []string{"name"}, `SELECT * FROM "items"`, nil},
		{"bob", []string{"name"}, `SELECT * FROM "items" WHERE "name" ILIKE $1`, []any{"%bob%"}},
		{" bob ", []string{"name", "email"}, `SELECT * FROM "items" WHERE ("name" ILIKE $1 OR "email" ILIKE $2)`, []any{"%bob%", "%bob%"}},
		{`50%_off\`, []string{"name"}, `SELECT * FROM "items" WHERE "name" ILIKE $1`, []any{`%50\%\_off\\%`}},
	}

	for _, tt := range tests {
		stmt := db.Scopes(Scope(tt.term, tt.columns...)).Find(&[]item{}).Statement
		if got := stmt.SQL.String(); got != tt.wantSQL {
			t.Errorf("Scope(%q) SQL\n got: %s\nwant: %s", tt.term, got, tt.wantSQL)
		}
		if len(stmt.Vars) != len(tt.wantVars) {
			t.Fatalf("Scope(%q) vars = %v, want %v", tt.term, stmt.Vars, tt.wantVars)
		}
		for i := range tt.wantVars {
			if stmt.Vars[i] != tt.wantVars[i] {
				t.Errorf("Scope(%q) var %d = %v, want %v", tt.term, i, stmt.Vars[i], tt.wantVars[i])
			}
		}
	}
}
