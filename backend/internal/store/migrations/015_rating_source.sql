-- Which score a person sees on posters (tmdb, imdb, metacritic or rottenTomatoes); '' = TMDB.

ALTER TABLE users ADD COLUMN rating_source VARCHAR(16) NOT NULL DEFAULT '';
