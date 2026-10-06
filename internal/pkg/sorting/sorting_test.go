package sorting

import (
	"errors"
	"testing"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type item struct{ ID int }

var allowed = map[string]string{"id": "id", "name": "name", "createdAt": "created_at"}

// dryRunDB builds SQL without connecting to a database
func dryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=invalid"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestApply(t *testing.T) {
	tests := []struct {
		sort, want string
	}{
		{"", `SELECT * FROM "items" ORDER BY "created_at" DESC,"items"."id"`},
		{",, ,", `SELECT * FROM "items" ORDER BY "created_at" DESC,"items"."id"`}, // no field: fallback
		{"name", `SELECT * FROM "items" ORDER BY "name","items"."id"`},
		{"-createdAt, name", `SELECT * FROM "items" ORDER BY "created_at" DESC,"name","items"."id"`},
		{"name,-name", `SELECT * FROM "items" ORDER BY "name","items"."id"`}, // duplicates ignored
		{"-id", `SELECT * FROM "items" ORDER BY "id" DESC`},                  // already unique: no tiebreaker
	}

	for _, tt := range tests {
		db, err := Apply(dryRunDB(t), tt.sort, allowed, "-createdAt")
		if err != nil {
			t.Fatalf("Apply(%q): %v", tt.sort, err)
		}
		got := db.Find(&[]item{}).Statement.SQL.String()
		if got != tt.want {
			t.Errorf("Apply(%q)\n got: %s\nwant: %s", tt.sort, got, tt.want)
		}
	}
}

func TestApplyRejectsUnknownField(t *testing.T) {
	for _, sort := range []string{"password", "name;DROP TABLE users", "-created_at"} {
		_, err := Apply(dryRunDB(t), sort, allowed, "")
		var appErr *apperror.Error
		if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation || appErr.Fields["sort"] == "" {
			t.Errorf("Apply(%q) must return a validation error, got %v", sort, err)
		}
	}
}

// The ORDER BY must not leak into the count query used by pagination
func TestApplyIsDroppedByCount(t *testing.T) {
	db, err := Apply(dryRunDB(t), "-createdAt", allowed, "")
	if err != nil {
		t.Fatal(err)
	}
	var n int64
	got := db.Model(&item{}).Count(&n).Statement.SQL.String()
	if want := `SELECT count(*) FROM "items"`; got != want {
		t.Fatalf("count SQL\n got: %s\nwant: %s", got, want)
	}
}
