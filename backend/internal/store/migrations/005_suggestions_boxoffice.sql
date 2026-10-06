-- Per-user suggestion rows (Netflix-style), the weekly box-office chart, and genre root folders.

CREATE TABLE suggestion_rows (
    id              VARCHAR(32) NOT NULL PRIMARY KEY,
    user_id         VARCHAR(64) NOT NULL,
    kind            VARCHAR(16) NOT NULL,
    seed_type       VARCHAR(8) NOT NULL,
    seed_tmdb_id    BIGINT NOT NULL,
    seed_title      VARCHAR(255) NOT NULL,
    position        INT NOT NULL,
    generated_at    BIGINT NOT NULL,
    history_version BIGINT NOT NULL,
    personal        INT NOT NULL
);

CREATE INDEX idx_suggestion_rows_user ON suggestion_rows (user_id);

-- Title snapshots so reading a row needs no TMDB call; availability is applied at read time.
CREATE TABLE suggestion_items (
    row_id        VARCHAR(32) NOT NULL,
    pos           INT NOT NULL,
    media_type    VARCHAR(8) NOT NULL,
    tmdb_id       BIGINT NOT NULL,
    title         VARCHAR(255) NOT NULL,
    poster_path   VARCHAR(255) NOT NULL,
    release_date  VARCHAR(16) NOT NULL,
    vote_tenths   INT NOT NULL,
    overview      TEXT NOT NULL,
    PRIMARY KEY (row_id, pos)
);

CREATE TABLE boxoffice_weeks (
    region     VARCHAR(8) NOT NULL,
    week_key   VARCHAR(12) NOT NULL,
    label      VARCHAR(64) NOT NULL,
    fetched_at BIGINT NOT NULL,
    PRIMARY KEY (region, week_key)
);

CREATE TABLE boxoffice_entries (
    region           VARCHAR(8) NOT NULL,
    week_key         VARCHAR(12) NOT NULL,
    pos              INT NOT NULL,
    title            VARCHAR(255) NOT NULL,
    weekend_gross    BIGINT NOT NULL,
    total_gross      BIGINT NOT NULL,
    weeks_in_release INT NOT NULL,
    tmdb_id          BIGINT NOT NULL,
    poster_path      VARCHAR(255) NOT NULL,
    release_date     VARCHAR(16) NOT NULL,
    vote_tenths      INT NOT NULL,
    overview         TEXT NOT NULL,
    PRIMARY KEY (region, week_key, pos)
);

-- JSON object {"<tmdb genre id>": "<root folder>"}; first matching genre of a movie wins.
ALTER TABLE servarr_instances ADD COLUMN genre_roots VARCHAR(4000) NOT NULL DEFAULT '';
