// Command migrate manages the database schema. Run it through the Makefile:
//
//	make db:init                    create the database (DB_NAME) if it does not exist
//	make db:migrate                 apply all pending migrations
//	make db:rollback                roll back the last applied migration
//	make db:status                  show applied and pending migrations
//	make db:create name=add_posts   create a new SQL migration file
//	make db:reset                   drop everything and re-apply all migrations (dev only)
//	make db:push                    sync the schema from GORM models without a migration (dev only)
//	make db:create-admin email=admin@example.com name="Admin"
//	                                create an admin user (password is prompted or read from ADMIN_PASSWORD)
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/config"
	"github.com/gibran/go-gin-boilerplate/internal/database"
	"github.com/gibran/go-gin-boilerplate/internal/modules"
	"github.com/gibran/go-gin-boilerplate/internal/modules/auth"
	"github.com/gibran/go-gin-boilerplate/internal/modules/user"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/ratelimit"
	"github.com/pressly/goose/v3"
	"golang.org/x/term"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const usage = `Usage: migrate [-force] <command> [args]

Commands:
  init           Create the database (DB_NAME) if it does not exist
  up             Apply all pending migrations
  down           Roll back the last applied migration
  status         Show applied and pending migrations
  create <name>  Create a new SQL migration file in ` + database.MigrationsDir + `
  reset          Drop all tables and re-apply every migration (not allowed in production)
  push           Sync the schema from GORM models without creating a migration (not allowed in production)
  create-admin -email <email> -name <name>
                 Create an admin user. The password is read from ADMIN_PASSWORD,
                 or prompted (hidden) when it is not set.

Flags:
  -force         Skip the confirmation prompt of reset
`

func main() {
	force := flag.Bool("force", false, "skip confirmation prompts")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(context.Background(), flag.Arg(0), flag.Args()[1:], *force); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, command string, args []string, force bool) error {
	// create only writes a file, so it does not need a database connection
	if command == "create" {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return errors.New("usage: migrate create <name>")
		}
		goose.SetSequential(true)
		return goose.Create(nil, database.MigrationsDir, args[0], "sql")
	}

	cfg := config.Load()

	// init connects to the server's maintenance database, since DB_NAME may not exist yet
	if command == "init" {
		return initDatabase(ctx, cfg)
	}

	conn, err := database.Open(cfg, cfg.DBName)
	if err != nil {
		return err
	}
	// Keep CLI output readable: only log slow queries and errors, not every statement
	db := conn.Session(&gorm.Session{Logger: database.NewLogger(logger.Warn, true)})
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	switch command {
	case "up":
		return up(ctx, db)
	case "down":
		return down(ctx, db)
	case "status":
		return status(ctx, db)
	case "reset":
		if cfg.IsProduction() {
			return errors.New("reset is not allowed when APP_ENV=production")
		}
		if !force && !confirm(fmt.Sprintf("This will DROP ALL DATA in database %q. Continue?", cfg.DBName)) {
			return errors.New("aborted")
		}
		return reset(ctx, db)
	case "push":
		if cfg.IsProduction() {
			return errors.New("push is not allowed when APP_ENV=production, use migrations instead")
		}
		return push(db)
	case "create-admin":
		return createAdmin(ctx, db, args)
	default:
		flag.Usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func initDatabase(ctx context.Context, cfg *config.Config) error {
	created, err := database.CreateDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("created database %q\n", cfg.DBName)
	} else {
		fmt.Printf("database %q already exists\n", cfg.DBName)
	}
	return nil
}

func up(ctx context.Context, db *gorm.DB) error {
	migrator, err := database.NewMigrator(db)
	if err != nil {
		return err
	}

	results, err := migrator.Up(ctx)
	for _, r := range results {
		fmt.Printf("applied  %s (%s)\n", r.Source.Path, r.Duration)
	}
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println("no pending migrations, database is up to date")
	}
	return nil
}

func down(ctx context.Context, db *gorm.DB) error {
	migrator, err := database.NewMigrator(db)
	if err != nil {
		return err
	}

	result, err := migrator.Down(ctx)
	if errors.Is(err, goose.ErrNoNextVersion) {
		fmt.Println("no applied migrations to roll back")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Printf("rolled back  %s (%s)\n", result.Source.Path, result.Duration)
	return nil
}

func status(ctx context.Context, db *gorm.DB) error {
	migrator, err := database.NewMigrator(db)
	if err != nil {
		return err
	}

	statuses, err := migrator.Status(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VERSION\tSTATE\tAPPLIED AT\tFILE")
	for _, s := range statuses {
		appliedAt := "-"
		if s.State == goose.StateApplied {
			appliedAt = s.AppliedAt.Local().Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Source.Version, s.State, appliedAt, s.Source.Path)
	}
	return w.Flush()
}

// reset drops the whole public schema (including tables created by push) and re-applies all migrations
func reset(ctx context.Context, db *gorm.DB) error {
	if err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;").Error; err != nil {
		return fmt.Errorf("drop schema: %w", err)
	}
	fmt.Println("dropped all tables")
	return up(ctx, db)
}

// push syncs tables with the GORM models, similar to `prisma db push`.
// It is meant for quick prototyping: it does not create a migration file,
// so write one with `make db:create` before committing schema changes.
func push(db *gorm.DB) error {
	if err := db.AutoMigrate(modules.Models()...); err != nil {
		return err
	}
	fmt.Println("schema pushed from models")
	fmt.Println("note: no migration file was created; run `make db:create name=...` before committing")
	return nil
}

// createAdmin creates a user with the admin role, so the admin-only endpoints can be used
// on a fresh database. It goes through auth.Service, so the same validation and password
// hashing as the register endpoint apply.
func createAdmin(ctx context.Context, db *gorm.DB, args []string) error {
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	email := fs.String("email", "", "admin email (required)")
	name := fs.String("name", "Admin", "admin display name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*email) == "" {
		return errors.New("usage: migrate create-admin -email <email> [-name <name>]")
	}

	password, err := adminPassword()
	if err != nil {
		return err
	}

	// Token settings and the login limiter are not used to create a user
	svc := auth.NewService(user.NewRepository(db), ratelimit.NewFailureCounter(1, time.Minute), auth.Config{})
	u, err := svc.CreateUser(ctx, *name, *email, password, user.RoleAdmin)
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && len(appErr.Fields) > 0 {
			for field, msg := range appErr.Fields {
				fmt.Fprintf(os.Stderr, "  %s: %s\n", field, msg)
			}
		}
		return err
	}

	fmt.Printf("created admin %s (%s)\n", u.Email, u.ID)
	return nil
}

// adminPassword reads the password from ADMIN_PASSWORD, or prompts for it twice without echo.
// It is never taken from a command-line flag, which would end up in the shell history.
func adminPassword() (string, error) {
	if password, ok := os.LookupEnv("ADMIN_PASSWORD"); ok {
		return password, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", errors.New("set ADMIN_PASSWORD or run the command in a terminal to be prompted")
	}

	fmt.Print("Password: ")
	first, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	fmt.Print("Confirm password: ")
	second, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", errors.New("passwords do not match")
	}
	return string(first), nil
}

func confirm(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
