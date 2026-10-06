package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/config"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/termcolor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// maintenanceDB is the default database every PostgreSQL server has.
// It is used to create the application database when it does not exist yet.
const maintenanceDB = "postgres"

// Connect establishes a connection to the PostgreSQL database.
// It does not touch the schema; see NewMigrator and `make db:migrate`.
func Connect(cfg *config.Config) *gorm.DB {
	db, err := Open(cfg, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")

	return db
}

// Open connects to the given database on the configured PostgreSQL server
func Open(cfg *config.Config, dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, dbName, cfg.DBPort, cfg.DBSSLMode)

	// Logging every SQL query is useful in development but noisy and may leak data in production
	logLevel := logger.Info
	if cfg.IsProduction() {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Silent while connecting: connection errors are returned to the caller instead of logged twice
		Logger: logger.Default.LogMode(logger.Silent),
		// Translate driver errors (e.g. unique violation) into gorm errors such as gorm.ErrDuplicatedKey
		TranslateError: true,
	})
	if err != nil {
		if IsDatabaseMissing(err) {
			return nil, fmt.Errorf("database %q does not exist, create it with `make db:init`: %w", dbName, err)
		}
		return nil, err
	}

	db.Logger = NewLogger(logLevel, cfg.PrettyLogs() && termcolor.Enabled(os.Stdout))

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

// NewLogger returns a GORM logger. "Record not found" is not logged: callers handle it
// (e.g. checking whether an email is taken), so logging it would only add false errors.
func NewLogger(level logger.LogLevel, colorful bool) logger.Interface {
	return logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		Colorful:                  colorful,
	})
}

// IsDatabaseMissing reports whether err means the target database does not exist
func IsDatabaseMissing(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "3D000" // invalid_catalog_name
}

// CreateDatabase creates cfg.DBName if it does not exist yet. It reports whether the database was created.
// The configured DB_USER needs the CREATEDB privilege.
func CreateDatabase(ctx context.Context, cfg *config.Config) (bool, error) {
	db, err := Open(cfg, maintenanceDB)
	if err != nil {
		return false, fmt.Errorf("connect to %q database: %w", maintenanceDB, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDB.Close()

	var exists bool
	if err := sqlDB.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	// CREATE DATABASE does not accept parameters, so the name is quoted as an identifier
	if _, err := sqlDB.ExecContext(ctx, "CREATE DATABASE "+pgx.Identifier{cfg.DBName}.Sanitize()); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" { // insufficient_privilege
			return false, fmt.Errorf("user %q is not allowed to create databases; grant it with `ALTER ROLE %s CREATEDB;` "+
				"or ask a database admin to create %q: %w", cfg.DBUser, pgx.Identifier{cfg.DBUser}.Sanitize(), cfg.DBName, err)
		}
		return false, err
	}
	return true, nil
}
