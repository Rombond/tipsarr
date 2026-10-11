-- Titles a person already swiped past in the Teasers feed, so the feed never repeats them (forgotten after 60 days).

CREATE TABLE teaser_seen (
    user_id    VARCHAR(191) NOT NULL,
    media_type VARCHAR(16)  NOT NULL,
    tmdb_id    BIGINT       NOT NULL,
    seen_at    BIGINT       NOT NULL,
    PRIMARY KEY (user_id, media_type, tmdb_id)
);
