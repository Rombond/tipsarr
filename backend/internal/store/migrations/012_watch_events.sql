-- One row per play session, copied from Jellyfin's Playback Reporting plugin (optional).
-- source_rowid is the plugin's own rowid, so the next sync only asks for newer rows.
-- tmdb_id is the movie, or the show an episode belongs to (0 when it is not in the library).

CREATE TABLE watch_events (
    source_rowid BIGINT NOT NULL PRIMARY KEY,
    user_id      VARCHAR(64) NOT NULL,
    jellyfin_id  VARCHAR(64) NOT NULL,
    media_type   VARCHAR(8) NOT NULL,
    tmdb_id      BIGINT NOT NULL,
    title        VARCHAR(255) NOT NULL,
    played_at    BIGINT NOT NULL,
    seconds      INT NOT NULL
);

CREATE INDEX idx_watch_events_user ON watch_events (user_id, played_at);
