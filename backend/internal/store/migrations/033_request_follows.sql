-- Requests followed from a phone (Live Activity on iOS, ongoing notification on Android), one row per session
-- and request. The push token is the Live Activity token (iOS) or the FCM token (Android).
CREATE TABLE request_follows (
    id           VARCHAR(32)   NOT NULL PRIMARY KEY,
    request_id   VARCHAR(32)   NOT NULL,
    user_id      VARCHAR(64)   NOT NULL,
    session_id   VARCHAR(191)  NOT NULL,
    platform     VARCHAR(16)   NOT NULL,
    push_token   VARCHAR(1024) NOT NULL,
    sandbox      INT           NOT NULL,
    server       VARCHAR(255)  NOT NULL,
    activity_id  VARCHAR(64)   NOT NULL,
    last_stage   VARCHAR(16)   NOT NULL,
    last_percent INT           NOT NULL,
    last_sent_at BIGINT        NOT NULL,
    created_at   BIGINT        NOT NULL,
    expires_at   BIGINT        NOT NULL
);
CREATE UNIQUE INDEX idx_request_follows_session_request ON request_follows (session_id, request_id);
CREATE INDEX idx_request_follows_request ON request_follows (request_id);
