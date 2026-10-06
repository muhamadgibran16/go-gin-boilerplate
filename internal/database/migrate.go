package database

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// MigrationsDir is where SQL migration files live, relative to the project root
const MigrationsDir = "internal/database/migrations"

//go:embed migrations/*.sql
var migrationsFS embed.FS

// NewMigrator returns a goose provider for the SQL migrations embedded in the binary.
// Migrations are never applied automatically; run them with `make db:migrate`.
func NewMigrator(db *gorm.DB) (*goose.Provider, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	return goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
}
