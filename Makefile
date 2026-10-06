GO_BIN ?= go
BINARY_NAME=go-gin-boilerplate
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/api/main.go
MIGRATE=$(GO_BIN) run ./cmd/migrate

.PHONY: all run build test test-cover tidy clean help swag vet \
	db\:init db\:setup db\:migrate db\:rollback db\:status db\:create db\:reset db\:push db\:create-admin

all: build

## run | Run the application
run:
	$(GO_BIN) run $(MAIN_PATH)

## build | Build the application and migration binaries
build:
	$(GO_BIN) build -o $(BINARY_PATH) $(MAIN_PATH)
	$(GO_BIN) build -o ./bin/migrate ./cmd/migrate

## test | Run all tests with the race detector
test:
	$(GO_BIN) test -race ./...

## test-cover | Run tests with coverage
test-cover:
	$(GO_BIN) test -coverprofile=coverage.out ./...
	$(GO_BIN) tool cover -html=coverage.out

## tidy | Tidy go modules
tidy:
	$(GO_BIN) mod tidy

## swag | Generate swagger documentation
swag:
	$(GO_BIN) run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d ./cmd/api,./internal/modules,./internal/httpx,./internal/pkg -o ./docs

## vet | Run go vet
vet:
	$(GO_BIN) vet ./...

# ---------------------------------------------------------------------------
# Database (similar to Prisma: migrate deploy / migrate reset / db push)
# Migrations are never applied when the app starts; run them explicitly.
# ---------------------------------------------------------------------------

## db:init | Create the database (DB_NAME) if it does not exist
db\:init:
	@$(MIGRATE) init

## db:setup | Create the database if needed and apply all migrations
db\:setup: db\:init db\:migrate

## db:migrate | Apply all pending migrations
db\:migrate:
	@$(MIGRATE) up

## db:rollback | Roll back the last applied migration
db\:rollback:
	@$(MIGRATE) down

## db:status | Show applied and pending migrations
db\:status:
	@$(MIGRATE) status

## db:create | Create a new SQL migration, e.g. make db:create name=add_posts_table
db\:create:
	@test -n "$(name)" || (echo "usage: make db:create name=<migration_name>" && exit 1)
	@$(MIGRATE) create $(name)

## db:reset | Drop all tables and re-apply every migration (dev only, add force=1 to skip prompt)
db\:reset:
	@$(MIGRATE) $(if $(force),-force) reset

## db:push | Sync schema from GORM models without a migration file (dev only, prototyping)
db\:push:
	@$(MIGRATE) push

## db:create-admin | Create an admin user, e.g. make db:create-admin email=admin@example.com name="Admin" (password is prompted)
db\:create-admin:
	@test -n "$(email)" || (echo 'usage: make db:create-admin email=<email> [name="<name>"]' && exit 1)
	@$(MIGRATE) create-admin -email "$(email)" $(if $(name),-name "$(name)")

## clean | Clean build artifacts
clean:
	rm -rf ./bin
	rm -f coverage.out

## help | Show this help message
help:
	@echo "Usage: make [target]"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s '|' | sed -e 's/^/  /'
