// Pending-request counter for the navigation badge (admins), kept fresh by SSE.
import { api, unwrap } from '$lib/api/client';
import { onEvent, stream } from '$lib/events.svelte';

class Counts {
	pending = $state(0);

	async refresh() {
		try {
			this.pending = (await unwrap(api.GET('/requests/counts'))).pending;
		} catch {
			/* not critical */
		}
	}

	/** Start tracking; returns a stop function. Call from an $effect. */
	track(): () => void {
		this.refresh();
		void stream.reconnects;
		return onEvent('request.updated', () => this.refresh());
	}
}

export const counts = new Counts();
