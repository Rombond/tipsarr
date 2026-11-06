-- Whose watch history a person's recommendations are built from, besides their own (an admin sets it, see the plan
-- Account Mapping). One row per (person, source account).

CREATE TABLE history_sources (
    user_id        VARCHAR(191) NOT NULL,
    source_user_id VARCHAR(191) NOT NULL,
    created_at     BIGINT       NOT NULL,
    PRIMARY KEY (user_id, source_user_id)
);

CREATE INDEX idx_history_sources_source ON history_sources (source_user_id);
