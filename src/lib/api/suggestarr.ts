// SuggestArr API client — JWT via Authorization header + httpOnly refresh cookie
// Endpoints: /api/auth/*, /api/jobs/suggestions, /api/jobs/suggestions/{action}, /api/automation/requests/workflow/*

const BASE = import.meta.env.VITE_SUGGESTARR_URL || 'http://localhost:8080/api';

let accessToken: string | null = null;

export function getAccessToken() {
  return accessToken;
}

export function setAccessToken(token: string | null) {
  accessToken = token;
}

async function readJsonOrThrow(r: Response) {
  if (!r.ok) {
    const body = await r.json().catch(() => null);
    throw new Error(body?.error || `${r.status} ${r.statusText}`);
  }
  return r.json();
}

async function refreshAccessToken() {
  const r = await fetch(`${BASE}/auth/refresh`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!r.ok) {
    accessToken = null;
    return null;
  }
  const data = await r.json();
  accessToken = data.access_token;
  return accessToken;
}

function withAuthHeaders(options: RequestInit): RequestInit {
  return {
    ...options,
    credentials: 'include',
    headers: {
      ...(options.headers || {}),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    },
  };
}

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  let r = await fetch(url, withAuthHeaders(options));
  if (r.status === 401) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      r = await fetch(url, withAuthHeaders(options));
    }
  }
  return readJsonOrThrow(r);
}

export function getAuthStatus() {
  return fetch(`${BASE}/auth/status`, { credentials: 'include' }).then((r) => r.json());
}

export async function login(username: string, password: string) {
  const r = await fetch(`${BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password }),
  });
  const data = await readJsonOrThrow(r);
  accessToken = data.access_token;
  return data;
}

export async function logout() {
  try {
    await fetchWithAuth(`${BASE}/auth/logout`, { method: 'POST' });
  } finally {
    accessToken = null;
  }
}

export function register(username: string, password: string) {
  return fetch(`${BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password }),
  }).then(readJsonOrThrow);
}

export function getAuthMe() {
  return fetchWithAuth(`${BASE}/auth/me`);
}

export function getJellyfinLinkCandidates() {
  return fetchWithAuth(`${BASE}/users/me/link/jellyfin/users`);
}

export function linkJellyfin(externalUserId: string, externalUsername: string) {
  return fetchWithAuth(`${BASE}/users/me/link/jellyfin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ external_user_id: externalUserId, external_username: externalUsername }),
  });
}

export function getSuggestarrUsers() {
  return fetchWithAuth(`${BASE}/users`);
}

export function updateUserPermissions(id: number, patch: Record<string, unknown>) {
  return fetchWithAuth(`${BASE}/users/${id}/permissions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  });
}

export function getSuggestions() {
  return fetchWithAuth(`${BASE}/jobs/suggestions`).then((data) => data.items ?? []);
}

export function retrySuggestions(ids) {
  return fetchWithAuth(`${BASE}/jobs/suggestions/retry`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
}

export function blacklistSuggestions(ids) {
  return fetchWithAuth(`${BASE}/jobs/suggestions/blacklist`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
}

export function getWorkflowSuggestions(status = 'awaiting_approval') {
  return fetchWithAuth(`${BASE}/automation/requests/workflow?status=${status}`);
}

export function approveWorkflowSuggestions(ids) {
  return fetchWithAuth(`${BASE}/automation/requests/workflow/approve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
}

export function rejectWorkflowSuggestions(ids) {
  return fetchWithAuth(`${BASE}/automation/requests/workflow/reject`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  });
}
