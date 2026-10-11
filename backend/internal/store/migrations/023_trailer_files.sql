-- Trailer files fetched by the optional trailer-resolver sidecar (Teasers feed): one row per YouTube key.
-- status is queued, running, ready or failed; permanent = 1 means retrying is pointless (private or removed video).

CREATE TABLE trailer_files (
    youtube_key  VARCHAR(191) NOT NULL PRIMARY KEY,
    status       VARCHAR(16)  NOT NULL,
    bytes        BIGINT       NOT NULL DEFAULT 0,
    height       INT          NOT NULL DEFAULT 0,
    duration     INT          NOT NULL DEFAULT 0,
    error        VARCHAR(255) NOT NULL DEFAULT '',
    permanent    INT          NOT NULL DEFAULT 0,
    attempts     INT          NOT NULL DEFAULT 0,
    requested_at BIGINT       NOT NULL DEFAULT 0,
    ready_at     BIGINT       NOT NULL DEFAULT 0,
    last_used_at BIGINT       NOT NULL DEFAULT 0,
    next_try_at  BIGINT       NOT NULL DEFAULT 0
);

CREATE INDEX idx_trailer_files_status ON trailer_files (status);
