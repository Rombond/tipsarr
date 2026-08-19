// Radarr API v3 client — direct browser calls (LAN-only; API key ships in bundle)
// Endpoints: /api/v3/system/status, /api/v3/movie, /api/v3/movie/{id}, /api/v3/movie/lookup

const BASE = import.meta.env.VITE_RADARR_URL || 'http://localhost:7878';
const API_KEY = import.meta.env.VITE_RADARR_API_KEY;

function radarrFetch(path: string) {
  return fetch(`${BASE}${path}`, {
    headers: { 'X-Api-Key': API_KEY },
  }).then((r) => {
    if (!r.ok) throw new Error(`${r.status} ${r.statusText}`);
    return r.json();
  });
}

export function getRadarrMovie(radarrId: number) {
  return radarrFetch(`/api/v3/movie/${radarrId}`);
}

export function lookupRadarrMovie(term: string) {
  return radarrFetch(`/api/v3/movie/lookup?term=${encodeURIComponent(term)}`);
}

export function getRadarrPosterUrl(movie: { images?: Array<{ coverType: string; remoteUrl?: string; url?: string }> }) {
  const poster = movie.images?.find((i) => i.coverType === 'poster');
  return poster?.remoteUrl || poster?.url || null;
}
