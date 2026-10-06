-- Manual TMDB matches for box-office titles the automatic search gets wrong.
CREATE TABLE boxoffice_aliases (
    title_key  VARCHAR(255) NOT NULL PRIMARY KEY,
    tmdb_id    BIGINT NOT NULL,
    created_at BIGINT NOT NULL
);
