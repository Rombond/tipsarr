CREATE TABLE tmdb_cache (
    ckey       VARCHAR(191) NOT NULL PRIMARY KEY,
    body       TEXT NOT NULL,
    fetched_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL
);

CREATE INDEX idx_tmdb_cache_expires_at ON tmdb_cache (expires_at);
