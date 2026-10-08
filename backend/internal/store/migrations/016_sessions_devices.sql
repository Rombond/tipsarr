-- Which device a session belongs to (web browser, iOS app, Android app) and when it was last used.

ALTER TABLE sessions ADD COLUMN platform VARCHAR(16) NOT NULL DEFAULT 'web';
ALTER TABLE sessions ADD COLUMN device_name VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN app_version VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN last_seen_at BIGINT NOT NULL DEFAULT 0;

UPDATE sessions SET last_seen_at = created_at;
