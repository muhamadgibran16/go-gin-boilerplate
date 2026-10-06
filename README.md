# Go Gin Boilerplate

REST API boilerplate in Go with [Gin](https://github.com/gin-gonic/gin), organized by feature modules,
with JWT authentication, role-based access, versioned migrations and secure defaults.

## Contents

- [Features](#features)
- [Quick start](#quick-start)
- [API](#api)
- [Project structure](#project-structure)
- [Development guide](#development-guide)
- [Database migrations](#database-migrations)
- [Testing](#testing)
- [Configuration](#configuration)
- [Deploying to production](#deploying-to-production)
- [Known limitations](#known-limitations)
- [Tech stack](#tech-stack)

## Features

- **Authentication**: register, login, refresh and logout with JWT. Access and refresh tokens are distinct
  (a refresh token cannot be used as a Bearer token), and refresh reloads the user from the database, so
  deleted or demoted users cannot get new tokens.
- **Users**: admin-only CRUD with pagination, sorting and search. The first admin is created with
  `make db:create-admin`.
- **Security**: bcrypt passwords, UUID primary keys, layered rate limiting, login lockout per email,
  1 MiB request body limit, 8s request timeout, security headers, CORS allow-list, trusted proxies,
  and a startup check that refuses a weak `JWT_SECRET` in production. Internal errors are logged but
  never shown to clients.
- **Database**: PostgreSQL with GORM and versioned SQL migrations ([goose](https://github.com/pressly/goose)),
  run explicitly with Prisma-like `make db:*` commands, never on startup.
- **Developer experience**: one JSON format for every response, validation messages using JSON field names,
  request-scoped logs with `request_id` / `user_id`, Swagger docs, tests and Docker.

## Quick start

Requirements: Go 1.25+, Docker and Docker Compose.

### Run locally (PostgreSQL in Docker, API with Go)

```bash
cp .env.example .env
docker compose up -d db                                      # PostgreSQL only
make db:setup                                                # create the database + apply migrations
make db:create-admin email=admin@example.com name="Admin"    # first admin, password is prompted
make run                                                     # http://localhost:8080
```

### Run everything in Docker

```bash
cp .env.example .env
docker compose up -d db
docker compose run --rm api ./migrate up                                         # apply migrations
docker compose run --rm -it api ./migrate create-admin -email admin@example.com  # first admin
docker compose up -d --build api
```

Swagger UI: http://localhost:8080/swagger/index.html (enabled by default outside production).

### Make targets

| Target                    | Description                                                     |
| ------------------------- | --------------------------------------------------------------- |
| `make run`                | Run the API                                                     |
| `make build`              | Build `bin/go-gin-boilerplate` and `bin/migrate`                |
| `make test`               | Run all tests with the race detector (`make test-cover` for coverage) |
| `make vet`, `make tidy`   | Run `go vet` / `go mod tidy`                                    |
| `make swag`               | Regenerate the Swagger docs after changing handler annotations  |
| `make db:*`               | Database commands, see [Database migrations](#database-migrations) |
| `make help`               | List every target                                               |

## API

### Endpoints

| Method | Endpoint                      | Auth  | Description |
| ------ | ----------------------------- | ----- | ----------- |
| GET    | `/health`, `/api/v1/health`   | -     | Liveness: the process is up |
| GET    | `/health/ready`               | -     | Readiness: pings the database, 503 if it is unreachable |
| POST   | `/api/v1/auth/register`       | -     | Create an account with the `user` role |
| POST   | `/api/v1/auth/login`          | -     | Returns an access token, a refresh token and the user |
| POST   | `/api/v1/auth/refresh`        | -     | Returns a new access token for a refresh token |
| POST   | `/api/v1/auth/logout`         | user  | Logout (stateless, see [Known limitations](#known-limitations)) |
| GET    | `/api/v1/users`               | admin | List users, see the query parameters below |
| GET    | `/api/v1/users/:id`           | admin | Get a user |
| PUT    | `/api/v1/users/:id`           | admin | Update a user's `name` and/or `role` (`admin`, `user`) |
| DELETE | `/api/v1/users/:id`           | admin | Soft delete a user |

Authenticated endpoints need the header `Authorization: Bearer <access token>`.
Admins cannot change their own role or delete their own account.

**List query parameters** (`GET /api/v1/users`):

| Parameter | Default      | Description |
| --------- | ------------ | ----------- |
| `page`    | `1`          | Page number |
| `perPage` | `10`         | Items per page, at most 100 (above is rejected with 400) |
| `sort`    | `-createdAt` | Comma-separated fields, `-` prefix for descending: `name`, `email`, `role`, `createdAt`, `updatedAt` |
| `search`  | -            | Case-insensitive search in name and email |

Example: `GET /api/v1/users?page=2&perPage=20&sort=name,-createdAt&search=bob`

**Validation rules**: name is required (at most 255 characters, at least 3 when updating, surrounding spaces
are trimmed); email must be valid; password needs at least 6 characters and at most 72 bytes (bcrypt limit;
non-ASCII characters count as several bytes).

### Response format

Every response, including errors, unknown routes (404) and wrong methods (405), uses this JSON format.

Success:

```json
{
    "status": "success",
    "message": "Get user successfully",
    "data": { "id": "…", "name": "Bob", "email": "bob@example.com", "role": "user", "createdAt": "…", "updatedAt": "…" }
}
```

List:

```json
{
    "status": "success",
    "message": "Get users successfully",
    "data": [ … ],
    "meta": { "currentPage": 1, "perPage": 10, "totalCurrentPage": 10, "totalPage": 2, "totalData": 12 }
}
```

Error (`errors` is only present for validation errors, keyed by JSON field name):

```json
{ "status": "error", "message": "Validation failed", "errors": { "email": "email must be a valid email address" } }
```

### Status codes

| Status | When |
| ------ | ---- |
| 400 | Invalid JSON, query string or path parameter; validation failed (`errors` lists the fields) |
| 401 | Missing, invalid or expired token; wrong email or password |
| 403 | Missing role (e.g. not an admin), or an admin changing their own role / deleting their own account |
| 404 | Resource or route not found |
| 405 | Known route called with the wrong method |
| 409 | Email already registered |
| 413 | Request body larger than 1 MiB |
| 429 | Rate limit exceeded, or too many failed logins for an email |
| 500 | Unexpected error. The client only gets `"Internal server error"`; the details are logged with the `request_id` |
| 503 | Database unreachable, or the request exceeded the 8s timeout |

### Rate limits

Every limit uses a key the client cannot change:

| Scope                        | Key                   | Limit |
| ---------------------------- | --------------------- | ----- |
| Every request                | IP                    | 300 / minute (flood protection) |
| Register and login           | IP                    | 20 / minute, shared |
| Authenticated endpoints      | User ID from the JWT  | 100 / minute, so users behind one IP do not share a quota |
| Failed logins                | Email                 | 5 per 15 minutes, then the email is locked (even for the right password) until the window ends |

Limits are constants in `internal/app/app.go`. The client IP is read from `X-Forwarded-For` only when the
request comes from a proxy listed in `TRUSTED_PROXIES`.

## Project structure

Code is organized **by feature**: each module in `internal/modules` owns its handler, service, repository,
model and DTOs. `internal/app` only wires everything together.

```
go-gin-boilerplate/
├── cmd/
│   ├── api/main.go                 # API server entry point
│   └── migrate/main.go             # Database CLI used by `make db:*`
├── internal/
│   ├── app/                        # Composition root: wiring, routes, limits, http.Server
│   ├── config/                     # Environment configuration
│   ├── database/
│   │   ├── database.go             # GORM connection, CREATE DATABASE
│   │   ├── migrate.go              # Migration provider (embedded SQL files)
│   │   ├── tx.go                   # Transactions carried in context (Transactor, Conn)
│   │   └── migrations/             # Versioned SQL migrations (goose)
│   ├── httpx/                      # Shared HTTP layer: JSON responses, binding and validation messages,
│   │                               # ErrorHandler, middleware (logger, recovery, request ID, CORS,
│   │                               # security headers, rate limits, body limit, timeout), current user
│   ├── pkg/                        # Generic utilities shared by many modules
│   │   ├── apperror/               # Typed application errors (not found, conflict, ...)
│   │   ├── ctxlog/                 # Request-scoped logger carried in context.Context
│   │   ├── pagination/             # page/perPage binding, pagination.Find[T], response meta
│   │   ├── ratelimit/              # Failure counter (failed logins per email)
│   │   ├── search/                 # Case-insensitive search over several columns
│   │   └── sorting/                # Safe ?sort=-createdAt,name with a field whitelist
│   └── modules/
│       ├── modules.go              # Models() registry used by `make db:push`
│       ├── auth/                   # Register, login, refresh, JWT, passwords, auth middleware, RequireRoles
│       ├── user/                   # User model, repository, service, handler, DTOs
│       └── health/                 # Liveness and readiness checks
├── docs/                           # Generated Swagger docs (`make swag`)
├── .env.example
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

Dependency direction: `cmd → app → modules → httpx / database / config / pkg`.
Modules never import `app`; `auth` imports `user` (it needs the user model), never the other way around.

### Where to put code

| Code                                              | Location              |
| ------------------------------------------------- | --------------------- |
| Used by a single module                           | Inside that module    |
| HTTP related and shared (responses, middleware)   | `internal/httpx`      |
| Database related and shared                       | `internal/database`   |
| Generic utility shared by many modules            | `internal/pkg/<name>` |

Rules for `internal/pkg`:

- One sub-package per concern, named after what it provides (`pagination`, `password`, `stringx`, ...).
  No catch-all `utils` package.
- It must not import `internal/modules` or `internal/app`.
- Start a helper inside the module that needs it, and move it to `internal/pkg` once a second module needs it.

## Development guide

### Adding a new module

1. Create `internal/modules/<name>/` with `model.go`, `dto.go`, `repository.go`, `service.go` and `handler.go`
   (see `internal/modules/user` as the reference).
2. Register the model in `internal/modules/modules.go`.
3. Wire the repository, service and handler in `internal/app/app.go`, and add the routes in `internal/app/routes.go`.
4. Create the table: `make db:create name=create_<name>_table`, write the SQL, then `make db:migrate`.
5. Add Swagger annotations to the handlers and run `make swag`.
6. Write service tests with a fake repository (see [Testing](#testing)).

### Handlers and errors

Services return `apperror` errors for expected failures; any other error is treated as internal.
Handlers pass errors to `httpx.Abort`, and `httpx.ErrorHandler` writes the response:

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
httpx.Success(c, "Post updated successfully", NewPostResponse(post))
```

| apperror                       | Status |
| ------------------------------ | ------ |
| `BadRequest`, `Validation`     | 400    |
| `Unauthorized`                 | 401    |
| `Forbidden`                    | 403    |
| `NotFound`                     | 404    |
| `Conflict`                     | 409    |
| `PayloadTooLarge`              | 413    |
| `TooManyRequests`              | 429    |
| `Unavailable`                  | 503    |
| `Internal` or any other error  | 500, message hidden from the client and the error logged |

Always pass `c.Request.Context()` down to the repository: it carries the 8s request deadline (queries are
cancelled when it expires), the transaction and the request logger.

### List endpoints

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
var q ListQuery
if !httpx.BindQuery(c, &q) {
	return
}
posts, total, err := h.service.List(c.Request.Context(), q)
...
httpx.SuccessPaginated(c, "Get posts successfully", NewPostResponses(posts), pagination.NewMeta(q.Query, len(posts), total))
```

- Add `page`, `perPage`, `sort` and `search` to the route's `httpx.ValidateQueryParams` whitelist.
- Order with `sorting.Apply` or `db.Order`, never through `db.Scopes`: GORM would keep the `ORDER BY` in the
  count query, which PostgreSQL rejects.
- `sorting.Apply` always appends `id` as the last sort column, so rows with equal values keep a stable order
  and never repeat or disappear across pages.
- The defaults (`page=1`, `perPage=10`) live only in `pagination.DefaultPage` / `DefaultPerPage`.

### Transactions

Repositories get their handle with `database.Conn(ctx, r.db)`, so they join a transaction started by a service:

```go
err := s.tx.WithinTx(ctx, func(ctx context.Context) error { // s.tx is a *database.Transactor
	if err := s.orders.Create(ctx, order); err != nil {
		return err // rollback
	}
	return s.items.CreateMany(ctx, items)
}) // commit
```

### Logging

Logs are human-readable in development and structured JSON in production (`APP_ENV=production`).
Every request writes an access log entry with its `request_id`.

`ctxlog.From(ctx)` returns a logger that already carries `request_id`, and `user_id` once authenticated,
so logs written by services can be traced back to their request:

```go
ctxlog.From(ctx).Info("user created", zap.String("created_user_id", u.ID.String()))
```

## Database migrations

Migrations are plain SQL files in `internal/database/migrations`, embedded into the binaries. They are
**never applied when the app starts**; the app only logs a warning when migrations are pending.

| Command                           | Prisma equivalent                   | Description |
| --------------------------------- | ----------------------------------- | ----------- |
| `make db:init`                    | (done by `migrate dev`)             | Create the database `DB_NAME` if it does not exist (safe to re-run) |
| `make db:setup`                   | -                                   | `db:init` + `db:migrate`, for a fresh environment |
| `make db:migrate`                 | `prisma migrate deploy`             | Apply all pending migrations |
| `make db:rollback`                | -                                   | Roll back the last applied migration |
| `make db:status`                  | `prisma migrate status`             | Show applied and pending migrations |
| `make db:create name=add_posts`   | `prisma migrate dev --create-only`  | Create a new empty SQL migration file |
| `make db:reset`                   | `prisma migrate reset`              | Drop all tables and re-apply every migration (asks for confirmation; `force=1` skips it) |
| `make db:push`                    | `prisma db push`                    | Sync the schema from the GORM models (`modules.Models()`) without a migration file |
| `make db:create-admin email=...`  | -                                   | Create an admin user (`name="..."` optional). The password is prompted without echo, or read from `ADMIN_PASSWORD` |

Workflow for a schema change:

```bash
make db:create name=add_phone_to_users   # creates internal/database/migrations/0000N_add_phone_to_users.sql
# write the SQL under "-- +goose Up" and its reverse under "-- +goose Down"
make db:migrate
make db:rollback && make db:migrate      # optional: check that Down works
```

Notes:

- SQL is not generated from the models (unlike Prisma); write it by hand.
- `db:push` is for prototyping only. Before committing, write a migration and run `make db:reset` to check that
  the migrations alone produce the schema.
- Never edit a migration that was already applied in a shared environment; add a new one.
- `db:reset` and `db:push` refuse to run when `APP_ENV=production`.
- `db:init` runs `CREATE DATABASE` from the server's default `postgres` database, so `DB_USER` needs the
  `CREATEDB` privilege (`ALTER ROLE <user> CREATEDB;`).
- In Docker, use the `migrate` binary shipped in the image: `docker compose run --rm api ./migrate up`
  (also `down`, `status`, `create-admin`).

## Testing

```bash
make test
```

Tests live in the feature modules and cover the code with the most logic:

| File                                        | Covers |
| ------------------------------------------- | ------ |
| `internal/modules/auth/service_test.go`     | Register, login, refresh, login lockout, password and name rules |
| `internal/modules/auth/middleware_test.go`  | Bearer token checks (a refresh token is rejected), roles |
| `internal/modules/user/service_test.go`     | Admin rules, update, delete |
| `internal/modules/user/handler_test.go`     | HTTP status for each error, validation messages, pagination defaults, no internal error leaks |
| `internal/modules/user/repository_test.go`  | SQL of the list query: sorting, search, pagination, stable order, count without `ORDER BY` |

Services are tested with in-memory fake repositories, and the repository test builds SQL with GORM's dry-run
mode, so no database is needed. Shared code in `internal/httpx` and `internal/pkg` is covered through these
module tests. When adding a module, write service tests for its logic, and a handler test when it maps
errors or validates input in a specific way.

## Configuration

Settings are read from environment variables, or from `.env` (see [.env.example](.env.example)).

| Variable                   | Default                     | Description |
| -------------------------- | --------------------------- | ----------- |
| `APP_ENV`                  | `development`               | `production` enables stricter checks and JSON logs, and disables Swagger, `db:reset` and `db:push` |
| `APP_PORT`                 | `8080`                      | HTTP port |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | `localhost`, `5432`, `postgres`, `postgres`, `go_gin_boilerplate`, `disable` | PostgreSQL connection |
| `JWT_SECRET`               | (insecure placeholder)      | Signing key. In production, at least 32 random characters (`openssl rand -hex 32`) or the app refuses to start |
| `JWT_ACCESS_EXPIRE_HOURS`  | `1`                         | Access token lifetime |
| `JWT_REFRESH_EXPIRE_DAYS`  | `7`                         | Refresh token lifetime |
| `TRUSTED_PROXIES`          | (none)                      | Comma-separated proxy IPs/CIDRs allowed to set `X-Forwarded-For` |
| `CORS_ALLOWED_ORIGINS`     | `*` in development, none in production | Comma-separated browser origins |
| `SWAGGER_ENABLED`          | `true` outside production   | Expose `/swagger` |
| `ADMIN_PASSWORD`           | -                           | Only for `make db:create-admin` in scripts/CI; otherwise the password is prompted |

## Deploying to production

- Set `APP_ENV=production` and a random `JWT_SECRET` of at least 32 characters.
- Set `CORS_ALLOWED_ORIGINS` to your frontend origins, and `TRUSTED_PROXIES` to your load balancer or reverse
  proxy. Without it, every request looks like it comes from the proxy and shares one rate limit quota.
- Use `DB_SSLMODE=require` (or stricter) when the database is not on a private network.
- Run `./migrate up` before starting the new version, e.g. as a release step or an init container.
- Create the first admin once with `./migrate create-admin -email ...`.
- Use `/health` for liveness and `/health/ready` for readiness probes.
- Read the [known limitations](#known-limitations) before running more than one instance.

## Known limitations

- **Logout does not revoke tokens.** Tokens are stateless: an access token stays valid until it expires (1 hour by
  default), even after logout, a role change or deletion. Revoking them requires a token blacklist (e.g. in Redis).
- **Rate limit counters are in memory.** With several instances, each one counts separately, so the effective
  limits are multiplied. Use a shared store such as Redis.
- **Login lockout can be triggered by others.** Anyone who knows an email can lock it for 15 minutes by failing
  5 logins. This is the usual trade-off against brute force; a CAPTCHA after a few failures is an alternative.
- **Deleted users' emails cannot be reused.** The unique index on `email` also covers soft-deleted users.

## Tech stack

| Package                                                     | Version | Usage                    |
| ----------------------------------------------------------- | ------- | ------------------------ |
| [gin-gonic/gin](https://github.com/gin-gonic/gin)           | v1.11   | HTTP framework           |
| [gorm.io/gorm](https://github.com/go-gorm/gorm)             | v1.31   | ORM (PostgreSQL via pgx) |
| [pressly/goose](https://github.com/pressly/goose)           | v3.27   | SQL migrations           |
| [golang-jwt/jwt](https://github.com/golang-jwt/jwt)         | v5.3    | JWT                      |
| [ulule/limiter](https://github.com/ulule/limiter)           | v3.11   | Rate limiting            |
| [uber-go/zap](https://github.com/uber-go/zap)               | v1.27   | Structured logging       |
| [swaggo/swag](https://github.com/swaggo/swag)               | v1.16   | Swagger docs generation  |
| [google/uuid](https://github.com/google/uuid)               | v1.6    | UUID primary keys        |
