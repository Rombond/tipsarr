// Unified session store — merges Seerr (session cookie) and SuggestArr (JWT) identities.
// Seerr's Jellyfin login is the primary gate; SuggestArr is a secondary integration that
// degrades gracefully (suggestions simply unavailable) if it can't be provisioned.

import * as seerr from '$lib/api/seerr';
import * as suggestarr from '$lib/api/suggestarr';

// Overseerr/Jellyseerr permission bitmask (from upstream permission.ts) — ADMIN and
// MANAGE_USERS are the only bits this app currently checks.
const PERMISSION_ADMIN = 2;
const PERMISSION_MANAGE_USERS = 8;

type SeerrUser = { id: number; username?: string; jellyfinUsername?: string; permissions: number };
type SuggestarrUser = { id: number; username: string; role: string };

class AuthState {
	seerrUser: SeerrUser | null = $state(null);
	suggestarrUser: SuggestarrUser | null = $state(null);
	status: 'unknown' | 'checking' | 'authenticated' | 'unauthenticated' = $state('unknown');
	error: string | null = $state(null);
	suggestarrError: string | null = $state(null);

	get isAdmin() {
		const seerrAdmin =
			!!this.seerrUser &&
			((this.seerrUser.permissions & PERMISSION_ADMIN) !== 0 ||
				(this.seerrUser.permissions & PERMISSION_MANAGE_USERS) !== 0);
		const suggestarrAdmin = this.suggestarrUser?.role === 'admin';
		return seerrAdmin || suggestarrAdmin;
	}

	get username() {
		return this.seerrUser?.username || this.seerrUser?.jellyfinUsername || this.suggestarrUser?.username || null;
	}

	get suggestarrLinked() {
		return this.suggestarrUser !== null;
	}

	async bootstrap() {
		this.status = 'checking';
		try {
			this.seerrUser = await seerr.getMe();
		} catch {
			this.seerrUser = null;
		}
		try {
			this.suggestarrUser = await suggestarr.getAuthMe();
		} catch {
			this.suggestarrUser = null;
		}
		this.status = this.seerrUser ? 'authenticated' : 'unauthenticated';
	}

	async login(username: string, password: string) {
		this.error = null;
		this.suggestarrError = null;
		this.status = 'checking';

		// 1. Seerr/Jellyfin is the primary gate — real credential check.
		await seerr.loginJellyfin(username, password);
		this.seerrUser = await seerr.getMe();

		// 2. SuggestArr is best-effort: log in, auto-register + link if needed.
		try {
			await this.loginOrProvisionSuggestarr(username, password);
		} catch (e) {
			this.suggestarrUser = null;
			this.suggestarrError = (e as Error).message;
		}

		this.status = 'authenticated';
	}

	private async loginOrProvisionSuggestarr(username: string, password: string) {
		try {
			await suggestarr.login(username, password);
		} catch {
			await suggestarr.register(username, password);
			await suggestarr.login(username, password);
			try {
				const candidates = await suggestarr.getJellyfinLinkCandidates();
				const match = (candidates?.items || candidates || []).find(
					(u: any) => (u.name || u.username || '').toLowerCase() === username.toLowerCase(),
				);
				if (match) {
					await suggestarr.linkJellyfin(match.id || match.external_user_id, match.name || match.username);
				}
			} catch {
				// Linking is a nice-to-have — ignore failures.
			}
		}
		this.suggestarrUser = await suggestarr.getAuthMe();
	}

	async logout() {
		try {
			await seerr.logoutSeerr();
		} catch {
			// ignore
		}
		try {
			await suggestarr.logout();
		} catch {
			// ignore
		}
		this.seerrUser = null;
		this.suggestarrUser = null;
		this.status = 'unauthenticated';
	}
}

export const auth = new AuthState();
