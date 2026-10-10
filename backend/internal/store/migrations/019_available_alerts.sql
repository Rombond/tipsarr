-- "Now available" alerts wait until the library shows the title, so they need to remember when a request
-- turned available and whether its alert went out. Requests that are already available count as alerted.

ALTER TABLE requests ADD COLUMN available_at BIGINT NOT NULL DEFAULT 0;
ALTER TABLE requests ADD COLUMN notified_at BIGINT NOT NULL DEFAULT 0;
UPDATE requests SET available_at = updated_at, notified_at = updated_at WHERE status = 'available';

-- Shows: each requested season can be announced when it is complete, before the whole request is.
ALTER TABLE request_seasons ADD COLUMN done_at BIGINT NOT NULL DEFAULT 0;
ALTER TABLE request_seasons ADD COLUMN notified_at BIGINT NOT NULL DEFAULT 0;
