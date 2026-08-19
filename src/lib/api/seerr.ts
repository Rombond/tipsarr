// Seerr API client — session-cookie auth (Jellyfin login sets a Seerr session cookie).
// Requests go through the /seerr-api dev/preview proxy (see vite.config.ts) since
// Seerr does not send CORS headers and can't be called directly from the browser.
// credentials: 'include' is required on every call so the session cookie rides along
// through the proxy.

const BASE = '/seerr-api';
const TMDB_IMAGE_BASE = import.meta.env.VITE_TMDB_IMAGE_BASE || 'https://image.tmdb.org/t/p/original';

async function seerrFetch(path: string, options: RequestInit = {}) {
	const r = await fetch(`${BASE}/api/v1${path}`, {
		...options,
		credentials: 'include',
		headers: {
			...(options.body ? { 'Content-Type': 'application/json' } : {}),
			...(options.headers || {}),
		},
	});
	if (!r.ok) {
		const body = await r.json().catch(() => null);
		throw new Error(body?.message || `${r.status} ${r.statusText}`);
	}
	if (r.status === 204) return null;
	return r.json();
}

// --- Auth ---

export function loginJellyfin(username: string, password: string) {
	return seerrFetch('/auth/jellyfin', {
		method: 'POST',
		body: JSON.stringify({ username, password }),
	});
}

export function getMe() {
	return seerrFetch('/auth/me');
}

export function logoutSeerr() {
	return seerrFetch('/auth/logout', { method: 'POST' });
}

// --- Discover ---

export function getTrending(page = 1) {
	return seerrFetch(`/discover/trending?page=${page}&mediaType=all`);
}

export function getDiscoverMovies(page = 1) {
	return seerrFetch(`/discover/movies?page=${page}&sortBy=popularity.desc`);
}

export function getDiscoverTv(page = 1) {
	return seerrFetch(`/discover/tv?page=${page}&sortBy=popularity.desc`);
}

export function getMovieDetails(tmdbId: number) {
	return seerrFetch(`/movie/${tmdbId}`);
}

export function getTvDetails(tmdbId: number) {
	return seerrFetch(`/tv/${tmdbId}`);
}

export function getMovieRecommendations(tmdbId: number, page = 1) {
	return seerrFetch(`/movie/${tmdbId}/recommendations?page=${page}`);
}

export function getTvRecommendations(tmdbId: number, page = 1) {
	return seerrFetch(`/tv/${tmdbId}/recommendations?page=${page}`);
}

export function getCollection(collectionId: number) {
	return seerrFetch(`/collection/${collectionId}`);
}

export function search(query: string, page = 1) {
	return seerrFetch(`/search?query=${encodeURIComponent(query)}&page=${page}`);
}

export function getPersonDetails(personId: number) {
	return seerrFetch(`/person/${personId}`);
}

export function getPersonCombinedCredits(personId: number) {
	return seerrFetch(`/person/${personId}/combined_credits`);
}

export function posterUrl(posterPath?: string | null) {
	return posterPath ? `${TMDB_IMAGE_BASE}${posterPath}` : null;
}

// --- Requests ---

export function getRequests(params: { take?: number; skip?: number; filter?: string; requestedBy?: number } = {}) {
	const query = new URLSearchParams();
	if (params.take) query.set('take', String(params.take));
	if (params.skip) query.set('skip', String(params.skip));
	if (params.filter) query.set('filter', params.filter);
	if (params.requestedBy) query.set('requestedBy', String(params.requestedBy));
	query.set('sort', 'added');
	return seerrFetch(`/request?${query.toString()}`);
}

export function createRequest(payload: {
	mediaType: 'movie' | 'tv';
	mediaId: number;
	seasons?: number[] | 'all';
	is4k?: boolean;
}) {
	return seerrFetch('/request', { method: 'POST', body: JSON.stringify(payload) });
}

export function updateRequestStatus(requestId: number, status: 'approve' | 'decline') {
	return seerrFetch(`/request/${requestId}/${status}`, { method: 'POST' });
}

export function retryRequest(requestId: number) {
	return seerrFetch(`/request/${requestId}/retry`, { method: 'POST' });
}

export function deleteRequest(requestId: number) {
	return seerrFetch(`/request/${requestId}`, { method: 'DELETE' });
}

// --- Users ---

export function getUsers() {
	return seerrFetch('/user?take=100');
}

export function getUser(userId: number) {
	return seerrFetch(`/user/${userId}`);
}

export function getUserPermissions(userId: number) {
	return seerrFetch(`/user/${userId}/settings/permissions`);
}

export function setUserPermissions(userId: number, permissions: number) {
	return seerrFetch(`/user/${userId}/settings/permissions`, {
		method: 'POST',
		body: JSON.stringify({ permissions }),
	});
}
