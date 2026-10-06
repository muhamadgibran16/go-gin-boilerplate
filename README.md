# Go Gin Boilerplate

Go REST API boilerplate using [Gin](https://github.com/gin-gonic/gin), organized by feature modules,
with JWT authentication, role-based access, versioned migrations and security defaults.

## Main Features

- **Authentication**: register, login, refresh, logout with JWT. Access and refresh tokens are
  distinct (a refresh token cannot be used as a Bearer token), and refresh reloads the user from the
  database so deleted or demoted users cannot get new tokens.
- **Users**: admin-only CRUD with pagination, sorting and search
  (`?page=1&perPage=10&sort=-createdAt,name&search=bob`). The first admin is created with `make db:create-admin`.
- **Security**:
  - bcrypt password hashing, UUID primary keys (no ID enumeration).
  - Layered rate limiting with keys clients cannot change:
    - per IP for every request (300/min, flood protection);
    - per IP for register and login (20/min);
    - per user ID from the JWT for authenticated endpoints (100/min), so users behind one IP do not share a quota;
    - per email for failed logins (5 per 15 min, then the email is locked even for the right password),
      against brute force from many IPs.
  - Internal errors are logged but never shown to clients; request bodies are limited to 1 MiB.
  - Security headers, CORS allow-list, trusted proxies, and a production check that refuses a weak `JWT_SECRET`.
- **Database**: PostgreSQL with GORM and versioned SQL migrations ([goose](https://github.com/pressly/goose)),
  run explicitly with Prisma-like `make db:*` commands (never on startup).
- **Developer experience**: consistent JSON responses, validation messages using JSON field names,
  request-scoped logs with `request_id` / `user_id`, Swagger docs, tests, Docker.

Rate limits are constants in `internal/app/app.go`. Counters are kept in memory: use a shared store such as
Redis when running several instances. Set `TRUSTED_PROXIES` when running behind a load balancer.

## Tech Stack

| Package                                                     | Version | Usage                       |
| ----------------------------------------------------------- | ------- | --------------------------- |
| [gin-gonic/gin](https://github.com/gin-gonic/gin)           | v1.11   | HTTP framework              |
| [gorm.io/gorm](https://github.com/go-gorm/gorm)             | v1.31   | ORM (PostgreSQL via pgx)    |
| [pressly/goose](https://github.com/pressly/goose)           | v3.27   | SQL migrations              |
| [golang-jwt/jwt](https://github.com/golang-jwt/jwt)         | v5.3    | JWT                         |
| [ulule/limiter](https://github.com/ulule/limiter)           | v3.11   | Rate limiting               |
| [uber-go/zap](https://github.com/uber-go/zap)               | v1.27   | Structured logging          |
| [swaggo/swag](https://github.com/swaggo/swag)               | v1.16   | Swagger docs generation     |
| [google/uuid](https://github.com/google/uuid)               | v1.6    | UUID primary keys           |

## Project Structure

Code is organized **by feature**: each module in `internal/modules` owns its handler, service,
repository, model and DTOs. `internal/app` only wires everything together.

```
go-gin-boilerplate/
├── cmd/
│   ├── api/main.go                 # API server entry point
│   └── migrate/main.go             # Migration CLI (used by `make db:*`)
├── internal/
│   ├── app/                        # Composition root: dependency wiring, routes, http.Server
│   ├── config/                     # Environment configuration
│   ├── database/
│   │   ├── database.go             # GORM Postgres connection, CREATE DATABASE
│   │   ├── migrate.go              # Migration provider (embedded SQL files)
│   │   ├── tx.go                   # Transactions carried in context (Transactor, Conn)
│   │   └── migrations/             # Versioned SQL migrations (goose)
│   ├── httpx/                      # Shared HTTP layer: JSON responses, binding & validation
│   │                               # messages, ErrorHandler, middleware (CORS, logger, recovery,
│   │                               # request ID, rate limit, security headers), current user
│   ├── pkg/                        # Generic utilities shared by many modules
│   │   ├── apperror/               # Typed application errors (not found, conflict, ...)
│   │   ├── ctxlog/                 # Request-scoped logger carried in context.Context
│   │   ├── pagination/             # page/perPage binding, pagination.Find[T], response meta
│   │   ├── ratelimit/              # Failure counter (failed logins per email)
│   │   ├── search/                 # Case-insensitive search over several columns
│   │   └── sorting/                # Safe ?sort=-createdAt,name with a field whitelist
│   └── modules/
│       ├── modules.go              # Models() registry used by `make db:push`
│       ├── auth/                   # Register/login/refresh, JWT, password hashing,
│       │                           # auth middleware & RequireRoles
│       ├── user/                   # User model, repository, service, handler, DTOs
│       └── health/                 # Liveness and readiness checks
├── docs/                           # Generated Swagger docs (`make swag`)
├── .env.example
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

Dependency direction: `cmd → app → modules → httpx / database / config / pkg`.
Modules never import `app`; `auth` may import `user` (it needs the user model), but not the other way around.

### Where to put code

| Code                                                   | Location                         |
| ------------------------------------------------------ | -------------------------------- |
| Used by a single module                                | Inside that module               |
| HTTP related, shared (responses, middleware)           | `internal/httpx`                 |
| Database related, shared (connection, migrations)      | `internal/database`              |
| Generic utility shared by many modules                 | `internal/pkg/<name>`            |

Rules for `internal/pkg`:

- One sub-package per concern, named after what it provides (`pagination`, `password`, `stringx`, ...). No catch-all `utils` package.
- It must not import `internal/modules` or `internal/app`; modules import `pkg`, never the other way around.
- Start a helper inside the module that needs it and move it to `internal/pkg` once a second module needs it.

### Writing handlers and services

**Errors.** Services return `apperror` errors for expected failures; anything else is an internal error.
Handlers just pass errors to `httpx.Abort`, and `httpx.ErrorHandler` writes the response:

```go
// service
var ErrNotFound = apperror.NotFound("post not found")

// handler
id, ok := httpx.ParamUUID(c, "id")          // 400 with {"errors":{"id":"..."}} if invalid
if !ok {
	return
}
var req UpdateRequest
if !httpx.BindJSON(c, &req) {                // 400 with per-field messages using JSON names
	return
}
post, err := h.service.Update(c.Request.Context(), id, req)
if err != nil {
	httpx.Abort(c, err)                      // apperror -> its status; other errors -> logged 500
	return
}
```

| apperror                     | HTTP status |
| ---------------------------- | ----------- |
| `BadRequest`, `Validation`   | 400         |
| `Unauthorized`               | 401         |
| `Forbidden`                  | 403         |
| `NotFound`                   | 404         |
| `Conflict`                   | 409         |
| `PayloadTooLarge`            | 413         |
| `TooManyRequests`            | 429         |
| `Unavailable`                | 503         |
| `Internal` / any other error | 500 (message hidden from the client, error logged) |

**List endpoints** (`?page=1&perPage=10&sort=-createdAt,name&search=bob`):

```go
// dto.go
type ListQuery struct {
	pagination.Query
	Sort   string `form:"sort" binding:"omitempty,max=200"`
	Search string `form:"search" binding:"omitempty,max=100"`
}

// repository.go
var sortFields = map[string]string{"title": "title", "createdAt": "created_at"} // whitelist

func (r *Repository) FindAll(ctx context.Context, q ListQuery) ([]Post, int64, error) {
	db, err := sorting.Apply(database.Conn(ctx, r.db), q.Sort, sortFields, "-createdAt")
	if err != nil {
		return nil, 0, err // 400 listing the allowed fields
	}
	return pagination.Find[Post](db.Scopes(search.Scope(q.Search, "title", "body")), q.Query)
}

// handler.go
httpx.SuccessPaginated(c, "Get posts successfully", NewPostResponses(posts), pagination.NewMeta(q.Query, len(posts), total))
```

Response `meta`:

```json
"meta": { "currentPage": 1, "perPage": 10, "totalCurrentPage": 10, "totalPage": 2, "totalData": 12 }
```

Add `sort` and `search` to the route's `httpx.ValidateQueryParams` whitelist. Always order with
`sorting.Apply` or `db.Order`, not through `db.Scopes`, or the count query will fail.

`sorting.Apply` always appends the table's `id` as the last sort column, so rows with equal values
(same name, same `created_at`) keep a stable order and never repeat or disappear across pages.
Defaults (`page=1`, `perPage=10`) live only in `pagination.DefaultPage` / `DefaultPerPage`;
`perPage` above 100 is rejected with 400.

Request bodies are limited to 1 MiB (`maxBodyBytes` in `internal/app/app.go`); larger requests get 413.

**Transactions.** Repositories get their handle with `database.Conn(ctx, r.db)`, so they automatically
join a transaction started by a service:

```go
err := s.tx.WithinTx(ctx, func(ctx context.Context) error { // s.tx is a *database.Transactor
	if err := s.orders.Create(ctx, order); err != nil {
		return err // rollback
	}
	return s.items.CreateMany(ctx, items)
}) // commit
```

**Logging.** `ctxlog.From(ctx)` returns a logger that already carries `request_id` (and `user_id` once
authenticated), so logs from services can be traced back to the request:

```go
ctxlog.From(ctx).Info("user registered", zap.String("registered_user_id", u.ID.String()))
```

### Adding a new module

1. Create `internal/modules/<name>/` with `model.go`, `dto.go`, `repository.go`, `service.go`, `handler.go`.
2. Register the model in `internal/modules/modules.go`.
3. Wire the repository, service and handler in `internal/app/app.go` and add routes in `internal/app/routes.go`.
4. Create the table with `make db:create name=create_<name>_table`, then `make db:migrate`.

## Running the Project

### 1. Prerequisites

- Go 1.25+
- Docker & Docker Compose

### 2. Run locally (PostgreSQL in Docker, API with Go)

```bash
cp .env.example .env
docker compose up -d db                                      # PostgreSQL only
make db:setup                                                # create the database + apply migrations
make db:create-admin email=admin@example.com name="Admin"    # first admin, password is prompted
make run                                                     # http://localhost:8080
```

### 3. Or run everything in Docker

```bash
cp .env.example .env
docker compose up -d db
docker compose run --rm api ./migrate up                                       # apply migrations
docker compose run --rm -it api ./migrate create-admin -email admin@example.com  # first admin
docker compose up -d --build api
```

Swagger UI: http://localhost:8080/swagger/index.html (development only by default).
Run `make help` to see every Make target.

## API Endpoints

### Public Endpoints

| Method | Endpoint                | Description   |
| ------ | ----------------------- | ------------- |
| GET    | `/`                     | Hello World   |
| GET    | `/health`, `/api/v1/health` | Liveness check (process is up) |
| GET    | `/health/ready`         | Readiness check (pings the database, 503 if down) |
| GET    | `/swagger/index.html`   | API Documentation UI (disabled in production unless `SWAGGER_ENABLED=true`) |
| POST   | `/api/v1/auth/register` | Register user |
| POST   | `/api/v1/auth/login`    | Login user |
| POST   | `/api/v1/auth/refresh`  | Get a new access token from a refresh token |

### Protected Endpoints (Header: `Authorization: Bearer <token>`)

| Method | Endpoint               | Description                       |
| ------ | ---------------------- | --------------------------------- |
| POST   | `/api/v1/auth/logout`  | Logout (stateless: the client deletes its tokens; the access token stays valid until it expires) |
| GET    | `/api/v1/users`        | List users (admin). Query: `page`, `perPage` (max 100), `sort` (`name`, `email`, `role`, `createdAt`, `updatedAt`, `-` for descending), `search` (name, email) |
| GET    | `/api/v1/users/:id`    | Get user detail (admin)           |
| PUT    | `/api/v1/users/:id`    | Update user (admin)               |
| DELETE | `/api/v1/users/:id`    | Delete user (admin)               |

Admins cannot change their own role or delete their own account.

### Response format

Success:

```json
{ "status": "success", "message": "Get user successfully", "data": { "id": "...", "name": "Bob", "email": "bob@example.com", "role": "user", "createdAt": "...", "updatedAt": "..." } }
```

List (paginated):

```json
{
    "status": "success",
    "message": "Get users successfully",
    "data": [ ... ],
    "meta": { "currentPage": 1, "perPage": 10, "totalCurrentPage": 10, "totalPage": 2, "totalData": 12 }
}
```

Error (`errors` only for validation errors, keyed by JSON field name):

```json
{ "status": "error", "message": "Validation failed", "errors": { "email": "email must be a valid email address" } }
```

## Database Migrations

Migrations live in `internal/database/migrations` as plain SQL files and are embedded into the binaries.
They are **never applied when the app starts**; the app only logs a warning if migrations are pending.

| Command                           | Prisma equivalent        | Description |
| --------------------------------- | ------------------------ | ----------- |
| `make db:init`                    | (done by `migrate dev`)  | Create the database `DB_NAME` if it does not exist (safe to re-run) |
| `make db:setup`                   | -                        | `db:init` + `db:migrate` in one step, for a fresh environment |
| `make db:migrate`                 | `prisma migrate deploy`  | Apply all pending migrations |
| `make db:rollback`                | -                        | Roll back the last applied migration |
| `make db:status`                  | `prisma migrate status`  | Show applied and pending migrations |
| `make db:create name=add_posts`   | `prisma migrate dev --create-only` | Create a new empty SQL migration file |
| `make db:reset`                   | `prisma migrate reset`   | Drop all tables and re-apply every migration (dev only, asks for confirmation; `force=1` skips it) |
| `make db:push`                    | `prisma db push`         | Sync the schema from GORM models (`modules.Models()`) without a migration file (dev only) |
| `make db:create-admin email=...`  | -                        | Create an admin user (`name="..."` optional). The password is prompted without echo, or read from `ADMIN_PASSWORD` for scripts/CI |

Typical workflow for a schema change:

```bash
make db:create name=add_phone_to_users   # creates internal/database/migrations/0000N_add_phone_to_users.sql
# write the SQL under "-- +goose Up" and its reverse under "-- +goose Down"
make db:migrate
make db:rollback && make db:migrate      # optional: check that Down works
```

Notes:

- `db:init` connects to the server's default `postgres` database to run `CREATE DATABASE`, so `DB_USER` needs the `CREATEDB` privilege (`ALTER ROLE <user> CREATEDB;`).
- Unlike Prisma, SQL is not generated from the models; write it by hand in the migration file.
- `db:push` is for quick prototyping only. Before committing, write a proper migration and run `make db:reset` to make sure the migrations alone produce the schema.
- Never edit a migration that has already been applied in a shared environment; add a new one instead.
- `db:reset` and `db:push` refuse to run when `APP_ENV=production`.
- In Docker, the image ships a `migrate` binary: `docker compose run --rm api ./migrate up` (also `down`, `status`).

## Testing

```bash
make test

# Integration tests (transactions) run only against a real PostgreSQL database:
TEST_DATABASE_DSN="host=localhost user=postgres password=postgres dbname=postgres sslmode=disable" \
  go test ./internal/database/
```

Tests live next to the code they cover: services are unit-tested with in-memory fake repositories
(`internal/modules/*/service_test.go`), and the shared layers have their own tests (auth middleware and JWT,
HTTP error mapping and validation, pagination/sorting/search SQL, rate limiting). When adding a module,
service tests are usually enough, since the shared layers are already covered.

## Generating Documentation

If you add new endpoints or change existing ones, update the Swagger docs:

```bash
make swag
```

## Environment Variables

See the [.env.example](.env.example) file for the full list. Notable settings:

| Variable               | Description |
| ---------------------- | ----------- |
| `APP_ENV`              | `development` or `production` (stricter checks, no Swagger, no `db:reset`/`db:push`). |
| `DB_*`                 | PostgreSQL connection (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`). |
| `JWT_SECRET`           | Required in production, at least 32 random characters (`openssl rand -hex 32`). |
| `JWT_ACCESS_EXPIRE_HOURS`, `JWT_REFRESH_EXPIRE_DAYS` | Token lifetimes (default 1 hour and 7 days). |
| `TRUSTED_PROXIES`      | Proxy IPs/CIDRs allowed to set `X-Forwarded-For`. Set this behind a load balancer. |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins. Empty = `*` in development, none in production. |
| `SWAGGER_ENABLED`      | Expose `/swagger`. Defaults to `true` outside production. |
| `ADMIN_PASSWORD`       | Only for `make db:create-admin` in scripts/CI; otherwise the password is prompted. |
