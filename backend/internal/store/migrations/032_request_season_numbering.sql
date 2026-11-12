-- Numbering of a show request's seasons: 'tvdb' (Sonarr's, what Tipsarr uses now) or 'tmdb' (requests made before,
-- translated or refused when they are sent). Movies keep ''.
ALTER TABLE requests ADD COLUMN season_numbering VARCHAR(8) NOT NULL DEFAULT '';

-- Sent or imported requests already reached Sonarr with these numbers: they are Sonarr's.
UPDATE requests SET season_numbering = 'tvdb' WHERE media_type = 'tv' AND (sent_at > 0 OR source <> '');
UPDATE requests SET season_numbering = 'tmdb' WHERE media_type = 'tv' AND season_numbering = '';
