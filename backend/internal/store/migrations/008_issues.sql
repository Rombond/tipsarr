-- Problems reported on a title (bad video, missing audio/subtitles...), with a comment thread.

CREATE TABLE issues (
    id             VARCHAR(32) NOT NULL PRIMARY KEY,
    media_type     VARCHAR(8) NOT NULL,
    tmdb_id        BIGINT NOT NULL,
    title          VARCHAR(255) NOT NULL,
    poster_path    VARCHAR(255) NOT NULL,
    kind           VARCHAR(12) NOT NULL,
    season_number  INT NOT NULL,
    episode_number INT NOT NULL,
    status         VARCHAR(12) NOT NULL,
    created_by     VARCHAR(64) NOT NULL,
    resolved_by    VARCHAR(64) NOT NULL,
    created_at     BIGINT NOT NULL,
    updated_at     BIGINT NOT NULL
);

CREATE INDEX idx_issues_status ON issues (status, created_at);
CREATE INDEX idx_issues_created_by ON issues (created_by);
CREATE INDEX idx_issues_media ON issues (media_type, tmdb_id);

CREATE TABLE issue_comments (
    id         VARCHAR(32) NOT NULL PRIMARY KEY,
    issue_id   VARCHAR(32) NOT NULL,
    user_id    VARCHAR(64) NOT NULL,
    message    TEXT NOT NULL,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_issue_comments_issue ON issue_comments (issue_id, created_at);
