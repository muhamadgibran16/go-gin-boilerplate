-- +goose Up
-- IF NOT EXISTS keeps this compatible with databases previously created by GORM AutoMigrate.
CREATE TABLE IF NOT EXISTS users (
    id         uuid PRIMARY KEY,
    name       varchar(255) NOT NULL,
    email      varchar(255) NOT NULL,
    password   varchar(255) NOT NULL,
    role       varchar(50)  NOT NULL DEFAULT 'user',
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS users;
