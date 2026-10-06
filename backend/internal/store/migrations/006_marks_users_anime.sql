-- Watchlist / blocklist per user, and a separate root folder for anime on Sonarr.

CREATE TABLE user_marks (
    user_id      VARCHAR(64) NOT NULL,
    kind         VARCHAR(12) NOT NULL,
    media_type   VARCHAR(8) NOT NULL,
    tmdb_id      BIGINT NOT NULL,
    title        VARCHAR(255) NOT NULL,
    poster_path  VARCHAR(255) NOT NULL,
    release_date VARCHAR(16) NOT NULL,
    vote_tenths  INT NOT NULL,
    created_at   BIGINT NOT NULL,
    PRIMARY KEY (user_id, kind, media_type, tmdb_id)
);

CREATE INDEX idx_user_marks_list ON user_marks (user_id, kind, created_at);

ALTER TABLE servarr_instances ADD COLUMN anime_root VARCHAR(512) NOT NULL DEFAULT '';
