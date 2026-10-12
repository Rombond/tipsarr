-- The trailers the server always keeps ready for the Teasers feed: about 75 movies and 25 shows per language and region.
-- scope is "<language>|<region>" as saved on the users; item is the JSON of the title as the lists show it.

CREATE TABLE teaser_pool (
    scope       VARCHAR(64)  NOT NULL,
    media_type  VARCHAR(8)   NOT NULL,
    tmdb_id     BIGINT       NOT NULL,
    youtube_key VARCHAR(191) NOT NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    item        TEXT         NOT NULL,
    updated_at  BIGINT       NOT NULL DEFAULT 0,
    PRIMARY KEY (scope, media_type, tmdb_id)
);
