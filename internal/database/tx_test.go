package database

import (
	"context"
	"errors"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Integration test: runs only when TEST_DATABASE_DSN points to a PostgreSQL database, e.g.
// TEST_DATABASE_DSN="host=localhost user=postgres password=postgres dbname=app_test sslmode=disable" go test ./internal/database/
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TEMP TABLE tx_items (name text)").Error; err != nil {
		t.Fatal(err)
	}
	// Temp tables are per connection; pin the pool to one connection
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	return db
}

func count(t *testing.T, db *gorm.DB) int64 {
	var n int64
	if err := db.Table("tx_items").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func insert(ctx context.Context, db *gorm.DB, name string) error {
	return Conn(ctx, db).Exec("INSERT INTO tx_items (name) VALUES (?)", name).Error
}

func TestWithinTx(t *testing.T) {
	db := testDB(t)
	tx := NewTransactor(db)
	ctx := context.Background()

	err := tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := insert(ctx, db, "a"); err != nil {
			return err
		}
		return insert(ctx, db, "b")
	})
	if err != nil || count(t, db) != 2 {
		t.Fatalf("commit: err=%v count=%d", err, count(t, db))
	}

	boom := errors.New("boom")
	err = tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := insert(ctx, db, "c"); err != nil {
			return err
		}
		// nested call joins the outer transaction
		return tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := insert(ctx, db, "d"); err != nil {
				return err
			}
			return boom
		})
	})
	if !errors.Is(err, boom) || count(t, db) != 2 {
		t.Fatalf("rollback: err=%v count=%d", err, count(t, db))
	}
}
