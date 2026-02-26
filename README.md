# Go Gin Boilerplate

Go boilerplate project using [Gin](https://github.com/gin-gonic/gin) framework with clean architecture, complete authentication, and maximum security.

## Main Features

- **Authentication**: Register, Login, Logout (JWT based).
- **User CRUD**: User management with pagination meta.
- **Security**:
  - Password hashing with `bcrypt`.
  - JWT v5 tokens (Access & Refresh tokens).
  - UUID v4 for Primary Keys (to avoid ID enumeration).
  - JWT middleware for protected routes.
- **Database**: PostgreSQL with GORM ORM (Auto-migration).
- **Architecture**: Clean Architecture (Handler, Service, Repository, Model).
- **DevOps**: Integrated Docker & Docker Compose.
- **Rate Limiting**: Global rate limiting (100 req/min/IP) to prevent abuse.

## Tech Stack

| Package                                                | Version | Usage             |
| ------------------------------------------------------ | ------- | ----------------- |
| [gin-gonic/gin](https://github.com/gin-gonic/gin)      | v1.11+  | HTTP Framework    |
| [gorm.io/gorm](https://github.com/gorm.io/gorm)        | v1.25+  | ORM               |
| [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) | v5.0+   | JWT Token         |
| [google/uuid](https://github.com/google/uuid)          | v1.6+   | UUID Generator    |
| [uber-go/zap](https://github.com/uber-go/zap)          | v1.27+  | Structured logger |

## Project Structure

```
go-gin-boilerplate/
├── cmd/api/main.go                    # Entry point
├── config/config.go                   # Load environment config
├── database/database.go               # GORM Postgres connection
├── internal/
│   ├── handler/                       # HTTP Handlers (auth, user, health)
│   ├── middleware/                   # Middlewares (auth, cors, logger, recovery)
│   ├── model/                         # GORM Models (user)
│   ├── repository/                    # Data access layer
│   ├── routes/routes.go               # Route registration
│   ├── server/server.go               # Gin engine & server setup
│   └── service/                       # Business logic layer
├── pkg/
│   ├── response/response.go           # Standard JSON response helper
│   └── security/                      # JWT & Password helper
├── .env.example
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## Running the Project

### 1. Prerequisites

- Go 1.22+
- Docker & Docker Compose

### 2. Steps

```bash
# Copy environment file
cp .env.example .env

# Run infrastructure (PostgreSQL)
docker-compose up -d

# Run application
make run
# or
go run cmd/api/main.go
```

## API Endpoints

### Public Endpoints

| Method | Endpoint                | Description   |
| ------ | ----------------------- | ------------- |
| GET    | `/`                     | Hello World   |
| GET    | `/health`               | Health check  |
| POST   | `/api/v1/auth/register` | Register user |
| POST   | `/api/v1/auth/login`    | Login user    |

### Protected Endpoints (Header: `Authorization: Bearer <token>`)

| Method | Endpoint               | Description            |
| ------ | ---------------------- | ---------------------- |
| GET    | `/swagger/*any`        | API Documentation UI   |
| GET    | `/api/v1/users`        | List users (paginated) |
| GET    | `/api/v1/users/:id`    | Get user detail        |
| PUT    | `/api/v1/users/:id`    | Update user            |
| DELETE | `/api/v1/users/:id`    | Delete user            |
| POST   | `/api/v1/users/logout` | Logout                 |

## Generating Documentation

If you add new endpoints or change existing ones, update the Swagger docs:

```bash
make swag
```

## Environment Variables

See the [.env.example](.env.example) file for the full list of required environment variables (DB, JWT, App Settings).

# go-gin-boilerplate
