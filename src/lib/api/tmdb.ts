// TMDB API client — fallback poster lookup when Radarr has no match
// v3 key auth via query string. Free tier: 40 req/10s.

const API_URL = import.meta.env.VITE_TMDB_API_URL || 'https://api.themoviedb.org/3';
const IMAGE_BASE = import.meta.env.VITE_TMDB_IMAGE_BASE || 'https://image.tmdb.org/t/p/original';
const API_KEY = import.meta.env.VITE_TMDB_API_KEY;

export async function searchTmdbMoviePoster(title: string, year?: number) {
  if (!API_KEY) return null;

  const query = year ? `${title} ${year}` : title;
  const url = `${API_URL}/search/movie?query=${encodeURIComponent(query)}&api_key=${API_KEY}`;

  let r: Response;
  try {
    r = await fetch(url);
  } catch {
    return null;
  }

  if (!r.ok) return null; // 401 invalid key, 404 not found, 429 rate limited — all treated as no poster

  const data = await r.json();
  const posterPath = data.results?.[0]?.poster_path;
  return posterPath ? `${IMAGE_BASE}${posterPath}` : null;
}
