-- Phones and tablets that receive push notifications. One row per app session, so signing a
-- session out (or letting it expire) removes the device; the token itself is never shown by the API.

CREATE TABLE devices (
    id          VARCHAR(32) NOT NULL PRIMARY KEY,
    user_id     VARCHAR(64) NOT NULL,
    session_id  VARCHAR(191) NOT NULL,
    platform    VARCHAR(16) NOT NULL,
    push_token  VARCHAR(1024) NOT NULL,
    sandbox     INT NOT NULL,
    app_version VARCHAR(32) NOT NULL,
    language    VARCHAR(16) NOT NULL,
    categories  INT NOT NULL,
    created_at  BIGINT NOT NULL,
    updated_at  BIGINT NOT NULL
);

CREATE UNIQUE INDEX idx_devices_session ON devices (session_id);
CREATE INDEX idx_devices_user ON devices (user_id);
