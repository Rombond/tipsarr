// Session store: the backend owns the session cookie; this only mirrors "who am I".
import { api, unwrap, type User } from '$lib/api/client';

class AuthState {
	user: User | null = $state(null);
	status: 'unknown' | 'checking' | 'authenticated' | 'unauthenticated' = $state('unknown');
	/** false until setup has been completed (Jellyfin URL saved). */
	configured: boolean | null = $state(null);
	dryRun = $state(false);
	/** non-admins may choose the quality profile and folder when requesting */
	userRequestOptions = $state(false);
	/** the setup call needs the token from the server log (opt-in on the server) */
	setupTokenRequired = $state(false);

	get isAdmin() {
		return this.user?.role === 'admin';
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
			this.userRequestOptions = st.userRequestOptions;
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
