-- Requests mirrored from what Radarr/Sonarr already monitor ('' = made by a user in Tipsarr).

ALTER TABLE requests ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT '';
