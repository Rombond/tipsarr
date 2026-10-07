-- What the Library page sorts and filters on, read from Jellyfin by the library sync.
-- rating10 is the community rating times 10 (78 = 7.8); genres is '|Action|Comedy|' so a plain LIKE works on every engine.

ALTER TABLE library_items ADD COLUMN year INT NOT NULL DEFAULT 0;
ALTER TABLE library_items ADD COLUMN runtime_min INT NOT NULL DEFAULT 0;
ALTER TABLE library_items ADD COLUMN rating10 INT NOT NULL DEFAULT 0;
ALTER TABLE library_items ADD COLUMN added_at BIGINT NOT NULL DEFAULT 0;
ALTER TABLE library_items ADD COLUMN genres VARCHAR(500) NOT NULL DEFAULT '';
ALTER TABLE library_items ADD COLUMN image_tag VARCHAR(64) NOT NULL DEFAULT '';
