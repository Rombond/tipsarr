// Session store: the backend owns the session cookie; this only mirrors "who am I".
import { api, unwrap, type User } from '$lib/api/client';
import { i18n } from '$lib/i18n/index.svelte';
import { P, has } from '$lib/permissions';

/** true when tag a (v1.2.3) is a later release than b; false if either is not a version. */
export function isNewer(a: string, b: string): boolean {
	const parse = (v: string) => /^v?(\d+)\.(\d+)\.(\d+)$/.exec(v.trim())?.slice(1).map(Number);
	const x = parse(a);
	const y = parse(b);
	if (!x || !y) return false;
	for (let i = 0; i < 3; i++) if (x[i] !== y[i]) return x[i] > y[i];
	return false;
}

class AuthState {
	user: User | null = $state(null);
	status: 'unknown' | 'checking' | 'authenticated' | 'unauthenticated' = $state('unknown');
	/** false until setup has been completed (Jellyfin URL saved). */
	configured: boolean | null = $state(null);
	dryRun = $state(false);
	/** a throwaway public demo sandbox: one-click sign-in, banner */
	demo = $state(false);
	/** the server release ("dev" for a local build) and the newest one on GitHub ("" while unknown) */
	version = $state('');
	latestVersion = $state('');
	/** bumped when a picture changes so browsers refetch it */
	avatarV = $state(0);
	/** the setup call needs the token from the server log (opt-in on the server) */
	setupTokenRequired = $state(false);

	/** The role: admins can do everything. For what a person may do, use the can* getters. */
	get isAdmin() {
		return this.user?.role === 'admin';
	}

	/** What the person may do: the sum of their permission flags (everything for an admin). */
	get permissions() {
		return this.user?.effectivePermissions ?? 0;
	}

	can(...flags: number[]) {
		return has(this.permissions, ...flags);
	}

	/** may request this kind of title */
	mayRequest(type: 'movie' | 'tv') {
		return this.can(P.request) || this.can(type === 'movie' ? P.requestMovie : P.requestTV);
	}

	get canManageSettings() {
		return this.can(P.manageSettings);
	}
	get canManageUsers() {
		return this.can(P.manageUsers);
	}
	get canManageRequests() {
		return this.can(P.manageRequests);
	}
	/** sees everyone's requests */
	get canViewRequests() {
		return this.can(P.requestView);
	}
	get canViewIssues() {
		return this.can(P.viewIssues);
	}
	get canManageIssues() {
		return this.can(P.manageIssues);
	}
	get canCreateIssues() {
		return this.can(P.createIssues);
	}
	/** any admin-area page: users or settings */
	get canAdminister() {
		return this.canManageUsers || this.canManageSettings;
	}

	/** a newer release than the running one is published (never for a dev build) */
	get updateAvailable() {
		return isNewer(this.latestVersion, this.version);
	}

	get username() {
		return this.user?.name ?? null;
	}

	async bootstrap() {
		if (this.status !== 'unknown') return;
		this.status = 'checking';
		try {
			const [setup, st] = await Promise.all([
				unwrap(api.GET('/setup/status')),
				unwrap(api.GET('/status')),
			]);
			this.configured = setup.configured;
			this.setupTokenRequired = setup.tokenRequired;
			this.dryRun = st.dryRun;
			this.demo = st.demo;
			this.version = st.version;
			this.latestVersion = st.latestVersion;
			i18n.setAppDefault(st.defaultLanguage);
		} catch {
			this.configured = null;
		}
		try {
			this.user = await unwrap(api.GET('/me'));
			this.status = 'authenticated';
		} catch {
			this.user = null;
			this.status = 'unauthenticated';
		}
	}

	async login(username: string, password: string) {
		this.user = await unwrap(api.POST('/auth/login', { body: { username, password } }));
		this.status = 'authenticated';
	}

	/** demo only: sign in as the seeded admin or plain user */
	async demoLogin(role: 'admin' | 'user') {
		this.user = await unwrap(api.POST('/auth/demo', { body: { role } }));
		this.status = 'authenticated';
	}

	async logout() {
		try {
			await api.POST('/auth/logout');
		} finally {
			this.user = null;
			this.status = 'unauthenticated';
		}
	}

	markConfigured() {
		this.configured = true;
	}
}

export const auth = new AuthState();
