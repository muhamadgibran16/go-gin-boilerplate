# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries: the API server and the migration CLI
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/migrate ./cmd/migrate

# Final Stage
FROM alpine:3.22

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -H -u 10001 app

# Copy binaries from builder
COPY --from=builder /bin/server /bin/migrate ./

# Configuration (including JWT_SECRET) must be provided at runtime via environment
# variables, never baked into the image.
# Migrations are embedded in ./migrate and run explicitly, e.g. `./migrate up`.

USER app

EXPOSE 8080

CMD ["./server"]
