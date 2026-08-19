// BoxArr API client — no auth required
// Endpoints: /api/boxoffice/current, /api/movies/{id}, /api/movies/status, /api/weeks, /api/scheduler/trigger

const BASE = import.meta.env.VITE_BOXARR_URL || 'http://localhost:8080/api';

export function getBoxOfficeCurrent() {
  return fetch(`${BASE}/boxoffice/current`).then((r) => r.json());
}

export function getMovie(id) {
  return fetch(`${BASE}/movies/${id}`).then((r) => r.json());
}

export function getMovieStatus(movieIds) {
  return fetch(`${BASE}/movies/status`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ movie_ids: movieIds }),
  }).then((r) => r.json());
}

export function getWeeks() {
  return fetch(`${BASE}/weeks`).then((r) => r.json());
}

export function triggerRefresh() {
  return fetch(`${BASE}/scheduler/trigger`, { method: 'POST' }).then((r) => r.json());
}
