-- Status history of a request (one row per step, with when and by whom), so the timeline shows a date on every step.

CREATE TABLE request_events (
    id         VARCHAR(32)  NOT NULL PRIMARY KEY,
    request_id VARCHAR(32)  NOT NULL,
    kind       VARCHAR(16)  NOT NULL,
    actor_id   VARCHAR(191) NOT NULL,
    note       TEXT         NOT NULL,
    at         BIGINT       NOT NULL
);

CREATE INDEX idx_request_events_request ON request_events (request_id, at);
