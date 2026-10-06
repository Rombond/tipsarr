-- Portable SQL: must run unchanged on SQLite, PostgreSQL and MySQL.
-- No AUTOINCREMENT / SERIAL / vendor types; timestamps are unix seconds (BIGINT).

CREATE TABLE users (
    id            VARCHAR(64) NOT NULL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    role          VARCHAR(16) NOT NULL,
    region        VARCHAR(8) NOT NULL DEFAULT '',
    language      VARCHAR(16) NOT NULL DEFAULT '',
    created_at    BIGINT NOT NULL,
    last_login_at BIGINT NOT NULL
);

CREATE TABLE sessions (
    id         VARCHAR(64) NOT NULL PRIMARY KEY,
    user_id    VARCHAR(64) NOT NULL,
    user_agent VARCHAR(255) NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE settings (
    skey   VARCHAR(191) NOT NULL PRIMARY KEY,
    svalue TEXT NOT NULL
);
