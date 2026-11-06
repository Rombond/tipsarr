-- The address people open Radarr/Sonarr at in their browser (the url column is the one Tipsarr itself uses, often a
-- container name). Empty = fall back to url.
ALTER TABLE servarr_instances ADD COLUMN public_url VARCHAR(512) NOT NULL DEFAULT '';
