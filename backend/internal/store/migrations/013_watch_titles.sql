-- Titles that were played in Jellyfin but are no longer in the library (removed by clean-up tools):
-- looked up once on TMDB so the Stats page can still show their poster, year and genres.
-- tmdb_id 0 = no match found (tried again after a week).

CREATE TABLE watch_titles (
    media_type VARCHAR(8) NOT NULL,
    title      VARCHAR(191) NOT NULL,
    tmdb_id    BIGINT NOT NULL,
    poster     VARCHAR(255) NOT NULL,
    year       INT NOT NULL,
    rating10   INT NOT NULL,
    genres     VARCHAR(500) NOT NULL,
    checked_at BIGINT NOT NULL,
    PRIMARY KEY (media_type, title)
);
