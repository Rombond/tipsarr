-- What Jellyfin has (TMDB id <-> Jellyfin item), who watched what, and background job state.

CREATE TABLE library_items (
    media_type  VARCHAR(8) NOT NULL,
    tmdb_id     BIGINT NOT NULL,
    jellyfin_id VARCHAR(64) NOT NULL,
    title       VARCHAR(255) NOT NULL,
    PRIMARY KEY (media_type, tmdb_id)
);

CREATE INDEX idx_library_items_jellyfin_id ON library_items (jellyfin_id);

CREATE TABLE library_seasons (
    tmdb_id       BIGINT NOT NULL,
    season_number INT NOT NULL,
    episode_count INT NOT NULL,
    PRIMARY KEY (tmdb_id, season_number)
);

CREATE TABLE watch_history (
    user_id        VARCHAR(64) NOT NULL,
    media_type     VARCHAR(8) NOT NULL,
    tmdb_id        BIGINT NOT NULL,
    last_played_at BIGINT NOT NULL,
    play_count     INT NOT NULL,
    PRIMARY KEY (user_id, media_type, tmdb_id)
);

CREATE INDEX idx_watch_history_recent ON watch_history (user_id, last_played_at);

-- Bumped whenever a user's synced history actually changed; suggestions use it to
-- know when to recompute.
CREATE TABLE user_history_state (
    user_id    VARCHAR(64) NOT NULL PRIMARY KEY,
    version    BIGINT NOT NULL,
    hash       VARCHAR(64) NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE job_runs (
    name             VARCHAR(64) NOT NULL PRIMARY KEY,
    last_started_at  BIGINT NOT NULL,
    last_finished_at BIGINT NOT NULL,
    status           VARCHAR(16) NOT NULL,
    message          TEXT NOT NULL
);
