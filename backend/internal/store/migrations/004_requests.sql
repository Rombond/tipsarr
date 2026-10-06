-- Radarr/Sonarr instances, requests, outgoing webhooks.

CREATE TABLE servarr_instances (
    id                 VARCHAR(32) NOT NULL PRIMARY KEY,
    kind               VARCHAR(8) NOT NULL,
    name               VARCHAR(100) NOT NULL,
    url                VARCHAR(512) NOT NULL,
    api_key            VARCHAR(255) NOT NULL,
    quality_profile_id INT NOT NULL,
    root_folder        VARCHAR(512) NOT NULL,
    is_default         INT NOT NULL,
    created_at         BIGINT NOT NULL
);

CREATE TABLE requests (
    id             VARCHAR(32) NOT NULL PRIMARY KEY,
    media_type     VARCHAR(8) NOT NULL,
    tmdb_id        BIGINT NOT NULL,
    title          VARCHAR(255) NOT NULL,
    poster_path    VARCHAR(255) NOT NULL,
    release_date   VARCHAR(16) NOT NULL,
    requested_by   VARCHAR(64) NOT NULL,
    status         VARCHAR(16) NOT NULL,
    decided_by     VARCHAR(64) NOT NULL,
    decline_reason TEXT NOT NULL,
    instance_id    VARCHAR(32) NOT NULL,
    servarr_id     BIGINT NOT NULL,
    sent_at        BIGINT NOT NULL,
    dry_run        INT NOT NULL,
    error          TEXT NOT NULL,
    created_at     BIGINT NOT NULL,
    updated_at     BIGINT NOT NULL
);

CREATE INDEX idx_requests_status ON requests (status);
CREATE INDEX idx_requests_requested_by ON requests (requested_by);
CREATE INDEX idx_requests_media ON requests (media_type, tmdb_id);

CREATE TABLE request_seasons (
    request_id    VARCHAR(32) NOT NULL,
    season_number INT NOT NULL,
    PRIMARY KEY (request_id, season_number)
);

CREATE TABLE webhooks (
    id         VARCHAR(32) NOT NULL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    url        VARCHAR(1024) NOT NULL,
    secret     VARCHAR(255) NOT NULL,
    events     VARCHAR(255) NOT NULL,
    enabled    INT NOT NULL,
    created_at BIGINT NOT NULL
);
