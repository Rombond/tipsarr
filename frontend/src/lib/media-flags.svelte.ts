// Per-person flags on a title (watchlist, hidden), shared by the detail page and the Discover popup.
import { SvelteSet } from 'svelte/reactivity';
import { t } from '$lib/i18n/index.svelte';
import { api, unwrap, expectOk, errorText } from '$lib/api/client';
import { toast } from '$lib/toast.svelte';
import { auth } from '$lib/stores/auth.svelte';

type MediaType = 'movie' | 'tv';

// keyed by person too: signing out and in as someone else keeps the page (no reload), not their hidden titles
const key = (type: MediaType, tmdbId: number) => `${auth.user?.id ?? ''}:${type}:${tmdbId}`;

/** Titles hidden during this session: Discover rows drop them right away instead of waiting for a reload. */
export const hiddenTitles = new SvelteSet<string>();
export const isHidden = (item: { type: MediaType; tmdbId: number }) => hiddenTitles.has(key(item.type, item.tmdbId));

export class MediaFlags {
	watchlisted = $state(false);
	blocklisted = $state(false);
	busy = $state(false);
	#type: MediaType;
	#tmdbId: number;

	constructor(type: MediaType, tmdbId: number) {
		this.#type = type;
		this.#tmdbId = tmdbId;
	}

	/** Reads the current flags; on failure they stay off. */
	async load() {
		try {
			const f = await unwrap(api.GET('/media/{type}/{id}/flags', { params: { path: { type: this.#type, id: this.#tmdbId } } }));
			this.watchlisted = f.watchlisted;
			this.blocklisted = f.blocklisted;
		} catch {
			// flags are a nicety: the buttons still work from the default
		}
	}

	/** Flips one flag and toasts the result. Resolves to true when the server accepted it. */
	async toggle(kind: 'watchlist' | 'blocklist'): Promise<boolean> {
		const type = this.#type;
		const id = this.#tmdbId;
		this.busy = true;
		try {
			if (kind === 'watchlist') {
				const on = this.watchlisted;
				if (on) await expectOk(api.DELETE('/watchlist/{type}/{id}', { params: { path: { type, id } } }));
				else await expectOk(api.POST('/watchlist', { body: { type, tmdbId: id } }));
				this.watchlisted = !on;
				toast.success(t(on ? 'actions.toast_watch_removed' : 'actions.toast_watch_added'));
			} else {
				const on = this.blocklisted;
				if (on) await expectOk(api.DELETE('/blocklist/{type}/{id}', { params: { path: { type, id } } }));
				else await expectOk(api.POST('/blocklist', { body: { type, tmdbId: id } }));
				this.blocklisted = !on;
				if (on) hiddenTitles.delete(key(type, id));
				else hiddenTitles.add(key(type, id));
				toast.success(t(on ? 'actions.toast_unhidden' : 'actions.toast_hidden'));
			}
			return true;
		} catch (e) {
			toast.error(errorText(e));
			return false;
		} finally {
			this.busy = false;
		}
	}
}
